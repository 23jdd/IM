package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

// 编译期断言：确保 customUserModel 实现了 UserModel 接口。
var _ UserModel = (*customUserModel)(nil)

type (
	// UserModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserModel.
	// UserModel 用户表模型接口，可在此扩展自定义方法并在 customUserModel 中实现。
	UserModel interface {
		userModel
		withSession(session sqlx.Session) UserModel
	}

	// customUserModel 自定义用户模型，内嵌默认实现以便扩展。
	customUserModel struct {
		*defaultUserModel
	}
)

// NewUserModel returns a model for the database table.
// NewUserModel 基于数据库连接创建用户表模型。
func NewUserModel(conn sqlx.SqlConn) UserModel {
	return &customUserModel{
		defaultUserModel: newUserModel(conn),
	}
}

// withSession 返回绑定指定事务会话的用户模型，用于在同一事务中执行操作。
func (m *customUserModel) withSession(session sqlx.Session) UserModel {
	return NewUserModel(sqlx.NewSqlConnFromSession(session))
}
