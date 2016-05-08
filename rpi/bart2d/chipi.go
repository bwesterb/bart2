package main

import (
	"fmt"
	"time"
)

type ChipiReport struct {
	Time       time.Time
	Chip       byte
	Resistance uint
	Heating    bool
	OK         bool
	TempLow    bool
	TempHigh   bool
	BudyDied   bool
}

func chipiReportFrom(msg MuxiMsg) (r ChipiReport) {
	r.Time = time.Now()
	r.Chip = msg.Chip
	r.Resistance = uint(msg.Data[0])<<2 + uint(msg.Data[1])>>6
	r.Heating = msg.GetBit(10)
	r.OK = msg.GetBit(11)
	r.TempLow = msg.GetBit(12)
	r.TempHigh = msg.GetBit(13)
	r.BudyDied = msg.GetBit(14)
	return
}

type Chipi struct {
	Reports <-chan ChipiReport
	Err     <-chan error

	reports    chan ChipiReport
	err        chan error
	closer     chan bool
	muxi       *Muxi
	out1, out2 chan MuxiMsg
}

func ChipiOpen() (chipi *Chipi, err error) {
	chipi = &Chipi{
		reports: make(chan ChipiReport),
		err:     make(chan error),
		closer:  make(chan bool),
		out1:    make(chan MuxiMsg),
		out2:    make(chan MuxiMsg),
	}
	chipi.Reports = chipi.reports
	chipi.Err = chipi.err
	if chipi.muxi, err = MuxiOpen(); err != nil {
		return
	}
	go chipi.doGetReports(1)
	go chipi.doGetReports(2)
	go chipi.doGetErrors()
	go chipi.doSortMessages()
	return
}

func (chipi *Chipi) Close () error {
    close(chipi.closer)
    chipi.muxi.Close()
    return nil
}

func (chipi *Chipi) doGetReports(chip byte) {
	var out <-chan MuxiMsg
	switch chip {
	case 1:
		out = chipi.out1
	case 2:
		out = chipi.out2
	default:
		chipi.err <- fmt.Errorf("chipi: chip neither 1 nor 2")
		return
	}

outerLoop:
	for {
		chipi.muxi.In <- MuxiMsg{
			Chip:   chip,
			Length: 1,
			Data:   [4]byte{1, 0, 0, 0},
		}
		response := MuxiMsg{Chip: chip}
		for response.Length < 16 {
			select {
			case msg := <-out:
				response = MuxiMsgJoin(response, msg)
            case _ = <-time.After(5*time.Second):
				chipi.err <- fmt.Errorf("chipi: chip %v did not respond\n",
					chip)
				continue outerLoop
			case _ = <-chipi.closer:
				return
			}
		}
		if response.Length != 16 {
			chipi.err <- fmt.Errorf("chipi: chip %v send a message of size %v",
				chip, response.Length)
			continue outerLoop
		}

		chipi.reports <- chipiReportFrom(response)
	}
}

func (chipi *Chipi) doGetErrors() {
	for {
		select {
		case err := <-chipi.muxi.Err:
			chipi.err <- err
		case _ = <-chipi.closer:
			return
		}
	}
}

func (chipi *Chipi) doSortMessages() {
	for {
		select {
		case msg := <-chipi.muxi.Out:
			switch msg.Chip {
			case 1:
				chipi.out1 <- msg
			case 2:
				chipi.out2 <- msg
			default:
				chipi.err <- fmt.Errorf("chipi: chip neither 1 nor 2")
			}
		case _ = <-chipi.closer:
			return
		}
	}
}
