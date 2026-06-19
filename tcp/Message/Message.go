package Message

import "encoding/json"

type Message struct {
	t    MsgType
	key  uint32
	len  uint32
	Data []byte
}

func NewMessage(t MsgType, key uint32, data []byte) *Message {
	return &Message{
		t:    t,
		key:  key,
		len:  uint32(len(data)),
		Data: data,
	}
}
func AckMessage(key uint32) *Message {
	return NewMessage(ACK, key, nil)
}
func HeartMessage(key uint32) *Message {
	return NewMessage(HeartBeat, key, nil)
}
func JsonMessage(key uint32, target any) (*Message, error) {
	data, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	return NewMessage(Json, key, data), nil
}
func TextMessage(key uint32, text string) *Message {
	return NewMessage(Text, key, []byte(text))
}
func BlobMessage(key uint32, blob []byte) *Message {
	return NewMessage(Blob, key, blob)
}
func NackMessage(key uint32) *Message {
	return NewMessage(Nack, key, nil)
}
func AuthMessage(key uint32, token string) *Message {
	return NewMessage(Auth, key, []byte(token))
}
func (m *Message) Len() uint32 {
	return m.len
}
func (m *Message) GetMsgType() MsgType {
	return m.t
}
func (m *Message) GetKey() uint32 {
	return m.key
}
