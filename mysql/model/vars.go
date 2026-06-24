package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

// ErrNotFound 查询无结果时返回的错误（等同于 sqlx.ErrNotFound）。
var ErrNotFound = sqlx.ErrNotFound
