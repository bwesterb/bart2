#pragma once

#define F_CPU 9400000       // XXX calibrate 

#define byte unsigned char

#include <avr/interrupt.h>
#include <util/atomic.h>
#include <util/delay.h>
#include <avr/io.h>

// microsecond delay of the basic time period in the "draad" one-wire protocol.
#define DRAAD_DELAY 30

// Set the ATTINY13 clock to ~9.4MHz, by clearing the clock divisor
inline void full_speed_clock()
{
    CLKPR = (1 << CLKPCE);
    CLKPR = 0;
}
