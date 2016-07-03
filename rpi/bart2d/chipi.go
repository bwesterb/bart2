package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ourThermistor returns the type of thermistor we use in our Bar T2.
func ourThermistor() Thermistor {
	return Thermistor{A: 1.270e-3, B: 2.229e-4, C: 3.948e-8}
}

func ourRMeter() RMeter {
	return RMeter{Resistor: 997}
}

func ourVRatioMeter() VRatioMeter {
	return VRatioMeter{MaxNo: 1023}
}

// Thermistor models a thermistor by its Steinhart--Hart coefficients (A,B,C).
type Thermistor struct {
	A float64
	B float64
	C float64
}

// TempC returns the temperature (in Celsius) at the given resistance.
func (t Thermistor) TempC(R float64) float64 {
	logR := math.Log(R)
	tempKrec := 1.270e-3 + 2.229e-4*logR + 3.948e-8*math.Pow(logR, 3)
	return (1 / tempKrec) - 273.15
}

// RMeter models the way we measure the resistance of the
// thermistor: by measuring the ratio of the voltage  over the thermistor
// to the voltage over the thermistor and a resistor put in series.
type RMeter struct {
	// Resistance of the resistor (in Ohm)
	Resistor float64
}

// R returns the measured resistance at the given voltage ratio.
func (r RMeter) R(ratio float64) float64 {
	return (1/ratio - 1) * r.Resistor
}

// VRatioMeter models the way the ICs measure the ratio of the
// voltage on a pin to the voltage provided: it returns a number
// from 0 to MaxNo; 0 indicates the ratio 0, and MaxNo indicates 1.
type VRatioMeter struct {
	MaxNo uint
}

// Ratio returns the ratio that corresponds to the given number.
func (v VRatioMeter) Ratio(no uint) float64 {
	return float64(no) / float64(v.MaxNo)
}

// ChipiReport models a report on temperature of the boiler (among other
// things) send by the chips.
type ChipiReport struct {
	Time      time.Time
	Chip      byte
	VoltageNo uint
	TempC     float64
	Heating   bool
	OK        bool
	TempLow   bool
	TempHigh  bool
	BuddyDied bool

	// for debug purposes:
	Msg MuxiMsg
}

func (r ChipiReport) String() string {
	return strings.Join(r.toRecord(), " ")
}

const TIME_LAYOUT = "15:04:05.0"

func (rep ChipiReport) toRecord() (rec []string) {
	rec = make([]string, 0, 9)
	rec = append(rec, strconv.FormatFloat(rep.TempC, 'f', 1, 64))
	rec = append(rec, rep.Time.Format(TIME_LAYOUT))
	rec = append(rec, strconv.FormatUint(uint64(rep.Chip), 10))
	rec = append(rec, strconv.FormatUint(uint64(rep.VoltageNo), 10))
	if rep.Heating {
		rec = append(rec, "Heating")
	}
	if rep.OK {
		rec = append(rec, "OK")
	}
	if rep.TempLow {
		rec = append(rec, "TempLow")
	}
	if rep.TempHigh {
		rec = append(rec, "TempHigh")
	}
	if rep.BuddyDied {
		rec = append(rec, "BuddyDied")
	}
	return
}

func (c *Chipi) reportFrom(msg MuxiMsg) (r ChipiReport) {
	r.Time = time.Now()
	r.Chip = msg.Chip
	r.VoltageNo = msg.UintX(0, 10, true) // true = least significant bit first
	r.Heating = msg.Bool(10)
	r.OK = msg.Bool(11)
	r.TempLow = msg.Bool(12)
	r.TempHigh = msg.Bool(13)
	r.BuddyDied = msg.Bool(14)
	r.computeTempC(c)
	r.Msg = msg
	return
}

func (r *ChipiReport) computeTempC(c *Chipi) {
	ratio := c.voltageRatioMeter.Ratio(r.VoltageNo)
	R := c.resistanceMeter.R(ratio)
	r.TempC = c.thermistor.TempC(R)
}

// Chipi is the interface to the two chips which measure the temperature
// of the boiler.
type Chipi struct {
	Reports <-chan ChipiReport
	Err     <-chan error

	thermistor        Thermistor
	resistanceMeter   RMeter
	voltageRatioMeter VRatioMeter
	reports           chan ChipiReport
	err               chan error
	closer            chan bool
	muxi              *Muxi
	out0, out1        chan MuxiMsg
}

// ChipiOpen opens an interface to the chips.
func ChipiOpen() (chipi *Chipi, err error) {
	chipi = &Chipi{
		reports:           make(chan ChipiReport),
		err:               make(chan error),
		closer:            make(chan bool),
		out0:              make(chan MuxiMsg),
		out1:              make(chan MuxiMsg),
		thermistor:        ourThermistor(),
		resistanceMeter:   ourRMeter(),
		voltageRatioMeter: ourVRatioMeter(),
	}
	chipi.Reports = chipi.reports
	chipi.Err = chipi.err
	if chipi.muxi, err = MuxiOpen(); err != nil {
		return
	}
	go chipi.doGetReports(0)
	go chipi.doGetReports(1)
	go chipi.doGetErrors()
	go chipi.doSortMessages()
	return
}

func (chipi *Chipi) Close() error {
	close(chipi.closer)
	chipi.muxi.Close()
	return nil
}

func (chipi *Chipi) doGetReports(chip byte) {
	var out <-chan MuxiMsg
	switch chip {
	case 0:
		out = chipi.out0
	case 1:
		out = chipi.out1
	default:
		chipi.err <- fmt.Errorf("chipi: chip neither 0 nor 1")
		return
	}

outerLoop:
	for {
		chipi.muxi.In <- MuxiMsg{
			Chip: chip,
			Bits: "1",
		}
		response := MuxiMsg{Chip: chip}
		for response.Length() < 16 {
			select {
			case msg := <-out:
				response = MuxiMsgJoin(response, msg)
			case _ = <-time.After(5 * time.Second):
				chipi.err <- fmt.Errorf("chipi: chip %v did not respond\n",
					chip)
				continue outerLoop
			case _ = <-chipi.closer:
				return
			}
		}
		if response.Length() != 16 {
			chipi.err <- fmt.Errorf("chipi: chip %v send a message of size %v",
				chip, response.Length)
			continue outerLoop
		}

		chipi.reports <- chipi.reportFrom(response)
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
			case 0:
				chipi.out0 <- msg
			case 1:
				chipi.out1 <- msg
			default:
				chipi.err <- fmt.Errorf("chipi: chip neither 0 nor 1")
			}
		case _ = <-chipi.closer:
			return
		}
	}
}
