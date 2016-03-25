""" Connects using SPI  to the ATTINY13 running the draadmaster test firmware
    and output its input. """

import spidev

def main():
    spi = spidev.SpiDev()
    spi.open(0, 0)
    spi.mode = 1
    spi.max_speed_hz = 10000

    print map(bin, spi.xfer([0b10110100,0b11111111,0b01010101,0b01101001]))

if __name__ == '__main__':
    main()
