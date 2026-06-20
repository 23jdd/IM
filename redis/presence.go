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

func NewRedisPresence() *RedisPresence { return &RedisPresence{} }

func (p *RedisPresence) SetOnline(ctx context.Context, uid, instance string) error {
	return Client.Set(ctx, "online:"+uid, instance, presenceTTL).Err()
}

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

func (p *RedisPresence) SetOffline(ctx context.Context, uid, instance string) error {
	v, err := Client.Get(ctx, "online:"+uid).Result()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	if v == instance {
		return Client.Del(ctx, "online:"+uid).Err()
	}
	return nil
}

type routePayload struct {
	ToUid string `json:"to_uid"`
	Frame []byte `json:"frame"`
}

// RedisForwarder 用 Redis Pub/Sub 实现 tcp.Forwarder（跨实例转发）。
type RedisForwarder struct{}

func NewRedisForwarder() *RedisForwarder { return &RedisForwarder{} }

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
			continue
		}
		deliver(p.ToUid, p.Frame)
	}
}
