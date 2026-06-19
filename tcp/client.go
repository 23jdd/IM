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
	finished bool
}

const WorkerSize int = 200 //
func NewClient(con net.Conn, server *Server) *Client {
	return &Client{
		con:     con,
		server:  server,
		worker:  make(chan *Message.Message, WorkerSize),
		context: NewContext(),
	}
}

func (c *Client) HeartBeat() {
	ticker := time.NewTicker(c.server.t)
	//
	for {
		s := <-ticker.C
		fmt.Println(s.String())
		c.OnTicker()
	}
}

func (c *Client) Start() {
	go c.HeartBeat()
	go c.MessageHandler()
	for {
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
	for {
		message, ok := <-c.worker
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

func (c *Client) Process(m *Message.Message) {
	c.worker <- m
}

func (c *Client) Context() *Context {
	return c.context
}
func (c *Client) OnTicker() {
	fmt.Println("OnTicker")
}
func (c *Client) Close() {
	err := c.con.Close()
	if err != nil {
		log.Println(err)
	}
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
	header := c.server.pool.Get(8)
	_, err := io.ReadFull(c.con, header)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(header[4:8])
	buf := c.server.pool.Get(int(length) + 8)
	copy(buf, header)
	c.server.pool.Put(header)
	_, err = io.ReadFull(c.con, buf[8:])
	if err != nil {
		return nil, err
	}
	message, err := Message.Decode(buf)
	c.server.pool.Put(buf)
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
