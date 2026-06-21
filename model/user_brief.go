package model

// UserBrief 是用户的公开基本信息（用于搜索/添加好友、查看资料）。
type UserBrief struct {
	Uid       string `db:"uid" json:"uid"`
	Name      string `db:"name" json:"name"`
	Avatar    string `db:"avatar" json:"avatar"`
	Gender    uint64 `db:"gender" json:"gender"`
	Signature string `db:"signature" json:"signature"`
}
