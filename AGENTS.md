# AGENTS

本仓库是 OneBot11 + OpenAI 的 QQ 消息代办收集器。

## 关键模块

- `backend/main.go` 启动入口，加载配置、启动 SSE 监听与 HTTP 服务
- `backend/sse.go` OneBot11 SSE 监听与消息处理
- `backend/llm.go` OpenAI Responses API 调用与 JSON 解析
- `backend/storage.go` SQLite 存储
- `backend/api.go` 登录与待办 API
- `backend/config.go` `backend/config.toml` 解析

## 运行方式

```bash
cd frontend
pnpm install
pnpm build

cd ../backend
go run . -config config.toml
```

## 开发提示

- 配置文件固定在 `backend/config.toml`
- SSE 仅处理 `message` 事件，过滤 `self_id`
- 前端通过 `/api/*` 访问后端（Vite 已配置代理）
- 若 OpenAI 未配置，会跳过抽取
