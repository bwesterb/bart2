// Mirrors the LED to the free pin on the programmer board.
// Used to determine the switch from logic 0 to logic 1 on the inputs
// of the ATtiny13 (~1.3V)

#include "common.h"

#include <avr/io.h>
#include <util/delay.h>


int main(void)
{
    full_speed_clock();

    DDRB |= _BV(DDB4);  // LED-pin is output (pin 3)

    for(;;) {
        if (PINB & _BV(DDB3)) {
            PORTB |= _BV(DDB4);
        } else {
            PORTB &= ~_BV(DDB4);
        }
    }

    return 0;
}
