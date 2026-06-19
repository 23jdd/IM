package mongdb

import (
	"IM/model"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MessageDoc struct {
	MsgId     string    `bson:"msg_id"`
	FromUid   string    `bson:"from_uid"`
	ToUid     string    `bson:"to_uid"`
	GroupId   string    `bson:"group_id"`
	MsgType   byte      `bson:"msg_type"`
	Content   string    `bson:"content"`
	Status    byte      `bson:"status"`
	CreatedAt time.Time `bson:"created_at"`
}

func SaveMessage(ctx context.Context, msg *model.ChatMessage) error {
	doc := &MessageDoc{
		MsgId:     msg.MsgId,
		FromUid:   msg.FromUid,
		ToUid:     msg.ToUid,
		GroupId:   msg.GroupId,
		MsgType:   msg.MsgType,
		Content:   msg.Content,
		Status:    msg.Status,
		CreatedAt: msg.CreatedAt,
	}
	_, err := MsgCol.InsertOne(ctx, doc)
	return err
}

func GetChatHistory(ctx context.Context, uid1, uid2 string, before time.Time, limit int64) ([]*MessageDoc, error) {
	filter := bson.D{
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "from_uid", Value: uid1}, {Key: "to_uid", Value: uid2}},
			bson.D{{Key: "from_uid", Value: uid2}, {Key: "to_uid", Value: uid1}},
		}},
	}
	if !before.IsZero() {
		filter = append(filter, bson.E{Key: "created_at", Value: bson.D{{Key: "$lt", Value: before}}})
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cursor, err := MsgCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []*MessageDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func GetGroupChatHistory(ctx context.Context, groupId string, before time.Time, limit int64) ([]*MessageDoc, error) {
	filter := bson.D{{Key: "group_id", Value: groupId}}
	if !before.IsZero() {
		filter = append(filter, bson.E{Key: "created_at", Value: bson.D{{Key: "$lt", Value: before}}})
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cursor, err := MsgCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []*MessageDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func GetOfflineMessages(ctx context.Context, uid string) ([]*MessageDoc, error) {
	filter := bson.D{
		{Key: "to_uid", Value: uid},
		{Key: "status", Value: model.MsgStatusUnread},
	}
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}).SetLimit(200)
	cursor, err := MsgCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []*MessageDoc
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func MarkMessagesRead(ctx context.Context, msgIds []string) error {
	filter := bson.D{{Key: "msg_id", Value: bson.D{{Key: "$in", Value: msgIds}}}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: model.MsgStatusRead}}}}
	_, err := MsgCol.UpdateMany(ctx, filter, update)
	return err
}
