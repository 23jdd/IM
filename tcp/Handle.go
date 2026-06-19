package tcp

import (
	"IM/tcp/Message"
	"IM/utils"
)

const Finsh string = "Finsh"

type Handler func(m *Message.Message, c *Client)

func Echo(m *Message.Message, c *Client) {
	err := c.SendBlob(0, m.Data)
	if err != nil {
		panic(err)
	}
}
func Verify(m *Message.Message, c *Client) {
	if m.GetMsgType() == Message.Auth {
		c.finished = true
		if c.uid == "" {
			token := string(m.Data)
			claim, err := utils.ParseToken(token)
			if err != nil {
				err := c.SendNack(m.GetKey())
				c.SetClose(err)
				return
			}
			c.uid = claim.Uid
			err = c.SendAck(m.GetKey())
			c.SetClose(err)
		}
	}
}
