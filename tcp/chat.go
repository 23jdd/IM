package tcp

import (
	"IM/service"
	"IM/tcp/Message"
	"context"
	"encoding/json"
	"log"
	"time"
)

// 通过函数变量注入，便于离线投递 / 群聊扇出逻辑的单元测试（不依赖真实 DB）。
var (
	getOfflineMessages = service.GetOfflineMessages
	markMessagesRead   = service.MarkMessagesRead
	getGroupMembers    = service.GetGroupMembers
	sendGroupMessage   = service.SendGroupMessage
	isBlocked          = service.IsBlocked
)

type TextChatPayload struct {
	ToUid    string   `json:"to_uid"`
	GroupId  string   `json:"group_id"`
	Content  string   `json:"content"`
	Mentions []string `json:"mentions"`
}

// RealtimeTextPayload 是服务端路由给接收方的实时文本帧体，携带发送者信息，
// 使接收端能正确归属消息（修复"实时帧丢失 from_uid"问题）。群聊时带 group_id。
type RealtimeTextPayload struct {
	FromUid   string    `json:"from_uid"`
	GroupId   string    `json:"group_id,omitempty"`
	MsgId     string    `json:"msg_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// BuildRealtimeText 构造单聊实时文本帧体（JSON）。
func BuildRealtimeText(fromUid, msgId, content string, createdAt time.Time) []byte {
	data, _ := json.Marshal(RealtimeTextPayload{
		FromUid:   fromUid,
		MsgId:     msgId,
		Content:   content,
		CreatedAt: createdAt,
	})
	return data
}

// BuildGroupText 构造群聊实时文本帧体（带 group_id）。
func BuildGroupText(fromUid, groupId, msgId, content string, createdAt time.Time) []byte {
	data, _ := json.Marshal(RealtimeTextPayload{
		FromUid:   fromUid,
		GroupId:   groupId,
		MsgId:     msgId,
		Content:   content,
		CreatedAt: createdAt,
	})
	return data
}

func ChatMessageHandler(m *Message.Message, c *Client) {
	var payload TextChatPayload
	if err := json.Unmarshal(m.Data, &payload); err != nil {
		log.Println("chat: unmarshal failed:", err)
		return
	}

	if payload.Content == "" {
		return
	}

	// 群聊：群成员扇出
	if payload.GroupId != "" {
		handleGroupMessage(m, c, payload)
		return
	}

	if payload.ToUid == "" {
		return
	}

	ctx := context.Background()

	// 黑名单拦截：任一方向拉黑则拒绝单聊投递，并提示发送方。
	if blocked, _ := isBlocked(ctx, c.UID(), payload.ToUid); blocked {
		c.finished = true
		c.SendNack(m.GetKey())
		bp, _ := json.Marshal(map[string]any{"event": "blocked", "to_uid": payload.ToUid})
		c.Send(Message.NewMessage(Message.Json, 0, bp))
		return
	}

	msg, err := service.SendChatMessage(ctx, c.UID(), payload.ToUid, Message.Text, payload.Content)
	if err != nil {
		log.Println("chat: save message failed:", err)
		c.SendNack(m.GetKey())
		return
	}

	c.Send(Message.NewMessage(Message.ACK, m.GetKey(), []byte(msg.MsgId)))
	c.finished = true // 已消费，短路 Echo，避免把发送帧回显给发送方

	err = c.server.RouteTo(payload.ToUid, Message.NewMessage(
		Message.Text, 0,
		BuildRealtimeText(c.UID(), msg.MsgId, msg.Content, msg.CreatedAt),
	))
	if err != nil {
		log.Printf("chat: route to %s failed (offline): %v", payload.ToUid, err)
	}
}

// handleGroupMessage 处理群聊：持久化群消息并扇出给所有在线群成员（跳过发送者）。
func handleGroupMessage(m *Message.Message, c *Client, payload TextChatPayload) {
	c.finished = true // 已消费，短路 Echo
	ctx := context.Background()

	members, err := getGroupMembers(ctx, payload.GroupId)
	if err != nil {
		log.Println("group: members query failed:", err)
		c.SendNack(m.GetKey())
		return
	}

	// 禁言拦截：发送者在禁言期内则拒绝并提示。
	for _, mem := range members {
		if mem.Uid == c.UID() && service.IsMuted(mem) {
			c.SendNack(m.GetKey())
			muted, _ := json.Marshal(map[string]any{"event": "group_muted_self", "group_id": payload.GroupId})
			c.Send(Message.NewMessage(Message.Json, 0, muted))
			return
		}
	}

	msg, err := sendGroupMessage(ctx, c.UID(), payload.GroupId, Message.Text, payload.Content)
	if err != nil {
		log.Println("group: save message failed:", err)
		c.SendNack(m.GetKey())
		return
	}

	c.Send(Message.NewMessage(Message.ACK, m.GetKey(), []byte(msg.MsgId)))

	frame := Message.NewMessage(Message.Text, 0,
		BuildGroupText(c.UID(), payload.GroupId, msg.MsgId, msg.Content, msg.CreatedAt))
	for _, mem := range members {
		if mem.Uid == c.UID() {
			continue
		}
		if err := c.server.RouteTo(mem.Uid, frame); err != nil {
			log.Printf("group: route to %s failed (offline): %v", mem.Uid, err)
		}
	}

	// @提醒：给被 @ 的在群成员额外推一条 mention 通知（在线即时弹出）。
	if len(payload.Mentions) > 0 {
		memberSet := make(map[string]bool, len(members))
		for _, mem := range members {
			memberSet[mem.Uid] = true
		}
		mentionPayload, _ := json.Marshal(map[string]any{
			"event":    "mention",
			"group_id": payload.GroupId,
			"from_uid": c.UID(),
		})
		for _, uid := range payload.Mentions {
			if uid == c.UID() || !memberSet[uid] {
				continue
			}
			_ = c.server.RouteTo(uid, Message.NewMessage(Message.Json, 0, mentionPayload))
		}
	}
}

func OfflineSyncHandler(m *Message.Message, c *Client) {
	c.finished = true // 同步请求已消费，短路 Echo

	// Json 帧分流：带 action 的为实时信号（typing/read），即发即弃，不落库、不归档。
	if len(m.Data) > 0 {
		var req struct {
			Action  string `json:"action"`
			ToUid   string `json:"to_uid"`
			GroupId string `json:"group_id"`
			UpTo    int64  `json:"up_to"`
		}
		if json.Unmarshal(m.Data, &req) == nil {
			switch req.Action {
			case "typing":
				handleTyping(c, req.ToUid, req.GroupId)
				return
			case "read":
				handleRead(c, req.ToUid, req.GroupId, req.UpTo)
				return
			}
		}
	}

	ctx := context.Background()
	msgs, err := getOfflineMessages(ctx, c.UID())
	if err != nil {
		log.Println("offline sync: query failed:", err)
		return
	}

	// 逐条发送并分配非 0 key，记录待确认；只有收到客户端 ACK 后才标记已读，
	// 避免"发完即标记"在客户端未收到时丢消息（实现 at-least-once）。
	for _, msg := range msgs {
		data, _ := json.Marshal(msg)
		key := c.nextOfflineKey()
		if err := c.SendBlob(key, data); err != nil {
			log.Println("offline sync: send failed:", err)
			return
		}
		c.trackOffline(key, msg.MsgId)
	}
}

// handleTyping 转发“正在输入”信号给单聊对端或群在线成员（即发即弃，best-effort，不落库）。
func handleTyping(c *Client, toUid, groupId string) {
	if groupId != "" {
		members, err := getGroupMembers(context.Background(), groupId)
		if err != nil {
			return
		}
		payload, _ := json.Marshal(map[string]any{"event": "typing", "from_uid": c.UID(), "group_id": groupId})
		frame := Message.NewMessage(Message.Json, 0, payload)
		for _, mem := range members {
			if mem.Uid == c.UID() {
				continue
			}
			_ = c.server.RouteTo(mem.Uid, frame)
		}
		return
	}
	if toUid == "" {
		return
	}
	payload, _ := json.Marshal(map[string]any{"event": "typing", "from_uid": c.UID()})
	_ = c.server.RouteTo(toUid, Message.NewMessage(Message.Json, 0, payload))
}

// handleRead 转发“已读”回执：单聊回给对端，群聊扇出给除阅读者外的在线成员。
// up_to 为阅读者已读到的最新消息时间戳（毫秒），由接收端据此标记自己发出的消息。
func handleRead(c *Client, toUid, groupId string, upTo int64) {
	if groupId != "" {
		members, err := getGroupMembers(context.Background(), groupId)
		if err != nil {
			return
		}
		payload, _ := json.Marshal(map[string]any{
			"event": "group_read", "from_uid": c.UID(), "group_id": groupId, "up_to": upTo,
		})
		frame := Message.NewMessage(Message.Json, 0, payload)
		for _, mem := range members {
			if mem.Uid == c.UID() {
				continue
			}
			_ = c.server.RouteTo(mem.Uid, frame)
		}
		return
	}
	if toUid == "" {
		return
	}
	payload, _ := json.Marshal(map[string]any{"event": "read", "from_uid": c.UID(), "up_to": upTo})
	_ = c.server.RouteTo(toUid, Message.NewMessage(Message.Json, 0, payload))
}

// AckHandler 处理客户端对离线消息的确认：收到 ACK(key) 后才将对应消息标记已读。
func AckHandler(m *Message.Message, c *Client) {
	c.finished = true // ACK 已消费，短路 Echo
	msgId, ok := c.takeOffline(m.GetKey())
	if !ok {
		return
	}
	if err := markMessagesRead(context.Background(), []string{msgId}); err != nil {
		log.Println("offline ack: mark read failed:", err)
	}
}

func init() {
	RegisterRoute(Message.Text, ChatMessageHandler)
	RegisterRoute(Message.Json, OfflineSyncHandler)
	RegisterRoute(Message.ACK, AckHandler)
}
