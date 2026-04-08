# GoChat

基于 Go + React 的全栈实时聊天应用，支持私聊、群聊、文件传输和音视频通话。

## 功能特性

- **即时通讯**：基于 WebSocket 的实时私聊与群聊
- **消息管理**：支持文本、图片、文件发送，消息撤回，已读标记
- **好友管理**：申请添加好友、审核、删除、拉黑/解除拉黑
- **群组管理**：创建群组、加入/退出群组、移除成员、解散群组，支持群名/公告/头像修改
- **音视频通话**：基于 WebRTC 的点对点音视频通话
- **多种登录方式**：密码登录 + 邮箱验证码登录
- **管理后台**：用户管理、封禁、群组管理、系统统计
- **在线状态**：实时显示好友在线状态与未读消息数

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端语言 | Go 1.24 |
| Web 框架 | Gin |
| ORM | GORM + MySQL 8.0 |
| 缓存 | Redis 7 |
| 消息队列 | Apache Kafka 3.7 |
| 实时通信 | WebSocket (gorilla/websocket) |
| 认证 | JWT |
| 前端框架 | React 18 + TypeScript |
| 构建工具 | Vite |

## 项目结构

```
GoChat/
├── back/
│   ├── cmd/main.go              # 后端入口
│   ├── internal/
│   │   ├── chat/                # WebSocket Hub、Kafka 生产/消费者
│   │   ├── config/              # 配置加载、DB/Redis 初始化
│   │   ├── controller/v1/       # HTTP 控制器
│   │   ├── dao/                 # 数据访问层
│   │   ├── dto/                 # 请求/响应结构体
│   │   ├── middleware/          # JWT 鉴权、CORS、管理员检查
│   │   ├── model/               # GORM 数据模型
│   │   ├── router/              # 路由注册
│   │   └── service/             # 业务逻辑层
│   └── utils/                   # JWT、Redis、邮件等工具函数
├── web/
│   └── src/
│       ├── api/                 # Axios 封装 + WebSocket 客户端
│       ├── components/          # UI 组件
│       ├── context/             # React Context（认证状态）
│       ├── hooks/               # 自定义 Hooks（WebRTC 等）
│       ├── pages/               # 页面组件
│       ├── types/               # TypeScript 类型定义
│       └── utils/               # 工具函数
├── static/                      # 静态资源（头像、文件）
├── docker-compose.yml           # 基础服务编排
└── back/internal/config/config.toml  # 应用配置文件
```

## 消息流转架构

```
WebSocket 客户端
      │
      ▼
  Kafka 生产者
      │
      ├──▶ Dispatcher（实时推送到 WebSocket 客户端）
      ├──▶ Persist（写入 MySQL）
      └──▶ Cache（更新 Redis）
```

## 快速开始

### 1. 启动基础服务

使用 Docker Compose 启动 MySQL、Redis 和 Kafka：

```bash
docker-compose up -d
```

> 默认端口：MySQL `:3307`（映射容器 3306）、Redis `:6379`、Kafka `:9092`

### 2. 配置应用

编辑 `back/internal/config/config.toml`，按需修改数据库、Redis、Kafka 连接信息及邮件 SMTP 配置。

### 3. 启动后端

```bash
# 在项目根目录执行
go run ./back/cmd/main.go
```

后端启动后监听 `http://localhost:8000`，首次启动会自动执行数据库迁移。

### 4. 启动前端

```bash
cd web
npm install
npm run dev
```

前端开发服务器运行在 `http://localhost:5173`。

## API 说明

- 所有受保护接口需携带请求头：`Authorization: Bearer <JWT>`
- Token 刷新：`POST /auth/refresh`
- WebSocket 连接：`ws://localhost:8000/wss?token=<JWT>`
- 文件上传：`POST /message/uploadFile`（multipart/form-data）

### 主要接口分组

| 分组 | 前缀 | 说明 |
|------|------|------|
| 认证 | `/login` `/register` | 登录、注册、邮箱验证码登录 |
| 联系人 | `/contact/` | 好友申请、列表、删除、拉黑 |
| 群组 | `/group/` | 创建、加入、退出、成员管理 |
| 会话 | `/session/` | 打开/删除会话、获取会话列表 |
| 消息 | `/message/` | 消息列表、文件上传、撤回、标记已读 |
| 管理 | `/admin/` | 用户封禁、群组管理、系统统计（管理员专用）|

## 环境要求

- Go 1.24+
- Node.js 18+
- Docker & Docker Compose（用于运行基础服务）




<img width="1823" height="1069" alt="image" src="https://github.com/user-attachments/assets/d99bd5f1-fc7b-4edd-9521-60378b6fea72" />
<img width="1826" height="1061" alt="image" src="https://github.com/user-attachments/assets/656aa3a2-e2a9-4091-95df-c9b38f8ad27e" />
<img width="1825" height="1062" alt="image" src="https://github.com/user-attachments/assets/2c53184e-6721-412e-a229-59b09201768e" />
<img width="1828" height="1057" alt="image" src="https://github.com/user-attachments/assets/01c46061-a5f8-4726-bfe1-5651c43063ef" />
<img width="1824" height="1063" alt="image" src="https://github.com/user-attachments/assets/353c7ece-20d2-4055-82ac-e16a015bfc56" />




