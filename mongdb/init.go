package mongdb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	Client *mongo.Client
	MsgCol *mongo.Collection
)

func InitMongoDB(uri, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return err
	}

	Client = client
	MsgCol = client.Database(dbName).Collection("messages")
	AvatarCol = client.Database(dbName).Collection("avatars")
	return nil
}

func CloseMongoDB() {
	if Client != nil {
		Client.Disconnect(context.Background())
	}
}
