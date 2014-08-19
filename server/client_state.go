package main

import (
	"goodyear/frame"
	// XXX - We need to not use this directly,
	// since we need to support levels.
	"log"
	"strconv"
	"strings"
)

type clientStatePhase int

const (
	opened clientStatePhase = iota
	connected
	disconnected
	errorPhase
)

type clientState struct {
	phase    clientStatePhase
	id       int
	version  string
	outgoing chan *frame.Frame
}

func (cs *clientState) Error(ct string, body []byte) error {
	f := frame.NewFrame()

	f.Cmd = "ERROR"
	f.Body = body

	f.Headers.Add("content-type", ct)
	f.Headers.Add("content-length", strconv.FormatUint(uint64(len(f.Body)), 10))

	cs.outgoing <- f
	cs.phase = errorPhase

	return nil
}

func (cs *clientState) ErrorString(msg string) error {
	msg += "\r\n"
	return cs.Error("text/plain", []byte(msg))
}

type frameProvider func() *frame.Frame

func (cs *clientState) HandleIncomingFrames(getFrame frameProvider) {
	defer func() {
		// Signal the outgoing goroutine to close things out.
		close(cs.outgoing)
	}()

	var curFrame *frame.Frame
	processFrame := func() {
		curFrame = getFrame()
		if curFrame != nil {
			log.Printf("conn %d cmd %s", cs.id, curFrame.Cmd)
			return
		}

		curFrame = nil
		cs.ErrorString("failed to parse frame.  good bye!")
		return
	}

	handleReceipt := func() {
		if v, ok := curFrame.Headers.Get("receipt"); ok {
			resp := frame.NewFrame()
			resp.Cmd = "RECEIPT"
			resp.Headers.Add("receipt-id", v)
			cs.outgoing <- resp
		}
	}

	// Before connection.
	for cs.phase == opened {
		processFrame()
		if curFrame == nil {
			return
		}

		switch curFrame.Cmd {
		case "CONNECT", "STOMP":
			if _, ok := curFrame.Headers.Get("receipt"); ok {
				cs.ErrorString("receipt not allowed during connect.")
				break
			}

			supVersion, ok := curFrame.Headers.Get("version")

			if !ok {
				cs.ErrorString("a version header is required")
				break
			}

			validVersion := false

			for _, v := range strings.Split(supVersion, ",") {
				if v == "1.2" {
					validVersion = true
				}
			}

			if !validVersion {
				cs.ErrorString("this server only supports standard version 1.2")
				break
			}

			cs.version = "1.2"
			cs.phase = connected
			resp := frame.NewFrame()
			resp.Cmd = "CONNECTED"
			resp.Headers.Add("version", cs.version)
			if _, ok := curFrame.Headers.Get("heart-beat"); ok {
				resp.Headers.Add("heart-beat", "0,0")
			}

			cs.outgoing <- resp
		default:
			cs.ErrorString("unknown/unallowed command.")
		}
	}

	// Now we're connected.
	for cs.phase == connected {
		processFrame()
		if curFrame == nil {
			return
		}

		switch curFrame.Cmd {
		case "CONNECT":
			cs.ErrorString("you're already connected.")

		case "DISCONNECT":
			log.Printf("conn %d requested disconnect", cs.id)
			cs.phase = disconnected

		default:
			cs.ErrorString("unknown command.")
		}

		handleReceipt()
	}
}

func newClientState(connId int) *clientState {
	cs := &clientState{}
	cs.phase = opened
	cs.id = connId
	cs.version = ""
	cs.outgoing = make(chan *frame.Frame, 0)

	return cs
}
