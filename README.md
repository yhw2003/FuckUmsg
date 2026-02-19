# QQ 消息代办收集器

通过 OneBot11 SSE 监听 QQ 消息，调用 OpenAI Responses API 抽取“他人交代给我的杂事”，汇总为待办列表，并提供网页查看与完成/撤销。

## 架构概览

```
QQ消息 -> OneBot11 SSE -> 后端解析/过滤 -> OpenAI 抽取 -> SQLite -> Web UI
```

## 目录结构

- `backend/` Go 后端（SSE 监听、LLM 抽取、REST API、SQLite）
- `frontend/` Vite + React 前端
- `backend/config.toml` 运行配置

## 配置

编辑 `backend/config.toml`：

```toml
[server]
port = 8080
password = "change_me"
static_dir = "../frontend/dist"
token_ttl_minutes = 1440

[onebot]
sse_url = "http://127.0.0.1:5700/onebot/v11/sse"
access_token = ""

[openai]
base_url = "https://api.openai.com/v1"
api_key = ""
model = "gpt-4o-mini"
format = "json_schema"
timeout_seconds = 30

[storage]
sqlite_path = "./data.db"
```

要点：
- 程序启动时会通过 OneBot API `get_login_info` 自动获取并使用自己的 QQ 号（用于过滤自己发送的消息）。
- `onebot.sse_url` 请填写 OneBot11 的 SSE 接口地址。
- `server.password` 是网页登录口令。
- `openai.api_key` 为敏感信息，请勿提交到公共仓库。
- LLM 输出采用固定两行格式（`IS_TODO` / `TITLE`），解析失败会记录日志且不会入库。

## 运行方式

1. 安装前端依赖并构建：

```bash
cd frontend
pnpm install
pnpm build
```

2. 启动后端：

```bash
cd backend
go mod tidy
go run . -config config.toml
```

3. 浏览器访问：

```
http://localhost:8080
```

## 开发模式

前端本地开发：

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
