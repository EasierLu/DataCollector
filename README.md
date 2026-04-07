# DataCollector

一个基于 Go + Vue 3 的轻量级 HTTP 数据收集系统，提供通用数据采集、管理后台和实时监控能力。支持零依赖单文件部署，开箱即用。

## 特性

- **零依赖部署** - 单二进制文件即可运行，前端资源通过 `go:embed` 嵌入
- **多数据库支持** - 默认 SQLite（零配置），可切换 PostgreSQL
- **实时监控** - WebSocket 推送，仪表盘实时展示数据流入统计
- **安全设计** - Data Token SHA-256 哈希存储、JWT 认证、RBAC 权限控制、双维度限流
- **动态数据验证** - 基于数据源 Schema 配置的字段动态校验
- **数据导出** - 支持采集数据的查询与导出
- **多平台构建** - 支持 Windows / macOS / Linux（amd64 / arm64）

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.23, Gin, slog |
| 前端 | Vue 3, TypeScript, Element Plus, ECharts, Pinia |
| 数据库 | SQLite (默认) / PostgreSQL |
| 实时通信 | WebSocket |
| 构建 | Vite, Docker 多阶段构建 |

## 快速开始

### 环境要求

- Go 1.21+
- Node.js 20+
- GCC（SQLite 需要 CGO）

### 本地开发

```bash
# 克隆项目
git clone https://github.com/datacollector/datacollector.git
cd datacollector

# 安装前端依赖
make web-install

# 启动前端开发服务器
make web-dev

# 启动后端
make run
```

### 构建

**Linux / macOS:**

```bash
# 完整构建（前端 + 后端）
make build

# 仅构建后端（跳过前端，开发调试用）
make build-go

# 多平台交叉编译
make build-all
```

**Windows:**

```bat
build.bat
```

构建产物位于 `dist/` 目录。

### 运行

```bash
./dist/datacollector
```

服务默认监听 `0.0.0.0:8080`，首次启动访问管理界面完成初始化配置。

## Docker 部署

### SQLite 模式（默认）

```bash
docker compose up -d
```

### PostgreSQL 模式

编辑 `docker-compose.yml`，取消 PostgreSQL 相关服务的注释，然后：

```bash
docker compose up -d
```

### 单独构建镜像

```bash
make docker-build
```

## 配置

配置文件位于 `configs/config.yaml`，支持通过环境变量覆盖。

### 主要配置项

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `server.port` | `8080` | 服务监听端口 |
| `server.mode` | `debug` | 运行模式 (debug / release) |
| `database.driver` | `sqlite` | 数据库驱动 (sqlite / postgres) |
| `database.sqlite.path` | `./data/datacollector.db` | SQLite 数据库文件路径 |
| `jwt.secret` | - | JWT 签名密钥（生产环境请修改） |
| `jwt.expiration` | `24h` | JWT Token 过期时间 |
| `collector.rate_limit_per_token` | `100` | 每个 Token 每分钟请求上限 |
| `collector.rate_limit_per_ip` | `200` | 每个 IP 每分钟请求上限 |
| `log.output` | `stdout` | 日志输出 (stdout / file) |

### 环境变量（Docker）

| 环境变量 | 说明 |
|----------|------|
| `DB_DRIVER` | 数据库驱动 |
| `DB_SQLITE_PATH` | SQLite 文件路径 |
| `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` | PostgreSQL 连接参数 |
| `LOG_OUTPUT` | 日志输出方式 |
| `LOG_FILE_PATH` | 日志文件路径 |

## API 概览

### 数据采集

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/collect/:source_id` | 提交单条数据 |
| `POST` | `/api/v1/collect/:source_id/batch` | 批量提交数据 |

数据采集接口通过 `X-Data-Token` 请求头进行认证。

### 管理后台

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/api/v1/admin/login` | 管理员登录 |
| `GET` | `/api/v1/admin/dashboard` | 获取仪表盘数据 |
| `GET` | `/api/v1/admin/dashboard/trend` | 获取趋势数据 |
| `GET/POST/PUT/DELETE` | `/api/v1/admin/sources` | 数据源 CRUD |
| `POST/GET` | `/api/v1/admin/sources/:id/tokens` | Token 管理 |
| `GET` | `/api/v1/admin/data` | 查询采集数据 |
| `GET` | `/api/v1/admin/data/export` | 导出数据 |

管理后台接口通过 `Authorization: Bearer <JWT>` 认证。

### 系统

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/api/v1/health` | 健康检查 |
| `GET` | `/api/v1/setup/status` | 初始化状态检查 |
| `POST` | `/api/v1/setup/init` | 系统初始化 |

## 项目结构

```
DataCollector/
├── cmd/server/          # 程序入口
├── internal/
│   ├── api/             # HTTP Handler
│   ├── auth/            # JWT 鉴权
│   ├── collector/       # 数据采集核心逻辑
│   ├── config/          # 配置管理
│   ├── middleware/       # Gin 中间件 (CORS, 限流, 日志等)
│   ├── model/           # 数据模型
│   ├── monitor/         # 实时监控 (WebSocket + 统计聚合)
│   ├── server/          # HTTP Server 配置
│   ├── storage/         # 数据存储抽象层 (SQLite / PostgreSQL)
│   └── web/             # 前端资源 (go:embed)
├── web/                 # Vue 3 前端源码
├── configs/             # 配置文件
├── scripts/             # 构建脚本
├── docs/                # 文档
├── Dockerfile           # Docker 多阶段构建
├── docker-compose.yml
└── Makefile
```

## License

MIT
