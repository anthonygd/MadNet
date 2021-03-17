package utils

import (
	"bufio"
	"os"
)

// Writer writes a migration file one line at a time
type Writer struct {
	Writer *bufio.Writer
}

// WriteLine writes a *CommandObj to a file as a line
func (w *Writer) WriteLine(cmd *CommandObj) error {
	s, err := cmd.Marshal()
	if err != nil {
		return err
	}
	s += "\n"
	if w.Writer.Available() < len(s) {
		if err := w.Writer.Flush(); err != nil {
			return err
		}
	}
	for wc := 0; wc < len(s); {
		wcc, err := w.Writer.WriteString(s[wc:])
		if err != nil {
			return err
		}
		wc += wcc
	}
	return nil
}

// OpenFileAsWriter opens the file at located at path as a *bufio.Writer
func OpenFileAsWriter(path string) (w *bufio.Writer, close func() error, err error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return nil, nil, err
	}
	w = bufio.NewWriter(file)
	close = func() error {
		defer file.Close()
		if err := w.Flush(); err != nil {
			return err
		}
		return nil
	}
	return w, close, nil
}
