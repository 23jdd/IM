package tcp

import (
	"IM/tcp/Message"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Client struct {
	uid       string
	con       net.Conn
	context   *Context
	server    *Server
	closed    bool
	worker    chan *Message.Message
	heart     chan any
	finished  bool
	key       uint32
	writeMu   sync.Mutex
	closeOnce sync.Once
}

const WorkerSize int = 200

func NewClient(con net.Conn, server *Server) *Client {
	return &Client{
		con:     con,
		server:  server,
		worker:  make(chan *Message.Message, WorkerSize),
		context: NewContext(),
		heart:   make(chan any, 1),
	}
}

func (c *Client) HeartBeat() {
	ticker := time.NewTicker(c.server.t)
	defer ticker.Stop()
	for !c.closed {
		select {
		case s := <-ticker.C:
			fmt.Println(s.String())
			c.OnTicker()
		}
	}
}

func (c *Client) Start() {
	go c.HeartBeat()
	go c.MessageHandler()

	defer c.Close()
	for {
		message, err := c.ReadMessage()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				close(c.worker)
				return
			}
			c.closed = true
			close(c.worker)
			return
		}
		c.worker <- message
	}
}

func (c *Client) MessageHandler() {
	for {
		select {
		case _, ok := <-c.heart:
			if !ok {
				return
			}
			c.IncrKey()
			if err := c.SendHeart(c.key); err != nil {
				c.closed = true
				log.Println(err)
				return
			}
		case message, ok := <-c.worker:
			if !ok {
				return
			}
			for _, h := range c.server.clientHandlers {
				h(message, c)
				if c.finished {
					break
				}
			}
			c.finished = false
		}
	}
}

func (c *Client) Process(m *Message.Message) {
	c.worker <- m
}

func (c *Client) Context() *Context {
	return c.context
}

func (c *Client) UID() string {
	return c.uid
}

func (c *Client) OnTicker() {
	select {
	case c.heart <- nil:
	default:
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.closed = true
		if c.uid != "" {
			c.server.clients.Delete(c.uid)
		}
		c.server.count.Add(-1)
		if err := c.con.Close(); err != nil {
			log.Println("close conn:", err)
		}
	})
}

func (c *Client) Send(message *Message.Message) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err := c.con.Write(Message.Encode(message))
	return err
}

func (c *Client) SendJson(key uint32, target any) error {
	message, err := Message.JsonMessage(key, target)
	if err != nil {
		return err
	}
	return c.Send(message)
}

func (c *Client) SendHeart(key uint32) error {
	message := Message.HeartMessage(key)
	return c.Send(message)
}

func (c *Client) SendAck(key uint32) error {
	message := Message.AckMessage(key)
	return c.Send(message)
}

func (c *Client) SendText(key uint32, text string) error {
	message := Message.TextMessage(key, text)
	return c.Send(message)
}

func (c *Client) SendBlob(key uint32, blob []byte) error {
	message := Message.BlobMessage(key, blob)
	return c.Send(message)
}

func (c *Client) SendAuth(key uint32, token string) error {
	message := Message.AuthMessage(key, token)
	return c.Send(message)
}

func (c *Client) SendNack(key uint32) error {
	message := Message.NackMessage(key)
	return c.Send(message)
}

func (c *Client) ReadMessage() (*Message.Message, error) {
	header := c.server.bufPool.Get(8)
	_, err := io.ReadFull(c.con, header)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header[4:8])
	buf := c.server.bufPool.Get(int(length) + 8)
	copy(buf, header)
	c.server.bufPool.Put(header)
	_, err = io.ReadFull(c.con, buf[8:])
	if err != nil {
		return nil, err
	}
	message, err := Message.Decode(buf)
	c.server.bufPool.Put(buf)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (c *Client) IncrKey() {
	c.key++
}
