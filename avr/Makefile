MCU=attiny13
CC=avr-gcc
CFLAGS=-Os -Wall -mcall-prologues -mmcu=$(MCU)
OBJ2HEX=avr-objcopy

default: blink.hex

%.hex: %.c
	$(CC) $(CFLAGS) $< -o $<.o
	$(OBJ2HEX) -R .eeprom -O ihex $<.o $@

clean:
	rm *.o *.hex
