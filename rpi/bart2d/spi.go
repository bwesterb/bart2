package main

// Partial interface to the linux's Serial Peripheral Interface driver.
//
// See:
//  - https://www.kernel.org/doc/Documentation/spi/spi-summary
//  - https://www.kernel.org/doc/Documentation/spi/spidev
//  - http://lxr.free-electrons.com/source/include/uapi/linux/spi/spidev.h
//
// TODO: Add funtions for reading settings from the device such as
//       the max transfer speed via SPI_IOC_RD_MAX_SPEED_HZ.
//       My ioctl calls return "bad address" for now.

import (
	"fmt"
	"os"
	s "syscall"
	"unsafe"
)

type SpiDevice os.File

func (d *SpiDevice) Close() error {
	return (*os.File)(d).Close()
}

// The following functions configure the given SPI device.

func (d *SpiDevice) WrMode(mode uint8) error {
	_, _, ern := s.Syscall(s.SYS_IOCTL, d.fd(), SPI_IOC_WR_MODE,
		uintptr(unsafe.Pointer(&mode)))
	return fixNil(ern)
}

func (d *SpiDevice) WrLsbFirst(value bool) error {
	var valueAsByte uint8 = 0
	if value {
		valueAsByte = 1
	}
	_, _, ern := s.Syscall(s.SYS_IOCTL, d.fd(), SPI_IOC_WR_LSB_FIRST,
		uintptr(unsafe.Pointer(&valueAsByte)))
	return fixNil(ern)
}

func (d *SpiDevice) WrBitsPerWord(bitsPerWord uint8) error {
	_, _, ern := s.Syscall(s.SYS_IOCTL, d.fd(), SPI_IOC_WR_BITS_PER_WORD,
		uintptr(unsafe.Pointer(&bitsPerWord)))
	return fixNil(ern)
}

func (d *SpiDevice) WrMaxSpeedHz(maxSpeedHz uint32) error {
	_, _, ern := s.Syscall(s.SYS_IOCTL, d.fd(), SPI_IOC_WR_BITS_PER_WORD,
		uintptr(unsafe.Pointer(&maxSpeedHz)))
	return fixNil(ern)
}

// Message transfers tbuf to the other end, while receiving in rbuf.
func (d *SpiDevice) Message(rbuf, tbuf []byte, args SpiMessageArgs) error {
	if len(rbuf) != len(tbuf) {
		return fmt.Errorf("Slices rbuf and tbuf should have the same length")
	}
	_, _, ern := s.Syscall(s.SYS_IOCTL, d.fd(), SPI_IOC_MESSAGE_1,
		uintptr(unsafe.Pointer(&spiTransfer{
			TxBuf:          uint64(uintptr(unsafe.Pointer(&tbuf[0]))),
			RxBuf:          uint64(uintptr(unsafe.Pointer(&rbuf[0]))),
			Len:            uint32(len(rbuf)),
			SpiMessageArgs: args,
		})))
	return fixNil(ern)
}

func (d *SpiDevice) fd() uintptr {
	return (*os.File)(d).Fd()
}

type SpiMessageArgs struct {
	SpeedHz     uint32
	DelayUsecs  uint16
	BitsPerWord uint8
	CSChange    uint8
	TxNBits     uint8
	RxNBits     uint8
}

type spiTransfer struct {
	TxBuf uint64
	RxBuf uint64
	Len   uint32
	SpiMessageArgs
	Pad uint16
	// 32 bytes in total
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
	SPI_IOC_WR_LSB_FIRST     = 0x40016b02 //01 00000000000001 01101011 00000010
	SPI_IOC_WR_BITS_PER_WORD = 0x40016b03 //01 00000000000001 01101011 00000011
	SPI_IOC_WR_MAX_SPEED_HZ  = 0x40046b04 //01 00000000000100 01101011 00000100
)

type SpiConfiguredDevice struct {
	Device *SpiDevice
	SpiMessageArgs
}

func (d *SpiConfiguredDevice) Close() error {
	return d.Device.Close()
}

// SpiOpen opens the named SPI device (e.g. /dev/spidev0.0) for reading.
func SpiOpen(name string,
	mode uint8, lsbFirst bool, bitsPerWord uint8, speedHz uint32,
) (d *SpiConfiguredDevice, err error) {
	f, err := os.Open(name)
	if err != nil {
		return
	}

	d = &SpiConfiguredDevice{
		Device: (*SpiDevice)(f),
		SpiMessageArgs: SpiMessageArgs{
			BitsPerWord: bitsPerWord,
			SpeedHz:     speedHz,
		},
	}

	// configure device
	err = d.Device.WrMode(mode)
	if err != nil {
		return
	}
	err = d.Device.WrLsbFirst(lsbFirst)
	if err != nil {
		return
	}
	err = d.Device.WrBitsPerWord(bitsPerWord)
	if err != nil {
		return
	}
	err = d.Device.WrMaxSpeedHz(speedHz)
	if err != nil {
		return
	}
	return
}

func (d *SpiConfiguredDevice) Message(rbuf, tbuf []byte) error {
	return d.Device.Message(rbuf, tbuf, d.SpiMessageArgs)
}
