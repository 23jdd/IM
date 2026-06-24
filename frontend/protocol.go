package main

import (
	"encoding/binary"
	"errors"
)

// 二进制协议（与后端 tcp/Message 一致，自包含实现，不依赖后端 module）
//
// ┌──────┬──────────┬──────────┬──────────┐
// │ 1B   │ 3B       │ 4B       │ N bytes  │
// │ type │ key      │ body len │ body     │
// └──────┴──────────┴──────────┴──────────┘

// headSize 帧头固定长度：1B 类型 + 3B key + 4B body 长度。
const headSize = 8

const (
	msgACK       byte = 0 // 确认
	msgNack      byte = 1 // 拒绝
	msgAuth      byte = 2 // JWT 认证
	msgHeartBeat byte = 3 // 心跳
	msgJson      byte = 4 // 系统消息（触发离线同步）
	msgText      byte = 5 // 文本聊天
	msgBlob      byte = 6 // 二进制（离线消息 JSON）
)

// frame 一帧消息：类型、用于匹配 ack 的 key 及消息体。
type frame struct {
	Type byte
	Key  uint32
	Data []byte
}

// encodeFrame 将类型、key 与消息体编码为协议字节流。
func encodeFrame(t byte, key uint32, data []byte) []byte {
	buf := make([]byte, headSize+len(data))
	buf[0] = t
	// 将 key 的低 24 位按大端拆分写入 3 个字节
	buf[1] = byte((key >> 16) & 0xFF)
	buf[2] = byte((key >> 8) & 0xFF)
	buf[3] = byte(key & 0xFF)
	binary.BigEndian.PutUint32(buf[4:8], uint32(len(data)))
	if len(data) > 0 {
		copy(buf[8:], data)
	}
	return buf
}

// parseHeader 解析帧头，返回类型、key 与消息体长度。
func parseHeader(head []byte) (t byte, key uint32, bodyLen uint32, err error) {
	if len(head) < headSize {
		return 0, 0, 0, errors.New("header too short")
	}
	t = head[0]
	// 由 3 个字节大端还原出 24 位 key
	key = uint32(head[1])<<16 | uint32(head[2])<<8 | uint32(head[3])
	bodyLen = binary.BigEndian.Uint32(head[4:8])
	return t, key, bodyLen, nil
}
