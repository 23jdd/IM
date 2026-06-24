package mongdb

import (
	"IM/model"
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// MomentCol 存放朋友圈动态，在 InitMongoDB 中初始化。
var MomentCol *mongo.Collection

// InsertMoment 新增一条朋友圈动态。
func InsertMoment(ctx context.Context, m *model.Moment) error {
	_, err := MomentCol.InsertOne(ctx, m)
	return err
}

// FindMoments 按发布者 uid 集合查询动态（时间倒序）。
func FindMoments(ctx context.Context, uids []string, limit int64) ([]*model.Moment, error) {
	filter := bson.D{{Key: "uid", Value: bson.D{{Key: "$in", Value: uids}}}}
	// 按创建时间倒序，并限制返回条数
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cursor, err := MomentCol.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var docs []*model.Moment
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

// FindMoment 按 momentId 查询单条动态。
func FindMoment(ctx context.Context, momentId string) (*model.Moment, error) {
	var m model.Moment
	if err := MomentCol.FindOne(ctx, bson.D{{Key: "moment_id", Value: momentId}}).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// UpdateMomentLike add=true 点赞($addToSet)，add=false 取消($pull)。
func UpdateMomentLike(ctx context.Context, momentId, uid string, add bool) error {
	op := "$pull"
	if add {
		// 点赞使用 $addToSet，避免重复添加同一用户
		op = "$addToSet"
	}
	update := bson.D{{Key: op, Value: bson.D{{Key: "likes", Value: uid}}}}
	_, err := MomentCol.UpdateOne(ctx, bson.D{{Key: "moment_id", Value: momentId}}, update)
	return err
}

// AddComment 向指定动态追加一条评论。
func AddComment(ctx context.Context, momentId string, comment model.MomentComment) error {
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "comments", Value: comment}}}}
	_, err := MomentCol.UpdateOne(ctx, bson.D{{Key: "moment_id", Value: momentId}}, update)
	return err
}

// DeleteMoment 按 momentId 删除一条动态。
func DeleteMoment(ctx context.Context, momentId string) error {
	_, err := MomentCol.DeleteOne(ctx, bson.D{{Key: "moment_id", Value: momentId}})
	return err
}
