package Message

import "encoding/json"

type Message struct {
	t    MsgType
	key  uint64
	len  int
	Data []byte
}

func NewMessage(t MsgType, key uint64, data []byte) *Message {
	return &Message{
		t:    t,
		key:  key,
		len:  len(data),
		Data: nil,
	}
}
func AckMessage(key uint64) *Message {
	return NewMessage(ACK, key, nil)
}
func HeartMessage(key uint64) *Message {
	return NewMessage(HeartBeat, key, nil)
}
func JsonMessage(key uint64, target any) (*Message, error) {
	data, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	return NewMessage(Json, key, data), nil
}
func TextMessage(key uint64, text string) *Message {
	return NewMessage(Text, key, []byte(text))
}
func BlobMessage(key uint64, blob []byte) *Message {
	return NewMessage(Blob, key, blob)
}
func (m *Message) Len() int {
	return m.len
}
