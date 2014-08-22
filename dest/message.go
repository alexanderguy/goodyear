package dest

import (
	"goodyear/frame"
)

type Message struct {
	Frame *frame.Frame
	Id uint64
}

func Ack(m *Message) {
}

func Nack(m *Message) {
}

func NewMessage(f *frame.Frame) *Message {
	m := &Message{}
	m.Frame = f

	return m
}
