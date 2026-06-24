package service

import "encoding/json"

// pushNotification 由 main 注入（绑定到 tcp.Server.RouteTo），用于向在线用户推送系统通知。
// 解耦 service 与 tcp，避免循环依赖；离线用户由 RouteTo 返回错误，通知 best-effort 丢弃。
var pushNotification func(toUid string, payload []byte)

// SetNotifier 由 main 在启动时注入推送实现（绑定到在线用户的下行通道）。
func SetNotifier(fn func(toUid string, payload []byte)) {
	pushNotification = fn
}

// notify 构造 {event, ...data} 的 JSON 通知并推送给 toUid（若已注入 notifier）。
func notify(toUid, event string, data map[string]any) {
	if pushNotification == nil {
		return
	}
	m := map[string]any{"event": event}
	for k, v := range data {
		m[k] = v
	}
	payload, err := json.Marshal(m)
	if err != nil {
		return
	}
	pushNotification(toUid, payload)
}
