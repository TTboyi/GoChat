# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A full-stack real-time chat application with Go backend and React frontend, supporting private/group messaging, file sharing, and audio/video calling.

## Development Commands

### Backend (Go)
```bash
# Run from project root
go run ./back/cmd/main.go

# Build
go build -o chatapp-server ./back/cmd/main.go

# Tidy dependencies
go mod tidy
```

### Frontend (React + Vite)
```bash
cd web
npm install
npm run dev      # Dev server at http://localhost:5173
npm run build    # Production build to web/dist
npm run preview  # Preview production build
```

### Required Services
The backend expects these running locally:
- **MySQL** on `:3306`, database `gochat`, user `root`, password `123456`
- **Redis** on `:6379`, password `123456`
- **Kafka** on `:9092`, topic `chat_message`

Configuration lives in `back/internal/config/config.toml`.

## Architecture

### Backend (`back/`)

**Entry point**: `back/cmd/main.go` — loads config, initializes DB/Redis/Kafka, starts HTTP server on `:8000`.

**Package layout**:
- `internal/config/` — TOML config loading, DB/Redis initialization
- `internal/router/` — Gin route registration (REST + WebSocket at `/wss`)
- `internal/controller/v1/` — HTTP handlers (auth, user, group, contact, message, session, admin, WebSocket)
- `internal/service/` — Business logic layer
- `internal/dao/` — Data access layer (GORM queries)
- `internal/model/` — GORM models: `UserInfo`, `GroupInfo`, `UserContact`, `ContactApply`, `Message`, `Session`
- `internal/chat/` — WebSocket hub, Kafka producer/consumers, real-time routing
- `internal/middleware/` — JWT auth, CORS, admin checks
- `internal/dto/` — Request/response structs
- `utils/` — JWT, Redis client, email (SMTP), ID generation

**Message flow**: WebSocket client → Kafka producer → three parallel consumers:
1. **Dispatcher** (`kafka_consumer_dispatch.go`) — pushes to connected WebSocket clients in real-time
2. **Persist** (`kafka_consumer_persist.go`) — writes to MySQL
3. **Cache** (`kafka_consumer_cache.go`) — updates Redis

**WebSocket message format**: JSON with `type` field (`0`=text, `1`=file, `2`=call) and `action` field for signaling (`join_group`, `call_invite`, `call_answer`, `call_candidate`, `call_end`, `group_dismiss`).

### Frontend (`web/src/`)

**Entry**: `main.tsx` → `App.tsx` (React Router with routes: `/`, `/register`, `/captcha-login`, `/chat`, `/profile`).

**Key modules**:
- `api/api.ts` — Axios instance + all REST API calls
- `api/socket.ts` — WebSocket client class and helper functions
- `context/` — `AuthContext` for user authentication state
- `hooks/` — Custom hooks (`useAuth`, `useContactActions`, `useWebRTC`, etc.)
- `pages/Chat.tsx` — Main chat page (most complex component)
- `types/` — TypeScript type definitions
- `utils/chatUtils.ts` — Chat session utilities

**Path alias**: `@/` maps to `web/src/` (configured in `vite.config.ts` and `tsconfig.json`).

### Static Assets
- `static/avatars/` — User avatar uploads served by the backend

## API Conventions

- All protected endpoints require `Authorization: Bearer <JWT>` header
- JWT tokens are refreshed via `/auth/refresh`
- WebSocket connects as `ws://localhost:8000/wss?token=<JWT>`
- File uploads use multipart form to `/message/uploadFile` or `/message/uploadAvatar`
- Group members are stored as a JSON array in `group_info.members`
