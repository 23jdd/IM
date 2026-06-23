package mysql

import (
	"IM/mysql/model"
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var conn sqlx.SqlConn

func ConfigInit(dataSource string) {
	conn = sqlx.MustNewConn(sqlx.SqlConf{
		DataSource: dataSource,
		DriverName: "mysql",
		Replicas:   nil,
		Policy:     "",
	})
	createTables(conn)
	UserModel = model.NewUserModel(conn)
}

// createTables 自动建表（IF NOT EXISTS），保证表结构与代码一致，避免手动建表导致的 schema 漂移。
// DDL 中以 § 代替反引号（Go raw string 无法包含反引号），执行前替换回来。
func createTables(c sqlx.SqlConn) {
	ddls := []string{
		userDDL,
		chatMessageDDL,
		friendDDL,
		groupInfoDDL,
		groupMemberDDL,
		groupJoinRequestDDL,
	}
	ctx := context.Background()
	for _, ddl := range ddls {
		if _, err := c.ExecCtx(ctx, strings.ReplaceAll(ddl, "§", "`")); err != nil {
			panic(fmt.Errorf("create table failed: %w", err))
		}
	}
}

const userDDL = `
CREATE TABLE IF NOT EXISTS §user§ (
    §uid§           VARCHAR(32)      NOT NULL,
    §password_hash§ VARCHAR(128)     NOT NULL,
    §name§          VARCHAR(64)      NOT NULL,
    §avatar§        VARCHAR(512)     NOT NULL DEFAULT '',
    §gender§        TINYINT UNSIGNED NOT NULL DEFAULT 0,
    §birthday§      DATE             NULL,
    §signature§     VARCHAR(256)     NOT NULL DEFAULT '',
    §phone§         VARCHAR(32)      NOT NULL DEFAULT '',
    §email§         VARCHAR(128)     NOT NULL DEFAULT '',
    §last_ip§       VARCHAR(64)      NOT NULL DEFAULT '',
    §status§        TINYINT UNSIGNED NOT NULL DEFAULT 0,
    §last_seen_at§  DATETIME(3)      NULL,
    §created_at§    DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    §updated_at§    DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
    PRIMARY KEY (§uid§),
    KEY §idx_phone§ (§phone§),
    KEY §idx_last_ip§ (§last_ip§)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const chatMessageDDL = `
CREATE TABLE IF NOT EXISTS §chat_message§ (
    §msg_id§     VARCHAR(20)  NOT NULL PRIMARY KEY,
    §from_uid§   VARCHAR(20)  NOT NULL,
    §to_uid§     VARCHAR(20)  NOT NULL DEFAULT '',
    §group_id§   VARCHAR(20)  NOT NULL DEFAULT '',
    §msg_type§   TINYINT      NOT NULL DEFAULT 0,
    §content§    TEXT         NOT NULL,
    §status§     TINYINT      NOT NULL DEFAULT 0,
    §created_at§ DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX §idx_to_uid_status§ (§to_uid§, §status§),
    INDEX §idx_from_uid§ (§from_uid§),
    INDEX §idx_group_id§ (§group_id§),
    INDEX §idx_created_at§ (§created_at§)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const friendDDL = `
CREATE TABLE IF NOT EXISTS §friend_relation§ (
    §id§          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    §uid§         VARCHAR(20)     NOT NULL,
    §friend_uid§  VARCHAR(20)     NOT NULL,
    §status§      TINYINT         NOT NULL DEFAULT 0,
    §remark§      VARCHAR(64)     NOT NULL DEFAULT '',
    §created_at§  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    §updated_at§  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE INDEX §uk_uid_friend§ (§uid§, §friend_uid§),
    INDEX §idx_uid_status§ (§uid§, §status§),
    INDEX §idx_friend_uid§ (§friend_uid§)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const groupInfoDDL = `
CREATE TABLE IF NOT EXISTS §group_info§ (
    §group_id§     VARCHAR(20)   NOT NULL PRIMARY KEY,
    §name§         VARCHAR(128)  NOT NULL,
    §avatar§       VARCHAR(512)  NOT NULL DEFAULT '',
    §owner_uid§    VARCHAR(20)   NOT NULL,
    §description§  VARCHAR(512)  NOT NULL DEFAULT '',
    §announcement§ VARCHAR(1024) NOT NULL DEFAULT '' COMMENT '群公告',
    §member_count§ INT UNSIGNED  NOT NULL DEFAULT 0,
    §status§       TINYINT       NOT NULL DEFAULT 0,
    §created_at§   DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    §updated_at§   DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX §idx_owner_uid§ (§owner_uid§)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const groupMemberDDL = `
CREATE TABLE IF NOT EXISTS §group_member§ (
    §id§         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    §group_id§   VARCHAR(20)     NOT NULL,
    §uid§        VARCHAR(20)     NOT NULL,
    §role§       TINYINT         NOT NULL DEFAULT 0,
    §nickname§   VARCHAR(64)     NOT NULL DEFAULT '',
    §mute_until§ DATETIME        NULL,
    §joined_at§  DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE INDEX §uk_group_uid§ (§group_id§, §uid§),
    INDEX §idx_uid§ (§uid§)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const groupJoinRequestDDL = `
CREATE TABLE IF NOT EXISTS §group_join_request§ (
    §id§         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    §group_id§   VARCHAR(20)     NOT NULL,
    §uid§        VARCHAR(20)     NOT NULL,
    §status§     TINYINT         NOT NULL DEFAULT 0,
    §created_at§ DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE INDEX §uk_group_uid§ (§group_id§, §uid§),
    INDEX §idx_group_status§ (§group_id§, §status§)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`
