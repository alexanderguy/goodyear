package main

import (
	"bufio"
	"container/list"
	"goodyear/frame"
	// XXX - We need to not use this directly,
	// since we need to support levels.
	"log"
	"net"
	"sync"
	"goodyear/dest"
)

type serverState struct {
	connsLock sync.RWMutex
	conns     *list.List
	serial    int
}

const LISTENING_ADDR = ":61613"

func main() {
	state := serverState{}
	state.serial = 0
	state.conns = list.New()

	d := dest.NewBroadcast()
	dest.AddDest("everyone", d)

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
		cs := newClientState(state.serial)
		state.serial += 1
		state.connsLock.Lock()
		thisConn := state.conns.PushBack(cs)
		state.connsLock.Unlock()
		log.Printf("accepting connection %d", cs.id)

		// Outgoing Frames
		go func(conn net.Conn, cs *clientState, myElement *list.Element) {
			defer func() {
				log.Print("taking down conn ", cs.id)
				state.connsLock.Lock()
				state.conns.Remove(myElement)
				state.connsLock.Unlock()
				conn.Close()
			}()

			for f := range cs.outgoing {
				b := f.Bytes()
				n, err := conn.Write(b)

				if err != nil {
					log.Printf("Error writing to conn %d: %s", cs.id, err)
					cs.phase = errorPhase
				} else if n != len(b) {
					log.Printf("Short write while sending to client conn %d", cs.id)
					cs.phase = errorPhase
				}
			}
		}(conn, cs, thisConn)

		// Incoming Frame Processing
		go func(conn net.Conn, cs *clientState) {
			r := bufio.NewReader(conn)

			getFrame := func() *frame.Frame {
				f, err := frame.NewFrameFromReader(r)
				if err == nil {
					return f
				}
				log.Printf("Failed parsing frame: %s", err)
				return nil
			}

			cs.HandleIncomingFrames(getFrame)
		}(conn, cs)
	}
}
