package main

import (
	"fmt"
	"os"
	"os/user"
	"path"
)

type Dir struct {
	pth string
}

const DIR_DEFAULT_FILEMODE os.FileMode = 0640 // rw- r-- ---
const DIR_DEFAULT_DIRMODE os.FileMode = 0750  // rwx r-x ---

func DirOpen() (d Dir, err error) {
	usr, err := user.Current()
	if err != nil {
		return
	}
	pth := path.Join(usr.HomeDir, ".bart2d")
	d = Dir{pth: pth}

	err = ensureDir(pth)
	if err != nil {
		return
	}

	err = ensureDir(d.Reports())
	if err != nil {
		return
	}

	return // err=nil
}

func (d Dir) Reports() string {
	return path.Join(d.pth, "reports")
}

func ensureDir(name string) error {
	fi, err := os.Stat(name)
	if err == nil {
		if fi.IsDir() {
			return nil
		} else {
			return fmt.Errorf("%s is a file, but should be a directory.",
				name)
		}
	}
	if !os.IsNotExist(err) {
		return WrapErr(err, "Could not stat %s.", name)
	}
	if err := os.Mkdir(name, DIR_DEFAULT_DIRMODE); err != nil {
		return WrapErr(err, "Could not mkdir %s.", name)
	}
	if _, err = os.Stat(name); err != nil {
		return WrapErr(err, "Could not stat %s.", name)
	}
	return nil
}
