package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	fmt.Println("testing")
	args := SpiArgs{SpeedHz: 10000}
	f, err := os.Open("/dev/spidev0.0")
	defer f.Close()
	if err != nil {
		panic(fmt.Sprintf("could not open spidev, because", err))
	}
	fmt.Println(SpiWrMode(f, 1))
	// 133 1
	SpiMessage(f, []byte{0, 0}, []byte{133, 1}, args)
	time.Sleep(50 * time.Millisecond)
	rbuf := []byte{0, 0, 0, 0, 0}
	SpiMessage(f, rbuf, []byte{0, 0, 0, 0, 0}, args)
	fmt.Printf("%v\n", rbuf)
}
