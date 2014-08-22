package dest

import (
	"errors"
	"goodyear/frame"
)

type Sub interface {
	Send(*Message) error
}

type DestId string

type Dest interface {
	Subscribe(Sub) error
	Unsubscribe(Sub) error
	Send(*Message) error
}

func Subscribe(id DestId, s Sub) error {
	if dst, exists := destManager.dests[id]; exists {
		return dst.Subscribe(s)
	}

	return nil
}

func Unsubscribe(id DestId, s Sub) error {
	if dst, exists := destManager.dests[id]; exists {
		return dst.Unsubscribe(s)
	}

	return nil
}

func Send(id DestId, f *frame.Frame) error {
	m := NewMessage(f)

	if dst, exists := destManager.dests[id]; exists {
		return dst.Send(m)
	}

	return nil
}

func AddDest(id DestId, d Dest) error {
	if _, exists := destManager.dests[id]; exists {
		return errors.New("destination already exists")
	}

	destManager.dests[id] = d

	return nil
}

type destNamespace struct {
	dests map[DestId]Dest
}

var destManager *destNamespace

func init() {
	destManager = &destNamespace{
		make(map[DestId]Dest),
	}
}
