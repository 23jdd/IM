package tcp

import "IM/tcp/Message"

type Handler func(m *Message.Message, c *Client)

func Echo(m *Message.Message, c *Client) {
	err := c.SendBlob(0, m.Data)
	if err != nil {
		panic(err)
	}
}
