// This is test firmware for the single-wire communication protocol
// between the mux mC and the other mCs.  This is the slave.

#define F_CPU 9400000

#include <avr/interrupt.h>
#include <util/atomic.h>
#include <util/delay.h>
#include <avr/io.h>

#define DRAAD_DELAY 30

#define DRAAD       DDB3
#define LED         DDB4

typedef unsigned char byte;
typedef unsigned char bool;

unsigned int buffer = 0;
byte to_send = 0;

int main(void)
{
    // Set clock to 9.4MHz
    CLKPR = (1 << CLKPCE);
    CLKPR = 0;
    
    DDRB |= 1 << LED;       // LED-pin is output (pin 3)

    for (;;) {
        // Wait until the master pulls the draad high
        while (!(PINB & (1 << DRAAD)));

        _delay_us(DRAAD_DELAY * 1.25);

        // Is master sending a write(0) or write(1)?
        // If so, see what the master is writing and put it into a buffer
        // to echo back.
        if (PINB & (1 << DRAAD)) {
            bool received = 0;

            _delay_us(DRAAD_DELAY * 1.50);

            if (PINB & (1 << DRAAD))
                received = 1;

            if (to_send != 16) {
                buffer |= received << to_send;
                to_send++;
                // XXX error mode
            }

            while (PINB & (1 << DRAAD));

            _delay_us(DRAAD_DELAY * 0.25);
            continue;
        }

        // Master sends a read, send a bit if we have one to send.
        if (to_send == 0) {
            _delay_us(DRAAD_DELAY * 2.25);
            continue;
        }

        PORTB |= 1 << DRAAD;

        bool bit = !!(buffer & 1);
        to_send--;
        buffer >>= 1;

        if (bit)
            _delay_us(DRAAD_DELAY * 2);
        else
            _delay_us(DRAAD_DELAY);

        PORTB &= ~(1 << DRAAD);
        _delay_us(DRAAD_DELAY * 0.25);
    }

    return 0;
}


