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
	uid := c.UID()
	if uid == "" {
		return
	}

	t := m.GetMsgType()
	h, ok := bizRoutes[t]
	if !ok {
		log.Printf("no route for message type %d from %s", t, uid)
		return
	}

	h(m, c)
}
