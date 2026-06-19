package tcp

import (
	"IM/tcp/Message"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type Client struct {
	uid      string
	con      net.Conn
	context  *Context
	server   *Server
	closed   bool
	worker   chan *Message.Message
	heart    chan any
	finished bool
	key      uint32
}

const WorkerSize int = 200 //
func NewClient(con net.Conn, server *Server) *Client {
	return &Client{
		con:     con,
		server:  server,
		worker:  make(chan *Message.Message, WorkerSize),
		context: NewContext(),
		heart:   make(chan any),
	}
}

func (c *Client) HeartBeat() {
	ticker := time.NewTicker(c.server.t)
	//
	for !c.closed {
		s := <-ticker.C
		fmt.Println(s.String())
		c.OnTicker()
	}
}

func (c *Client) Start() {
	err := c.server.workerPool.Submit(c.HeartBeat)
	if err != nil {
		log.Println("submit heartbeat failed:", err)
		return
	}
	err = c.server.workerPool.Submit(c.MessageHandler)
	if err != nil {
		log.Println("submit message handler failed:", err)
		return
	}
	defer c.Close()
	for !c.closed {
		message, err := c.ReadMessage()
		if err != nil {
			if errors.Is(err, io.EOF) {
				close(c.worker)
				return
			} else {
				log.Println(err)
				continue
			}
		}
		c.worker <- message
	}
}
func (c *Client) MessageHandler() {
	for !c.closed {
		select {
		case _, ok := <-c.heart:
			if !ok {
				return
			}
			c.IncrKey()
			err := c.SendHeart(c.key)
			c.SetClose(err)
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
func (c *Client) OnTicker() {
	c.heart <- nil
}
func (c *Client) Close() {
	err := c.con.Close()
	if err != nil {
		log.Println(err)
	}
	c.server.count.Add(-1)
	c.server.clients.Delete(c.uid)
}
func (c *Client) Send(message *Message.Message) error {
	_, err := c.con.Write(Message.Encode(message))
	return err
}
func (c *Client) SendJson(key uint32, target any) error {
	message, err := Message.JsonMessage(key, target)
	if err != nil {
		return err
	}
	err = c.Send(message)
	if err != nil {
		return err
	}
	return nil
}
func (c *Client) SendHeart(key uint32) error {
	message := Message.HeartMessage(key)
	err := c.Send(message)
	if err != nil {
		return err
	}
	return nil
}
func (c *Client) SendAck(key uint32) error {
	message := Message.AckMessage(key)
	err := c.Send(message)
	if err != nil {
		return err
	}
	return nil
}
func (c *Client) SendText(key uint32, text string) error {
	message := Message.TextMessage(key, text)
	err := c.Send(message)
	if err != nil {
		return err
	}
	return nil
}
func (c *Client) SendBlob(key uint32, blob []byte) error {
	message := Message.BlobMessage(key, blob)
	err := c.Send(message)
	if err != nil {
		return err
	}
	return nil
}
func (c *Client) SendAuth(key uint32, token string) error {
	message := Message.AuthMessage(key, token)
	err := c.Send(message)
	if err != nil {
		return err
	}
	return nil
}
func (c *Client) SendNack(key uint32) error {
	message := Message.NackMessage(key)
	err := c.Send(message)
	if err != nil {
		return err
	}
	return nil
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
func (c *Client) SetClose(err error) {
	if errors.Is(err, io.EOF) {
		c.closed = true
	}
}

func (c *Client) IncrKey() {
	c.key++
}
