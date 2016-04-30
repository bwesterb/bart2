package main

import (
	"fmt"
	"os"
	s "syscall"
	"unsafe"
)

// transfer tbuf to the other end, while receiving in rbuf
func SpiMessage(f *os.File, rbuf, tbuf []byte, args SpiArgs) error {
	if len(rbuf) != len(tbuf) {
		return fmt.Errorf("Slices rbuf and tbuf should have the same length")
	}
	_, _, ern := s.Syscall(s.SYS_IOCTL, f.Fd(), SPI_IOC_MESSAGE_1,
		uintptr(unsafe.Pointer(&SpiTransfer{
			TxBuf:   uint64(uintptr(unsafe.Pointer(&tbuf[0]))),
			RxBuf:   uint64(uintptr(unsafe.Pointer(&rbuf[0]))),
			Len:     uint32(len(rbuf)),
			SpiArgs: args,
		})))
	return fixNil(ern)
}

type SpiArgs struct {
	SpeedHz     uint32
	DelayUsecs  uint16
	BitsPerWord uint8
	CSChange    uint8
	TxNBits     uint8
	RxNBits     uint8
}

type SpiTransfer struct {
	TxBuf uint64
	RxBuf uint64
	Len   uint32
	SpiArgs
	Pad uint16
	// 32 bytes in total
}

func SpiWrMode(f *os.File, mode uint8) error {
	_, _, ern := s.Syscall(s.SYS_IOCTL, f.Fd(), SPI_IOC_WR_MODE,
		uintptr(unsafe.Pointer(&mode)))
	return fixNil(ern)
}

func fixNil(ern s.Errno) error {
	// since ern's Kind is uintptr, it is 0 on success rather than nil
	if ern == 0 {
		return nil
	}
	return ern
}

const (
	SPI_IOC_MESSAGE_1        = 0x40206b00 //01 00000000100000 01101011 00000000
	SPI_IOC_WR_MODE          = 0x40016b01 //01 00000000000001 01101011 00000001
	SPI_IOC_RD_MODE          = 0x80016b01 //10 00000000000001 01101011 00000001
	SPI_IOC_WR_LSB_FIRST     = 0x40016b02 //01 00000000000001 01101011 00000010
	SPI_IOC_RD_LSB_FIRST     = 0x80016b02 //10 00000000000001 01101011 00000010
	SPI_IOC_WR_BITS_PER_WORD = 0x40016b03 //01 00000000000001 01101011 00000011
	SPI_IOC_RD_BITS_PER_WORD = 0x80016b03 //10 00000000000001 01101011 00000011
	SPI_IOC_WR_MAX_SPEED_HZ  = 0x40046b04 //01 00000000000100 01101011 00000100
	SPI_IOC_RD_MAX_SPEED_HZ  = 0x80046b04 //10 00000000000100 01101011 00000100
)
