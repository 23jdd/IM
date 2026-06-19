package tcp

import "IM/tcp/Message"

type Handler func(m *Message.Message, c *Client)
