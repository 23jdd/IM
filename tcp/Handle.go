package tcp

import (
	"IM/tcp/Message"
	"IM/utils"
	"context"
	"log"
)

type Handler func(m *Message.Message, c *Client)

func Echo(m *Message.Message, c *Client) {
	if err := c.SendBlob(m.GetKey(), m.Data); err != nil {
		log.Println("echo send error:", err)
		c.Close()
	}
}

func Verify(m *Message.Message, c *Client) {
	if m.GetMsgType() != Message.Auth {
		return
	}
	c.finished = true

	if c.UID() != "" {
		if err := c.SendNack(m.GetKey()); err != nil {
			c.Close()
		}
		return
	}

	token := string(m.Data)
	claim, err := utils.ParseToken(token)
	if err != nil {
		if err := c.SendNack(m.GetKey()); err != nil {
			c.Close()
		}
		return
	}

	c.setUID(claim.Uid)
	c.server.clients.Store(claim.Uid, c)
	if c.server.presence != nil {
		_ = c.server.presence.SetOnline(context.Background(), claim.Uid, c.server.instanceID)
	}
	if err := c.SendAck(m.GetKey()); err != nil {
		log.Println("verify ack send error:", err)
		c.Close()
	}
}
