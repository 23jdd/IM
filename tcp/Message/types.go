package Message

// MsgType 消息类型别名（单字节），标识帧的语义。
type MsgType = byte

// 消息类型常量：占用帧头第 1 字节。
const (
	ACK       byte = iota // 确认
	Nack                  // 否定确认（拒绝/失败）
	Auth                  // 鉴权
	HeartBeat             // 心跳
	Json                  // JSON 数据帧
	Text                  // 文本消息
	Blob                  // 二进制数据
)
