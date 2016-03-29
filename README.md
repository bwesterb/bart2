Controller for the Bar T2
-------------------------

We replaced the mechanical pressure switch of the Pavoni Bart T2 coffee
machine with two thermometers, three microcontrollers and a raspberry pi.

See `avr/ctrl.c` for the firmware of the two microcontrollers responsible
for switching the heater on and off depending on the temperature.

The third micrcontroller lets the raspberry pi communicate with the
other microcontroller.  Its firmware can be found in `avr/mux.c`.

![pcb](/pcb/brd.png?raw=true)
