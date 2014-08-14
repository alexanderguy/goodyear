package frame

import (
	"bufio"
	"strings"
	"errors"
	"fmt"
)

func readLine(r *bufio.Reader) (s string, err error) {
	s, err = r.ReadString('\n')
	if err != nil {
		return
	}

	endIdx := len(s) - 1
	if endIdx != 0 && s[endIdx - 1] == '\r' {
		endIdx--
	}

	s = s[:endIdx]

	return
}

type FrameHeader map[string][]string

func (h FrameHeader) Add(key, value string) {
	h[key] = append(h[key], value)
}

type Frame struct {
	complete bool
	cmd string
	headers FrameHeader
	body []byte
}

func (f *Frame) readPreface(r *bufio.Reader) error {
	var (
		s string
		err error
	)

	s, err = readLine(r)

	if err != nil {
		return err
	}

	f.cmd = s

	done := false

	for !done {
		s, err = readLine(r)

		if err != nil {
			return err
		}

		if len(s) == 0 {
			done = true
			continue
		}

		i := strings.IndexByte(s, ':')
		if i < 0 {
			return errors.New("no key/value delimiter found.")
		}
		k := s[:i]
		v := s[i + 1:]
		f.headers.Add(k, v)
	}

	return nil
}

func (f *Frame) readBody (r *bufio.Reader) (err error) {
	var s string
	s, err = r.ReadString('\000')
	fmt.Printf("body is %d %s", len(s), s)
	if err != nil {
		return
	}

	s = s[:len(s) - 1]

	f.body = []byte(s)
	f.complete = true

	return
}

func NewFrame() (f *Frame) {
	f = &Frame{
		false,
		"", 
		make(FrameHeader), 
		nil}
	return f
}
func NewFrameFromReader(r *bufio.Reader) (f *Frame, err error) {
	f = NewFrame()
	if err = f.readPreface(r); err != nil {
		return
	}

	if err = f.readBody(r); err != nil {
		return
	}

	return
}
