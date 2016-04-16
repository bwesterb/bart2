// This is test firmware for the single-wire communication protocol
// between the mux mC and the other mCs.  This is the master.

#define F_CPU 9400000

#include <avr/interrupt.h>
#include <util/atomic.h>
#include <util/delay.h>
#include <avr/io.h>

#define DRAAD_DELAY 30

#define SPI_MOSI    DDB0
#define SPI_MISO    DDB1
#define SPI_CLOCK   DDB2
#define DRAAD       DDB3
#define LED         DDB4

typedef unsigned char byte;
typedef unsigned char bool;

volatile byte last_pinb = 0;

bool in_error = 0;
volatile byte draad_to_send = 0;
volatile unsigned int draad_buffer = 0;
volatile byte spi_to_send = 0;
volatile unsigned int spi_buffer = 0;

void draad_tick()
{
    if (draad_to_send > 0) {
        // we got a bit to send
        bool to_send;
       
        ATOMIC_BLOCK(ATOMIC_FORCEON)
        {
            to_send = draad_buffer & 1;
            draad_buffer >>= 1;
            draad_to_send--;
        }
        PORTB |= (1 << DRAAD);
        if (to_send)
            _delay_us(DRAAD_DELAY * 3);
        else
            _delay_us(DRAAD_DELAY * 2);
        PORTB &= ~(1 << DRAAD);
        if (to_send)
            _delay_us(DRAAD_DELAY * 1);
        else
            _delay_us(DRAAD_DELAY * 2);
        return;
    }

    // Ask the slave for a bit.
    bool received = 0;
    PORTB |= (1 << DRAAD);
    _delay_us(DRAAD_DELAY);
    PORTB &= ~(1 << DRAAD);
    _delay_us(DRAAD_DELAY);
    if (!(PINB & (1 << DRAAD))) {
        _delay_us(DRAAD_DELAY * 2);
        return;
    }
    _delay_us(DRAAD_DELAY);
    if (PINB & (1 << DRAAD)) {
        received = 1;
    }
    if (spi_to_send < 16) {
        ATOMIC_BLOCK(ATOMIC_FORCEON)
        {
            spi_buffer |= received << spi_to_send;
            spi_to_send++;
        }
    }
    _delay_us(DRAAD_DELAY);
}

int main(void)
{
    // Set clock to 9.4MHz
    CLKPR = (1 << CLKPCE);
    CLKPR = 0;
    
    DDRB |= 1 << LED;       // LED-pin is output (pin 3)
    DDRB |= 1 << SPI_MISO;  // MISO (for SPI) is output

    PCMSK |= 1 << PCINT2;   // interrupt on SPI clock
    GIMSK |= 1 << PCIE;     // enable pin-change interrupts

    sei();  // enable interrupts

    while (1) {
        if(in_error)
            PINB |= 1 << LED;
        draad_tick();
    }

    return 0;
}


// A pin has changed interrupt.
ISR(PCINT0_vect)
{
    byte changes = PINB ^ last_pinb;
    last_pinb = PINB;

    if (changes & (1 << SPI_CLOCK)) {
        // SPI clock changed
        if (PINB & (1 << SPI_CLOCK)) {
            // SPI clock is high.  Set output bit.
            volatile bool to_send = 0;

            if (spi_to_send > 0) {
                ATOMIC_BLOCK(ATOMIC_FORCEON)
                {
                    to_send = spi_buffer & 1;
                    spi_buffer >>= 1;
                    spi_to_send--;
                }
            }

            if (to_send) {
                PORTB |= 1 << SPI_MISO;
            } else {
                PORTB &= ~(1 << SPI_MISO);
            }
        } else {
            if (draad_to_send != 16) {
                ATOMIC_BLOCK(ATOMIC_FORCEON)
                {
                    draad_buffer |= (PINB & (1 << SPI_MOSI)) << draad_to_send;
                    draad_to_send++;
                }
            } else
                in_error = 1;
        }
    }
}
