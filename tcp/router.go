package tcp

import (
	"IM/tcp/Message"
	"log"
)

type BusinessHandler func(m *Message.Message, c *Client)

var bizRoutes = make(map[byte]BusinessHandler)

func RegisterRoute(msgType byte, h BusinessHandler) {
	bizRoutes[msgType] = h
}

func Router(m *Message.Message, c *Client) {
	if c.uid == "" {
		return
	}

	t := m.GetMsgType()
	h, ok := bizRoutes[t]
	if !ok {
		log.Printf("no route for message type %d from %s", t, c.uid)
		return
	}

	h(m, c)
}
