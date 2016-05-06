package main

import (
	"bufio"
	"fmt"
	"io"
)

type MuxiMsg struct {
	Data   [4]byte
	Length byte
	Chip   byte
}

func (msg MuxiMsg) String() string {
	return fmt.Sprintf("%v @ %v (length: %v)",
		msg.Data, msg.Chip, msg.Length)
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

type Muxi struct {
	spiDevice      *SpiConfiguredDevice
	receivedWriter *io.PipeWriter
	messageScanner *bufio.Scanner
	out, in        chan MuxiMsg
	closer         chan bool // closed is closed if the muxi is closed
	err            chan error
	rbuf, tbuf     [5]byte
}

func (m *Muxi) Out() <-chan MuxiMsg {
	return m.out
}

func (m *Muxi) In() chan<- MuxiMsg {
	return m.in
}

func (m *Muxi) Err() <-chan error {
	return m.err
}

func MuxiOpen() (muxi *Muxi, err error) {
	spidev, err := SpiOpen("/dev/spidev0.0", 1, false, 8, 10e5)
	if err != nil {
		return
	}

	muxi = &Muxi{
		spiDevice: spidev,
		out:       make(chan MuxiMsg),
		in:        make(chan MuxiMsg),
		closer:    make(chan bool),
		err:       make(chan error),
	}

	reader, writer := io.Pipe()
	muxi.receivedWriter = writer
	muxi.messageScanner = bufio.NewScanner(reader)
	muxi.messageScanner.Split(func(data []byte, atEOF bool) (advance int,
		token []byte, err error) {
		if len(data) == 0 {
			return // 0, nil, nil
		}
		if data[advance] == 0 {
			for ; advance < len(data); advance++ {
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
			return
		}

		token = data[:advance]
		return
	})

	go muxi.doTransmit()
	go muxi.doProcess()
	return
}

func (m *Muxi) Close() error {
	close(m.closer)
	m.receivedWriter.Close()
	return nil
}

func (m *Muxi) doProcess() {
	var msg MuxiMsg
	for {
		msg = MuxiMsg{}
		m.messageScanner.Scan()
		msg.readFrom(m.messageScanner.Bytes())
		m.out <- msg
	}
}

func (m *Muxi) doTransmit() {
	for {
		select {
		case msg := <-m.in:
			if err := m.transmit(msg); err != nil {
				m.err <- err
				return
			}
		case _ = <-m.closer:
			break
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
