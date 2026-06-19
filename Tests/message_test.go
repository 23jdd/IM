package Tests

import (
	"IM/tcp/Message"
	"bytes"
	"testing"
)

func TestEncodeDecodeRoundtrip(t *testing.T) {
	testCases := []struct {
		name string
		t    byte
		key  uint32
		data []byte
	}{
		{"ACK", Message.ACK, 1, nil},
		{"ACK_empty", Message.ACK, 1, []byte{}},
		{"HeartBeat", Message.HeartBeat, 2, nil},
		{"Json", Message.Json, 100, []byte(`{"msg":"hello"}`)},
		{"Text", Message.Text, 200, []byte("hello world")},
		{"Blob", Message.Blob, 300, []byte{0x01, 0x02, 0x03, 0xFF}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := Message.NewMessage(tc.t, tc.key, tc.data)
			encoded := Message.Encode(m)

			decoded, err := Message.Decode(encoded)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}

			if !bytes.Equal(decoded.Data, tc.data) {
				t.Errorf("data = %v, want %v", decoded.Data, tc.data)
			}

			if len(decoded.Data) != len(tc.data) {
				t.Errorf("decoded data len = %d, want %d", len(decoded.Data), len(tc.data))
			}
		})
	}
}

func TestDecodePacketTooShort(t *testing.T) {
	_, err := Message.Decode([]byte{0x00, 0x01, 0x02})
	if err == nil {
		t.Error("expected error for packet too short")
	}
}

func TestDecodeIncompleteBody(t *testing.T) {
	data := make([]byte, 12)
	data[0] = Message.Text
	data[1] = 0
	data[2] = 0
	data[3] = 1
	data[4] = 0
	data[5] = 0
	data[6] = 0
	data[7] = 100

	_, err := Message.Decode(data)
	if err == nil {
		t.Error("expected error for incomplete packet body")
	}
}

func TestDecodeEmptyInput(t *testing.T) {
	_, err := Message.Decode([]byte{})
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestFullPacketSize(t *testing.T) {
	data := []byte{
		Message.Text, 0, 0, 1,
		0, 0, 0, 5,
		'h', 'e', 'l', 'l', 'o',
	}

	size, ok := Message.FullPacketSize(data)
	if !ok {
		t.Fatal("expected full packet")
	}
	if size != 13 {
		t.Errorf("size = %d, want 13", size)
	}
}

func TestFullPacketSizeIncomplete(t *testing.T) {
	data := []byte{
		Message.Text, 0, 0, 1,
		0, 0, 0, 5,
		'h', 'e',
	}

	expected, ok := Message.FullPacketSize(data)
	if ok {
		t.Error("expected incomplete packet")
	}
	if expected != 13 {
		t.Errorf("expected size = 13, got %d", expected)
	}
}

func TestFullPacketSizeNoHeader(t *testing.T) {
	size, ok := Message.FullPacketSize([]byte{0x00, 0x01})
	if ok {
		t.Error("expected incomplete (no header)")
	}
	if size != 0 {
		t.Errorf("expected size = 0, got %d", size)
	}
}

func TestAckMessage(t *testing.T) {
	m := Message.AckMessage(42)
	if m.Len() != 0 {
		t.Errorf("len = %d, want 0", m.Len())
	}
	if m.Data != nil {
		t.Error("expected nil data for ACK")
	}
}

func TestHeartMessage(t *testing.T) {
	m := Message.HeartMessage(99)
	if m.Len() != 0 {
		t.Errorf("len = %d, want 0", m.Len())
	}
}

func TestJsonMessage(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	p := payload{Name: "alice", Age: 30}

	m, err := Message.JsonMessage(77, p)
	if err != nil {
		t.Fatalf("JsonMessage failed: %v", err)
	}
	if m.Len() == 0 {
		t.Error("expected non-zero length")
	}

	encoded := Message.Encode(m)
	decoded, err := Message.Decode(encoded)
	if err != nil {
		t.Fatalf("roundtrip decode failed: %v", err)
	}
	if !bytes.Equal(decoded.Data, m.Data) {
		t.Error("roundtrip data mismatch")
	}
}

func TestTextMessage(t *testing.T) {
	m := Message.TextMessage(10, "hello 世界")
	if m.Len() != uint32(len("hello 世界")) {
		t.Errorf("len = %d, want %d", m.Len(), len("hello 世界"))
	}
	if !bytes.Equal(m.Data, []byte("hello 世界")) {
		t.Errorf("data = %v, want 'hello 世界'", m.Data)
	}
}

func TestBlobMessage(t *testing.T) {
	blob := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	m := Message.BlobMessage(255, blob)
	if m.Len() != 4 {
		t.Errorf("len = %d, want 4", m.Len())
	}
	if !bytes.Equal(m.Data, blob) {
		t.Errorf("data = %v, want %v", m.Data, blob)
	}
}

func TestNewMessage(t *testing.T) {
	m := Message.NewMessage(Message.Text, 12345, []byte("test"))
	if m.Len() != 4 {
		t.Errorf("len = %d, want 4", m.Len())
	}
	if !bytes.Equal(m.Data, []byte("test")) {
		t.Errorf("data = %v, want 'test'", m.Data)
	}
}

func TestKeyEncodingLimits(t *testing.T) {
	m := Message.NewMessage(Message.Text, 0xFFFFFF, []byte("max"))
	encoded := Message.Encode(m)

	if encoded[1] != 0xFF || encoded[2] != 0xFF || encoded[3] != 0xFF {
		t.Errorf("key encoding mismatch: [%02x %02x %02x], want [ff ff ff]",
			encoded[1], encoded[2], encoded[3])
	}

	decoded, err := Message.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if !bytes.Equal(decoded.Data, []byte("max")) {
		t.Errorf("data mismatch after max-key roundtrip")
	}
}

func TestEmptyBodyMessage(t *testing.T) {
	m := Message.NewMessage(Message.ACK, 1, nil)
	encoded := Message.Encode(m)
	if len(encoded) != 8 {
		t.Errorf("encoded length = %d, want 8 (header only)", len(encoded))
	}

	decoded, err := Message.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if len(decoded.Data) != 0 {
		t.Errorf("data length = %d, want 0", len(decoded.Data))
	}
}

func TestEncodeLargePayload(t *testing.T) {
	data := make([]byte, 65536)
	for i := range data {
		data[i] = byte(i % 256)
	}

	m := Message.NewMessage(Message.Blob, 0, data)
	encoded := Message.Encode(m)

	decoded, err := Message.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if !bytes.Equal(decoded.Data, data) {
		t.Error("large payload roundtrip data mismatch")
	}
}

func TestLen(t *testing.T) {
	m := Message.NewMessage(Message.Text, 0, nil)
	if m.Len() != 0 {
		t.Errorf("Len() = %d, want 0 for nil data", m.Len())
	}

	m = Message.NewMessage(Message.Text, 0, []byte{})
	if m.Len() != 0 {
		t.Errorf("Len() = %d, want 0 for empty data", m.Len())
	}

	m = Message.NewMessage(Message.Text, 0, []byte("abc"))
	if m.Len() != 3 {
		t.Errorf("Len() = %d, want 3", m.Len())
	}
}

func TestAllMessageTypesRoundtrip(t *testing.T) {
	types := []struct {
		name string
		t    byte
	}{
		{"ACK", Message.ACK},
		{"HeartBeat", Message.HeartBeat},
		{"Json", Message.Json},
		{"Text", Message.Text},
		{"Blob", Message.Blob},
	}

	for _, tt := range types {
		t.Run(tt.name, func(t *testing.T) {
			payload := []byte("test_payload")
			m := Message.NewMessage(tt.t, 100, payload)
			encoded := Message.Encode(m)
			decoded, err := Message.Decode(encoded)
			if err != nil {
				t.Fatalf("Decode failed: %v", err)
			}
			if !bytes.Equal(decoded.Data, payload) {
				t.Errorf("data mismatch for type %s", tt.name)
			}
		})
	}
}

func TestHeaderByteValues(t *testing.T) {
	m := Message.NewMessage(Message.ACK, 0, nil)
	enc := Message.Encode(m)
	if enc[0] != Message.ACK {
		t.Errorf("ACK header byte = %d, want %d", enc[0], Message.ACK)
	}

	m = Message.NewMessage(Message.Nack, 0, nil)
	enc = Message.Encode(m)
	if enc[0] != Message.Nack {
		t.Errorf("Nack header byte = %d, want %d", enc[0], Message.Nack)
	}

	m = Message.NewMessage(Message.Auth, 0, nil)
	enc = Message.Encode(m)
	if enc[0] != Message.Auth {
		t.Errorf("Auth header byte = %d, want %d", enc[0], Message.Auth)
	}

	m = Message.NewMessage(Message.HeartBeat, 0, nil)
	enc = Message.Encode(m)
	if enc[0] != Message.HeartBeat {
		t.Errorf("HeartBeat header byte = %d, want %d", enc[0], Message.HeartBeat)
	}

	m = Message.NewMessage(Message.Json, 0, nil)
	enc = Message.Encode(m)
	if enc[0] != Message.Json {
		t.Errorf("Json header byte = %d, want %d", enc[0], Message.Json)
	}

	m = Message.NewMessage(Message.Text, 0, nil)
	enc = Message.Encode(m)
	if enc[0] != Message.Text {
		t.Errorf("Text header byte = %d, want %d", enc[0], Message.Text)
	}

	m = Message.NewMessage(Message.Blob, 0, nil)
	enc = Message.Encode(m)
	if enc[0] != Message.Blob {
		t.Errorf("Blob header byte = %d, want %d", enc[0], Message.Blob)
	}
}
