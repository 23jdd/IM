package tcp

import (
	"IM/tcp/Message"
	"log"
)

// BusinessHandler 业务消息处理器函数类型，按消息类型分发。
type BusinessHandler func(m *Message.Message, c *Client)

// bizRoutes 消息类型到业务处理器的路由表。
var bizRoutes = make(map[byte]BusinessHandler)

// RegisterRoute 注册某消息类型对应的业务处理器。
func RegisterRoute(msgType byte, h BusinessHandler) {
	bizRoutes[msgType] = h
}

// Router 按消息类型把帧分发给对应业务处理器（仅处理已鉴权连接）。
func Router(m *Message.Message, c *Client) {
	// 未鉴权（无 uid）的连接不进入业务路由。
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
