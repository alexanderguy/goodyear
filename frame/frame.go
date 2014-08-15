package frame

import (
	"bufio"
	"strings"
	"errors"
	"strconv"
	"fmt"
	"bytes"
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
	Complete bool
	Cmd string
	Headers FrameHeader
	Body []byte
}

func (f *Frame) readPreface(r *bufio.Reader) error {
	var (
		s string
		err error
	)

	var done bool

	// Consume any newlines that might have come
	// in after a previous null.
	done = false
	for !done {
		s, err = readLine(r)

		if err != nil {
			return err
		}

		if len(s) != 0 {
			done = true
		}
	}

	f.Cmd = s

	// Grab any headers.
	done = false
	for !done {
		s, err = readLine(r)

		if err != nil {
			return err
		}

		// If we find an empty line, we're done with headers.
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
		f.Headers.Add(k, v)
	}

	return nil
}

func (f *Frame) readBody (r *bufio.Reader) error {
	var (
		err error
		s string
	)

	if val, exists := f.Headers["content-length"]; exists {
		var (
			v int
			c byte
		)

		if v, err = strconv.Atoi(val[0]); err != nil {
			return err
		}

		fmt.Printf("content length is %d", v)
		b := make([]byte, v)
		var count int

		count, err = r.Read(b)
		if err != nil {
			return err
		}

		if count != v {
			return errors.New("couldn't read frame body")
		}

		if c, err = r.ReadByte(); err != nil || c != '\x00' {
			return errors.New("body incorrectly null terminated")
		}

		f.Body = b
	} else {
		s, err = r.ReadString('\000')
		if err != nil {
			return err
		}

		s = s[:len(s) - 1]
		f.Body = []byte(s)
	}

	f.Complete = true

	return nil
}

func (f *Frame) ToNetwork() []byte {
	var buf bytes.Buffer
	buf.Write([]byte(fmt.Sprintf("%s\r\n", f.Cmd)))
	for k, v := range f.Headers {
		for _, w := range v {
			buf.Write([]byte(fmt.Sprintf("%s:%s\r\n", k, w)))
		}
	}

	buf.Write([]byte("\r\n"))
	buf.Write(f.Body)
	buf.Write([]byte("\x00"))

	return buf.Bytes()
}

func NewFrame() *Frame {
	f := &Frame{
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
