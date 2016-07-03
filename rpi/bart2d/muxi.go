package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

type MuxiMsg struct {
	Bits string
	Chip byte
}

func (msg *MuxiMsg) Bool(idx int) bool {
	return msg.Bits[idx] == "1"[0]
}

func (msg *MuxiMsg) Length() int {
	return len(msg.Bits)
}

func (msg *MuxiMsg) UintX(idx, length int, lsbFirst bool) uint {
	return parseBinary(msg.Bits[idx:idx+length], lsbFirst)
}

func parseBinary(text string, lsbFirst bool) (res uint) {
	var idx int
	for i := 0; i < len(text); i++ {
		if lsbFirst {
			idx = len(text) - i - 1
		} else {
			idx = i
		}
		res = res << 1
		if text[idx] == "1"[0] {
			res++
		}
	}
	return
}

func (msg MuxiMsg) String() string {
	return fmt.Sprintf("%v@%v", msg.Bits, msg.Chip)
}

func (msg *MuxiMsg) Vet() error {
	if msg.Chip > 1 {
		return fmt.Errorf("muxi: invalid MuxiMsg: Chip should be 0 or 1.")
	}
	if len(msg.Bits) > 31 { // 11111
		return fmt.Errorf("muxi: invalid MuxiMsg: " +
			"length of Bits should be below 31")
	}
	for i := 0; i < len(msg.Bits); i++ {
		if msg.Bits[i] != "0"[0] && msg.Bits[i] != "1"[0] {
			return fmt.Errorf("muxi: invalid MuxiMsg: Bits should consist" +
				" of only '0' and '1'.")
		}
	}
	return nil
}

func (msg *MuxiMsg) writeTo(buf []byte) {
	// assume len(buf) >= 5
	// write header
	buf[0] = 1<<7 | byte(msg.Length())<<2 | msg.Chip
	// write body
	var byteIdx, bitsLeft int // 0, 0
	for i := 0; i < len(msg.Bits); i++ {
		if bitsLeft == 0 {
			byteIdx++
			bitsLeft = 8
		}
		buf[byteIdx] = buf[byteIdx] << 1
		if msg.Bits[i] == "1"[0] {
			buf[byteIdx]++
		}
		bitsLeft--
	}
}

func (msg *MuxiMsg) readFrom(buf []byte) error {
	// read header
	if len(buf) == 0 {
		return fmt.Errorf("invalid frame: no header")
	}
	header := buf[0]
	if header>>7 != 1 {
		return fmt.Errorf("invalid frame: first bit of the header should be 1")
	}
	msg.Chip = header & 3        // pick out 000000xx
	length := (header >> 2) & 31 // pick out 0xxxxx00

	// read body
	var byteIdx, bitsLeft int // 0, 0
	var curByte byte

	s := make([]string, 0, length)

	for length > 0 {
		if bitsLeft == 0 {
			byteIdx++
			curByte = buf[byteIdx]
			bitsLeft = 8
		}

		var bit string
		switch curByte % 2 {
		case 1:
			bit = "1"
		case 0:
			bit = "0"
		}
		s = append(s, bit)

		length--
		bitsLeft--
		curByte = curByte >> 1
	}
	msg.Bits = strings.Join(s, "")
	return nil
}

func MuxiMsgJoin(msg1, msg2 MuxiMsg) (res MuxiMsg) {
	// assume msg1.Chip = msg2.Chip
	// assume msg1.Length + msg2.Length <= 31
	res.Chip = msg1.Chip
	res.Bits = msg1.Bits + msg2.Bits
	return
}

type Muxi struct {
	Out <-chan MuxiMsg
	In  chan<- MuxiMsg
	Err <-chan error

	spiDevice      *SpiConfiguredDevice
	receivedWriter *io.PipeWriter
	messageScanner *bufio.Scanner
	out, in        chan MuxiMsg
	closer         chan bool // closer is closed if the muxi is closed
	err            chan error
	rbuf, tbuf     [5]byte
	ticker         *time.Ticker
}

func MuxiOpen() (muxi *Muxi, err error) {
	spidev, err := SpiOpen("/dev/spidev0.0", 1, false, 8, 8192)
	if err != nil {
		return
	}

	muxi = &Muxi{
		spiDevice: spidev,
		out:       make(chan MuxiMsg),
		in:        make(chan MuxiMsg),
		closer:    make(chan bool),
		err:       make(chan error),
		ticker:    time.NewTicker(500 * time.Millisecond),
	}

	muxi.Err = muxi.err
	muxi.In = muxi.in
	muxi.Out = muxi.out

	reader, writer := io.Pipe()
	muxi.receivedWriter = writer
	muxi.messageScanner = bufio.NewScanner(reader)
	muxi.messageScanner.Split(func(data []byte, atEOF bool) (advance int,
		token []byte, err error) {
		if len(data) == 0 {
			return // 0, nil, nil
		}
		if data[0] == 0 {
			for ; advance < len(data) && data[advance] == 0; advance++ {
			}
			return // advance, nil, nil
		}

		numofbits := (data[0] >> 2) & 31 // pick out 01111100

		advance += int(numofbits)/8 + 1 // +1 is for the header
		if numofbits%8 > 0 {
			advance++
		}

		if advance > len(data) {
			advance = 0
			return // 0, nil, nil
		}

		token = data[:advance]
		return // advance, <frame>, nil
	})

	go muxi.doTransfer()
	go muxi.doProcess()
	return
}

func (m *Muxi) Close() error {
	close(m.closer)
	return nil
}

func (m *Muxi) doProcess() {
	var msg MuxiMsg
	for {
		msg = MuxiMsg{}
		m.messageScanner.Scan()
		if len(m.messageScanner.Bytes()) == 0 {
			return
		}
		if err := msg.readFrom(m.messageScanner.Bytes()); err != nil {
			m.err <- err
			return
		}
		select {
		case m.out <- msg:
		case _ = <-m.closer:
			return
		}
	}
}

func (m *Muxi) doTransfer() {
	for {
		select {
		case msg := <-m.in:
			if err := m.transmit(msg); err != nil {
				m.err <- err
				return
			}
		case _ = <-m.ticker.C:
			m.tbuf = [5]byte{0, 0, 0, 0, 0}
			if err := m.transfer(); err != nil {
				m.err <- err
				return
			}
		case _ = <-m.closer:
			m.receivedWriter.Close()
			m.ticker.Stop()
			m.spiDevice.Close()
			return
		}
	}
}

func (m *Muxi) transmit(msg MuxiMsg) error {
	if err := msg.Vet(); err != nil {
		return err
	}
	msg.writeTo(m.tbuf[:])
	return m.transfer()
}

func (m *Muxi) transfer() error {
	if err := m.spiDevice.Message(m.rbuf[:], m.tbuf[:]); err != nil {
		return err
	}
	//fmt.Printf("muxi: received %v; transferred %v\n", m.rbuf, m.tbuf)
	if _, err := m.receivedWriter.Write(m.rbuf[:]); err != nil {
		return err
	}
	return nil
}
