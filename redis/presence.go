package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// presenceTTL 在线登记的过期时间；由心跳周期性续期。
const presenceTTL = 90 * time.Second

// RedisPresence 用 Redis 实现 tcp.Presence（在线注册表），支持多实例共享。
type RedisPresence struct{}

// NewRedisPresence 创建一个基于 Redis 的在线注册表实例。
func NewRedisPresence() *RedisPresence { return &RedisPresence{} }

// SetOnline 将用户标记为在线，并记录其所在的实例标识，带 TTL 自动过期。
func (p *RedisPresence) SetOnline(ctx context.Context, uid, instance string) error {
	return Client.Set(ctx, "online:"+uid, instance, presenceTTL).Err()
}

// GetInstance 获取用户当前所在的实例标识；不在线时返回空字符串且不报错。
func (p *RedisPresence) GetInstance(ctx context.Context, uid string) (string, error) {
	v, err := Client.Get(ctx, "online:"+uid).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return v, nil
}

// SetOffline 将用户置为离线；仅当登记的实例与传入实例一致时才删除，避免误删其他实例的登记。
func (p *RedisPresence) SetOffline(ctx context.Context, uid, instance string) error {
	v, err := Client.Get(ctx, "online:"+uid).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	if v == instance { // 仅当当前登记仍属于本实例时才执行删除
		return Client.Del(ctx, "online:"+uid).Err()
	}
	return nil
}

// routePayload 跨实例转发的消息载荷：目标用户 ID 与原始帧数据。
type routePayload struct {
	ToUid string `json:"to_uid"`
	Frame []byte `json:"frame"`
}

// RedisForwarder 用 Redis Pub/Sub 实现 tcp.Forwarder（跨实例转发）。
type RedisForwarder struct{}

// NewRedisForwarder 创建一个基于 Redis Pub/Sub 的跨实例转发器实例。
func NewRedisForwarder() *RedisForwarder { return &RedisForwarder{} }

// Forward 将帧数据发布到目标实例的转发通道，由目标实例订阅后完成本地投递。
func (f *RedisForwarder) Forward(ctx context.Context, instance, toUid string, frame []byte) error {
	body, err := json.Marshal(routePayload{ToUid: toUid, Frame: frame})
	if err != nil {
		return err
	}
	return Client.Publish(ctx, "route:"+instance, body).Err()
}

// SubscribeRoutes 订阅本实例的转发通道，收到帧后调用 deliver 进行本地投递。
// 阻塞运行，通常以 goroutine 启动。
func SubscribeRoutes(ctx context.Context, instance string, deliver func(toUid string, frame []byte)) {
	sub := Client.Subscribe(ctx, "route:"+instance)
	defer sub.Close()
	for msg := range sub.Channel() {
		var p routePayload
		if err := json.Unmarshal([]byte(msg.Payload), &p); err != nil {
			continue // 反序列化失败则跳过该消息
		}
		deliver(p.ToUid, p.Frame)
	}
}
