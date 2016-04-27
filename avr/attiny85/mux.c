// Firmware for the MUX ATTINY85 microcontroller.
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
// XXX send status

#include "common.h"

// Pinout
#define PIN_SPI_MOSI   DDB0
#define PIN_SPI_MISO   DDB1
#define PIN_SPI_CLOCK  DDB2
#define PIN_DRAAD1     DDB3
#define PIN_DRAAD2     DDB4

register unsigned long spi_tx_buffer asm("r10"); // r10, r11, r12, r13
register byte spi_tx_buffer_size asm("r14");    // XXX only need 5 bits

volatile unsigned long draad_tx_buffer[2] = {0,0};
volatile byte draad_tx_buffer_size[2] = {0,0};

unsigned long draad_rx_buffer[2] = {0,0};
byte draad_rx_buffer_size[2] = {0,0};


struct status {
    byte draad_tx_overflow : 1;
    byte spi_rx_overflow : 1;
};

register struct status status asm("r15");

#define SPI_RX_BUFFER_MAX   8       // max size of the SPI rx buffer
byte spi_rx_buffer[SPI_RX_BUFFER_MAX];
byte spi_rx_buffer_size asm("r16");
byte spi_rx_buffer_offset asm("r17");

void main(void) __attribute__ ((noreturn));

void main(void)
{
    byte spi_frame_body_bits_to_read = 0;
    byte spi_frame_body_size = 0;
    long spi_frame_body;
    byte spi_frame_who;

    full_speed_clock();

    spi_tx_buffer = 0;
    spi_tx_buffer_size = 0;

    spi_rx_buffer_size = 0;
    spi_rx_buffer_offset = 0;

    status.draad_tx_overflow = 0;
    status.spi_rx_overflow = 0;

    // Set up pins and interrupts for SPI.
    DDRB |= _BV(PIN_SPI_MISO);  // MISO (for SPI) is output

    // Set up universal serial interface
    USICR |= _BV(USIWM0) | _BV(USICS1) | _BV(USICS0) // SPI mode 01
          | _BV(USIOIE) | _BV(USISIE);   // with overflow and start int.

    sei();  // enable interrupts

    // We will loop and either
    //  (1) poll the first uC,
    //  (2) poll the second uC,
    //  (3) prepare the data to be send over SPI or
    //  (4) parse the data we received over SPI.
    // We process the SPI data in the main loop such that we don't spend
    // too much time in interrupt handlers, which would mess with the timing
    // on the draad.

    byte who = 0; // which uC is being polled?

    for(;;) {
        who = 1 - who;
        
        // Interpret SPI rx buffer if it's not empty
        while (spi_rx_buffer_size > 0) {
            byte spi_byte_received;

            ATOMIC_BLOCK(ATOMIC_FORCEON)
            {
                spi_byte_received = spi_rx_buffer[spi_rx_buffer_offset];
                spi_rx_buffer_size--;
                spi_rx_buffer_offset = (spi_rx_buffer_offset + 1)
                                                % SPI_RX_BUFFER_MAX;
            }

            // Are we in the middle of receiving a frame?
            if (spi_frame_body_bits_to_read > 0) {
                if (spi_frame_body_bits_to_read < 8)
                    spi_frame_body_bits_to_read = 0;
                else
                    spi_frame_body_bits_to_read -= 8;
                spi_frame_body |= (spi_byte_received
                                    << spi_frame_body_bits_to_read);

                if (spi_frame_body_bits_to_read > 0)
                    continue;

                // We received the whole frame.  Send it over draad.
                if (spi_frame_who > 1)
                    continue;

                // Check for overflow
                if (draad_tx_buffer_size[spi_frame_who]
                            + spi_frame_body_size > 32) {
                    status.draad_tx_overflow = 1;
                    continue;
                }

                // Again note that we can't be interrupted here.
                draad_tx_buffer[spi_frame_who] |= (spi_frame_body
                                        << draad_tx_buffer_size[spi_frame_who]);
                draad_tx_buffer_size[spi_frame_who] += spi_rx_buffer_size;
            }

            // Either there is no frame, or we are at the start of a frame.
            // Is there a frame?
            if (!(spi_byte_received & 128))
                continue;  // there is no frame

            // Start of a new frame.  Parse the header
            spi_frame_body_size = (spi_byte_received >> 2) & 31;
            spi_frame_body_bits_to_read = spi_frame_body_size;
            spi_frame_who = (spi_byte_received) & 3;
            spi_frame_body = 0;
        }

        // Fill SPI tx buffer if it's empty
        if (spi_tx_buffer_size == 0 && draad_rx_buffer_size[who] > 0) {
                byte n_bits_to_send = draad_rx_buffer_size[who];
                if (n_bits_to_send > 24)
                    n_bits_to_send = 24;
            ATOMIC_BLOCK(ATOMIC_FORCEON)
            {
                spi_tx_buffer = 128 | (n_bits_to_send << 2) | who
                                  | (draad_rx_buffer[who] << 8);
                spi_tx_buffer_size = 8 + n_bits_to_send;
            }
            draad_rx_buffer_size[who] -= n_bits_to_send;
            draad_rx_buffer[who] >>= n_bits_to_send;
        }

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
                _delay_us(DRAAD_DELAY * 3 - DRAAD_PULLDOWN_TIME);
            else
                _delay_us(DRAAD_DELAY * 2 - DRAAD_PULLDOWN_TIME);

            PORTB &= ~_BV(pin);
            if (to_send)
                _delay_us(DRAAD_DELAY * 1 + DRAAD_PULLDOWN_TIME);
            else
                _delay_us(DRAAD_DELAY * 2 + DRAAD_PULLDOWN_TIME);

            continue;
        }

        // Is our buffer empty enough to receive something from the uC?
        if (draad_rx_buffer_size[who] == 32)
            continue;

        byte received = 0;
        PORTB |= _BV(pin);
        _delay_us(DRAAD_DELAY - DRAAD_PULLDOWN_TIME);

        PORTB &= ~_BV(pin);
        _delay_us(DRAAD_DELAY + DRAAD_PULLDOWN_TIME);

        if (!(PINB & _BV(pin))) {
            _delay_us(DRAAD_DELAY * 2);
            continue;  // No reply
        }

        _delay_us(DRAAD_DELAY);
        if (PINB & _BV(pin))
            received = 1;

        draad_rx_buffer[who] |= (received << draad_rx_buffer_size[who]);
        draad_rx_buffer_size[who]++;
        _delay_us(DRAAD_DELAY);
    }
}

// Interrupt handlers.

// Called when the USI unit is about to send data over SPI.  We should provide
// the USI unit with the byte it should send next.
ISR(USI_START_vect)
{
    if (spi_tx_buffer_size == 0) {
        USIDR = 0;
        goto done;
    }

    USIDR = spi_tx_buffer & 8;  // next byte to send

    spi_tx_buffer >>= 8;

    if (spi_tx_buffer_size < 8)
        spi_tx_buffer_size = 0;
    else
        spi_tx_buffer_size -= 8;
done:
    USISR |= _BV(USISIF);  // clear interrupt flag
}

// Called when the USI unit just received data over SPI.  We store the data in
// a buffer.
ISR(USI_OVF_vect)
{
    if (spi_rx_buffer_size == SPI_RX_BUFFER_MAX) {
        status.spi_rx_overflow = 1;
        goto done;
    }
    
    byte i = (spi_rx_buffer_offset + spi_rx_buffer_size) % SPI_RX_BUFFER_MAX;
    spi_rx_buffer[i] = USIBR;
    spi_rx_buffer_size++;

done:
    USISR |= _BV(USIOIF);  // clear interrupt
}
