package main

import (
	"bufio"
	"container/list"
	"errors"
	"goodyear/frame"
	// XXX - We need to not use this directly,
	// since we need to support levels.
	"log"
	"net"
	"strconv"
)

type connStatePhase int

const (
	disconnected connStatePhase = iota
	connected
)

type connState struct {
	phase   connStatePhase
	conn    net.Conn
	id      int
	me      *list.Element
	version string
}

func (cs *connState) WriteFrame(f *frame.Frame) error {
	b := f.ToNetwork()
	n, err := cs.conn.Write(b)

	if err == nil && n != len(b) {
		err = errors.New("failed to flush complete write to network.")
	}

	return err
}

func (cs *connState) Receipt(id string) error {
	f := frame.NewFrame()
	f.Cmd = "RECEIPT"
	f.Headers.Add("receipt-id", id)

	err := cs.WriteFrame(f)
	return err
}

func (cs *connState) Error(ct string, body []byte) error {
	f := frame.NewFrame()

	f.Cmd = "ERROR"
	f.Body = body

	f.Headers.Add("content-type", ct)
	f.Headers.Add("content-length", strconv.FormatUint(uint64(len(f.Body)), 10))

	err := cs.WriteFrame(f)

	return err
}

func (cs *connState) ErrorString(msg string) error {
	msg += "\r\n"
	return cs.Error("text/plain", []byte(msg))
}

func newConnState(conn net.Conn, connId int) *connState{
	cs := &connState{}
	cs.phase = disconnected
	cs.conn = conn
	cs.id = connId
	cs.version = ""

	return cs
}

type serverState struct {
	conns  *list.List
	serial int
}

const LISTENING_ADDR = ":61613"

func main() {
	state := serverState{list.New(), 0}

	log.Print("Listening on address ", LISTENING_ADDR)
	l, err := net.Listen("tcp", LISTENING_ADDR)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		cs := newConnState(conn, state.serial)
		state.serial += 1
		cs.me = state.conns.PushBack(cs)
		log.Printf("accepting connection %d", cs.id)

		go func(cs *connState) {
			defer func() {
				log.Print("taking down conn ", cs.id)
				state.conns.Remove(cs.me)
				cs.conn.Close()
			}()

			r := bufio.NewReader(cs.conn)
			for {
				f, err := frame.NewFrameFromReader(r)
				if err != nil {
					log.Printf("Failed parsing frame, dropping conn: %s", err)
					cs.ErrorString("failed to parse frame.  good bye!")
					return
				}

				switch f.Cmd {
				case "CONNECT":
					switch cs.phase {
					case connected:
						cs.ErrorString("you're already connected.")
					case disconnected:
						cs.phase = connected
						rf := frame.NewFrame()
						rf.Cmd = "CONNECTED"
						cs.WriteFrame(rf)
					}

				case "DISCONNECT":
					log.Printf("conn %d requested disconnect", cs.id)
					cs.phase = disconnected
					cs.Receipt("derp")
					return
				}

				// log.Printf("from %d got: %v", cs.id, data)
				// for e := state.conns.Front(); e != nil; e = e.Next() {
				// 	t := e.Value.(*connState)
				// 	_, err := t.conn.Write(data)
				// 	if err != nil {
				// 		log.Fatal("had an error while writing!")
				// 	}
				// }

				log.Printf("conn %d cmd %s", cs.id, f.Cmd)
			}
		}(cs)
	}
}
