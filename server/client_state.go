package main

import (
	"goodyear/dest"
	"goodyear/frame"
	// XXX - We need to not use this directly,
	// since we need to support levels.
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
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
	subs     map[string]*clientSub
	incomingMsgs chan *clientSubMessage
	ackId    int
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

func (cs *clientState) handleCmdSubscribe(f *frame.Frame) {
	s := &clientSub{}
	s.client = cs

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

	if dst, ok := f.Headers.Get("destination"); ok && len(dst) > 1 {
		s.dest = dest.DestId(dst)
	} else {
		cs.ErrorString("a destination headers is required for SUBSCRIBE")
		return
	}

	if _, exists := cs.subs[s.id]; exists {
		cs.ErrorString(fmt.Sprintf("a subscription IDed '%s' already exists.", s.id))
		return
	}

	if err := dest.Subscribe(s.dest, s); err != nil {
		cs.ErrorString(fmt.Sprintf("failed to subscribe '%s'", err))
		return
	}

	cs.subs[s.id] = s
}

func (cs *clientState) handleCmdUnsubscribe(curFrame *frame.Frame) {
	if id, ok := curFrame.Headers.Get("id"); ok {
		if sub, exists := cs.subs[id]; exists {
			dest.Unsubscribe(sub.dest, sub)
			delete(cs.subs, id)
		} else {
			cs.ErrorString(fmt.Sprintf("subscription id '%s' doesn't exist.", id))
		}
	} else {
		cs.ErrorString("an id is required to UNSUBSCRIBE.")
	}
}

type frameProvider func() *frame.Frame

func (cs *clientState) HandleIncomingFrames(getFrame frameProvider) {
	defer func() {
		for _, sub := range(cs.subs) {
			dest.Unsubscribe(sub.dest, sub)
		}

		// Clean up everything.
		close(cs.outgoing)
		close(cs.incomingMsgs)
	}()

	go func() {
		for subMsg := range cs.incomingMsgs {
			sub := subMsg.sub
			msg := subMsg.msg

			f := frame.NewFrame()
			f.Cmd = "MESSAGE"
			f.Headers.Add("subscription", sub.id)
			if sub.ackMode != ackModeAuto {
				f.Headers.Add("ack", strconv.FormatUint(uint64(cs.ackId), 10))
				cs.ackId++
			}

			for k, values := range msg.Frame.Headers {
				for _, v := range values {
					f.Headers.Add(k, v)
				}
			}

			f.Body = subMsg.msg.Frame.Body

			// XXX - We need to implement ack handling here.

			cs.outgoing <-f
		}
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
			cs.handleCmdSubscribe(curFrame)

		case "UNSUBSCRIBE":
			cs.handleCmdUnsubscribe(curFrame)

		case "SEND":
			dst, ok := curFrame.Headers.Get("destination")
			if !ok {
				cs.ErrorString("SEND requires a destination.")
			}

			log.Printf("conn %d sending to destination %s", cs.id, dst)
			dest.Send(dest.DestId(dst), curFrame)
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
	cs.ackId = 0
	cs.version = ""
	cs.outgoing = make(chan *frame.Frame, 0)
	cs.subs = make(map[string]*clientSub)
	cs.incomingMsgs = make(chan *clientSubMessage)

	return cs
}
