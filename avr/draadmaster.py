""" Connects using SPI  to the ATTINY13 running the draadmaster test firmware
    and output its input. """

import spidev
import time

def main():
    spi = spidev.SpiDev()
    spi.open(0, 0)
    spi.mode = 1
    spi.max_speed_hz = 10000

    print map(bin, spi.xfer([0b10101010]))
    time.sleep(0.002)
    print map(bin, spi.xfer([0b11111111]))
    time.sleep(0.002)
    print map(bin, spi.xfer([0b00000000]))

if __name__ == '__main__':
    main()
