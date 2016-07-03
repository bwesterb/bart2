package main

import (
	"encoding/csv"
	"os"
    "fmt"
    "time"
    "path"
)

func DumperOpen(dir Dir) (d *Dumper, err error) {
	file := DumpFile{
        dirName: dir.Reports(),
    }
	d = &Dumper{
        file: file,
		csvWriter: csv.NewWriter(file),
	}
	return // nil
}

func (d *Dumper) Dump(r ChipiReport) error {
	// changes underlying csv-file, if necessary
	if err := d.file.Update(r.Time); err != nil {
		return err
	}
	return d.csvWriter.Write(r.toRecord())
}

func (d *Dumper) Close() error {
	d.csvWriter.Flush()
	err1 := d.csvWriter.Error() // get error produced by Flush()
	err2 := d.file.Close()
	return WrapErrs([]error{err1, err2}, "Could not close dumper")
}

type Dumper struct {
	Path      string
	file      DumpFile
	csvWriter *csv.Writer
}

// DumpFile represents the CSV-file into which the temperature reports
// are dumped; it implements io.Writer.  It switches every day to a new
// file internally.
type DumpFile struct {
	dirName          string
	file             *os.File
	year, day int
    month time.Month
}

func (d DumpFile) Update(time time.Time) error {
	year := time.Year()
	month := time.Month()
	day := time.Day()

	if d.day == day && d.month == month && d.year == year {
		return nil
	}

	// d.Close() only closes d.file (if possible).
	if err := d.Close(); err != nil {
		return err
	}

	fileName := path.Join(d.dirName, fmt.Sprintf("%4d-%02d-%02d.csv",
		year, month, day))
	file, err := os.OpenFile(fileName,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		DIR_DEFAULT_FILEMODE)

	if err != nil {
		return err
	}

	d.file = file
	d.year = year
	d.month = month
	d.day = day
    return nil
}

func (d DumpFile) Close() (err error) {
	if d.file == nil {
		return nil
	}
	return d.file.Close()
}

func (d DumpFile) Write(p []byte) (int, error) {
	return d.file.Write(p)
}
