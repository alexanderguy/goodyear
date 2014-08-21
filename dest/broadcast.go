package dest

import (
	"errors"
	"sync"
)

type Broadcast struct {
	subsLock sync.RWMutex
	subs []Sub
}

func (b *Broadcast) Subscribe(s Sub) error {
	b.subsLock.Lock()
	defer b.subsLock.Unlock()

	for _, v := range(b.subs) {
		if v == s {
			return errors.New("this subscription has already been created")
		}
	}

	b.subs = append(b.subs, s)

	return nil
}

func (b *Broadcast) Unsubscribe(s Sub) error {
	b.subsLock.Lock()
	defer b.subsLock.Unlock()

	count := len(b.subs)
	for i := 0; i < count; i++ {
		if b.subs[i] == s {
			b.subs[i], b.subs[count - 1] = b.subs[count - 1], b.subs[i]
			count--
		}
	}

	if count == len(b.subs) {
		return errors.New("this subscription didn't appear to be subscribed.")
	}

	b.subs = b.subs[:count:count]

	return nil
}

func (b *Broadcast) Send(m *Message) error {
	b.subsLock.RLock()
	defer b.subsLock.RUnlock()

	for _, sub := range b.subs {
		sub.Send(m)
	}

	return nil
}

func NewBroadcast() *Broadcast {
	b := &Broadcast{}
	b.subs = make([]Sub, 0)

	return b
}
