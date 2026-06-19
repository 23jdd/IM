package Tests

import (
	"IM/tcp"
	"IM/tcp/Message"
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

func newTestServer() *tcp.Server {
	s := tcp.NewServer("", 0, 10*time.Second)
	s.AddHandler(tcp.Echo)
	return s
}

func readFullMessage(conn net.Conn) (*Message.Message, error) {
	header := make([]byte, 8)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, err
	}
	bodyLen := binary.BigEndian.Uint32(header[4:8])
	buf := make([]byte, 8+bodyLen)
	copy(buf, header)
	_, err = io.ReadFull(conn, buf[8:])
	if err != nil {
		return nil, err
	}
	return Message.Decode(buf)
}

func TestEchoHandler(t *testing.T) {
	server := newTestServer()

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	sendData := []byte("hello echo")
	msg := Message.NewMessage(Message.Text, 1, sendData)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientConn.Write(Message.Encode(msg))
	}()

	received, err := readFullMessage(serverConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	go client.MessageHandler()
	client.Process(received)

	echoed, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read echoed message failed: %v", err)
	}

	if !bytes.Equal(echoed.Data, sendData) {
		t.Errorf("echoed data = %v, want %v", echoed.Data, sendData)
	}
}

func TestEchoHandlerEmptyData(t *testing.T) {
	server := newTestServer()

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	msg := Message.NewMessage(Message.ACK, 0, nil)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientConn.Write(Message.Encode(msg))
	}()

	received, err := readFullMessage(serverConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	go client.MessageHandler()
	client.Process(received)

	echoed, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read echoed message failed: %v", err)
	}

	if len(echoed.Data) != 0 {
		t.Errorf("expected empty data, got %v", echoed.Data)
	}
}

func TestEchoHandlerBinaryData(t *testing.T) {
	server := newTestServer()

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	binaryData := []byte{0x00, 0xFF, 0xAB, 0xCD, 0x12, 0x34}
	msg := Message.NewMessage(Message.Blob, 255, binaryData)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientConn.Write(Message.Encode(msg))
	}()

	received, err := readFullMessage(serverConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	go client.MessageHandler()
	client.Process(received)

	echoed, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read echoed message failed: %v", err)
	}

	if !bytes.Equal(echoed.Data, binaryData) {
		t.Errorf("echoed binary data mismatch")
	}
}

func TestEchoHandlerUTF8(t *testing.T) {
	server := newTestServer()

	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, server)
	go client.MessageHandler()

	utf8Data := []byte("你好，世界！🚀")
	msg := Message.NewMessage(Message.Text, 1, utf8Data)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientConn.Write(Message.Encode(msg))
	}()

	received, err := readFullMessage(serverConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	go client.MessageHandler()
	client.Process(received)

	echoed, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read echoed message failed: %v", err)
	}

	if !bytes.Equal(echoed.Data, utf8Data) {
		t.Errorf("UTF-8 echo mismatch")
	}
}

func TestClientSendText(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.SendText(3, "text message")
	}()

	received, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	if !bytes.Equal(received.Data, []byte("text message")) {
		t.Errorf("received = %v, want 'text message'", received.Data)
	}
}

func TestClientSendJson(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	type testPayload struct {
		Msg string `json:"msg"`
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.SendJson(7, testPayload{Msg: "hello json"})
	}()

	received, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	if !bytes.Contains(received.Data, []byte("hello json")) {
		t.Errorf("expected 'hello json' in received data, got %s", received.Data)
	}
}

func TestClientSendBlob(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	blob := []byte{0x01, 0x02, 0x03, 0x04}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.SendBlob(9, blob)
	}()

	received, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	if !bytes.Equal(received.Data, blob) {
		t.Errorf("received = %v, want %v", received.Data, blob)
	}
}

func TestClientSendAck(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.SendAck(15)
	}()

	received, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	if len(received.Data) != 0 {
		t.Errorf("expected empty data for ACK, got %d bytes", len(received.Data))
	}
}

func TestClientSendHeart(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.SendHeart(33)
	}()

	received, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	if len(received.Data) != 0 {
		t.Errorf("expected empty data for HeartBeat, got %d bytes", len(received.Data))
	}
}

func TestClientContext(t *testing.T) {
	_, serverConn := net.Pipe()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	client.SetContext("test_value")
	ctx := client.Context()
	if ctx.(string) != "test_value" {
		t.Errorf("context = %v, want 'test_value'", ctx)
	}
}

func TestClientContextNil(t *testing.T) {
	_, serverConn := net.Pipe()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	ctx := client.Context()
	if ctx != nil {
		t.Errorf("expected nil initial context, got %v", ctx)
	}
}

func TestNewClient(t *testing.T) {
	_, serverConn := net.Pipe()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestClientMultipleMessages(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	messages := []string{"one", "two", "three"}
	for i, msg := range messages {
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			client.SendText(uint32(i), msg)
		}()

		received, err := readFullMessage(clientConn)
		if err != nil {
			t.Fatalf("readFullMessage %d failed: %v", i, err)
		}
		wg.Wait()

		if !bytes.Equal(received.Data, []byte(msg)) {
			t.Errorf("msg %d: received = %v, want %v", i, received.Data, msg)
		}
	}
}
