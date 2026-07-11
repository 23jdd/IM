# ChatIM

ChatIM 是一个 Go + Wails3 + Vue3 的即时通讯项目。

后端提供 HTTP API、自研 TCP 长连接、网关代理和消息路由；桌面客户端使用 Wails3 + Vue3 + Element Plus，并用本地 SQLite 保存登录态和消息缓存。

## 功能

- 注册、登录、个人资料、头像上传
- 好友申请、好友备注、删除好友、黑名单
- 单聊、群聊、群成员管理、群公告、禁言、踢人、转让群主
- 文本、图片、文件、表情消息
- 离线消息同步，逐条 ACK 后标记已读
- 消息撤回、正在输入、已读回执
- 同账号多端在线和多端同步
- 跨实例在线路由，Redis Pub/Sub 转发
- 朋友圈动态、点赞、评论
- 一对一 WebRTC 视频通话
- 本地聊天记录搜索、双击复制、导出、清空本机缓存

## 技术栈

- 后端：Go、Gin、MySQL、MongoDB、Redis、RabbitMQ
- 长连接：自研 TCP 二进制协议
- 桌面端：Wails3、Vue3、Element Plus、Pinia、Vite
- 本地存储：SQLite
- 视频通话：浏览器原生 WebRTC API

## WebRTC 说明

视频通话使用前端原生 WebRTC：

- `RTCPeerConnection`
- `navigator.mediaDevices.getUserMedia`
- SDP offer / answer
- ICE candidate

Go 服务端不转发音视频流，只通过已有 TCP Json 通道转发信令：

```json
{
  "action": "video_signal",
  "to_uid": "target_uid",
  "signal_type": "offer|answer|candidate|reject|end",
  "sdp": "...",
  "candidate": {},
  "call_id": "..."
}
```

媒体流是客户端点对点传输。当前只配置 STUN，复杂 NAT 或生产公网环境建议增加 TURN 服务。

## 目录结构

```text
IM/
├── main.go / config.go        后端入口和配置
├── http/                      HTTP API
├── tcp/                       TCP 长连接、消息路由、离线同步、WebRTC 信令
├── service/                   业务逻辑
├── model/                     数据模型
├── mysql/                     MySQL 访问
├── mongdb/                    MongoDB 访问
├── redis/                     在线状态和跨实例转发
├── rabbitmq/                  消息归档队列
├── gateway/                   TCP 网关代理
├── Tests/                     外部包测试
└── frontend/                  Wails 桌面客户端
    ├── authservice.go         HTTP 桥接
    ├── chatservice.go         TCP 桥接
    ├── videosignal.go         WebRTC 信令桥接结构
    ├── localstore.go          本地 SQLite
    └── frontend/              Vue 前端
```

## 配置

复制或编辑根目录配置文件：

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
jwt_secret: "change-me"
```

也可以用 `docker-compose.yml` 启动依赖服务。

## 启动后端

```bash
go mod tidy
go run .
```

默认服务：

- HTTP：`127.0.0.1:8080`
- TCP：`127.0.0.1:9000`
- 网关：`127.0.0.1:8000`

## 启动桌面客户端

```bash
cd frontend
go mod tidy
wails3 dev
```

前端单独构建：

```bash
cd frontend/frontend
npm install
npm run build
```

如果修改了暴露给前端的 Go 方法，需要重新生成 Wails 绑定：

```bash
cd frontend
wails3 generate bindings -clean=true
```

## 多端本地数据隔离

同一台机器启动多个客户端时，可以用不同数据目录隔离登录态和本地消息：

```powershell
$env:IM_DATA_DIR="C:\tmp\im-a"; .\im-client.exe
$env:IM_DATA_DIR="C:\tmp\im-b"; .\im-client.exe
```

## 测试

```bash
go test ./service/ ./tcp/ ./gateway/ ./Tests/ -count=1
cd frontend && go test . -count=1
cd frontend/frontend && npm run build
```

常用专项测试：

```bash
go test ./tcp -run "TestVideoSignal|TestBuildVideoSignal" -count=1
cd frontend && go test . -run "TestLocalStore" -count=1
```

## 注意事项

- 离线消息是 at-least-once 投递，客户端 ACK 后才标记已读。
- 正在输入、已读回执、WebRTC 信令都是在线实时信号，不做离线补偿。
- 本地清空聊天记录只删除本机 SQLite 缓存，不删除服务器历史。
- WebRTC 目前没有内置 TURN，公网生产环境需要补充 TURN。
- Wails 绑定文件是生成物；如果手动改了 Go 暴露方法，记得重新生成绑定。