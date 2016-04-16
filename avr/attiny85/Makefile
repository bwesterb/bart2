MCU=attiny85
CC=avr-gcc
CFLAGS=-Os -Wall -mmcu=$(MCU) -std=gnu99 -Wno-main \
			-ffreestanding -fwhole-program -ffunction-sections \
			-fdata-sections -Wl,--relax,--gc-sections -fno-tree-scev-cprop \
			-fno-split-wide-types -fpack-struct
OBJ2HEX=avr-objcopy

default: blink.hex

%.hex: %.c
	$(CC) $(CFLAGS) $< -o $<.o
	$(OBJ2HEX) -R .eeprom -O ihex $<.o $@

clean:
	rm *.o *.hex
