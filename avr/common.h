#pragma once

#define F_CPU 9400000

#define byte unsigned char

#include <avr/interrupt.h>
#include <util/atomic.h>
#include <util/delay.h>
#include <avr/io.h>

// microsecond delay of the basic time period in the "draad" one-wire protocol.
#define DRAAD_DELAY 200

// microseconds it takes to pull the draad down.  Depends on the resistance
// of the pull down resistor on the "draad".  Current is for a 100Kohm.
#define DRAAD_PULLDOWN_TIME 4

// Set the ATTINY13 clock to ~9.4MHz, by clearing the clock divisor
inline void full_speed_clock()
{
    CLKPR = (1 << CLKPCE);
    CLKPR = 0;
}
