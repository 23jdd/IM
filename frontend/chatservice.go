package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// ChatService 是前端与后端 TCP 长连接之间的桥接层。
// 前端通过绑定调用其方法收发实时消息，后端推送的消息以事件形式发给前端。
type ChatService struct {
	app  *application.App
	mu   sync.Mutex
	conn net.Conn
	key  uint32

	defaultAddr string

	connectedMu sync.RWMutex
	connected   bool
}

// NewChatService 创建聊天桥接服务实例。
func NewChatService() *ChatService {
	return &ChatService{}
}

// SetApp 注入 Wails 应用，用于向前端推送事件。
func (s *ChatService) SetApp(app *application.App) {
	s.app = app
}

// SetDefaultAddr 设置 Connect("") 时使用的默认后端 TCP 地址（由命令行 flag 决定）。
func (s *ChatService) SetDefaultAddr(addr string) {
	s.defaultAddr = addr
}

// emit 向前端发送一个事件（app 未就绪时忽略）。
func (s *ChatService) emit(name string, data any) {
	if s.app != nil {
		s.app.Event.Emit(name, data)
	}
}

// nextKey 生成自增的消息 key，并截断到 3 字节（协议中 key 占 24 位）。
func (s *ChatService) nextKey() uint32 {
	return atomic.AddUint32(&s.key, 1) & 0xFFFFFF
}

// isConnected 返回当前是否处于已连接状态（读锁保护）。
func (s *ChatService) isConnected() bool {
	s.connectedMu.RLock()
	defer s.connectedMu.RUnlock()
	return s.connected
}

// setConnected 设置连接状态（写锁保护）。
func (s *ChatService) setConnected(v bool) {
	s.connectedMu.Lock()
	s.connected = v
	s.connectedMu.Unlock()
}

// Connect 拨号连接后端 TCP 服务（网关 :8000 或直连 :9000）。
func (s *ChatService) Connect(addr string) error {
	if s.isConnected() {
		return nil
	}
	// 命令行 flag 指定的地址优先，忽略前端传入的硬编码地址。
	if s.defaultAddr != "" {
		addr = s.defaultAddr
	}
	if addr == "" {
		addr = "127.0.0.1:9000"
	}
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		s.emit("im:status", "error:"+err.Error())
		return err
	}
	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()
	s.setConnected(true)
	s.emit("im:status", "connected")

	go s.readLoop(conn)
	return nil
}

// Disconnect 主动断开连接。
func (s *ChatService) Disconnect() {
	s.mu.Lock()
	conn := s.conn
	s.conn = nil
	s.mu.Unlock()
	if conn != nil {
		_ = conn.Close()
	}
	s.setConnected(false)
}

// write 按二进制协议编码并发送一帧（未连接时返回错误）。
func (s *ChatService) write(t byte, key uint32, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return errors.New("not connected")
	}
	_, err := s.conn.Write(encodeFrame(t, key, data))
	return err
}

// Auth 发送 JWT 认证帧。
func (s *ChatService) Auth(token string) error {
	return s.write(msgAuth, s.nextKey(), []byte(token))
}

// SendText 发送单聊文本消息，body 为 {to_uid, content} 的 JSON。
// 返回本条消息的 key，前端可用其匹配后续的 ack/nack。
func (s *ChatService) SendText(toUid, content string) (uint32, error) {
	payload, _ := json.Marshal(map[string]string{
		"to_uid":  toUid,
		"content": content,
	})
	key := s.nextKey()
	if err := s.write(msgText, key, payload); err != nil {
		return 0, err
	}
	return key, nil
}

// SendGroupText 发送群聊文本消息，body 为 {group_id, content, mentions} 的 JSON。
func (s *ChatService) SendGroupText(groupId, content string, mentions []string) (uint32, error) {
	payload, _ := json.Marshal(map[string]any{
		"group_id": groupId,
		"content":  content,
		"mentions": mentions,
	})
	key := s.nextKey()
	if err := s.write(msgText, key, payload); err != nil {
		return 0, err
	}
	return key, nil
}

// Sync 触发离线消息同步（发送 Json 帧）。
func (s *ChatService) Sync() error {
	return s.write(msgJson, s.nextKey(), []byte("{}"))
}

// SendTyping 发送“正在输入”信号（走通知通道：Json 帧，action=typing）。
// 单聊传 toUid，群聊传 groupId（另一个留空）。即发即弃，未连接时忽略。
func (s *ChatService) SendTyping(toUid, groupId string) error {
	if !s.isConnected() {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{
		"action":   "typing",
		"to_uid":   toUid,
		"group_id": groupId,
	})
	return s.write(msgJson, s.nextKey(), payload)
}

// SendRead 发送“已读”回执（走通知通道：Json 帧，action=read）。
// upTo 为已读到的最新消息时间戳（毫秒）。单聊传 toUid，群聊传 groupId。
func (s *ChatService) SendRead(toUid, groupId string, upTo int64) error {
	if !s.isConnected() {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{
		"action":   "read",
		"to_uid":   toUid,
		"group_id": groupId,
		"up_to":    upTo,
	})
	return s.write(msgJson, s.nextKey(), payload)
}

// SendVideoSignal forwards WebRTC signaling metadata over the existing Json channel.
func (s *ChatService) SendVideoSignal(toUid, signalType, sdp, candidateJSON, callID string) error {
	if !s.isConnected() {
		return nil
	}
	payload := map[string]any{
		"action":      "video_signal",
		"to_uid":      toUid,
		"signal_type": signalType,
		"call_id":     callID,
	}
	if sdp != "" {
		payload["sdp"] = sdp
	}
	if candidateJSON != "" {
		var candidate any
		if err := json.Unmarshal([]byte(candidateJSON), &candidate); err != nil {
			return err
		}
		payload["candidate"] = candidate
	}
	data, _ := json.Marshal(payload)
	return s.write(msgJson, s.nextKey(), data)
}

// SaveFile 弹出保存对话框，把 base64 数据写入用户选择的路径，返回保存路径（取消则空串）。
func (s *ChatService) SaveFile(suggestedName, dataBase64 string) (string, error) {
	if s.app == nil {
		return "", errors.New("app not ready")
	}
	data, err := base64.StdEncoding.DecodeString(dataBase64)
	if err != nil {
		return "", err
	}
	path, err := s.app.Dialog.SaveFile().SetFilename(suggestedName).PromptForSingleSelection()
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", nil // 用户取消
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// readLoop 在独立 goroutine 中循环读取后端帧并分发，连接断开后清理状态并通知前端。
func (s *ChatService) readLoop(conn net.Conn) {
	head := make([]byte, headSize)
	for {
		if _, err := io.ReadFull(conn, head); err != nil {
			break
		}
		t, key, bodyLen, err := parseHeader(head)
		if err != nil {
			break
		}
		var body []byte
		if bodyLen > 0 {
			// 按帧头声明的长度精确读取消息体
			body = make([]byte, bodyLen)
			if _, err := io.ReadFull(conn, body); err != nil {
				break
			}
		}
		s.dispatch(t, key, body)
	}

	s.setConnected(false)
	s.mu.Lock()
	if s.conn == conn {
		// 仅当未被新连接替换时才清空，避免误清新连接
		s.conn = nil
	}
	s.mu.Unlock()
	_ = conn.Close()
	s.emit("im:status", "disconnected")
}

// dispatch 根据帧类型把后端消息转换为前端事件。
func (s *ChatService) dispatch(t byte, key uint32, body []byte) {
	switch t {
	case msgACK:
		s.emit("im:ack", map[string]any{"key": key, "msg_id": string(body)})
	case msgNack:
		s.emit("im:nack", map[string]any{"key": key})
	case msgText:
		// 后端路由的实时文本帧 body 为 JSON {from_uid, group_id?, msg_id, content}。
		// 兼容旧格式：解析失败则按裸文本处理（from_uid 为空）。
		var p struct {
			FromUid string `json:"from_uid"`
			ToUid   string `json:"to_uid"`
			GroupId string `json:"group_id"`
			MsgId   string `json:"msg_id"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(body, &p); err == nil && (p.Content != "" || p.FromUid != "") {
			s.emit("im:text", map[string]any{
				"key":      key,
				"from_uid": p.FromUid,
				"to_uid":   p.ToUid,
				"group_id": p.GroupId,
				"msg_id":   p.MsgId,
				"content":  p.Content,
			})
		} else {
			s.emit("im:text", map[string]any{
				"key":      key,
				"from_uid": "",
				"content":  string(body),
			})
		}
	case msgBlob:
		// 离线同步：每个 blob 为一条 ChatMessage 的 JSON。
		// 回 ACK(key)，服务端收到确认后才将该消息标记为已读（可靠投递）。
		if key != 0 {
			_ = s.write(msgACK, key, nil)
		}
		var m map[string]any
		if err := json.Unmarshal(body, &m); err == nil {
			s.emit("im:offline", m)
		}
	case msgJson:
		// 服务端推送的系统通知（如好友申请/接受），body 为 {event, ...}。
		var n map[string]any
		if err := json.Unmarshal(body, &n); err == nil {
			s.emit("im:notify", n)
		}
	case msgHeartBeat:
		// 心跳，忽略。
	default:
		s.emit("im:blob", map[string]any{"type": t, "content": string(body)})
	}
}
