package service

import (
	"IM/model"
	"IM/mysql"
	"IM/utils"
	"context"
	"strconv"
	"time"
)

func SendChatMessage(ctx context.Context, fromUid, toUid string, msgType byte, content string) (*model.ChatMessage, error) {
	msg := &model.ChatMessage{
		MsgId:     strconv.FormatUint(utils.GenerateId(), 10),
		FromUid:   fromUid,
		ToUid:     toUid,
		MsgType:   msgType,
		Content:   content,
		Status:    model.MsgStatusUnread,
		CreatedAt: time.Now(),
	}

	err := mysql.InsertChatMessage(ctx, msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

func GetOfflineMessages(ctx context.Context, uid string) ([]*model.ChatMessage, error) {
	return mysql.FindOfflineMessages(ctx, uid)
}

func MarkMessagesRead(ctx context.Context, msgIds []string) error {
	return mysql.MarkMessagesRead(ctx, msgIds)
}
