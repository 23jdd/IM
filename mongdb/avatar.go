package mongdb

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// AvatarCol 存放头像图片二进制，在 InitMongoDB 中初始化。
var AvatarCol *mongo.Collection

type AvatarDoc struct {
	Id          bson.ObjectID `bson:"_id,omitempty"`
	Data        []byte        `bson:"data"`
	ContentType string        `bson:"content_type"`
	CreatedAt   time.Time     `bson:"created_at"`
}

// SaveImage 将图片存入 MongoDB，返回其 _id 的十六进制字符串（写入 MySQL.user.avatar）。
func SaveImage(ctx context.Context, data []byte, contentType string) (string, error) {
	doc := AvatarDoc{
		Data:        data,
		ContentType: contentType,
		CreatedAt:   time.Now(),
	}
	res, err := AvatarCol.InsertOne(ctx, doc)
	if err != nil {
		return "", err
	}
	oid, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		return "", errors.New("unexpected inserted id type")
	}
	return oid.Hex(), nil
}

// GetImage 按 _id 十六进制字符串读取图片。
func GetImage(ctx context.Context, id string) ([]byte, string, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, "", err
	}
	var doc AvatarDoc
	if err := AvatarCol.FindOne(ctx, bson.D{{Key: "_id", Value: oid}}).Decode(&doc); err != nil {
		return nil, "", err
	}
	return doc.Data, doc.ContentType, nil
}
