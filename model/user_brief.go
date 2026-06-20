package model

// UserBrief 是用户的公开基本信息（用于搜索/添加好友前预览）。
type UserBrief struct {
	Uid    string `db:"uid" json:"uid"`
	Name   string `db:"name" json:"name"`
	Avatar string `db:"avatar" json:"avatar"`
}
