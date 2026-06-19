package mysql

import (
	"IM/mysql/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var conn sqlx.SqlConn

func ConfigInit(dataSource string) {
	conn := sqlx.MustNewConn(sqlx.SqlConf{
		DataSource: dataSource,
		DriverName: "mysql",
		Replicas:   nil,
		Policy:     "",
	})
	userModel = model.NewUserModel(conn)
}
