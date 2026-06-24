package tcp

import (
	"IM/tcp/Message"
	"IM/utils"
	"context"
	"log"
)

// Handler 客户端消息处理器函数类型，按注册顺序组成处理链。
type Handler func(m *Message.Message, c *Client)

// Echo 回显处理器：将收到的消息体原样发回客户端。
func Echo(m *Message.Message, c *Client) {
	if err := c.SendBlob(m.GetKey(), m.Data); err != nil {
		log.Println("echo send error:", err)
		c.Close()
	}
}

// Verify 鉴权处理器：校验 Auth 帧中的 token，成功则绑定 uid 并登记在线。
func Verify(m *Message.Message, c *Client) {
	// 非鉴权帧不处理，交由后续处理器。
	if m.GetMsgType() != Message.Auth {
		return
	}
	c.finished = true // 鉴权帧已消费，短路后续处理器

	// 已鉴权过的连接再次发 Auth 视为非法，回 Nack。
	if c.UID() != "" {
		if err := c.SendNack(m.GetKey()); err != nil {
			c.Close()
		}
		return
	}

	// 解析并校验 JWT token，失败则回 Nack。
	token := string(m.Data)
	claim, err := utils.ParseToken(token)
	if err != nil {
		if err := c.SendNack(m.GetKey()); err != nil {
			c.Close()
		}
		return
	}

	// 鉴权通过：绑定 uid、登记到本实例连接表，并写入在线注册表。
	c.setUID(claim.Uid)
	c.server.addClient(claim.Uid, c)
	if c.server.presence != nil {
		_ = c.server.presence.SetOnline(context.Background(), claim.Uid, c.server.instanceID)
	}
	if err := c.SendAck(m.GetKey()); err != nil {
		log.Println("verify ack send error:", err)
		c.Close()
	}
}
