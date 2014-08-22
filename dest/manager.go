package dest

import (
	"errors"
	"goodyear/frame"
	"sync"
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
	m.Id = getNextMessageId()

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

func getNextMessageId() uint64 {
	defer destManager.messageIdLock.Unlock()
	destManager.messageIdLock.Lock()
	v := destManager.nextMessageId

	destManager.nextMessageId++

	return v

}

type destNamespace struct {
	dests map[DestId]Dest
	messageIdLock sync.RWMutex
	nextMessageId uint64
}

var destManager *destNamespace

func init() {
	destManager = &destNamespace{}
	destManager.dests = make(map[DestId]Dest)
}
