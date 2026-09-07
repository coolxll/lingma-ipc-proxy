# AGENTS.md

独立 Go 后端，通过 Lingma 本地 pipe/websocket 传输层暴露 OpenAI 兼容 API。

## 项目结构

```
lingma-ipc-proxy/
├── cmd/                    # 入口
├── internal/
│   ├── httpapi/            # HTTP API 服务（server_test.go）
│   ├── lingmaipc/          # IPC 传输层（transport_test.go, ws_integration_test.go）
│   └── ...
├── config.example.json
├── go.mod                  # module lingma-ipc-proxy, go 1.25
└── docs/
```

## API 端点

- `GET /v1/models`
- `POST /v1/messages`（Anthropic 风格）
- `POST /v1/chat/completions`（OpenAI 风格）

支持流式和非流式响应，一次处理一个请求，支持 Windows。

## 常用命令

```bash
go build -o lingma-ipc-proxy ./cmd/...
go test ./...                              # 全部测试
go test -v ./internal/lingmaipc/...        # 传输层测试
go run ./cmd/...                           # 开发运行
```

## 依赖

- `github.com/Microsoft/go-winio` — Windows named pipe
- `github.com/gorilla/websocket` — WebSocket 传输
