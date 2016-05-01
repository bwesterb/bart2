package main

import (
	"fmt"
	"time"
)

func main() {
    args := SpiMessageArgs{SpeedHz:10e5,BitsPerWord:8}
	fmt.Println("testing")
	d, err := SpiOpen("/dev/spidev0.0", 1, false, 8, 10e6)
	if err != nil {
		panic(fmt.Sprintf("could not open spidev, because", err))
	}
	defer d.Close()
    err = d.Message3([]byte{0, 0}, []byte{133, 1},args)
    if err != nil {
        panic(err)
    }
	time.Sleep(50 * time.Millisecond)
	rbuf := []byte{0, 0, 0, 0, 0}
	err = d.Message3(rbuf, []byte{0, 0, 0, 0, 0},args)
    if err != nil {
        panic(err)
    }
	fmt.Printf("%v\n", rbuf)
}
