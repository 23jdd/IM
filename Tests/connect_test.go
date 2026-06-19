package Tests

import (
	"IM/tcp"
	"IM/tcp/Message"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
)

func TestConnectEcho(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Echo)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		client := tcp.NewClient(conn, server)
		go client.Start()
	}()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	sendData := []byte("hello real network")
	msg := Message.NewMessage(Message.Text, 1, sendData)
	encoded := Message.Encode(msg)

	_, err = conn.Write(encoded)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	header := make([]byte, 8)
	_, err = io.ReadFull(conn, header)
	if err != nil {
		t.Fatalf("Read header failed: %v", err)
	}

	bodyLen := binary.BigEndian.Uint32(header[4:8])
	buf := make([]byte, 8+bodyLen)
	copy(buf, header)
	if bodyLen > 0 {
		_, err = io.ReadFull(conn, buf[8:])
		if err != nil {
			t.Fatalf("Read body failed: %v", err)
		}
	}

	received, err := Message.Decode(buf)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if !bytes.Equal(received.Data, sendData) {
		t.Errorf("echoed data = %v, want %v", received.Data, sendData)
	}
}

func TestConnectEchoMultipleMessages(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Echo)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		client := tcp.NewClient(conn, server)
		go client.Start()
	}()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	messages := []string{"ping", "hello", "world", "测试中文"}
	for i, msg := range messages {
		encoded := Message.Encode(Message.NewMessage(Message.Text, uint32(i), []byte(msg)))

		_, err = conn.Write(encoded)
		if err != nil {
			t.Fatalf("Write msg %d failed: %v", i, err)
		}

		header := make([]byte, 8)
		_, err = io.ReadFull(conn, header)
		if err != nil {
			t.Fatalf("Read header msg %d failed: %v", i, err)
		}

		bodyLen := binary.BigEndian.Uint32(header[4:8])
		buf := make([]byte, 8+bodyLen)
		copy(buf, header)
		if bodyLen > 0 {
			_, err = io.ReadFull(conn, buf[8:])
			if err != nil {
				t.Fatalf("Read body msg %d failed: %v", i, err)
			}
		}

		received, err := Message.Decode(buf)
		if err != nil {
			t.Fatalf("Decode msg %d failed: %v", i, err)
		}

		if !bytes.Equal(received.Data, []byte(msg)) {
			t.Errorf("msg %d: echoed = %v, want %v", i, received.Data, msg)
		}
	}
}

func TestConnectLargePayload(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Echo)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		client := tcp.NewClient(conn, server)
		go client.Start()
	}()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer conn.Close()

	data := make([]byte, 4096)
	for i := range data {
		data[i] = byte(i % 256)
	}

	msg := Message.NewMessage(Message.Blob, 7, data)
	encoded := Message.Encode(msg)

	_, err = conn.Write(encoded)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	header := make([]byte, 8)
	_, err = io.ReadFull(conn, header)
	if err != nil {
		t.Fatalf("Read header failed: %v", err)
	}

	bodyLen := binary.BigEndian.Uint32(header[4:8])
	buf := make([]byte, 8+bodyLen)
	copy(buf, header)
	if bodyLen > 0 {
		_, err = io.ReadFull(conn, buf[8:])
		if err != nil {
			t.Fatalf("Read body failed: %v", err)
		}
	}

	received, err := Message.Decode(buf)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if !bytes.Equal(received.Data, data) {
		t.Errorf("large payload echoed data mismatch")
	}
}

func TestConnectMultipleClients(t *testing.T) {
	server := tcp.NewServer("", 0, 10*time.Second)
	server.AddHandler(tcp.Echo)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			client := tcp.NewClient(conn, server)
			go client.Start()
		}
	}()

	for i := 0; i < 3; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			t.Fatalf("Dial client %d failed: %v", i, err)
		}

		msg := Message.TextMessage(uint32(i), "hello")
		_, err = conn.Write(Message.Encode(msg))
		if err != nil {
			t.Fatalf("Write client %d failed: %v", i, err)
		}

		header := make([]byte, 8)
		_, err = io.ReadFull(conn, header)
		if err != nil {
			t.Fatalf("Read header client %d failed: %v", i, err)
		}

		bodyLen := binary.BigEndian.Uint32(header[4:8])
		buf := make([]byte, 8+bodyLen)
		copy(buf, header)
		if bodyLen > 0 {
			_, err = io.ReadFull(conn, buf[8:])
			if err != nil {
				t.Fatalf("Read body client %d failed: %v", i, err)
			}
		}

		received, err := Message.Decode(buf)
		if err != nil {
			t.Fatalf("Decode client %d failed: %v", i, err)
		}

		if !bytes.Equal(received.Data, []byte("hello")) {
			t.Errorf("client %d: echoed = %v, want 'hello'", i, received.Data)
		}

		conn.Close()
	}
}
