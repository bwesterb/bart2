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

    def _bin(x):
        return bin(x)[2:].zfill(8)


    def send(who, what):
        b1 = (who << 6) | (8 << 1) | 1
        b2 = what
        print map(_bin, spi.xfer(map(rev, [b1,b2])))

    send(0, 0b1000000000000000)
    time.sleep(0.050)
    s = ''.join(map(_bin, spi.xfer([0]*80)))
    i = 0
    print s
    while i < len(s):
        if s[i] == '0':
            i += 1
            continue
        i += 1
        print ''.join(reversed(s[i:i+5]))
        l = int(''.join(reversed(s[i:i+5])), 2)
        i += 5
        w = s[i:i+2]
        i += 2
        d = s[i:i+l]
        i += l
        print w,':', l, d

if __name__ == '__main__':
    main()
