""" Connects using SPI  to the ATTINY13 running the spitherm test firmware
    and output the voltage it's reading """

import spidev
import math
import time
import sys

def main():
    spi = spidev.SpiDev()
    spi.open(0, 0)
    spi.mode = 1
    spi.max_speed_hz = 10000

    while True:
        # send a 1, synchs the uC
        spi.xfer([1])
        vals = []
        for i in xrange(100):
            hi, lo = spi.xfer([0,0])
            vals.append((hi << 8) + lo)
        avg = float(sum(vals)) / len(vals)
        sqavg = float(sum([v**2 for v in vals])) / len(vals)

        R = 9940 / ((1023 / avg) - 1)
        T = 1.0 / (math.log(R / 10000) / 4220 + (1.0 / 298.15)) - 273.15

        print '%.1f +-%.1f min %s max %s R %.0f %.1f' % (
                    avg, math.sqrt(sqavg - avg ** 2),
                    min(vals), max(vals), R, T)

if __name__ == '__main__':
    main()
