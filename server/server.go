package main

import (
	"bufio"
	"container/list"
	"goodyear/frame"
	// XXX - We need to not use this directly,
	// since we need to support levels.
	"log"
	"net"
)

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
			r := bufio.NewReader(cs.conn)

			getFrame := func() *frame.Frame {
				f, err := frame.NewFrameFromReader(r)
				if err == nil {
					return f
				}
				log.Printf("Failed parsing frame: %s", err)
				return nil
			}

			cs.HandleIncomingFrames(getFrame)
		}(cs)
	}
}
