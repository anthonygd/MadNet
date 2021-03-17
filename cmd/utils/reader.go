package utils

import (
	"bufio"
	"os"
	"strings"
)

// Reader reads a migration file one line at a time and converts each line
// into a *CommandObj
type Reader struct {
	Reader *bufio.Reader
}

// ReadLine reads one line from the reader and returns it as a *CommandObj
// when the last line of the file is read, io.EOF will be returned
func (p *Reader) ReadLine() (*CommandObj, error) {
	line, err := p.Reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\n")
	cmd := &CommandObj{}
	err = cmd.Unmarshal(line)
	if err != nil {
		return nil, err
	}
	return cmd, nil
}

// OpenFileAsReader opens the file at located at path as a *bufio.Reader
func OpenFileAsReader(path string) (r *bufio.Reader, close func() error, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	close = file.Close
	r = bufio.NewReader(file)
	return r, close, nil
}
