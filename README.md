# ChatIM

一个即时通讯系统：**Go 后端**（HTTP + 自研 TCP 长连接 + 网关） + **Wails3 桌面客户端**（Vue3 + Element Plus），客户端与后端完全分离。

支持单聊 / 群聊、好友与群组管理、朋友圈、图片 / 文件 / 表情、消息撤回、历史翻页、黑名单、正在输入提示、已读回执、**一对一 WebRTC 视频通话**、**同账号多端在线与多端同步**、离线可靠投递、跨实例消息路由，以及客户端本地 SQLite 持久化。

---

## ✨ 功能特性

**账号与资料**
- 注册、登录（支持用 UID / 手机 / 邮箱 / 昵称 任一标识登录）、改密
- 个人资料、头像上传（图片存 MongoDB，MySQL 仅存其 `_id`）
- 点击任意头像查看用户资料卡片

**好友**
- 搜索 UID 查到用户后发起好友申请
- "新的朋友"列表，对方接受后双向成为好友
- 删除好友；单聊仅能对好友发起
- **好友备注**：在资料卡片修改备注，会话名实时跟随更新
- **黑名单**：拉黑 / 移出黑名单（支持拉黑非好友）；拉黑后任一方向都无法单聊投递

**单聊**
- 实时收发，消息携带发送者信息，离线进入离线表、上线拉取
- 发送状态（发送中 / 已送达）、**已读回执（已读）**
- **消息撤回**：2 分钟内可撤回，双方与本人多端实时变为"已撤回"
- **一对一视频通话**：WebRTC 媒体点对点传输，TCP Json 通道只转发 offer / answer / ICE candidate / 挂断等信令

**群聊**
- 创建群、邀请好友入群
- **加群需群主审批**（申请 → 群主在群成员面板通过 / 拒绝）
- 群成员列表；群消息显示**各发送者的头像与昵称**
- `@` 群成员，被 @ 的人实时收到提醒
- **群管理**：退群、解散、踢人、群主转让、禁言（按时长 / 解除）、群公告
- **群已读回执**：自己发出的群消息显示"N 人已读"

**历史消息**
- 上拉加载更早的消息（HTTP 分页拉取 MongoDB 归档，游标翻页 + 去重）
- 本地无历史的设备首次进会话自动回源拉取

**实时辅助提示**（走通知通道，即发即弃、不落库）
- **正在输入**：对端 / 群成员在你输入时看到"对方正在输入…"
- **已读回执**：阅读后回执已读位点，发送方据此显示"已读 / N 人已读"

**朋友圈（Moments）**
- 发布动态（文字 + 多图）、时间线（自己 + 好友）、点赞、评论、删除自己的动态、图片大图预览

**消息能力**
- 文本、图片（缩略图 + 大图预览）、文件（卡片 + 一键下载到本地）、内置表情包（可自定义素材）
- 当前会话聊天记录搜索、双击复制文本消息、导出聊天记录、清空本机缓存记录

**实时通知**（复用 TCP 通道，`event` 区分）
- 好友申请 / 好友接受 / 群邀请 / 入群审批结果 / 被 @ 提醒
- 消息撤回 / 解散 / 被踢 / 群主变更 / 群公告 / 被禁言 / 被拉黑 / 正在输入 / 已读回执 / 视频通话信令

**可靠性与分布式**
- 离线消息**逐条 ACK 后才标记已读**（at-least-once）
- **同账号多端在线**：同一 uid 可建立多条连接，消息扇出到全部在线端
- **多端同步**：自己发出的消息实时同步到本端其他设备，历史靠翻页补齐
- **跨实例路由**：在线状态写 Redis，目标在其他实例时经 Pub/Sub 转发
- 网关 TCP 透明代理 + 负载均衡 + 健康检查

**桌面客户端**
- 本地 SQLite 存**登录态**（替代 localStorage）与**按账号分库的消息历史**，支持本地搜索、导出与按会话清空
- 撤回状态本地持久化，重启后仍显示"已撤回"
- 数据目录可隔离（`IM_DATA_DIR`），方便同机多账号 / 多端测试

---

## 🏗️ 架构

```
┌──────────────────────────┐         ┌─────────────────────────────────────────┐
│   Wails3 桌面客户端        │         │              Go 后端                       │
│  (Vue3 + Element Plus)    │         │                                           │
│                          │  HTTP   │  http/  (Gin :8080)  用户/好友/群/消息/朋友圈 │
│  ┌────────────────────┐  ├────────►│                                           │
│  │ AuthService (HTTP) │  │         │  tcp/   (:9000) 自研二进制协议长连接          │
│  │ ChatService (TCP)  │  │  TCP    │   Verify → Router → Echo / 心跳 / 离线ACK    │
│  │ LocalStore(SQLite) │  ├────────►│   单聊·群扇出·多端同步·跨实例(Presence/Forwarder)│
│  └────────────────────┘  │         │                                           │
│  本地: session.db /      │         │  gateway/ (:8000) TCP 代理 + 负载均衡        │
│        messages_<uid>.db │         │  service/ 业务逻辑   model/ 数据模型          │
└──────────────────────────┘         │  mysql/ mongdb/ redis/ rabbitmq/ log/      │
                                     └─────────────────────────────────────────┘
                  MySQL（关系数据）   MongoDB（消息归档/图片/朋友圈）   Redis（在线表/转发）   RabbitMQ（归档队列）
```

**分层**：表示层（Wails 客户端）→ 业务逻辑层（`service`）→ 数据访问层（`mysql`/`mongdb`/`redis`/`rabbitmq`）→ 基础设施（网关、缓存、队列、日志）。

### 关键流程时序图

**① 登录 → 建立连接 → 离线同步**

```mermaid
sequenceDiagram
    autonumber
    participant C as 客户端(Vue)
    participant B as 客户端Go桥接
    participant H as HTTP :8080
    participant T as TCP :9000
    participant DB as MySQL

    C->>B: 登录(账号, 密码)
    B->>H: POST /api/user/login
    H->>DB: 查用户 + bcrypt 校验
    DB-->>H: 用户记录
    H-->>B: { token, uid, name }
    B-->>C: 登录成功(token 存本地 SQLite)

    C->>B: connect() + authTcp(token)
    B->>T: TCP Dial + Auth 帧
    T->>T: ParseToken → addClient(uid) → presence.SetOnline
    T-->>B: ACK

    C->>B: sync()
    B->>T: Json 帧(触发离线同步)
    T->>DB: 查 to_uid=uid AND status=0
    DB-->>T: 离线消息
    loop 每条离线消息
        T-->>B: Blob(key, 消息)
        B->>T: ACK(key)
        T->>DB: 标记该条已读
    end
```

**② 单聊实时投递 + 多端同步（含发送回执）**

```mermaid
sequenceDiagram
    autonumber
    participant A1 as A 设备1
    participant A2 as A 设备2
    participant T as TCP Server
    participant DB as MySQL
    participant MQ as RabbitMQ→Mongo
    participant Bc as B 客户端

    A1->>T: Text 帧 { to_uid:B, content }
    T->>DB: 存消息(status=0)
    T-->>MQ: best-effort 归档(异步)
    T-->>A1: ACK(key)
    Note over A1: 气泡标记「已送达」
    alt B 在线
        T->>Bc: Text 帧 { from_uid:A, to_uid:B }
        Note over Bc: 按 from_uid 归属并显示
    else B 离线
        Note over T,DB: 留在离线表, B 上线后 sync 拉取
    end
    T->>A2: Text 帧 { from_uid:A, to_uid:B }
    Note over A2: from===自己 → 归属会话 B, 标记「自己发出」
```

**③ 跨实例消息路由（A 连实例1，B 连实例2）**

```mermaid
sequenceDiagram
    autonumber
    participant A as A 客户端
    participant S1 as 后端实例1
    participant R as Redis
    participant S2 as 后端实例2
    participant Bc as B 客户端

    A->>S1: Text 帧(发给 B)
    S1->>S1: 本地无 B 的连接
    S1->>R: GetInstance(B) → 实例2
    S1->>R: PUBLISH route:实例2 { to:B, frame }
    R-->>S2: 订阅收到
    S2->>S2: DeliverLocal(B, frame) → B 的全部本地连接
    S2->>Bc: Text 帧
```

**④ 实时通知与事件（好友 / 群 / @ / 撤回 / 拉黑 …）**

```mermaid
sequenceDiagram
    autonumber
    participant A as 发起方
    participant H as HTTP / TCP 业务
    participant DB as DB
    participant N as Notifier(→RouteTo)
    participant Bc as 接收方(多端在线)

    A->>H: 申请加好友 / 入群审批 / 撤回 / 解散 / 踢人 / @某人 …
    H->>DB: 写关系/状态
    H->>N: notify(目标uid, event, data)
    N->>Bc: Json 帧 { event, ... } → 该 uid 全部在线连接
    Note over Bc: ElNotification / 刷新列表 / 本地状态变更
```

**⑤ 消息撤回（2 分钟内）**

```mermaid
sequenceDiagram
    autonumber
    participant A as 发送方
    participant H as HTTP :8080
    participant DB as MySQL/Mongo
    participant N as Notifier
    participant Bc as 对端/群成员/本人多端

    A->>H: POST /api/message/recall { msg_id }
    H->>H: 校验 是本人 且 2 分钟内
    H->>DB: MySQL status→2(已撤回) + best-effort 同步 Mongo
    H->>N: notify(相关方, "recall", { msg_id })
    N->>Bc: Json { event:recall, msg_id }
    Note over Bc: 气泡变「已撤回」并写入本地 SQLite(重启仍生效)
```

**⑥ 正在输入 / 已读回执（即发即弃，走通知通道）**

```mermaid
sequenceDiagram
    autonumber
    participant B as 阅读/输入方
    participant T as TCP Server
    participant A as 发送方(多端)

    B->>T: Json { action:typing, to_uid/group_id }
    T->>A: Json { event:typing, from_uid }
    Note over A: 标题显示「对方正在输入…」(5s 过期)

    B->>T: Json { action:read, to_uid/group_id, up_to }
    T->>A: Json { event:read / group_read, from_uid, up_to }
    Note over A: 单聊「已读」；群聊本地累计「N 人已读」
```

**⑦ 一对一视频通话信令（WebRTC）**

```mermaid
sequenceDiagram
    autonumber
    participant A as 主叫客户端
    participant T as TCP Server
    participant B as 被叫客户端

    A->>T: Json { action:video_signal, signal_type:offer, to_uid, sdp, call_id }
    T->>B: Json { event:video_signal, signal_type:offer, from_uid, sdp, call_id }
    B->>T: Json { action:video_signal, signal_type:answer, to_uid, sdp, call_id }
    T->>A: Json { event:video_signal, signal_type:answer, from_uid, sdp, call_id }
    loop ICE candidate
        A->>T: Json { action:video_signal, signal_type:candidate, candidate }
        T->>B: Json { event:video_signal, signal_type:candidate, candidate }
        B->>T: Json { action:video_signal, signal_type:candidate, candidate }
        T->>A: Json { event:video_signal, signal_type:candidate, candidate }
    end
    Note over A,B: 音视频媒体由 WebRTC 点对点传输，服务端只中转信令
```

---

## 🧰 技术栈

| 层 | 技术 |
|---|---|
| 客户端 | Wails3 (alpha)、Vue 3、Element Plus、Vite、Pinia、Vue Router |
| 客户端本地存储 | `modernc.org/sqlite`（纯 Go，无 CGO） |
| 后端 Web | Gin |
| 实时通信 | 自研 TCP 二进制协议、WebRTC（客户端 P2P 媒体）、ants 协程池、分级内存池 |
| 鉴权 | JWT（HS256，密钥可配置） |
| 关系数据库 | MySQL（go-zero `sqlx`） |
| 文档/二进制存储 | MongoDB（消息归档 / 历史翻页、头像 / 图片 / 文件、朋友圈） |
| 缓存 / 在线状态 | Redis（go-redis v9） |
| 消息队列 | RabbitMQ（消息异步归档到 Mongo） |
| 日志 | zap（文件 + 按大小切割） |
| 其他 | 雪花 ID、viper 配置 |

---

## 📁 目录结构

```
IM/                          # 后端（Go module: IM）
├── main.go / config.go      # 入口与配置
├── config.yaml              # 运行配置
├── http/                    # Gin HTTP API（用户/好友/群/消息/朋友圈/头像/文件）
├── tcp/                     # TCP 长连接引擎
│   ├── server.go            # Accept / 多连接 connSet / RouteTo·RouteToOthers / 跨实例 / 优雅关闭
│   ├── client.go            # 读写协程 / 心跳 / Handler 链 / 离线 ACK / 引用计数下线
│   ├── Handle.go router.go  # Verify(addClient) / Echo / 路由分发
│   ├── chat.go              # 单聊·群扇出·@提醒·离线同步·禁言/黑名单拦截·typing/read·多端同步
│   ├── video_signal.go      # WebRTC 信令请求/载荷构造与 offer/answer/ICE 转发
│   ├── presence.go          # Presence/Forwarder 抽象 + 内存实现
│   ├── pool.go context.go   # 内存池 / 连接级 KV
│   └── Message/             # 二进制协议编解码
├── service/                 # 业务逻辑层（注入式，可单测）
│   ├── message.go           # 收发 / 离线 / 撤回 / 历史翻页
│   ├── friend.go            # 好友 / 备注 / 黑名单
│   ├── group.go             # 群创建/邀请/审批/退群/解散/踢人/转让/禁言/公告
│   └── ...                  # login / moment / avatar / notify / user_brief
├── model/                   # 共享数据模型
├── mysql/ mongdb/ redis/ rabbitmq/ log/ gateway/ utils/
├── Tests/                   # 外部包测试（黑盒）
└── frontend/                # 桌面客户端（独立 Go module: im-client）
    ├── main.go              # Wails 应用入口，注册 3 个 Service
    ├── authservice.go       # HTTP 桥接（注册/登录/好友/群/消息/朋友圈/文件…）
    ├── chatservice.go       # TCP 桥接（收发/事件推送/typing/read/保存文件对话框）
    ├── videosignal.go       # WebRTC 信令桥接请求结构与 action 常量
    ├── localstore.go        # 本地 SQLite（session + 消息历史 + 撤回标记 + 搜索/清空）
    └── frontend/            # Vue 前端
        ├── src/views/       # Login / Main
        ├── src/components/  # 会话列表 / 聊天面板 / 视频通话 / 通讯录 / 朋友圈 / 信息卡片 ...
        ├── src/store/       # Pinia（user / chat）
        ├── src/api/         # 统一封装 Wails 绑定 + 事件
        └── public/stickers/ # 表情包资源（index.json + 图片）
```

---

## 📡 TCP 协议

定长 8 字节头 + 变长 body：

```
┌──────┬──────────┬──────────┬──────────┐
│ 1B   │ 3B       │ 4B       │ N bytes  │
│ type │ key      │ body len │ body     │
└──────┴──────────┴──────────┴──────────┘
```

| type | 值 | 说明 |
|------|----|------|
| ACK / Nack | 0 / 1 | 确认 / 拒绝（离线消息逐条 ACK 也走此通道） |
| Auth | 2 | JWT 认证 |
| HeartBeat | 3 | 心跳保活（写超时探活、续期在线状态） |
| Json | 4 | **多路复用**：客户端 → 触发离线同步 / 实时信号（`action:typing`、`action:read`、`action:video_signal`）；服务端 → 系统通知与事件（`event` 区分，如 `recall` / `group_*` / `typing` / `read` / `group_read` / `video_signal` / `blocked` …） |
| Text | 5 | 文本（图片/文件/表情复用，body 为带标记的 JSON；服务端下发携带 `from_uid`/`to_uid`/`group_id` 以支持归属与多端同步） |
| Blob | 6 | 二进制（离线消息逐条下发） |

读取有最大包体校验（`MaxBodyLen`）防止 OOM。

> **Json 帧分流**：服务端 `Json` 处理器先看 body 的 `action` 字段——`typing` / `read` / `video_signal` 走即发即弃的实时信号转发（不落库、不归档），否则按"触发离线同步"处理。

---

## ⚙️ TCP 引擎工作原理

### 连接生命周期

```
客户端 ──TCP──→ Server.Accept ──ants.Pool.Submit──→ Client.Start()
                                                      ├─ HeartBeat()      心跳 ticker → heart chan
                                                      ├─ ReadMessage()    解帧(TieredPool) → worker chan
                                                      └─ MessageHandler() Handler 链 / 处理 heart
```

- `Server.Start()` 监听端口，每个新连接提交到 **ants 协程池**，运行 `Client.Start()`。
- `Client.Start()` 启动 **3 个协程**：读循环、心跳、消息处理。
- 读循环用 **`TieredPool` 分级内存池**（8B~64KB）复用缓冲读取二进制帧，并校验包体上限。

### 多连接在线表（同账号多端）

- `clients` 由"uid → 单连接"升级为 **"uid → `connSet`（连接集合）"**，同一账号可同时建立多条连接。
- `RouteTo(uid)` 投递给该 uid 的**全部本地连接**；`RouteToOthers(uid, except)` 用于把自己发出的消息同步到本端其他设备。
- `Close` 采用**引用计数**：仅当某 uid 的最后一条连接断开时才 `presence.SetOffline`，避免单端断开误判整体离线。

### Handler 链

`main` 按序注册 `Verify → Router → Echo`，每条消息依次流经；业务 Handler **成功消费后置 `finished` 短路后续**（避免被 Echo 回显）：

```
MessageHandler():
  for h := range clientHandlers {
      h(message, client)
      if client.finished { break }   // 消费即短路
  }
  client.finished = false            // 重置
```

| Handler | 职责 |
|---|---|
| **Verify** | 只处理 `Auth`：`ParseToken` → `addClient(uid)`（多端在线） → `presence.SetOnline` → ACK；置 `finished` |
| **Router** | 跳过未认证；按 `type` 查 `bizRoutes`，分发到 `ChatMessageHandler`（单聊 / 群扇出，含禁言/黑名单拦截、多端同步）/ `OfflineSyncHandler`（离线同步 + typing/read/video_signal 分流）/ `AckHandler` |
| **Echo** | 兜底回显未被消费的消息 |

### 心跳保活与优雅退出

```
HeartBeat() ──ticker(10s)──→ OnTicker() ──heart chan(cap=1, 非阻塞)──→ MessageHandler()
                                                                          ├─ SendHeart(key)
                                                                          ├─ 续期 presence 在线状态
                                                                          └─ 写失败 → Close()
```

- `heart` channel 容量为 1、非阻塞投递，防止 ticker 堆积。
- 每次写设置 **写超时**，对端假死时写会失败 → 触发 `Close()`，使心跳真正具备探活能力。
- 协程通过 `quit` channel 干净退出，无忙等、无泄漏。

### 优雅关闭

```
SIGINT/SIGTERM → ShutDown:
  1. close(quit)              → Accept 循环退出
  2. listener.Close()         → 拒绝新连接
  3. 快照全部连接后逐个 Close   → 关连接 + 引用计数下线
  4. workerPool.Release()     → 等待 ants 协程池
```

### 关键设计决策

1. **写串行化**：所有 `Send*` 共享 `writeMu` 串行化 TCP 写入，并带写超时。
2. **一次清理**：`sync.Once` 确保连接清理只执行一次，防止重复计数/重复下线。
3. **并发安全**：`closed` 用 `atomic.Bool`、`uid` 用 `RWMutex` 访问器、多连接集合用 `RWMutex`（`-race` 验证无竞争）。
4. **可靠离线投递**：离线消息逐条 `SendBlob(key)`，**客户端 ACK 后才标记已读**（at-least-once）。
5. **多端一致**：自己发出的消息经 `RouteToOthers` 同步到本端其他设备，离线设备的历史靠翻页补齐。
6. **内存复用**：`TieredPool` 分级内存池复用读缓冲，减少 GC。
7. **协程池边界**：ants 池只用于 `Start()` 的提交，心跳/消息处理为独立 goroutine。

---

## 💾 数据存储分工

- **MySQL**
  - `user`、`friend_relation`、`group_info`、`group_member`、`group_join_request`、`chat_message`（离线消息）。建表脚本见 `mysql/sql/*.sql`。
  - `friend_relation.status`：`0=待确认 / 1=已添加 / 2=已拉黑`；`remark` 为好友备注。
  - `group_member`：`role`（`0=成员 / 1=管理员 / 2=群主`）、`mute_until`（禁言截止时间）。
  - `group_info`：`status`（`0=正常 / 1=已解散`）、`announcement`（群公告，旧库升级见 `group.sql` 末尾 ALTER 注释）。
  - `chat_message.status`：`0=未读 / 1=已读 / 2=已撤回`。
- **MongoDB**：`messages`（全量消息归档 / 历史翻页，撤回时同步状态）、`avatars`（头像 / 图片 / 文件二进制）、`moments`（朋友圈）。
- **Redis**：在线注册表 `online:<uid> → 实例ID` + 跨实例转发 `route:<实例>`（可用时启用，否则单机内存表）。
- **RabbitMQ**：`im.message` 队列，消息异步归档到 MongoDB（best-effort，不阻断主流程）。

---

## 🚀 快速开始

### 依赖
- Go 1.25+
- Node.js + npm，Wails3 CLI（`wails3`）
- MySQL、MongoDB（必需）；Redis、RabbitMQ（可选，缺失时降级为单机/跳过归档）

### 1) 准备数据库
```sql
CREATE DATABASE IF NOT EXISTS Im DEFAULT CHARSET utf8mb4;
-- 依次执行 mysql/sql/ 下的建表脚本（user / friend / group / chat_message）
-- 既有库升级群公告列：见 mysql/sql/group.sql 末尾的 ALTER 注释
```
> 注意：各表 collation 需统一（建议默认 `utf8mb4_0900_ai_ci`），否则跨表 JOIN 会报 collation 冲突。

### 2) 配置
编辑根目录 `config.yaml`：
```yaml
http_address: 127.0.0.1
http_port: 8080
tcp_address: 127.0.0.1
tcp_port: 9000
data_source: "root:1234@tcp(127.0.0.1:3306)/Im?parseTime=true&loc=UTC"
mongo_uri: "mongodb://127.0.0.1:27017"
mongo_db: "im"
redis_addr: "127.0.0.1:6379"
rabbitmq_url: "amqp://guest:guest@127.0.0.1:5672/"
gateway_port: 8000
jwt_secret: "change-me-in-production"
backend_addrs: ["127.0.0.1:9000"]
```

### 3) 启动后端
```bash
go run .
```
将同时启动 HTTP(:8080)、TCP(:9000)、网关(:8000)。

### 4) 启动客户端
```bash
cd frontend
wails3 dev          # 开发模式（热重载）
# 或打包：
wails3 build
```

> 修改了暴露给前端的 Go 方法（`authservice.go` / `chatservice.go` / `localstore.go`）后，需重新生成绑定：
> ```bash
> cd frontend && wails3 generate bindings -clean=true
> ```

同机并行测试多账号 / 同账号多端时，给每个客户端实例指定独立数据目录：
```powershell
$env:IM_DATA_DIR="C:\tmp\imA"; .\im-client.exe
$env:IM_DATA_DIR="C:\tmp\imB"; .\im-client.exe
```

---

## 🎨 自定义表情包

1. 把图片放到 `frontend/frontend/public/stickers/`；
2. 在同目录 `index.json` 中列出文件名（JSON 数组）；
3. `npm run build` 后，聊天输入框的「😀」按钮即显示这些表情。

表情不经服务器上传，消息只携带文件名，故所有客户端需放置同一套素材。

---

## 🧪 测试

业务逻辑采用**依赖注入**（函数变量），可在不连真实数据库的情况下白盒单测：

```bash
go test ./service/ ./tcp/ ./gateway/ ./Tests/ -count=1
# 并发安全验证：
go test ./service/ ./tcp/ ./gateway/ -race -count=1
```

覆盖：连接引擎并发安全、协议编解码与长度防护、心跳/优雅关闭、离线 ACK 投递、**同账号多连接投递与多端自发同步**、登录、好友申请/接受/删除/**备注/黑名单与发送拦截**、群创建/邀请/审批、**退群/解散/踢人/转让/禁言/公告**、@提醒扇出、**消息撤回**、**历史翻页**、**本地消息搜索/清空**、**正在输入/已读回执转发**、**WebRTC 视频信令构造与转发**、跨实例路由、朋友圈、头像/查用户、网关半关闭等。

> `Tests/TestTieredPoolPutAndReuse` 偶发失败属 `sync.Pool` 在 `-race` 下被 GC 清空的非确定性问题，与业务无关。

---

## 📝 设计要点与已知限制

- **客户端/后端分离**：WebView 无法直接开原始 TCP，故客户端 Go 侧实现 TCP 桥接，向前端暴露方法并以事件推送消息；HTTP 也经 Go 侧转发以规避跨域。
- **可靠投递**：离线消息逐条 ACK 后才标记已读，避免"发完即标记"导致丢消息。
- **多端在线/同步**：同一 uid 多连接共存，消息扇出到全部在线端；自己发出的消息实时同步到本端其他设备，离线设备的历史由翻页补齐。
- **实时信号即发即弃**：正在输入、已读回执与 WebRTC 信令复用通知通道（`Json` + `RouteTo`），仅在线投递、零持久化；输入提示 5s 过期、发送端 2.5s 节流。
- **撤回一致性**：撤回同时更新 MySQL 与（best-effort）Mongo，并写入客户端本地 SQLite，重启后仍显示"已撤回"。
- **水平扩展**：通过 Redis 在线表 + Pub/Sub 转发实现跨实例消息路由；网关负载均衡到多后端。
- **限制**：
  - 跨实例多端：在线表为 `online:<uid> → 单实例`，同账号同时连不同实例时只向其一转发（如需完全支持可升级为 `GetInstances` + Redis SET，向所有相关实例转发）。
  - 离线撤回/已读/输入为 best-effort，离线期间不回溯；群"N 人已读"为客户端本地累计，不入库。
  - WebRTC 当前只内置公共 STUN，媒体走客户端点对点；复杂 NAT / 公网生产环境通常需要补 TURN 服务。
  - 网关的服务发现为静态 `backend_addrs`（非 etcd）；离线消息默认 `LIMIT 200`、历史每页默认 30（上限 100）；表情/文件大小有前端限制。

---

## 📦 主要 HTTP 接口（鉴权用 `Authorization: Bearer <token>`）

```
# 用户 / 资料 / 文件
POST /api/user/register | login                 GET /api/user/info?uid=
GET/PUT /api/user/profile   PUT /api/user/password
POST /api/user/avatar   GET /api/avatar?id=   GET /api/avatar/by-uid?uid=
POST /api/file/upload

# 消息
POST /api/message/recall                        GET /api/message/history?peer=|group=&before=&limit=

# 好友 / 黑名单 / 备注
GET  /api/friend/list   POST /api/friend/request | accept | remove   GET /api/friend/requests
POST /api/friend/block | unblock | remark        GET /api/friend/blocklist
GET  /api/conversation/list

# 群组与群管理
POST /api/group/create | join | invite | approve | reject
POST /api/group/leave | disband | kick | transfer | mute | announce
GET  /api/group/list | members | requests | info

# 朋友圈
POST /api/moment/publish | like | comment | delete   GET /api/moment/timeline
```

实时消息、离线同步、正在输入 / 已读回执、WebRTC 视频信令、好友/群/@/撤回/拉黑等通知均通过 TCP(:9000) 长连接。
