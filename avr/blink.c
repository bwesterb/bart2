// Blink the LED on the programmer board.

#define F_CPU 1200000

#include <avr/io.h>
#include <util/delay.h>

int main(void)
{
    DDRB = (1 << DDB4);

    while (1) {
        PORTB ^= (1 << DDB4);
        _delay_ms(500);
    }

    return 0;
}
