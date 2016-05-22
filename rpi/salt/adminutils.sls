adminutils packages:
  pkg.installed:
    - pkgs:
      - htop
      - iftop
      - iotop
      - ncdu
      - vim
      - ipython
      - psmisc
      - screen
      - unattended-upgrades
/etc/vim/vimrc.local:
  file.managed:
    - source: salt://vimrc
