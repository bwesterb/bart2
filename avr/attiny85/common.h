#pragma once

#define F_CPU 7970703

#define byte unsigned char

#include <avr/interrupt.h>
#include <util/atomic.h>
#include <util/delay.h>
#include <avr/io.h>

#include "../draad.h"


// Set the ATTINY85 clock to ~9MHz, by clearing the clock divisor
inline void full_speed_clock()
{
    CLKPR = (1 << CLKPCE);
    CLKPR = 0;
}
