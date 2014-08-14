package frame

import (
	"testing"
	"strings"
	"bufio"
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

	if f.cmd != "CONNECT" {
		t.Error("incorrect command parsed")
	}

	failedHeaders := false

	if len(f.headers["accept-version"]) != 1 || f.headers["accept-version"][0] != "1.2" {
		failedHeaders = true
	}

	if len(f.headers["host"]) != 1 || f.headers["host"][0] != "localhost" {
		failedHeaders = true
	}

	if failedHeaders {
		t.Error("bogus header parsing")	
	}


	if !f.complete {
		t.Error("failed to properly parse body.")
	} else {
		if len(f.body) != 0 {
			t.Error("body should have been empty but isn't.")
		}
	}
}

