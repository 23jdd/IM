package Message

import "encoding/json"

// Message 协议消息：t 为类型，key 为帧序号/关联号，len 为体长度，Data 为消息体。
type Message struct {
	t    MsgType
	key  uint32
	len  uint32
	Data []byte
}

// NewMessage 创建一条指定类型、key 和数据的消息。
func NewMessage(t MsgType, key uint32, data []byte) *Message {
	return &Message{
		t:    t,
		key:  key,
		len:  uint32(len(data)),
		Data: data,
	}
}

// AckMessage 构造一条 ACK 确认消息（无消息体）。
func AckMessage(key uint32) *Message {
	return NewMessage(ACK, key, nil)
}

// HeartMessage 构造一条心跳消息（无消息体）。
func HeartMessage(key uint32) *Message {
	return NewMessage(HeartBeat, key, nil)
}

// JsonMessage 将任意对象序列化为 JSON 并构造一条 Json 消息。
func JsonMessage(key uint32, target any) (*Message, error) {
	data, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}
	return NewMessage(Json, key, data), nil
}

// TextMessage 构造一条文本消息。
func TextMessage(key uint32, text string) *Message {
	return NewMessage(Text, key, []byte(text))
}

// BlobMessage 构造一条二进制数据消息。
func BlobMessage(key uint32, blob []byte) *Message {
	return NewMessage(Blob, key, blob)
}

// NackMessage 构造一条 Nack 否定确认消息（无消息体）。
func NackMessage(key uint32) *Message {
	return NewMessage(Nack, key, nil)
}

// AuthMessage 构造一条携带 token 的鉴权消息。
func AuthMessage(key uint32, token string) *Message {
	return NewMessage(Auth, key, []byte(token))
}

// Len 返回消息体长度。
func (m *Message) Len() uint32 {
	return m.len
}

// GetMsgType 返回消息类型。
func (m *Message) GetMsgType() MsgType {
	return m.t
}

// GetKey 返回消息 key。
func (m *Message) GetKey() uint32 {
	return m.key
}
