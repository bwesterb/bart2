package main

import (
	"fmt"
)

func ExampleMuxiMsg_String() {
	msg := MuxiMsg{Chip: 1, Bits: "101"}
	fmt.Printf("%s", msg)
	// Output: 101@1
}

func ExampleMuxiMsg_ReadFromBuf() {
	msg := &MuxiMsg{}
	msg.readFrom([]byte{188, 237, 6})
	fmt.Printf("%s", msg)
	// Output: 101101110110000@0
}
