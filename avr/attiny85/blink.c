// Blink the LED and free PIN on the programmer board.

#include "common.h"

#include <avr/io.h>
#include <util/delay.h>


int main(void)
{
    full_speed_clock();

    DDRB |= (1 << DDB4);  // LED-pin is output (pin 3)
    DDRB |= (1 << DDB3);  // Free pin is output (pin 2)
    PORTB |= (1 << DDB3); // start high

    while (1) {
        PINB |= (1 << DDB4) | (1 << DDB3);
        _delay_us(100);
    }

    return 0;
}
