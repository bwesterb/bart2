package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

func main() {
	fmt.Println("testing")
	c, err := ChipiOpen()
	if err != nil {
		fmt.Println("could not open Chipi: ", err)
		return
	}
	go watchChipi(c)
	time.Sleep(time.Second)
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	_ = <-ch
	c.Close()
	time.Sleep(2 * time.Second)
}

func watchChipi(c *Chipi) {
	for {
		select {
		case err := <-c.Err:
			fmt.Printf("chipi error: %v\n", err)
		case report := <-c.Reports:
			fmt.Printf("received report: %s\n", report.String())
		}
	}
}
