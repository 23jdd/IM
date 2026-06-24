package mongdb

import (
	"IM/model"
	"context"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MessageDoc 消息文档，对应 MongoDB messages 集合中归档的一条聊天消息。
type MessageDoc struct {
	MsgId     string    `bson:"msg_id"`     // 消息唯一 ID
	FromUid   string    `bson:"from_uid"`   // 发送者 UID
	ToUid     string    `bson:"to_uid"`     // 接收者 UID（私聊）
	GroupId   string    `bson:"group_id"`   // 群组 ID（群聊）
	MsgType   byte      `bson:"msg_type"`   // 消息类型
	Content   string    `bson:"content"`    // 消息内容
	Status    byte      `bson:"status"`     // 消息状态（未读/已读/撤回等）
	CreatedAt time.Time `bson:"created_at"` // 创建时间
}

// SaveMessage 将一条聊天消息归档写入 MongoDB。
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

// GetChatHistory 查询 uid1 与 uid2 之间的私聊历史，before 为分页游标（仅取该时间之前），limit 为条数上限。
func GetChatHistory(ctx context.Context, uid1, uid2 string, before time.Time, limit int64) ([]*MessageDoc, error) {
	//search
	// 双向匹配：(uid1->uid2) 或 (uid2->uid1)
	filter := bson.D{
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "from_uid", Value: uid1}, {Key: "to_uid", Value: uid2}},
			bson.D{{Key: "from_uid", Value: uid2}, {Key: "to_uid", Value: uid1}},
		}},
	}
	if !before.IsZero() {
		// 仅查询 before 时间之前的消息，用于向上翻页
		filter = append(filter, bson.E{Key: "created_at", Value: bson.D{{Key: "$lt", Value: before}}})
	}

	// 按时间倒序排序并限制返回条数
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

// GetGroupChatHistory 查询指定群组的聊天历史，before 为分页游标，limit 为条数上限。
func GetGroupChatHistory(ctx context.Context, groupId string, before time.Time, limit int64) ([]*MessageDoc, error) {
	filter := bson.D{{Key: "group_id", Value: groupId}}
	if !before.IsZero() {
		// 仅查询 before 时间之前的消息，用于向上翻页
		filter = append(filter, bson.E{Key: "created_at", Value: bson.D{{Key: "$lt", Value: before}}})
	}

	// 按时间倒序排序并限制返回条数
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

// GetOfflineMessages 拉取指定用户的离线消息（状态为未读，最多 200 条，按时间正序）。
func GetOfflineMessages(ctx context.Context, uid string) ([]*MessageDoc, error) {
	filter := bson.D{
		{Key: "to_uid", Value: uid},
		{Key: "status", Value: model.MsgStatusUnread},
	}
	// 按时间正序，保证离线消息按发生顺序下发
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

// MarkMessagesRead 将给定 msgId 列表的消息批量标记为已读。
func MarkMessagesRead(ctx context.Context, msgIds []string) error {
	filter := bson.D{{Key: "msg_id", Value: bson.D{{Key: "$in", Value: msgIds}}}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: model.MsgStatusRead}}}}
	_, err := MsgCol.UpdateMany(ctx, filter, update)
	return err
}

// UpdateMessageStatus 更新归档消息的状态（如撤回 = 2），使历史翻页能反映撤回。
func UpdateMessageStatus(ctx context.Context, msgId string, status byte) error {
	filter := bson.D{{Key: "msg_id", Value: msgId}}
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "status", Value: status}}}}
	_, err := MsgCol.UpdateMany(ctx, filter, update)
	return err
}
