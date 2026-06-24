package model

import "time"

// MomentComment 表示一条朋友圈评论。
type MomentComment struct {
	Uid       string    `bson:"uid" json:"uid"`
	Content   string    `bson:"content" json:"content"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// Moment 是一条朋友圈动态，存储于 MongoDB。
type Moment struct {
	MomentId  string          `bson:"moment_id" json:"moment_id"`
	Uid       string          `bson:"uid" json:"uid"`
	Content   string          `bson:"content" json:"content"`
	Images    []string        `bson:"images" json:"images"` // 图片 _id 列表（复用通用图片存储）
	Likes     []string        `bson:"likes" json:"likes"`   // 点赞用户 uid 列表
	Comments  []MomentComment `bson:"comments" json:"comments"`
	CreatedAt time.Time       `bson:"created_at" json:"created_at"`
}
