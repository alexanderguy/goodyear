package main

import (
	"container/list"
	"errors"
	"goodyear/frame"
	// XXX - We need to not use this directly,
	// since we need to support levels.
	"net"
	"strconv"
)

type connStatePhase int

const (
	unknown connStatePhase = iota
	connected
	disconnected
)

type connState struct {
	phase   connStatePhase
	conn    net.Conn
	id      int
	me      *list.Element
	version string
	outgoing chan *frame.Frame
}

func (cs *connState) WriteFrame(f *frame.Frame) error {
	b := f.ToNetwork()
	n, err := cs.conn.Write(b)

	if err == nil && n != len(b) {
		err = errors.New("failed to flush complete write to network.")
	}

	return err
}

func (cs *connState) Error(ct string, body []byte) error {
	f := frame.NewFrame()

	f.Cmd = "ERROR"
	f.Body = body

	f.Headers.Add("content-type", ct)
	f.Headers.Add("content-length", strconv.FormatUint(uint64(len(f.Body)), 10))

	cs.outgoing <- f
	return nil
}

func (cs *connState) ErrorString(msg string) error {
	msg += "\r\n"
	return cs.Error("text/plain", []byte(msg))
}

func newConnState(conn net.Conn, connId int) *connState{
	cs := &connState{}
	cs.phase = unknown
	cs.conn = conn
	cs.id = connId
	cs.version = ""
	cs.outgoing = make(chan *frame.Frame, 0)


	return cs
}
