package main

import (
	"testing"
	"goodyear/frame"
)

func BF(cmd string, headers map[string]string, body string) *frame.Frame {
	f := frame.NewFrame()
	f.Cmd = cmd

	for k, v := range headers {
		f.Headers.Add(k, v)
	}

	f.Body = []byte(body)
	f.Complete = true

	return f
} 

type hdr map[string]string


func Check(t *testing.T, f *frame.Frame, cmd string) {
	if f.Cmd != cmd {
		t.Errorf("command didn't match")
	}
}

func TestTesting(t *testing.T) {
	incoming := make(chan *frame.Frame, 0)
	defer func() {
		close(incoming)
	}()

	cs := newConnState(nil, 0)

	go func() {
		getFrame := func() *frame.Frame {
			f := <-incoming
			return f
		}

		cs.HandleIncomingFrames(getFrame)
	}()

	incoming<-BF("CONNECT", hdr{"version": "1.1"}, "")
	Check(t, <-cs.outgoing, "ERROR")
	incoming<-BF("CONNECT", hdr{"version": "1.2"}, "")
	Check(t, <-cs.outgoing, "CONNECTED")
	incoming<-BF("CONNECT", hdr{"version": "1.2"}, "")
	Check(t, <-cs.outgoing, "ERROR")
	incoming<-BF("DISCONNECT", hdr{"receipt": "yoh"}, "")
	Check(t, <-cs.outgoing, "RECEIPT")
	
	if _, ok := <-cs.outgoing; ok {
		t.Errorf("Outgoing channel should be closed.")
	}
}
