The configuration of the rpi is managed using salt. This is how to bootstrap it.

1. Write raspbian lite to SD-card
2. Boot rPi and log in with user `pi` (password `raspberry`)
3. Change `/etc/wpa_supplicant/wpa_supplicant.conf` and run
   `systemctl restart networking` to get WiFi.
4. Run `raspi-config` and check everything.  Do not forget to enable SSH and SPI
   *and* to disable Serial login.  `reboot`.
5. Login via SSH.  Set `authorized_keys` on root. Delete the pi user. Scramble
   root password.
6. `apt-get update`
7. `apt-get install git salt-minion`
8. set `file_client: local` and `state_verbose: False` in `/etc/salt/minion`
9. `cd srv && git clone git://github.com/bwesterb/bart2`
10. `ln -s /srv/bart2/rpi/salt /srv/salt`
11. `salt-call state.highstate`
