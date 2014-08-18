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
	"strings"
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

		// Outgoing Frames
		go func(cs *connState) {
			defer func() {
				log.Print("taking down conn ", cs.id)
				state.conns.Remove(cs.me)
				cs.conn.Close()
			}()

			for f := range cs.outgoing {
				err := cs.WriteFrame(f)
				if err != nil {
					log.Printf("Error writing to conn %d: %s", cs.id, err)
					return
				}
			}

		}(cs)

		// Incoming Frame Processing
		go func(cs *connState) {
			defer func() {
				// Signal the outgoing goroutine to close things out.
				close(cs.outgoing)
			}()

			r := bufio.NewReader(cs.conn)

			processFrame := func() (*frame.Frame) {
				f, err := frame.NewFrameFromReader(r)
				if err == nil {
					return f
				}

				cs.phase = unknown
				log.Printf("Failed parsing frame, dropping conn: %s", err)
				cs.ErrorString("failed to parse frame.  good bye!")
				return nil
			}

			// Before connection.
			for cs.phase != connected {
				f := processFrame()
				if f == nil {
					return
				}

				switch f.Cmd {
				case "CONNECT":
					supVersion, ok := f.Headers.Get("version")

					if !ok {
						cs.ErrorString("a version header is required")
					} else {
						validVersion := false

						for _, v := range(strings.Split(supVersion, ",")) {
							if v == "1.2" {
								validVersion = true
							}
						}

						if !validVersion {
							cs.ErrorString("this server only supports standard version 1.2")
						} else {
							cs.phase = connected
							rf := frame.NewFrame()
							rf.Cmd = "CONNECTED"
							cs.outgoing <- rf
						}
					}

				default:
					cs.ErrorString("unknown/unallowed command.")
				}
			}

			// Now we're connected.
			for cs.phase != disconnected {
				f := processFrame()
				if f == nil {
					return
				}

				switch f.Cmd {
				case "CONNECT":
					cs.ErrorString("you're already connected.")

				case "DISCONNECT":
					log.Printf("conn %d requested disconnect", cs.id)
					cs.phase = disconnected

				default:
					cs.ErrorString("unknown command.")
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

				if v, ok := f.Headers.Get("receipt"); ok {
					if f.Cmd == "CONNECT" {
						cs.ErrorString("receipt-id not allowed during connect.")
					} else {
						f := frame.NewFrame()
						f.Cmd = "RECEIPT"
						f.Headers.Add("receipt-id", v)
						cs.outgoing <- f
					}
				}
			}
		}(cs)
	}
}
