// This is test firmware, which can be run while the ATTINY13 is still in
// the programmer.  The firmware measures voltage on PB3 and writes it
// to the SPI interface (with SPI mode 1).

#define F_CPU 9400000

#include <avr/interrupt.h>
#include <util/delay.h>
#include <avr/io.h>

volatile unsigned int voltage = 0xf000;
volatile char voltage_spi_index = 15;

int main(void)
{
    // Set clock to 9.4MHz
    CLKPR = (1 << CLKPCE);
    CLKPR = 0;
    
    // Enable the ADC3
    ADCSRA |= (1 << ADPS1) | (1 << ADPS2) | (1 << ADEN) | (1 << ADIE);
    ADMUX |= (1 << MUX1) | (1 << MUX0);

    DDRB |= 1 << DDB4;      // LED-pin is output (pin 3)
    DDRB |= 1 << DDB1;      // MISO (for SPI) is output

    PCMSK |= 1 << PCINT2;   // interrupt on SPI clock
    GIMSK |= 1 << PCIE;     // enable pin-change interrupts

    sei();  // enable interrupts

    ADCSRA |= (1 << ADSC);

    while (1) {
        _delay_ms(500);
        PINB |= 1 << DDB4;
    }

    return 0;
}

// A pin has changed interrupt.  We only watch DDB2, which is the SPI clock.
ISR(PCINT0_vect)
{
    if (PINB & (1 << DDB2)) {
        // If we get a 1 as input, we synchronize
        if (PINB & (1 << DDB0))
            voltage_spi_index = 0;

        if ((1 << voltage_spi_index) & voltage)
            PORTB |= 1 << DDB1;
        else
            PORTB &= ~(1 << DDB1);

        voltage_spi_index = voltage_spi_index - 1;
        if (voltage_spi_index < 0)
            voltage_spi_index = 15;
    }
}

// ADC conversion is ready interrupt.
ISR(ADC_vect)
{
    voltage = ADCW;
    ADCSRA |= (1 << ADSC);
}
