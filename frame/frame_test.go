package frame

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
)

func _F(s string) string {
	return strings.Replace(s, "\n", "\r\n", -1)
}

func _FR(s string) *bufio.Reader {
	return bufio.NewReader(strings.NewReader(_F(s)))
}

func _N(s string) string {
	s += "\x00"
	return s
}

func TestConnect(t *testing.T) {
	r := _FR(_N(`CONNECT
accept-version:1.2
host:localhost

`))
	f, err := NewFrameFromReader(r)

	if err != nil {
		t.Error("the test frame didn't parse.", err)
		t.FailNow()
	}

	if f == nil {
		t.Error("Frame is nil with no error, this shouldn't be possible.")
		t.FailNow()
	}

	if f.Cmd != "CONNECT" {
		t.Error("incorrect command parsed")
	}

	failedHeaders := false

	if len(f.Headers["accept-version"]) != 1 || f.Headers["accept-version"][0] != "1.2" {
		failedHeaders = true
	}

	if len(f.Headers["host"]) != 1 || f.Headers["host"][0] != "localhost" {
		failedHeaders = true
	}

	if failedHeaders {
		t.Error("bogus header parsing")
	}

	if !f.Complete {
		t.Error("failed to properly parse body.")
	} else {
		if len(f.Body) != 0 {
			t.Error("body should have been empty but isn't.")
		}
	}
}

func TestSend1(t *testing.T) {
	r := _FR(_N(`SEND
destination:/queue/a
content-type:text/plain

hello queue a
`))
	f, err := NewFrameFromReader(r)
	if err != nil {
		t.Error("the test frame didn't parse", err)
		t.FailNow()
	}

	if f == nil {
		t.Error("test frame wasn't allocated")
		t.FailNow()
	}

	if f.Cmd != "SEND" {
		t.Error("command was parsed incorrectly.")
	}

	if len(f.Headers) != 2 {
		t.Error("we parsed an incorrect number of headers.")
	}

	if len(f.Headers["destination"]) != 1 || f.Headers["destination"][0] != "/queue/a" {
		t.Error("we parsed destination wrong.")
	}

	if len(f.Headers["content-type"]) != 1 || f.Headers["content-type"][0] != "text/plain" {
		t.Error("we parsed content-type wrong.")
	}

	if !f.Complete {
		t.Error("we didn't find a complete frame.")
	}

	if bytes.Compare(f.Body, []byte("hello queue a\r\n")) != 0 {
		t.Error("we didn't parse the body correctly.")
	}
}

func TestBody2(t *testing.T) {
	r := _FR(_N(`CONNECT
content-length:1

a`))
	f, err := NewFrameFromReader(r)
	if err != nil {
		t.Error("didn't parse", err)
		t.FailNow()
	}

	if f == nil {
		t.Error("could produce frame")
		t.FailNow()
	}

	if !f.Complete {
		t.Error("we didn't finish the frame.")
	}

	if bytes.Compare(f.Body, []byte("a")) != 0 {
		t.Error("we didn't parse a body correctly.")
	}
}

func TestBody3(t *testing.T) {
	r := _FR(`CONNECT
content-length:3

hey`)
	f, err := NewFrameFromReader(r)
	if err == nil {
		t.Error("we shouldn't have parsed correctly.")
	}

	if f == nil {
		t.Error("could produce frame")
		t.FailNow()
	}

	if f.Complete {
		t.Error("the frame wasn't finished, why are we reporting that it did?")
	}
}

func TestHeader1(t *testing.T) {
	r := _FR(_N(`CONNECT
a:hey
b:there
a:you
a:guys
b:hot
c:stuff

`))
	f, err := NewFrameFromReader(r)

	if err != nil {
		t.Error("we should have parsed.")
		t.FailNow()
	}

	h := f.Headers

	a, _ := h.Get("a")
	b, _ := h.Get("b")
	c, _ := h.Get("c")
	t.Log(a, b, c)

	if a != "hey" || b != "there" || c != "stuff" {
		t.Error("we didn't get the header value we expected.")
	}
}
