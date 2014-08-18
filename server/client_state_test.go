package main

import (
	"testing"
	"goodyear/frame"
)

type hdr map[string]string

func BF(cmd string, headers hdr, body string) *frame.Frame {
	f := frame.NewFrame()
	f.Cmd = cmd

	for k, v := range headers {
		f.Headers.Add(k, v)
	}

	f.Body = []byte(body)
	f.Complete = true

	return f
} 



type simpleSeq struct {
	t *testing.T
	incoming chan *frame.Frame
	cs *connState
}

func (f *simpleSeq) Finish() {
	close(f.incoming)
	if _, ok := <-f.cs.outgoing; ok {
		f.t.Error("we have additional responses when we shouldn't")
	}
}

func (f *simpleSeq) Send(cmd string, headers hdr, body string) {
	req := BF(cmd, headers, body)

	f.incoming <-req
}

func (f *simpleSeq) Expect(cmd string) {
	resp := <-f.cs.outgoing
	if resp.Cmd != cmd {
		f.t.Errorf("command didn't match")
	}
}

func (f *simpleSeq) ExpectHeaders(cmd string, headers hdr) {
	resp := <-f.cs.outgoing
	if resp.Cmd != cmd {
		f.t.Errorf("command didn't match")
	}

	for k, v := range(headers) {
		if h, ok := resp.Headers.Get(k); !ok || h != v {
			f.t.Errorf("header didn't match %s: %s != %s", k, v, h)
		}
	}
}

func newSimpleSeq(t *testing.T) *simpleSeq {
	f := &simpleSeq{}
	f.incoming = make(chan *frame.Frame, 0)
	f.t = t
	f.cs = newConnState(nil, 0)

	go func() {
		getFrame := func() *frame.Frame {
			req := <-f.incoming
			return req
		}

		f.cs.HandleIncomingFrames(getFrame)
	}()

	return f
}




func TestTesting(t *testing.T) {
	s := newSimpleSeq(t)

	s.Send("CONNECT", hdr{"version": "1.1"}, "")
	s.Expect("ERROR")
	s.Send("CONNECT", hdr{"version": "1.2"}, "")
	s.Expect("CONNECTED")
	s.Send("CONNECT", hdr{"version": "1.2"}, "")
	s.Expect("ERROR")
	s.Send("DISCONNECT", hdr{"receipt": "yoh"}, "")
	s.ExpectHeaders("RECEIPT", hdr{"receipt-id": "yoh"})
	
	s.Finish()
}
