# QQ 消息代办收集器

通过 OneBot11 SSE 监听 QQ 消息，调用 OpenAI Responses API 抽取“他人交代给我的杂事”，汇总为待办列表，并提供网页查看与完成/撤销。

## 架构概览

```
QQ消息 -> OneBot11 SSE -> 后端解析/过滤 -> OpenAI 抽取 -> SQLite -> Web UI
```

## 目录结构

- `backend/` Go 后端（SSE 监听、LLM 抽取、REST API、SQLite）
- `frontend/` Vite + React 前端
- `backend/config.toml` 运行配置（可参考 `backend/config.example.toml`）

## 配置

编辑 `backend/config.toml`（推荐先复制 `backend/config.example.toml`）：

```toml
[server]
port = 8080
password = "change_me"
static_dir = "internal/server/web_dist"
static_embed = true
token_ttl_minutes = 1440

[onebot]
# OneBot11 SSE 地址（用于接收消息事件）
sse_url = "http://127.0.0.1:5700/onebot/v11/sse"
# OneBot11 HTTP API 地址（用于查询群名、昵称、登录信息等）
api_url = "http://127.0.0.1:5700"
# 如果 OneBot 配置了访问令牌，请填写
access_token = ""

[openai]
base_url = "https://api.openai.com/v1"
api_key = ""
model = "gpt-4o-mini"
timeout_seconds = 30

[calendar]
# 周解释策略：academic=按学周体系解释“第13周周二”；natural=按自然周解释
week_mode = "academic"
# 当前学期第1周的星期一（week_mode=academic 时必填，用于解析“13周星期二”等相对学周时间）
week1_monday = "2026-02-23"

[storage]
sqlite_path = "./data.db"

[log]
# 日志级别：debug/info/warn/warning/error（留空时回退环境变量 LOG_LEVEL，再兜底 info）
level = "info"
# 日志格式：json/development
format = "json"
```

要点：
- 程序启动时会通过 OneBot API `get_login_info` 自动获取并使用自己的 QQ 号（用于过滤自己发送的消息）。
- `onebot.sse_url` 请填写 OneBot11 的 SSE 接口地址。
- `server.password` 是网页登录口令。
- `server.static_embed=true` 时优先使用编译进二进制的前端资源；`false` 时使用 `server.static_dir` 指向的磁盘目录。
- `openai.api_key` 为敏感信息，请勿提交到公共仓库。
- `calendar.week_mode` 仅支持 `academic` / `natural`。
- 当 `calendar.week_mode=academic` 时，`calendar.week1_monday` 必填，格式 `YYYY-MM-DD`，且必须是周一。
- 日志优先级规则：`log.level`（配置） > `LOG_LEVEL`（环境变量） > `info`（默认）。

## 运行方式

### 生产部署（推荐：前端资源内嵌到后端二进制）

1. 构建前端资源（会输出到 `backend/internal/server/web_dist`）：

```bash
cd frontend
pnpm install
pnpm build
```

2. 启动后端（`server.static_embed=true`）：

```bash
cd backend
go mod tidy
go run . -config config.toml
```

### 开发模式（使用磁盘静态目录）

1. 将 `server.static_embed` 设置为 `false`，并保持 `server.static_dir` 指向静态目录。
2. 启动前端开发服务器：

```bash
cd frontend
pnpm dev
```

前端已配置代理，请确保后端在 `http://localhost:8080`。

## API 简述

- `POST /api/login` 传 `{ password }` 获得 token
- `GET /api/todos` 获取待办列表
- `POST /api/todos/:id/complete`
- `POST /api/todos/:id/reopen`

## 说明

- 默认不去重，所有消息都会尝试抽取。
- LLM 抽取失败不会入库，会记录日志。
