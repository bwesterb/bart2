// Blink the LED on the programmer board
// and send test messages over UART.

#define F_CPU 1154842

#include <avr/io.h>
#include <util/delay.h>

static const long int BAUD_WAIT = 1000000 / 9600;

static inline void write_byte(unsigned char what, unsigned char pin)
{
    PORTB &= ~(1 << pin); // start bit
    _delay_us(BAUD_WAIT);
    for (unsigned char i = 0; i < 8; i++) {
        if (what & 1)
            PORTB |= 1 << pin;
        else
            PORTB &= ~(1 << pin);
        what >>= 1;
        _delay_us(BAUD_WAIT);
    }
    PORTB |= 1 << pin;  // stop bit
    _delay_us(BAUD_WAIT);
}

int main(void)
{
    DDRB |= (1 << DDB4);  // LED-pin is output (pin 3)
    DDRB |= (1 << DDB3);  // TX-pin is output (pin 2)
    PORTB |= (1 << DDB3); // start high

    while (1) {
        // PINB |= (1 << DDB4);
        write_byte('!', DDB3);
        _delay_ms(1000);
    }

    return 0;
}
