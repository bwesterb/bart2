package main

import (
    "fmt"
)

func ExampleMuxiMsg(){
    msg := MuxiMsg{Length:10,Data:[4]byte{255,3}}
    for i:=uint(0); i<uint(msg.Length); i++ {
        byteIdx, bitIdx := msg.getIdx(i)
        fmt.Printf("%v,%v-", byteIdx, bitIdx)
    }

    fmt.Println()
    for i:=uint(0); i<uint(msg.Length); i++ {
        fmt.Printf("%v-",msg.GetBit(i))
    }
    // Output: 
    // 0,0-0,1-0,2-0,3-0,4-0,5-0,6-0,7-1,6-1,7-
    // true-true-true-true-true-true-true-true-true-true-
}
