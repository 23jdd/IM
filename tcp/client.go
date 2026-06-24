package tcp

import (
	"IM/tcp/Message"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Client 表示一条 TCP 客户端连接，封装读写、心跳、处理链与离线确认等状态。
type Client struct {
	uidMu        sync.RWMutex
	uid          string
	con          net.Conn
	context      *Context
	server       *Server
	closed       atomic.Bool
	worker       chan *Message.Message
	heart        chan any
	finished     bool
	key          uint32
	writeMu      sync.Mutex
	closeOnce    sync.Once
	quit         chan struct{}
	writeTimeout time.Duration

	// 离线消息待确认表：key -> msgId。仅在 MessageHandler goroutine（handler 链）
	// 内访问（发送在 OfflineSyncHandler、确认在 AckHandler），故无需加锁。
	offlineKey  uint32
	offlineAcks map[uint32]string
}

// WorkerSize 单连接消息处理队列容量。
const WorkerSize int = 200

// defaultWriteTimeout 给每次写设置截止时间：对端假死时写最终会失败，
// 配合心跳即可在有限时间内发现并清理死连接。
const defaultWriteTimeout = 10 * time.Second

// MaxBodyLen 单条消息体最大长度，防止恶意/错误的超大长度字段导致 OOM 或溢出。
const MaxBodyLen = 1 << 20 // 1MB

// NewClient 基于已建立的连接创建一个 Client，并初始化各通道与状态。
func NewClient(con net.Conn, server *Server) *Client {
	return &Client{
		con:          con,
		server:       server,
		worker:       make(chan *Message.Message, WorkerSize),
		context:      NewContext(),
		heart:        make(chan any, 1),
		quit:         make(chan struct{}),
		writeTimeout: defaultWriteTimeout,
		offlineAcks:  make(map[uint32]string),
	}
}

// HeartBeat 按服务端配置的周期定时触发心跳，直到连接退出。
func (c *Client) HeartBeat() {
	ticker := time.NewTicker(c.server.t)
	defer ticker.Stop()
	for {
		select {
		case <-c.quit:
			return
		case <-ticker.C:
			c.OnTicker()
		}
	}
}

// Start 启动连接：开启心跳与处理协程，并在本协程循环读取消息入队。
func (c *Client) Start() {
	go c.HeartBeat()
	go c.MessageHandler()

	defer c.Close()
	for {
		message, err := c.ReadMessage()
		if err != nil {
			// 对端正常关闭或连接已关闭：关闭 worker 通道并退出读循环。
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				close(c.worker)
				return
			}
			c.closed.Store(true)
			close(c.worker)
			return
		}
		c.worker <- message
	}
}

// MessageHandler 单连接处理协程：消费心跳与消息队列，并跑完整条处理链。
func (c *Client) MessageHandler() {
	for {
		select {
		case <-c.quit:
			return
		case _, ok := <-c.heart:
			if !ok {
				return
			}
			c.IncrKey()
			if err := c.SendHeart(c.key); err != nil {
				log.Println("heartbeat send failed, closing:", err)
				c.Close()
				return
			}
			// 心跳续期在线登记（配合 Redis presence TTL 保活）。
			if uid := c.UID(); uid != "" && c.server.presence != nil {
				_ = c.server.presence.SetOnline(context.Background(), uid, c.server.instanceID)
			}
		case message, ok := <-c.worker:
			if !ok {
				return
			}
			// 依次执行处理链；任一处理器置 finished 即短路后续处理器。
			for _, h := range c.server.clientHandlers {
				h(message, c)
				if c.finished {
					break
				}
			}
			c.finished = false // 重置标志，准备处理下一条消息
		}
	}
}

// Process 将一条消息投入处理队列。
func (c *Client) Process(m *Message.Message) {
	c.worker <- m
}

// Context 返回连接级上下文。
func (c *Client) Context() *Context {
	return c.context
}

// UID 并发安全地读取连接绑定的用户 id。
func (c *Client) UID() string {
	c.uidMu.RLock()
	defer c.uidMu.RUnlock()
	return c.uid
}

// setUID 并发安全地设置连接绑定的用户 id。
func (c *Client) setUID(uid string) {
	c.uidMu.Lock()
	c.uid = uid
	c.uidMu.Unlock()
}

// IsClosed 返回连接是否已关闭。
func (c *Client) IsClosed() bool {
	return c.closed.Load()
}

// OnTicker 触发一次心跳信号（通道满时直接丢弃，非阻塞）。
func (c *Client) OnTicker() {
	select {
	case c.heart <- nil:
	default:
	}
}

// Close 关闭连接：幂等地清理在线登记、连接计数并关闭底层连接。
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		close(c.quit)
		uid := c.UID()
		if uid != "" {
			// 多端在线：仅当该 uid 已无任何连接时才标记离线。
			empty := c.server.removeClient(uid, c)
			if empty && c.server.presence != nil {
				_ = c.server.presence.SetOffline(context.Background(), uid, c.server.instanceID)
			}
		}
		c.server.count.Add(-1)
		if err := c.con.Close(); err != nil {
			log.Println("close conn:", err)
		}
	})
}

// Send 串行（加锁）地编码并写出一条消息，带写超时以应对对端假死。
func (c *Client) Send(message *Message.Message) error {
	if c.closed.Load() {
		return net.ErrClosed
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	// 设置写截止时间，避免在死连接上永久阻塞。
	if c.writeTimeout > 0 {
		_ = c.con.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}
	_, err := c.con.Write(Message.Encode(message))
	return err
}

// SendJson 发送一条 JSON 消息。
func (c *Client) SendJson(key uint32, target any) error {
	message, err := Message.JsonMessage(key, target)
	if err != nil {
		return err
	}
	return c.Send(message)
}

// SendHeart 发送一条心跳消息。
func (c *Client) SendHeart(key uint32) error {
	message := Message.HeartMessage(key)
	return c.Send(message)
}

// SendAck 发送一条 ACK 确认消息。
func (c *Client) SendAck(key uint32) error {
	message := Message.AckMessage(key)
	return c.Send(message)
}

// SendText 发送一条文本消息。
func (c *Client) SendText(key uint32, text string) error {
	message := Message.TextMessage(key, text)
	return c.Send(message)
}

// SendBlob 发送一条二进制消息。
func (c *Client) SendBlob(key uint32, blob []byte) error {
	message := Message.BlobMessage(key, blob)
	return c.Send(message)
}

// SendAuth 发送一条鉴权消息。
func (c *Client) SendAuth(key uint32, token string) error {
	message := Message.AuthMessage(key, token)
	return c.Send(message)
}

// SendNack 发送一条 Nack 否定确认消息。
func (c *Client) SendNack(key uint32) error {
	message := Message.NackMessage(key)
	return c.Send(message)
}

// ReadMessage 从连接读取一个完整帧：先读 8 字节头，再按长度读体并解码。
func (c *Client) ReadMessage() (*Message.Message, error) {
	header := c.server.bufPool.Get(8)
	// 先读满 8 字节固定帧头。
	if _, err := io.ReadFull(c.con, header); err != nil {
		c.server.bufPool.Put(header)
		return nil, err
	}
	length := binary.BigEndian.Uint32(header[4:8])
	// 超大长度字段直接拒绝，防止 OOM。
	if length > MaxBodyLen {
		c.server.bufPool.Put(header)
		return nil, fmt.Errorf("message body too large: %d > %d", length, MaxBodyLen)
	}
	buf := c.server.bufPool.Get(int(length) + 8)
	copy(buf, header)
	c.server.bufPool.Put(header)
	// 读满消息体。
	if _, err := io.ReadFull(c.con, buf[8:]); err != nil {
		c.server.bufPool.Put(buf)
		return nil, err
	}
	message, err := Message.Decode(buf)
	c.server.bufPool.Put(buf)
	if err != nil {
		return nil, err
	}
	return message, nil
}

// IncrKey 自增心跳/帧序号 key。
func (c *Client) IncrKey() {
	c.key++
}

// nextOfflineKey 返回下一个非 0 的离线消息 key（24bit 内）。
func (c *Client) nextOfflineKey() uint32 {
	c.offlineKey++
	if c.offlineKey == 0 {
		c.offlineKey = 1
	}
	return c.offlineKey & 0xFFFFFF
}

// trackOffline 记录一条待客户端确认的离线消息。
func (c *Client) trackOffline(key uint32, msgId string) {
	c.offlineAcks[key] = msgId
}

// takeOffline 取出并移除某 key 对应的离线消息 id（收到 ACK 时调用）。
func (c *Client) takeOffline(key uint32) (string, bool) {
	id, ok := c.offlineAcks[key]
	if ok {
		delete(c.offlineAcks, key)
	}
	return id, ok
}
