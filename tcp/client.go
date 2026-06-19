package tcp

import (
	"fmt"
	"log"
	"net"
	"time"
)

type Client struct {
	con     net.Conn
	context any
}

func NewClient(con net.Conn) *Client {
	return &Client{
		con: con,
	}
}

func (c *Client) HeartBeat() {
	ticker := time.NewTicker(2 * time.Second)
	for {
		s := <-ticker.C
		fmt.Println(s.String())
		c.OnTicker()
	}
}
func (c *Client) Start() {
	go c.HeartBeat()
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
