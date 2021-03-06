#pragma once

#define F_CPU 9400000

#define byte unsigned char

#include <avr/interrupt.h>
#include <util/atomic.h>
#include <util/delay.h>
#include <avr/io.h>

#include "../draad.h"

// Set the ATTINY13 clock to ~9.4MHz, by clearing the clock divisor
inline void full_speed_clock()
{
    CLKPR = (1 << CLKPCE);
    CLKPR = 0;
}
