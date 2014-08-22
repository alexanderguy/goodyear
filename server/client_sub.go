package main

import (
	"goodyear/dest"
)

type clientSubAckMode int

const (
	ackModeAuto clientSubAckMode = iota
	ackModeClient
	ackModeClientIndividual
)

type clientSub struct {
	client  *clientState
	id      string
	dest    dest.DestId
	ackMode clientSubAckMode
}

type clientSubMessage struct {
	sub *clientSub
	msg *dest.Message
}

func (sub *clientSub) Send(m *dest.Message) error {
	v := &clientSubMessage{sub, m}
	sub.client.incomingMsgs <- v

	return nil
}
