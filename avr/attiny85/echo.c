// SPI test firmware for the ATtiny85

#include "common.h"

// Pinout
#define PIN_SPI_MOSI   DDB0
#define PIN_SPI_MISO   DDB1
#define PIN_SPI_CLOCK  DDB2

void main(void) __attribute__ ((noreturn));

void main(void)
{
    full_speed_clock();

    DDRB |= _BV(PIN_SPI_MISO);  // MISO (for SPI) is output
    DDRB |= _BV(DDB4);  // LED-pin is output (pin 3)
    DDRB |= _BV(DDB3);  // Free pin is output (pin 2)
    DDRB |= _BV(PIN_SPI_MISO);

    USICR |= _BV(USIWM0) | _BV(USICS1) | _BV(USICS0) | _BV(USIOIE);

    sei();  // enable interrupts

    for (;;);
}

ISR(USI_OVF_vect)
{
    PINB |= _BV(DDB4) | _BV(DDB3);
    USISR |= _BV(USIOIF);  // clear interrupt
    USIDR = ~USIBR;
}
