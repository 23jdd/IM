package tcp

import (
	"IM/tcp/Message"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type Handler func()
type Client struct {
	uid     uint32 //  user id
	con     net.Conn
	context any // set expire timer
	server  *Server
	closed  bool
}

func NewClient(con net.Conn) *Client {
	return &Client{
		con: con,
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
	for {
		c.ReadMessage()
	}
}
func (c *Client) SetContext(ctx any) {
	c.context = ctx
}
func (c *Client) Context() any {
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
func (c *Client) ReadMessage() *Message.Message {
	// read solve message

	if err != nil {
		return nil
	}
	return nil
}
func (c *Client) ReadAll() {

}
