package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var Client *redis.Client

func InitRedis(addr, password string, db int) error {
	Client = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return Client.Ping(ctx).Err()
}

func CloseRedis() {
	if Client != nil {
		Client.Close()
	}
}

func SetOnlineStatus(ctx context.Context, uid string, online bool, ttl time.Duration) error {
	key := "online:" + uid
	if online {
		return Client.Set(ctx, key, "1", ttl).Err()
	}
	return Client.Del(ctx, key).Err()
}

func IsOnline(ctx context.Context, uid string) (bool, error) {
	key := "online:" + uid
	val, err := Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return val == "1", nil
}

func SetSession(ctx context.Context, token string, uid string, ttl time.Duration) error {
	return Client.Set(ctx, "session:"+token, uid, ttl).Err()
}

func GetSession(ctx context.Context, token string) (string, error) {
	return Client.Get(ctx, "session:"+token).Result()
}

func CacheRecentMessages(ctx context.Context, uid string, msgs []string, ttl time.Duration) error {
	key := "recent_msgs:" + uid
	pipe := Client.Pipeline()
	for _, msg := range msgs {
		pipe.LPush(ctx, key, msg)
	}
	pipe.LTrim(ctx, key, 0, 99)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}
