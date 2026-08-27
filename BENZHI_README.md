# waste-dispatch

这是一个基于 Go 实现的后端服务，用于承载生活垃圾收运调度的业务处理、数据管理与稳定运行。

## 环境要求

- Go 1.22+
- CGO 构建环境（用于 go-sqlite3，需要 gcc/musl-dev）
- Docker（可选，用于容器化部署）

## 构建

```bash
go build ./...
```

构建二进制文件：

```bash
CGO_ENABLED=1 go build -o waste-dispatch ./cmd/server
```

## 运行

```bash
# 直接运行
go run ./cmd/server

# 或使用构建好的二进制
./waste-dispatch
```

默认监听 `0.0.0.0:8080`，数据库文件位于 `./data/waste_dispatch.db`。

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `SERVER_PORT` | `8080` | 监听端口 |
| `DB_PATH` | `./data/waste_dispatch.db` | SQLite 数据库路径 |
| `DB_MIGRATIONS_PATH` | `file://migrations` | 迁移文件路径 |
| `AUTH_TOKEN_SECRET` | `change-me-in-production-secret-key` | 会话 Token 密钥 |
| `LOG_LEVEL` | `info` | 日志级别（debug/info/warn/error） |
| `LOG_PRETTY` | `false` | 是否使用人类可读日志格式 |

## 测试

```bash
# 运行全部测试
go test ./... -count=1

# 运行全部测试（含竞态检测）
go test -race ./... -count=1

# 静态检查
go vet ./...
```

## Docker 使用

使用评测专用 Dockerfile 构建：

```bash
# 构建镜像
bash build_benzhi_docker.sh waste-dispatch linux/amd64

# 运行容器（交互式 shell）
docker run -it waste-dispatch:latest
```

或手动构建：

```bash
docker build --platform linux/amd64 -f benzhi.Dockerfile -t waste-dispatch .
docker run -it --rm waste-dispatch bash
```

运行服务时建议挂载数据目录：

```bash
docker build --platform linux/amd64 -f Dockerfile -t waste-dispatch:server .
docker run -p 8080:8080 -v $(pwd)/data:/app/data waste-dispatch:server
```

## 健康检查

服务启动后可通过以下接口检查健康状态：

```
GET /health
```

## API 说明

所有 API 均在 `/api/v1` 前缀下，主要资源：

- `POST /api/v1/auth/login` — 用户登录，返回 Bearer Token
- `POST /api/v1/auth/logout` — 退出并撤销当前 Session
- `GET /api/v1/auth/me` — 获取当前用户信息
- `GET/POST /api/v1/vehicles` — 车辆管理
- `GET/POST /api/v1/points` — 投放点管理
- `GET/POST /api/v1/tasks` — 收运任务管理
- `GET/POST /api/v1/incidents` — 异常事件管理
- `GET/POST /api/v1/credits/:resident_id/earn` — 居民积分管理
