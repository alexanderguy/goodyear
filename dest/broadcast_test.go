package dest

import (
	"goodyear/frame"
	"testing"
)

type MockSub struct {
	t  *testing.T
	id string
}

func (s *MockSub) Id() string {
	return s.id
}

func (s *MockSub) Send(m *Message) error {
	return nil
}

func TestAssoc1(t *testing.T) {
	b := NewBroadcast()

	s1 := &MockSub{t, "s1"}
	s2 := &MockSub{t, "s2"}

	if err := b.Subscribe(s1); err != nil {
		t.Error("failed to subscribe")
		t.FailNow()

		return
	}

	if err := b.Subscribe(s2); err != nil {
		t.Error("failed to subscribe")
		t.FailNow()

		return
	}

	if err := b.Subscribe(s1); err == nil {
		t.Error("we shouldn't have been able to subscribe")
		t.FailNow()

		return
	}

	if err := b.Unsubscribe(s1); err != nil {
		t.Error("failed to unsubscribe")
		t.FailNow()

		return
	}

	if err := b.Unsubscribe(s2); err != nil {
		t.Error("failed to unsubscribe")
		t.FailNow()

		return
	}

	if err := b.Unsubscribe(s1); err == nil {
		t.Error("we should have failed to unsubscribe")
		t.FailNow()

		return
	}

	if err := b.Unsubscribe(s2); err == nil {
		t.Error("we should have failed to unsubscribe")
		t.FailNow()

		return
	}
}

func TestSend1(t *testing.T) {
	b := NewBroadcast()

	s1 := &MockSub{t, "s1"}

	if err := b.Subscribe(s1); err != nil {
		t.Error("why did this subscription fail?")
		t.FailNow()
	}

	s2 := &MockSub{t, "s2"}

	if err := b.Subscribe(s2); err != nil {
		t.Error("why did this subscription fail?")
		t.FailNow()
	}

	f := frame.NewFrame()
	m := NewMessage(f)

	if err := b.Send(m); err != nil {
		t.Error("there shouldn't have been an error sending.")
		t.FailNow()
	}
}
