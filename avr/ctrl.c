// Firmware for the uC1 and uC2 ATTINY13 microcontroller.
//
// Each of the uCs measures the temperature of the kettle and decides
// whether to turn on the boiler.  The uCs are connected to a third
// uC, the MUX, via a single wire.  See XXX

#include "common.h"

// Pinout
#define PIN_DRAAD      DDB0
#define PIN_WATCH_IN   DDB1
#define PIN_WATCH_OUT  DDB2
#define PIN_GO         DDB3
#define PIN_THERM      DDB4

// Limits on tempearture
#define TEMP_TARGET         500     // Heat if temp is below this
#define TEMP_LOWER_BOUND    100     // go into error mode if temp is below this
#define TEMP_UPPER_BOUND    800     // go into error mode if temp is above this

// We add 32 measurements and to get a neat average.  adc_accum cotains
// the partial sums of the current measurements and adc_cnt the amount we
// have performed.
register unsigned int adc_accum asm("r2");  // r2, r3
register byte adc_cnt asm("r4");

struct status {
    unsigned int temperature : 10;
    byte heating : 1;
    byte ok : 1;
    byte temp_way_too_low : 1;
    byte temp_way_too_high : 1;
    byte other_uC_not_responding : 1;
};

struct status status;

void main(void) __attribute__ ((noreturn));

void main(void)
{
    unsigned long draad_tx_buffer = 0;
    byte draad_tx_buffer_size = 0;

    full_speed_clock();

    adc_accum = 0;
    adc_cnt = 0;

    status.heating = 0;
    status.ok = 1;
    status.temperature = 0;
    status.temp_way_too_low = 0;
    status.temp_way_too_high = 0;
    status.other_uC_not_responding = 0;

    // Set up pins and interrupts for SPI.
    DDRB |= _BV(PIN_WATCH_OUT);  // Set WATCH_OUT to an output pin
    DDRB |= _BV(PIN_GO);         // Set GO to an output pin
    //
    // Enable the ADC
    ADCSRA |= (1 << ADPS1) | (1 << ADPS2) | (1 << ADEN) | (1 << ADIE);
    ADMUX |= (1 << MUX1);  // ADC2 (PB4)

    sei();  // enable interrupts

    ADCSRA |= (1 << ADSC);  // start first ADC conversion

    // XXX make protocol on draad more flexible.  I.e. to set THERM_* runtime.
    for (;;) {
        // Wait until the mux pulls the draad high
        while (!(PINB & _BV(PIN_DRAAD)));

        _delay_us(DRAAD_DELAY * 1.25);

        // Is master sending a write(0) or write(1)?
        // If so, see what the master is writing.
        if (PINB & _BV(PIN_DRAAD)) {
            byte received = 0;

            _delay_us(DRAAD_DELAY * 1.50);

            if (PINB & _BV(PIN_DRAAD))
                received = 1;

            // XXX here we could handle more complex incoming messages ---
            // for now we'll just send our status message everytime we
            // get any incoming bit.
            if (draad_tx_buffer_size == 0) {
                draad_tx_buffer = *((unsigned long*)(&status));
                draad_tx_buffer_size = 16;
            }

            while (PINB & _BV(PIN_DRAAD));

            _delay_us(DRAAD_DELAY * 0.25);
            continue;
        }

        // Master sends a read, send a bit if we have one to send.
        if (draad_tx_buffer_size == 0) {
            _delay_us(DRAAD_DELAY * 2.25);
            continue;
        }

        PORTB |= _BV(PIN_DRAAD);

        byte bit = !!(draad_tx_buffer & 1);
        draad_tx_buffer_size--;
        draad_tx_buffer >>= 1;

        if (bit)
            _delay_us(DRAAD_DELAY * 2);
        else
            _delay_us(DRAAD_DELAY);

        PORTB &= ~_BV(PIN_DRAAD);
        _delay_us(DRAAD_DELAY * 0.25);
    }
}

// ADC conversion is ready interrupt.
// XXX is the interrupt handler fast enough to not disturb draad communication?
ISR(ADC_vect)
{
    adc_accum += ADCW;
    adc_cnt++;

    if (adc_cnt == 32) {
        unsigned int temp = adc_accum >> 5;

        status.temperature = temp;
        adc_accum = 0;
        adc_cnt = 0;

        if (temp <= TEMP_LOWER_BOUND) {
            PORTB &= ~_BV(PIN_GO);
            status.heating = 0;
            status.ok = 0;
            status.temp_way_too_low = 1;
        } else if (temp >= TEMP_UPPER_BOUND) {
            PORTB &= ~_BV(PIN_GO);
            status.heating = 0;
            status.ok = 0;
            status.temp_way_too_high = 1;
        } else if (!status.ok) {
        } else if (temp < TEMP_TARGET) {
            PORTB |= _BV(PIN_GO);
            status.heating = 1;
        } else  {
            PORTB &= ~_BV(PIN_GO);
            status.heating = 0;
        }
    }

    ADCSRA |= (1 << ADSC);
}
