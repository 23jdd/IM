package main

import (
	"IM/gateway"
	"IM/http"
	"IM/log"
	"IM/model"
	"IM/mongdb"
	"IM/mysql"
	"IM/rabbitmq"
	"IM/redis"
	"IM/service"
	"IM/tcp"
	"IM/tcp/Message"
	"IM/utils"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

func main() {
	config := MustLoadConfig(".")

	log.InitLogger(config.LogPath, config.LogLevel)
	defer log.CloseLogger()

	utils.SetJWTSecret(config.JWTSecret)

	log.Info("starting IM server")

	mysql.ConfigInit(config.DataSource)
	mysql.InitMessageConn(config.DataSource)
	mysql.InitFriendConn(config.DataSource)
	mysql.InitGroupConn(config.DataSource)

	if err := mongdb.InitMongoDB(config.MongoURI, config.MongoDB); err != nil {
		log.Error("mongodb init failed", zap.Error(err))
	}
	defer mongdb.CloseMongoDB()

	redisOK := false
	if err := redis.InitRedis(config.RedisAddr, config.RedisPassword, config.RedisDB); err != nil {
		log.Warn("redis init failed", zap.Error(err))
	} else {
		redisOK = true
		defer redis.CloseRedis()
	}

	if err := rabbitmq.InitRabbitMQ(config.RabbitMQURL); err != nil {
		log.Warn("rabbitmq init failed", zap.Error(err))
	} else {
		defer rabbitmq.CloseRabbitMQ()
		rabbitmq.ConsumeMessages(func(event *rabbitmq.MessageEvent) error {
			msg := &model.ChatMessage{
				MsgId:     event.MsgId,
				FromUid:   event.FromUid,
				ToUid:     event.ToUid,
				GroupId:   event.GroupId,
				MsgType:   event.MsgType,
				Content:   event.Content,
				Status:    model.MsgStatusUnread,
				CreatedAt: event.CreatedAt,
			}
			return mongdb.SaveMessage(context.Background(), msg)
		})
	}

	server := tcp.NewServer(config.TCPAddr, config.TcpPort, 10*time.Second)
	server.SetInstanceID(fmt.Sprintf("%s:%d", config.TCPAddr, config.TcpPort))
	server.AddHandler(tcp.Verify)
	server.AddHandler(tcp.Router)
	server.AddHandler(tcp.Echo)

	// 跨实例路由：Redis 可用时用共享在线表 + Pub/Sub 转发，否则退化为单机内存表。
	if redisOK {
		server.SetPresence(redis.NewRedisPresence())
		server.SetForwarder(redis.NewRedisForwarder())
		go redis.SubscribeRoutes(context.Background(), server.InstanceID(), func(uid string, frame []byte) {
			_ = server.DeliverLocal(uid, frame)
		})
	} else {
		server.SetPresence(tcp.NewMemoryPresence())
	}

	// 让 HTTP/service 层能向在线用户实时推送通知（好友申请/接受等），
	// 复用 TCP 的 RouteTo（自动支持本地投递与跨实例转发）。
	service.SetNotifier(func(toUid string, payload []byte) {
		_ = server.RouteTo(toUid, Message.NewMessage(Message.Json, 0, payload))
	})

	go server.Start()

	go func() {
		http.NewServer(config.HttpAddress, config.HttpPort).Start()
	}()

	if len(config.BackendAddrs) > 0 {
		lb := gateway.NewLoadBalancer(config.BackendAddrs)
		go func() {
			gateway.StartTCPProxy(
				fmt.Sprintf("%s:%d", config.TCPAddr, config.GatewayPort),
				lb,
			)
		}()
	}

	log.Info("all services started")
	tcp.NotifyServer(server)
}