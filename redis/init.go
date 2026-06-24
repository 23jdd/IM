package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client 全局 Redis 客户端，由 InitRedis 初始化后供包内各函数复用。
var Client *redis.Client

// InitRedis 初始化全局 Redis 客户端，并通过 Ping 验证连通性。
func InitRedis(addr, password string, db int) error {
	Client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// 设置 5 秒超时，避免连接探测长时间阻塞
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return Client.Ping(ctx).Err()
}

// CloseRedis 关闭全局 Redis 客户端连接。
func CloseRedis() {
	if Client != nil {
		Client.Close()
	}
}

// SetOnlineStatus 设置用户在线状态：online 为真则写入带 TTL 的标记，否则删除该键。
func SetOnlineStatus(ctx context.Context, uid string, online bool, ttl time.Duration) error {
	key := "online:" + uid
	if online {
		return Client.Set(ctx, key, "1", ttl).Err()
	}
	return Client.Del(ctx, key).Err()
}

// IsOnline 查询用户是否在线；键不存在时返回离线且不报错。
func IsOnline(ctx context.Context, uid string) (bool, error) {
	key := "online:" + uid
	val, err := Client.Get(ctx, key).Result()
	if err == redis.Nil { // 键不存在，视为离线
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

// SetSession 以 token 为键存储会话对应的用户 ID，并设置过期时间。
func SetSession(ctx context.Context, token string, uid string, ttl time.Duration) error {
	return Client.Set(ctx, "session:"+token, uid, ttl).Err()
}

// GetSession 根据 token 获取会话对应的用户 ID。
func GetSession(ctx context.Context, token string) (string, error) {
	return Client.Get(ctx, "session:"+token).Result()
}

// CacheRecentMessages 缓存用户最近消息列表，仅保留最新 100 条并设置整体过期时间。
func CacheRecentMessages(ctx context.Context, uid string, msgs []string, ttl time.Duration) error {
	key := "recent_msgs:" + uid
	pipe := Client.Pipeline() // 使用管道批量执行，减少往返开销
	for _, msg := range msgs {
		pipe.LPush(ctx, key, msg)
	}
	pipe.LTrim(ctx, key, 0, 99) // 裁剪列表，仅保留最新的 100 条
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}
