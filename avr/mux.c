// Firmware for the MUX ATTINY13 microcontroller.
//
// The MUX is connected to the rPi and the two microcontrollers that actually
// measure the temperature and determine whether to switch on the heaters
// of the coffee machine.
//
// The job of the MUX uC is to transfer messages to and from the two
// other uCs to the rPi.
//
// The MUX is connected to the rPi using a 3-wire SPI connection in mode 01
// with a baudrate of ~10kHz.
//
// There is a single wire from the MUX to each of the other uCs.  With this
// single wire the uC and the MUX can communicate in half-duplex using a
// variation on the "one-wire protocol", we call "draad".  See XXX
// 
// XXX write draad_rx_buffer --> spi_tx_buffer code
// XXX fix bug in spi rx code

#include "common.h"

// Pinout
#define PIN_SPI_MOSI   DDB0
#define PIN_SPI_MISO   DDB1
#define PIN_SPI_CLOCK  DDB2
#define PIN_DRAAD1     DDB3
#define PIN_DRAAD2     DDB4

volatile unsigned long spi_tx_buffer = 0;
volatile byte spi_tx_buffer_size = 0;           // XXX only need 5 bits

// The header of an incoming frame will be stored in spi_rx_header_buffer.
// The body is stored in spi_rx_buffer.
byte spi_rx_header_buffer = 0;
byte spi_rx_header_buffer_size = 0;             // XXX only need 3 bits
unsigned long spi_rx_buffer = 0;
byte spi_rx_buffer_size = 0;                    // XXX only need 5 bits

volatile unsigned long draad_tx_buffer[2] = {0,0};
volatile byte draad_tx_buffer_size[2] = {0,0};
volatile unsigned long draad_rx_buffer[2] = {0,0};
volatile byte draad_rx_buffer_size[2] = {0,0};


struct status {
    byte draad_tx_overflow : 1;
};

volatile struct status status;

int main(void)
{
    full_speed_clock();

    status.draad_tx_overflow = 0;

    // Set up pins and interrupts for SPI.
    DDRB |= _BV(PIN_SPI_MISO);  // MISO (for SPI) is output
    PCMSK |= _BV(PCINT2);       // interrupt on SPI clock
    GIMSK |= _BV(PCIE);         // enable pin-change interrupts

    sei();  // enable interrupts

    byte who = 0; // which uC is being polled?

    for(;;) {
        who = 1 - who;
        byte pin = who == 0 ? PIN_DRAAD1 : PIN_DRAAD2;

        if (draad_tx_buffer_size[who] > 0) {
            byte to_send;
           
            // Fetch the bit to send
            ATOMIC_BLOCK(ATOMIC_FORCEON)
            {
                to_send = draad_tx_buffer[who] & 1;
                draad_tx_buffer[who] >>= 1;
                draad_tx_buffer_size[who]--;
            }

            // Send the bit
            PORTB |= _BV(pin);
            if (to_send)
                _delay_us(DRAAD_DELAY * 3);
            else
                _delay_us(DRAAD_DELAY * 2);

            PORTB &= ~_BV(pin);
            if (to_send)
                _delay_us(DRAAD_DELAY * 1);
            else
                _delay_us(DRAAD_DELAY * 2);

            continue;
        }

        // Is our buffer empty enough to receive something from the uC?
        if (draad_rx_buffer_size[who] == 32)
            continue;

        byte received = 0;
        PORTB |= _BV(pin);
        _delay_us(DRAAD_DELAY);

        PORTB &= ~_BV(pin);
        _delay_us(DRAAD_DELAY);

        if (!(PINB & _BV(pin))) {
            _delay_us(DRAAD_DELAY * 2);
            continue;  // No reply
        }

        _delay_us(DRAAD_DELAY);
        if (PINB & _BV(pin))
            received = 1;

        ATOMIC_BLOCK(ATOMIC_FORCEON)
        {
            draad_rx_buffer[who] |= (received << draad_rx_buffer_size[who]);
            draad_rx_buffer_size[who]++;
        }
        _delay_us(DRAAD_DELAY);
    }

    return 0;
}

// Interrupt handlers.


// Called when a watched pin has changed value.
// We only watch the SPI clock
ISR(PCINT0_vect)
{
    if (PINB & _BV(PIN_SPI_CLOCK)) {
        // SPI clock is high, we set the SPI output pin (MISO) to the correct
        // state.
        byte to_send = 0;

        if (spi_tx_buffer_size > 0) {
            to_send = spi_tx_buffer & 1;   // Note that we cannot be interrupted
            spi_tx_buffer_size--;          // here, as interrupts are disabled.
            spi_tx_buffer >>= 1;
        }

        if (to_send)
            PORTB |= _BV(PIN_SPI_MISO);
        else
            PORTB &= ~_BV(PIN_SPI_MISO);

        return;
    }

    // SPI clock is low --- we read the SPI input pin (MOSI).
    byte received = !!(PORTB & _BV(PIN_SPI_MOSI)); 

    // If spi_rx_header_buffer_size == 0 we are in the idle state, and we wait
    // for a 1 to start a frame.
    if (spi_rx_header_buffer_size == 0 && !received)
        return;

    // We're reading the header of a frame.
    if (spi_rx_header_buffer_size != 8) {
        spi_rx_header_buffer |= (received << spi_rx_header_buffer_size);
        spi_rx_header_buffer_size++;
        return;
    }

    // We're reading the body of the incoming frame.
    spi_rx_buffer |= (received << spi_rx_buffer_size);
    spi_rx_buffer_size++;

    if (spi_rx_buffer_size != ((spi_rx_header_buffer >> 1) & 31))
        return;

    // We received the whole frame.  Send it over draad.
    byte who = spi_rx_header_buffer >> 6;
    if (who > 1)
        goto reset;

    if (draad_tx_buffer_size[who] + spi_rx_buffer_size > 32) {
        status.draad_tx_overflow = 1;
        goto reset;
    }

    // Again note that we can't be interrupted here.
    draad_tx_buffer[who] |= (spi_rx_buffer << draad_tx_buffer_size[who]);
    draad_tx_buffer_size[who] += spi_rx_buffer_size;
    
reset:
    // Reset to the idle state
    spi_rx_buffer_size = 0;
    spi_rx_header_buffer_size = 0;
    spi_rx_buffer = 0;
    spi_rx_header_buffer = 0;
}
