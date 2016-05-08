package main

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

type MuxiMsg struct {
	Data   [4]byte
	Length byte
	Chip   byte
}

func (msg MuxiMsg) String() string {
	s := make([]string, 0, 6)
	for i := 0; 8*i < int(msg.Length); i++ {
		l := int(msg.Length) - 8*i
		if l >= 8 {
			l = 8
		}
		format := "%0" + fmt.Sprintf("%d", l) + "b"
		s = append(s, fmt.Sprintf(format, msg.Data[i]))
	}
	s = append(s, "@", fmt.Sprintf("%d", msg.Chip))
	return strings.Join(s, "")
}

func (msg MuxiMsg) Vet() error {
	if msg.Chip == 0 || msg.Chip > 2 {
		return fmt.Errorf("muxi: invalid MuxiMsg: Chip should be 1 or 2.")
	}
	if msg.Length > 31 { // 11111
		return fmt.Errorf("muxi: invalid MuxiMsg: Length should be below 31")
	}
	return nil
}

func (msg MuxiMsg) writeTo(buf []byte) {
	// assume len(buf) >= 5
	copy(buf[1:], msg.Data[:])
	buf[0] = 1<<7 | msg.Length<<2 | msg.Chip
}

func (msg *MuxiMsg) readFrom(buf []byte) error {
	if len(buf) == 0 {
		return fmt.Errorf("invalid frame: no header")
	}
	header := buf[0]
	if header>>7 != 1 {
		return fmt.Errorf("invalid frame: first bit of the header should be 1")
	}
	msg.Chip = header & 3           // pick out 000000xx
	msg.Length = (header >> 2) & 31 // pick out 0xxxxx00

	var numofbytes int = 0
	numofbytes += int(msg.Length) / 8
	if msg.Length%8 > 0 {
		numofbytes++
	}

	if len(buf) != numofbytes+1 {
		return fmt.Errorf("invalid frame: too many or too few bytes")
	}
	copy(msg.Data[:], buf[1:])
	return nil
}

func (msg MuxiMsg) getIdx(i uint) (byteIdx uint, bitIdx uint) {
	byteIdx = i / 8
	var shift uint = 0
	if uint(msg.Length)-byteIdx*8 < 8 {
		shift = 8 - (uint(msg.Length) - byteIdx*8)
	}
	bitIdx = i%8 + shift
	return
}

func (msg *MuxiMsg) SetBit(i uint, value bool) {
	byteIdx, bitIdx := msg.getIdx(i)
	if value {
		msg.Data[byteIdx] |= 1 << (7 - bitIdx)
	} else {
		msg.Data[byteIdx] &= 255 - 1<<(7-bitIdx)
	}
}

func (msg *MuxiMsg) GetBit(i uint) bool {
	byteIdx, bitIdx := msg.getIdx(i)
	return (msg.Data[byteIdx]>>(7-bitIdx))&1 == 1
}

func MuxiMsgJoin(msg1, msg2 MuxiMsg) (res MuxiMsg) {
	// assume msg1.Chip = msg2.Chip
	// assume msg1.Length + msg2.Length <= 31
	res.Chip = msg1.Chip
	res.Length = msg1.Length + msg2.Length
	for i := uint(0); i < uint(msg1.Length); i++ {
		res.SetBit(i, msg1.GetBit(i))
	}
	for i := uint(0); i < uint(msg2.Length); i++ {
		res.SetBit(uint(msg1.Length)+i, msg2.GetBit(i))
	}
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
	spidev, err := SpiOpen("/dev/spidev0.0", 1, false, 8, 10e4)
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
	fmt.Printf("muxi: received %v; transferred %v\n", m.rbuf, m.tbuf)
	if _, err := m.receivedWriter.Write(m.rbuf[:]); err != nil {
		return err
	}
	return nil
}
