# Create bart2d user
bart2d:
  user.present:
    - system: True
    - home: /var/bart2d
    - groups:
      - spi
      - gpio

# We need the toolchain for Go
bart2d packages:
  pkg.installed:
    - pkgs:
        - golang

# We set GOPATH to /srv/go
/srv/go:
  file.directory
/srv/go/src:
  file.directory
/srv/go/src/bart2d:
  file.symlink:
    - target: /srv/bart2/rpi/bart2d

# Compile and install bart2d
go install bart2d:
  cmd.run:
    - env:
      - GOPATH: /srv/go

# Create a systemd unit
/etc/systemd/system/bart2d.service:
  file.managed:
    - source: salt://bart2d.service
bart2d running:
  service.running:
    - name: bart2d
