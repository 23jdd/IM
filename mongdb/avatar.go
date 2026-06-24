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

// AvatarDoc 头像文档，对应 MongoDB avatars 集合中的一条记录。
type AvatarDoc struct {
	Id          bson.ObjectID `bson:"_id,omitempty"` // MongoDB 文档主键
	Data        []byte        `bson:"data"`          // 图片二进制数据
	ContentType string        `bson:"content_type"`  // 图片 MIME 类型
	CreatedAt   time.Time     `bson:"created_at"`    // 创建时间
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
	// 将插入返回的 _id 断言为 ObjectID，类型不符则报错
	oid, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		return "", errors.New("unexpected inserted id type")
	}
	return oid.Hex(), nil
}

// GetImage 按 _id 十六进制字符串读取图片。
func GetImage(ctx context.Context, id string) ([]byte, string, error) {
	// 将十六进制字符串还原为 ObjectID
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
