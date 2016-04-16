// Test timer of ATtiny13

#include "common.h"

#include <avr/io.h>
#include <util/delay.h>


int main(void)
{
    full_speed_clock();

    DDRB |= _BV(DDB4);          // Free pin is output (pin 2)
    PORTB |= _BV(DDB4);         // start high

    // Set up timer
    TCCR0B |= _BV(CS00) | _BV(CS02);
    TIMSK0 |= _BV(TOIE0);        // enable clock overflow interrupt

    sei();

    for(;;);

    return 0;
}

// Timer overflowed
ISR(TIM0_OVF_vect)
{
    PINB |= _BV(DDB4);
}
