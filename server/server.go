package main

import (
	// XXX - We need to not use this directly,
	// since we need to support levels.
	"bufio"
	"container/list"
	"log"
	"net"
	"goodyear/frame"
)

type connState struct {
	conn net.Conn
	id   int
	me   *list.Element
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
		cs := &connState{conn, state.serial, nil}
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
