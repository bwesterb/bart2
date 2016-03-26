""" Connects using SPI to the MUX and prints its messages. """

import spidev
import time

def rev(n):
    return int('{:08b}'.format(n)[::-1], 2)

def main():
    spi = spidev.SpiDev()
    spi.open(0, 0)
    spi.mode = 1
    spi.bits_per_word = 8
    spi.max_speed_hz = 10000

    def send(who, what):
        b1 = (who << 6) | (8 << 1) | 1
        b2 = what
        print map(bin, spi.xfer(map(rev, [b1,b2])))

    send(0, 0b01101001)

if __name__ == '__main__':
    main()
