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
	return tcp.NewServer("", 0, 10*time.Second)
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
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	sendData := []byte("hello echo test")
	msg := Message.NewMessage(Message.Text, 1, sendData)
	encoded := Message.Encode(msg)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := clientConn.Write(encoded)
		if err != nil {
			t.Errorf("Write failed: %v", err)
		}
	}()

	received, err := readFullMessage(serverConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tcp.Echo(received, client)
	}()

	echoed, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read echoed message failed: %v", err)
	}
	wg.Wait()

	if !bytes.Equal(echoed.Data, sendData) {
		t.Errorf("echoed data = %v, want %v", echoed.Data, sendData)
	}
}

func TestEchoHandlerEmptyData(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	msg := Message.NewMessage(Message.ACK, 0, nil)
	encoded := Message.Encode(msg)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientConn.Write(encoded)
	}()

	received, err := readFullMessage(serverConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tcp.Echo(received, client)
	}()

	echoed, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read echoed message failed: %v", err)
	}
	wg.Wait()

	if len(echoed.Data) != 0 {
		t.Errorf("expected empty data, got %v", echoed.Data)
	}
}

func TestEchoHandlerBinaryData(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	binaryData := []byte{0x00, 0xFF, 0xAB, 0xCD, 0x12, 0x34}
	msg := Message.NewMessage(Message.Blob, 255, binaryData)
	encoded := Message.Encode(msg)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientConn.Write(encoded)
	}()

	received, err := readFullMessage(serverConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tcp.Echo(received, client)
	}()

	echoed, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read echoed message failed: %v", err)
	}
	wg.Wait()

	if !bytes.Equal(echoed.Data, binaryData) {
		t.Errorf("echoed binary data mismatch")
	}
}

func TestEchoClientSendAndRead(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := client.SendText(1, "direct send")
		if err != nil {
			t.Errorf("SendText failed: %v", err)
		}
	}()

	received, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	if !bytes.Equal(received.Data, []byte("direct send")) {
		t.Errorf("received = %v, want 'direct send'", received.Data)
	}
}

func TestEchoClientSendJson(t *testing.T) {
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
		err := client.SendJson(7, testPayload{Msg: "hello json"})
		if err != nil {
			t.Errorf("SendJson failed: %v", err)
		}
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

func TestEchoClientSendText(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := client.SendText(3, "text message")
		if err != nil {
			t.Errorf("SendText failed: %v", err)
		}
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

func TestEchoClientSendBlob(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	blob := []byte{0x01, 0x02, 0x03, 0x04}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := client.SendBlob(9, blob)
		if err != nil {
			t.Errorf("SendBlob failed: %v", err)
		}
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

func TestEchoClientSendAck(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := client.SendAck(15)
		if err != nil {
			t.Errorf("SendAck failed: %v", err)
		}
	}()

	received, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	if received.Len() != 0 {
		t.Errorf("expected empty data for ACK, got %d bytes", received.Len())
	}
}

func TestEchoClientSendHeart(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := client.SendHeart(33)
		if err != nil {
			t.Errorf("SendHeart failed: %v", err)
		}
	}()

	received, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	if received.Len() != 0 {
		t.Errorf("expected empty data for HeartBeat, got %d bytes", received.Len())
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

func TestEchoClientMultipleMessages(t *testing.T) {
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
			err := client.SendText(uint32(i), msg)
			if err != nil {
				t.Errorf("SendText %d failed: %v", i, err)
			}
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

func TestNewClient(t *testing.T) {
	_, serverConn := net.Pipe()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestEchoHandlerUTF8(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	client := tcp.NewClient(serverConn, newTestServer())

	utf8Data := []byte("你好，世界！🚀")
	msg := Message.NewMessage(Message.Text, 1, utf8Data)
	encoded := Message.Encode(msg)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		clientConn.Write(encoded)
	}()

	received, err := readFullMessage(serverConn)
	if err != nil {
		t.Fatalf("readFullMessage failed: %v", err)
	}
	wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		tcp.Echo(received, client)
	}()

	echoed, err := readFullMessage(clientConn)
	if err != nil {
		t.Fatalf("read echoed message failed: %v", err)
	}
	wg.Wait()

	if !bytes.Equal(echoed.Data, utf8Data) {
		t.Errorf("UTF-8 echo mismatch")
	}
}
