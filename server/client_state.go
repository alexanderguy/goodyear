package main

import (
	"goodyear/frame"
	// XXX - We need to not use this directly,
	// since we need to support levels.
	"fmt"
	"log"
	"strconv"
	"strings"
)

type clientSubAckMode int

const (
	ackModeAuto clientSubAckMode = iota
	ackModeClient
	ackModeClientIndividual
)

type clientSub struct {
	id      string
	dest    string
	ackMode clientSubAckMode
}

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

func (cs *clientState) handleCmdSUBSCRIBE(f *frame.Frame) {
	s := clientSub{}

	if id, ok := f.Headers.Get("id"); ok && len(id) > 0 {
		s.id = id
	} else {
		cs.ErrorString("id header required on SUBSCRIBE")
		return
	}

	if ack, ok := f.Headers.Get("ack"); ok {
		switch ack {
		case "auto":
			s.ackMode = ackModeAuto
		case "client":
			s.ackMode = ackModeClient
		case "client-individual":
			s.ackMode = ackModeClientIndividual
		default:
			cs.ErrorString(fmt.Sprintf("ack mode '%s' invalid on SUBSCRIBE", ack))
			return
		}
	} else {
		s.ackMode = ackModeAuto
	}

	if dest, ok := f.Headers.Get("destination"); ok && len(dest) > 1 {
		s.dest = dest
	} else {
		cs.ErrorString("a destination headers is required for SUBSCRIBE")
	}
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

		if err := curFrame.ValidateFrame(); err != nil {
			cs.ErrorString(err.Error())
			break
		}

		switch curFrame.Cmd {
		case "CONNECT", "STOMP":
			if _, ok := curFrame.Headers.Get("receipt"); ok {
				cs.ErrorString("receipt not allowed during connect.")
				break
			}

			supVersion, ok := curFrame.Headers.Get("accept-version")

			if !ok {
				cs.ErrorString("an accept-version header is required")
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

		if err := curFrame.ValidateFrame(); err != nil {
			cs.ErrorString(err.Error())
			break
		}

		switch curFrame.Cmd {
		case "CONNECT":
			cs.ErrorString("you're already connected.")

		case "DISCONNECT":
			log.Printf("conn %d requested disconnect", cs.id)
			cs.phase = disconnected

		case "SUBSCRIBE":
			cs.handleCmdSUBSCRIBE(curFrame)

		case "SEND":
			dest, ok := curFrame.Headers.Get("destination")
			if !ok {
				cs.ErrorString("SEND requires a destination.")
			}

			log.Printf("conn %d sending to destination %s", cs.id, dest)

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
