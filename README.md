# GoChat

基于 **Go + React** 的全栈实时聊天应用，支持私聊、群聊、文件传输和音视频通话。

---

## 功能特性

- **即时通讯** — 基于 WebSocket 的实时私聊与群聊，消息经 Kafka 分发、持久化、缓存三路并行处理
- **消息管理** — 文本、图片、文件发送，消息撤回，已读标记，历史消息分页加载
- **好友管理** — 申请添加好友、审核通过/拒绝、拉黑/解除拉黑、删除好友
- **群组管理** — 创建群组、直接加入或提交申请、退出/解散、移除成员、修改群名/公告/头像
- **音视频通话** — 基于 WebRTC + TURN 中继的点对点音视频通话
- **多种登录方式** — 账号密码登录 + 邮箱验证码登录
- **会话管理** — 私聊/群聊会话列表，删除会话，未读消息数角标
- **管理后台** — 用户封禁/解封、群组强制解散、系统数据统计（管理员专用）
- **安全认证** — JWT 双 Token（access + refresh）机制，密码 bcrypt 加密存储，自动迁移旧账号

---

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端语言 | Go 1.24 |
| Web 框架 | Gin |
| ORM | GORM + MySQL 8.0 |
| 缓存 | Redis 7 |
| 消息队列 | Apache Kafka 3.7 (KRaft 模式) |
| 实时通信 | WebSocket (gorilla/websocket) |
| 认证 | JWT (golang-jwt/jwt v5) + bcrypt |
| 前端框架 | React 18 + TypeScript |
| 构建工具 | Vite |
| 音视频 | WebRTC + TURN (coturn) |

---

## 项目结构

```
GoChat/
├── back/
│   ├── cmd/main.go                  # 入口：加载配置、初始化服务、启动 HTTP
│   ├── internal/
│   │   ├── chat/                    # WebSocket Hub、Kafka 生产者/消费者（分发/持久化/缓存）
│   │   ├── config/                  # TOML 配置加载、DB/Redis 初始化
│   │   │   └── config.toml          # 应用配置文件
│   │   ├── controller/v1/           # HTTP 控制器（auth、user、group、contact、message、session、admin）
│   │   ├── dao/                     # 数据访问层（GORM 查询）
│   │   ├── dto/                     # 请求/响应结构体
│   │   ├── middleware/              # JWT 鉴权、CORS、管理员权限检查
│   │   ├── model/                   # GORM 数据模型
│   │   ├── router/                  # Gin 路由注册
│   │   └── service/                 # 业务逻辑层
│   └── utils/                       # JWT、Redis 客户端、邮件(SMTP)、ID 生成
├── web/
│   ├── src/
│   │   ├── api/
│   │   │   ├── api.ts               # Axios 封装 + 全部 REST API 调用（含自动刷新 Token 拦截器）
│   │   │   └── socket.ts            # WebSocket 客户端（重连、群订阅、消息收发）
│   │   ├── components/              # 可复用 UI 组件（侧边栏、聊天界面、弹窗等）
│   │   ├── context/                 # React Context（AuthContext 用户认证状态）
│   │   ├── hooks/                   # 自定义 Hooks（useAuth、useWebRTC、useContactActions 等）
│   │   ├── pages/                   # 页面组件（Login、Register、CaptchaLogin、Chat、Profile）
│   │   ├── types/                   # TypeScript 类型定义
│   │   ├── utils/                   # 工具函数（session token 存取）
│   │   └── config.ts                # API/WS 地址配置（读取环境变量）
│   └── .env.production              # 生产环境变量（VITE_API_BASE、VITE_WS_BASE）
├── static/
│   ├── avatars/                     # 用户/群头像上传目录
│   └── files/                       # 聊天文件上传目录
├── docker-compose.yml               # MySQL + Redis + Kafka 容器编排
└── turnserver.conf                  # coturn TURN 服务器配置
```

---

## 消息流转架构

```
浏览器 (WebSocket)
       │  发送消息
       ▼
  WebSocket Hub
       │
       ▼
  Kafka Producer  ──────────────────────┐
       │                                │
       ▼                                ▼
  Dispatcher Consumer          Persist Consumer        Cache Consumer
  （实时推送到目标客户端）       （写入 MySQL）          （更新 Redis）
```

---

## 快速开始

### 1. 启动基础服务

使用 Docker Compose 一键启动 MySQL、Redis 和 Kafka：

```bash
docker-compose up -d
```

> 默认端口映射：MySQL `127.0.0.1:3307`、Redis `127.0.0.1:6379`、Kafka `127.0.0.1:9092`

### 2. 配置应用

编辑 `back/internal/config/config.toml`，按需修改以下配置：

```toml
[mysqlConfig]
host = "127.0.0.1"
port = 3307
user = "root"
password = "123456"
databaseName = "gochat"

[redisConfig]
host = "127.0.0.1"
port = 6379
password = "123456"

[email]
smtp_host = "smtp.gmail.com"
smtp_port = 465
username   = "your@email.com"
password   = "your-app-password"   # 邮箱授权码，非登录密码
```

### 3. 启动后端

```bash
# 项目根目录
go run ./back/cmd/main.go
```

后端监听 `http://localhost:8000`，首次启动时 GORM AutoMigrate 自动建表并应用索引。

### 4. 启动前端

```bash
cd web
npm install
npm run dev      # 开发服务器：http://localhost:5173
```

### 5. （可选）启动 TURN 服务器

音视频通话需要 TURN 中继服务器（用于 NAT 穿透）。可使用 coturn：

```bash
turnserver -c turnserver.conf
```

---

## 生产部署

### 前端构建

```bash
cd web
# 修改 .env.production 中的域名
# VITE_API_BASE=https://yourdomain.com
# VITE_WS_BASE=wss://yourdomain.com
npm run build    # 产物输出到 web/dist/
```

### 后端构建

```bash
go build -o chatapp-server ./back/cmd/main.go
./chatapp-server
```

---

## API 说明

所有受保护接口需携带请求头：`Authorization: Bearer <access_token>`

| 分组 | 路径前缀 | 说明 |
|------|----------|------|
| 认证 | `/login` `/register` `/auth/refresh` | 登录、注册、Token 刷新、登出 |
| 邮箱验证码 | `/captcha/` | 发送验证码、验证码登录 |
| 用户 | `/user/` `/api/user/` | 更新用户信息、获取当前用户信息 |
| 联系人 | `/contact/` | 申请/审核/删除/拉黑好友，获取列表 |
| 群组 | `/group/` `/apply/` | 创建/加入/退出/解散群聊，成员管理，入群申请审核 |
| 会话 | `/session/` | 打开/删除会话，获取会话列表 |
| 消息 | `/message/` | 消息列表、文件上传、撤回、标记已读、清除聊天记录 |
| WebRTC | `/turn/credentials` | 获取 TURN 动态凭证 |
| 管理员 | `/admin/` | 用户封禁、群组解散、系统统计（需管理员权限）|

**WebSocket 连接：**
```
ws://localhost:8000/wss?token=<access_token>
```

**消息格式（JSON）：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `type` | int | 消息类型：`0`=文本，`1`=文件，`2`=通话信令 |
| `action` | string | 信令动作：`join_group`、`call_invite`、`call_answer`、`call_candidate`、`call_end`、`group_dismiss` |

---

## 环境要求

- **Go** 1.24+
- **Node.js** 18+
- **Docker & Docker Compose**（运行 MySQL/Redis/Kafka）
- **coturn**（可选，音视频通话 NAT 穿透）

---

## 截图

<img width="1823" height="1069" alt="image" src="https://github.com/user-attachments/assets/d99bd5f1-fc7b-4edd-9521-60378b6fea72" />
<img width="1826" height="1061" alt="image" src="https://github.com/user-attachments/assets/656aa3a2-e2a9-4091-95df-c9b38f8ad27e" />
<img width="1825" height="1062" alt="image" src="https://github.com/user-attachments/assets/2c53184e-6721-412e-a229-59b09201768e" />
<img width="1828" height="1057" alt="image" src="https://github.com/user-attachments/assets/01c46061-a5f8-4726-bfe1-5651c43063ef" />
<img width="1824" height="1063" alt="image" src="https://github.com/user-attachments/assets/353c7ece-20d2-4055-82ac-e16a015bfc56" />
