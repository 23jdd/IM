# IM 系统工作原理

## 架构概览

```
客户端 ──TCP──→ Server.Accept
                  │
            ants.Pool.Submit
                  │
             Client.Start()
             /       |        \
    HeartBeat()  ReadMessage()  MessageHandler()
        │             │               │
    heart chan    worker chan    Handler 链:
        │             │          Verify → Router → ChatMessage → Echo
    OnTicker()   Decode(buf)
        │
    SendHeart()
```

## 核心链路

### 1. 连接建立

1. `Server.Start()` 监听端口，每个连接提交到 ants 协程池
2. `Client.Start()` 启动 3 个协程：读循环、心跳、消息处理
3. 读循环通过 `TieredPool` 内存池读取二进制帧

### 2. 认证流程

```
客户端 ──Auth(token)──→ TCP Server
                         Verify Handler:
                           ParseToken(JWT)
                           ├─ 成功 → uid=claim.Uid → clients.Store(uid, client) → SendAck
                           └─ 失败 → SendNack
                           c.finished = true (短路后续 Handler)
```

### 3. 消息发送

```
发送方 ──Json({to_uid, content})──→ TCP Server
                                     Verify (通过)
                                     Router → ChatMessageHandler:
                                       1. json.Unmarshal → TextChatPayload
                                       2. service.SendChatMessage → MySQL 持久化
                                       3. SendAck(key) → 发送方确认
                                       4. RouteTo(to_uid, msg)
                                          ├─ 在线 → Send() → 目标客户端收到
                                          └─ 离线 → 消息已存 MySQL，等上线拉取
```

### 4. 离线消息

```
上线方 ──Auth──→ TCP Server
                 Verify Handler 通过
                 客户端发送 Json({action:"sync"})
                 Router → OfflineSyncHandler:
                   1. FindOfflineMessages(uid) → MySQL 查询
                   2. 逐条 SendBlob → 客户端
                   3. MarkMessagesRead(ids) → MySQL 更新
```

### 5. 心跳保活

```
HeartBeat() ──ticker(10s)──→ OnTicker() ──heart chan──→ MessageHandler()
                                                            │
                                                        IncrKey()
                                                        SendHeart(key)
                                                        ├─ 成功 → 继续
                                                        └─ 失败 → closed=true → 退出 → Close()
```

`heart` channel 容量为1，非阻塞投递，防止 ticker 堆积。

### 6. 优雅关闭

```
SIGINT/SIGTERM → NotifyServer → ShutDown:
  1. close(quit)          → Accept 循环退出
  2. listener.Close()     → 拒绝新连接
  3. Range clients        → 逐个 Client.Close()
  4. workerPool.Release() → 等待 ants 协程池
```

## 协议格式

```
┌──────┬──────────┬──────────┬──────────┐
│ 1B   │ 3B       │ 4B       │ N bytes  │
│ type │ key      │ body len │ body     │
└──────┴──────────┴──────────┴──────────┘
```

| type | 值 | 说明 |
|------|----|------|
| ACK | 0 | 确认响应 |
| Nack | 1 | 拒绝响应 |
| Auth | 2 | JWT 认证 |
| HeartBeat | 3 | 心跳 |
| Json | 4 | 系统消息（离线同步） |
| Text | 5 | 文本聊天 |
| Blob | 6 | 二进制数据 |

## 目录结构

```
IM/
├── main.go                入口
├── config.go/yaml         配置（HTTP/TCP/MySQL）
├── tcp/                   TCP 长连接引擎
│   ├── server.go          服务端（Accept/ants池/clients Map/RouteTo/优雅关闭）
│   ├── client.go          客户端连接（读写协程/心跳/Handler链/writeMu）
│   ├── Handle.go          内置 Handler (Echo/Verify)
│   ├── router.go          消息路由分发器 (RegisterRoute)
│   ├── chat.go            单聊处理器 (ChatMessageHandler/OfflineSyncHandler)
│   ├── context.go         连接级 KV 存储
│   ├── pool.go            分级内存池 (TieredPool)
│   └── Message/           二进制协议 (编解码/类型定义)
├── http/                  HTTP API
│   ├── server.go          Gin 路由注册
│   └── User/handle.go     用户 API (注册/登录/资料/改密)
├── service/              业务逻辑层
│   ├── user.go            用户服务
│   └── message.go         消息服务
├── model/                共享数据模型
│   └── message.go         ChatMessage 结构体
├── mysql/                MySQL 数据层
│   ├── init.go           初始化连接
│   ├── models.go         导出 UserModel
│   ├── message.go         消息 CRUD
│   ├── model/             goctl 生成的 User Model
│   └── sql/               DDL 脚本
├── utils/                工具
│   ├── snowflake.go      雪花 ID 生成器
│   └── jwt.go            JWT 生成/解析
├── Tests/                测试（80 个用例）
│   ├── message_test.go   协议编解码测试
│   ├── pool_test.go      内存池测试
│   ├── context_test.go   Context 测试
│   ├── handler_test.go   Handler/Client 发送测试
│   ├── server_test.go    Server 测试
│   ├── verify_test.go    Verify/RouteTo 测试
│   ├── router_test.go    Router 分发测试
│   └── connect_test.go   真实 TCP 连接测试
├── gateway/              WebSocket 接入（待建）
├── mongdb/                MongoDB 消息存储（待建）
├── redis/                Redis 缓存（待建）
├── rabbitmq/             消息队列（待建）
└── log/                  日志（待建）
```

## Handler 链执行顺序

```
main.go:
  server.AddHandler(tcp.Verify)    // 1. JWT 认证
  server.AddHandler(tcp.Router)    // 2. 消息路由分发
  server.AddHandler(tcp.Echo)      // 3. 兜底 echo

每个消息按顺序经过 Handler 链：
  MessageHandler() {
    for _, h := range clientHandlers {
      h(message, client)
      if client.finished { break }   // finished=true 短路
    }
    client.finished = false          // 重置
  }
```

**Verify**: 只处理 Auth，设 `finished=true` 阻止后续 Handler
**Router**: 跳过未认证，按 type 查 bizRoutes map，调用对应 BusinessHandler
**Echo**: 兜底，任何没被 Router 消费的消息原样回显

## 关键设计决策

1. **Write MuTex**: 所有 `Send*` 共享一个 `writeMu`，串行化 TCP 写入
2. **Close Once**: `sync.Once` 确保清理只执行一次，防止重复计数
3. **非阻塞心跳**: `select + default` 丢弃堆积心跳，防止 channel 阻塞 ticker
4. **内存池**: 7 级 `TieredPool`(8B~64KB)，读缓冲复用，减少 GC
5. **ants 池**: 只为 `Start()` 提交使用，HeartBeat/MessageHandler 是 raw goroutine
