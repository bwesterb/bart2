""" Connects using SPI to the MUX and prints its messages. """

import spidev

def main():
    spi = spidev.SpiDev()
    spi.open(0, 0)
    spi.mode = 1
    spi.bits_per_word = 8
    spi.max_speed_hz = 10000
    ret = spi.xfer(range(256))
    print ret
    assert ret[1:] == range(255,0,-1)

if __name__ == '__main__':
    main()
