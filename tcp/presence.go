package tcp

import (
	"context"
	"sync"
)

// Presence 在线注册表：记录某 uid 当前在哪个实例上线，用于跨实例消息路由。
type Presence interface {
	SetOnline(ctx context.Context, uid, instance string) error
	// GetInstance 返回 uid 所在实例；不在线时返回空字符串。
	GetInstance(ctx context.Context, uid string) (string, error)
	// SetOffline 仅当当前登记为本实例时才下线，避免误删用户在别处的新会话。
	SetOffline(ctx context.Context, uid, instance string) error
}

// Forwarder 跨实例转发器：把已编码的帧投递到目标实例。
type Forwarder interface {
	Forward(ctx context.Context, instance, toUid string, frame []byte) error
}

// MemoryPresence 是单机 / 测试用的内存在线表实现。
type MemoryPresence struct {
	mu sync.RWMutex
	m  map[string]string // uid -> instance
}

func NewMemoryPresence() *MemoryPresence {
	return &MemoryPresence{m: make(map[string]string)}
}

func (p *MemoryPresence) SetOnline(ctx context.Context, uid, instance string) error {
	p.mu.Lock()
	p.m[uid] = instance
	p.mu.Unlock()
	return nil
}

func (p *MemoryPresence) GetInstance(ctx context.Context, uid string) (string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.m[uid], nil
}

func (p *MemoryPresence) SetOffline(ctx context.Context, uid, instance string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.m[uid] == instance {
		delete(p.m, uid)
	}
	return nil
}
