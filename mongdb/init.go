package mongdb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	Client *mongo.Client     // MongoDB 客户端，全局共享
	MsgCol *mongo.Collection // messages 集合，存放聊天消息
)

// InitMongoDB 连接 MongoDB 并初始化各集合句柄，uri 为连接串，dbName 为数据库名。
func InitMongoDB(uri, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	// 通过 Ping 确认连接可用
	if err := client.Ping(ctx, nil); err != nil {
		return err
	}

	Client = client
	// 初始化各业务集合句柄
	MsgCol = client.Database(dbName).Collection("messages")
	AvatarCol = client.Database(dbName).Collection("avatars")
	MomentCol = client.Database(dbName).Collection("moments")
	return nil
}

// CloseMongoDB 断开 MongoDB 连接，释放资源。
func CloseMongoDB() {
	if Client != nil {
		Client.Disconnect(context.Background())
	}
}
