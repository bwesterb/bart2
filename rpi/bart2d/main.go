package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
)

type Bart2d struct {
	dir    Dir
	chipi  *Chipi
	dumper *Dumper
}

func (b *Bart2d) Run() error {
	{
		dir, err := DirOpen()
		if err != nil {
			return err
		}
		b.dir = dir
	}

	{
		chipi, err := ChipiOpen()
		if err != nil {
			return WrapErr(err, "Could not open Chipi")
		}
		b.chipi = chipi
	}

	{
		dumper, err := DumperOpen(b.dir)
		if err != nil {
			return WrapErr(err, "Could not open Dumper")
		}
		b.dumper = dumper
	}
	go b.pump()
	ch := make(chan os.Signal)
	signal.Notify(ch, os.Interrupt)
	_ = <-ch
	if err := b.Close(); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return nil
}

func (b *Bart2d) Close() error {
	err1 := b.chipi.Close()
	err2 := b.dumper.Close()
	return WrapErrs([]error{err1, err2}, "Closing failed")
}

func (b *Bart2d) pump() {
	for {
		select {
		case err := <-b.chipi.Err:
			fmt.Printf("!! chipi error: %v\n", err)
		case report := <-b.chipi.Reports:
			fmt.Printf("%s -- %s\n", report, report.Msg)
			b.dumper.Dump(report)
		}
	}
}

func main() {
	if err := (&Bart2d{}).Run(); err != nil {
		fmt.Println("FATAL ERROR: ", err)
	}
}
