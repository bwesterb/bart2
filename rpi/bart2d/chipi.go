package main

import (
	"fmt"
	"math"
	"time"
)

// ourThermistor returns the type of thermistor we use in our Bar T2.
func ourThermistor() Thermistor {
	return Thermistor{A: 1.270e-3, B: 2.229e-4, C: 3.948e-8}
}

func ourRMeter() RMeter {
	return RMeter{Resistor: 10e3}
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
	return ratio / (1 - ratio) * r.Resistor
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
	BudyDied  bool
}

func (r *ChipiReport) String() string {
	return fmt.Sprintf("Chip %v @ %.1f °C (%v/1023)",
		r.Chip,
		r.TempC,
		r.VoltageNo)
}

func (c *Chipi) reportFrom(msg MuxiMsg) (r ChipiReport) {
	fmt.Println(msg)
	r.Time = time.Now()
	r.Chip = msg.Chip
	r.VoltageNo = uint(msg.Data[0])<<2 + uint(msg.Data[1])>>6
	r.Heating = msg.GetBit(10)
	r.OK = msg.GetBit(11)
	r.TempLow = msg.GetBit(12)
	r.TempHigh = msg.GetBit(13)
	r.BudyDied = msg.GetBit(14)
	r.computeTempC(c)
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
	out1, out2        chan MuxiMsg
}

// ChipiOpen opens an interface to the chips.
func ChipiOpen() (chipi *Chipi, err error) {
	chipi = &Chipi{
		reports:           make(chan ChipiReport),
		err:               make(chan error),
		closer:            make(chan bool),
		out1:              make(chan MuxiMsg),
		out2:              make(chan MuxiMsg),
		thermistor:        ourThermistor(),
		resistanceMeter:   ourRMeter(),
		voltageRatioMeter: ourVRatioMeter(),
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

func (chipi *Chipi) Close() error {
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
			case _ = <-time.After(5 * time.Second):
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
