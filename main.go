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
	"IM/tcp"
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

	if err := redis.InitRedis(config.RedisAddr, config.RedisPassword, config.RedisDB); err != nil {
		log.Warn("redis init failed", zap.Error(err))
	} else {
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
	server.AddHandler(tcp.Verify)
	server.AddHandler(tcp.Router)
	server.AddHandler(tcp.Echo)

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