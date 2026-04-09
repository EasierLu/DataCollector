<p align="center">
  <h1 align="center">DataCollector</h1>
  <p align="center">轻量级 HTTP 数据收集服务 — 基于 Go + Vue 3 构建</p>
  <p align="center">通用数据采集 · 管理后台 · 实时监控 · 零依赖单文件部署</p>
</p>

---

## 简介

DataCollector 是一个开箱即用的通用 HTTP 数据收集系统。外部客户端（脚本、IoT 设备、表单、微服务等）通过简单的 HTTP POST 将 JSON 数据推送到采集接口，服务端根据预定义的 Schema 自动校验并持久化，同时提供管理后台和实时仪表盘。

单二进制文件包含完整的前后端，下载即运行，无需额外环境依赖。

## 特性

- **零依赖部署** — 单二进制文件运行，前端资源通过 `go:embed` 嵌入
- **多数据库** — 默认 SQLite（零配置），可选 PostgreSQL
- **实时监控** — WebSocket 推送，仪表盘实时展示数据流入趋势与统计
- **安全设计** — Data Token SHA-256 哈希存储、JWT 认证、RBAC 权限控制、IP + Token 双维度限流
- **动态 Schema 校验** — 按数据源配置的 Schema 对采集数据进行字段级验证
- **数据管理** — 查询、筛选、批量删除与 CSV/JSON 导出
- **TLS 支持** — 内置 HTTPS，配置证书即可启用
- **多平台** — 支持 Windows / macOS / Linux（amd64 / arm64 / armv7）

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.25, Gin, slog |
| 前端 | Vue 3, TypeScript, Element Plus, ECharts, Pinia |
| 数据库 | SQLite（默认）/ PostgreSQL |
| 实时通信 | WebSocket（gorilla/websocket） |
| 构建 | Vite 6, Docker 多阶段构建 |

## 快速开始

### 从二进制运行

从 [Releases](https://github.com/datacollector/datacollector/releases) 下载对应平台的二进制文件：

```bash
# Linux / macOS
chmod +x datacollector
./datacollector

# Windows
datacollector.exe
```

服务默认监听 `0.0.0.0:8080`，首次访问将引导完成初始化配置（创建管理员账户等）。

### Docker 部署

**SQLite 模式（默认）：**

```bash
docker compose up -d
```

**PostgreSQL 模式：**

编辑 `docker-compose.yml`，取消 PostgreSQL 相关服务的注释，然后：

```bash
docker compose up -d
```

**单独构建镜像：**

```bash
make docker-build
# 或
docker build -t datacollector:latest .
```

## 开发指南

### 环境要求

- Go 1.25+
- Node.js 20+
- GCC（SQLite 的 CGO 编译需要）

### 本地开发

```bash
# 克隆项目
git clone https://github.com/datacollector/datacollector.git
cd datacollector

# 安装前端依赖
make web-install

# 启动前端开发服务器（端口 5173，自动代理 /api 到 8080）
make web-dev

# 启动后端（另一个终端）
make run
```

### 构建

```bash
# 完整构建（前端 + 后端），产物位于 dist/
make build

# 仅构建后端（跳过前端，开发调试用）
make build-go

# 多平台交叉编译（Windows / macOS / Linux × amd64 / arm64 / armv7）
make build-all

# 清理构建产物
make clean

# 运行测试
make test
```

Windows 环境可使用：

```bat
build.bat
```

## 配置

配置文件：`configs/config.yaml`，所有配置项均支持环境变量覆盖。

### 完整配置参考

```yaml
server:
  host: "0.0.0.0"
  port: 8080
  mode: "debug"              # debug / release

tls:
  enabled: false
  cert_file: ""
  key_file: ""

database:
  driver: "sqlite"           # sqlite / postgres
  sqlite:
    path: "./data/datacollector.db"
  postgres:
    host: "localhost"
    port: 5432
    user: "datacollector"
    password: ""
    dbname: "datacollector"
    sslmode: "disable"

jwt:
  secret: "change-me-to-a-secure-random-string"
  expiration: "24h"

collector:
  max_body_size: 1048576     # 请求体上限，默认 1MB
  rate_limit_per_token: 100  # 每 Token 每分钟请求上限
  rate_limit_per_ip: 200     # 每 IP 每分钟请求上限
  allowed_origins:
    - "*"

log:
  level: "info"              # debug / info / warn / error
  format: "json"
  output: "stdout"           # stdout / file
  file_path: "./logs/datacollector.log"
  max_size: 100              # 日志文件大小上限（MB）
  max_age: 30                # 日志保留天数
```

### Docker 环境变量

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `DB_DRIVER` | 数据库驱动 | `sqlite` |
| `DB_SQLITE_PATH` | SQLite 文件路径 | `/app/data/datacollector.db` |
| `DB_HOST` | PostgreSQL 主机 | - |
| `DB_PORT` | PostgreSQL 端口 | - |
| `DB_USER` | PostgreSQL 用户 | - |
| `DB_PASSWORD` | PostgreSQL 密码 | - |
| `DB_NAME` | PostgreSQL 数据库名 | - |
| `LOG_OUTPUT` | 日志输出方式 | `file` |
| `LOG_FILE_PATH` | 日志文件路径 | `/app/logs/datacollector.log` |

## API 参考

所有接口基础路径：`/api/v1`

### 数据采集

通过 `X-Data-Token` 请求头认证。`collect_id` 是创建数据源时自动生成的 8 位随机标识。

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/collect/:collect_id` | 提交单条数据 |
| `POST` | `/collect/:collect_id/batch` | 批量提交数据 |

**示例：**

```bash
curl -X POST http://localhost:8080/api/v1/collect/aB3xK9mZ \
  -H "X-Data-Token: your-data-token" \
  -H "Content-Type: application/json" \
  -d '{"temperature": 23.5, "humidity": 68}'
```

### 系统

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/health` | 健康检查 |
| `GET` | `/setup/status` | 初始化状态 |
| `POST` | `/setup/test-db` | 测试数据库连接 |
| `POST` | `/setup/init` | 系统初始化 |
| `POST` | `/setup/reinit` | 重新初始化（需 admin） |

### 管理后台

通过 `Authorization: Bearer <JWT>` 认证。

**认证**

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/admin/login` | 登录 |
| `POST` | `/admin/refresh-token` | 刷新 Token |
| `POST` | `/admin/change-password` | 修改密码 |

**仪表盘**

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/admin/dashboard` | 仪表盘统计 |
| `GET` | `/admin/dashboard/trend` | 趋势数据 |
| `WS` | `/admin/ws/monitor` | 实时监控推送 |

**数据源管理**

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/admin/sources` | 列出数据源 |
| `GET` | `/admin/sources/:id` | 数据源详情 |
| `POST` | `/admin/sources` | 创建数据源 |
| `PUT` | `/admin/sources/:id` | 更新数据源 |
| `DELETE` | `/admin/sources/:id` | 删除数据源 |
| `POST` | `/admin/sources/:id/tokens` | 创建 Data Token |
| `GET` | `/admin/sources/:id/tokens` | 列出 Data Token |

**Token 管理**

| 方法 | 路径 | 说明 |
|------|------|------|
| `PUT` | `/admin/tokens/:id/status` | 更新 Token 状态 |
| `DELETE` | `/admin/tokens/:id` | 删除 Token |

**数据管理**

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/admin/data` | 查询采集数据 |
| `DELETE` | `/admin/data/:id` | 删除单条记录 |
| `POST` | `/admin/data/batch-delete` | 批量删除 |
| `GET` | `/admin/data/export` | 导出数据 |

**系统设置**

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/admin/settings/rate-limit` | 查看限流配置 |
| `PUT` | `/admin/settings/rate-limit` | 更新限流配置 |

## 项目结构

```
DataCollector/
├── cmd/server/             # 程序入口
├── internal/
│   ├── api/                # HTTP Handler 与路由注册
│   ├── auth/               # JWT 鉴权与中间件
│   ├── collector/          # 数据采集核心（校验、处理、落库）
│   ├── config/             # 配置加载（YAML + 环境变量）
│   ├── middleware/          # Gin 中间件（CORS、限流、日志、请求体限制）
│   ├── model/              # 数据模型与 DTO
│   ├── monitor/            # 实时监控（统计聚合 + WebSocket Hub）
│   ├── server/             # HTTP Server 装配与 SPA 静态资源
│   ├── storage/            # 数据存储抽象层
│   │   ├── migrations/     # 数据库迁移 SQL
│   │   ├── sqlite/         # SQLite 实现
│   │   └── postgres/       # PostgreSQL 实现
│   └── web/                # 前端资源嵌入（go:embed）
├── web/                    # Vue 3 前端源码
│   └── src/
│       ├── views/          # 页面（Dashboard、Sources、Data、Settings 等）
│       ├── stores/         # Pinia 状态管理
│       ├── composables/    # 组合式函数（WebSocket 等）
│       └── router/         # 路由配置
├── configs/                # 默认配置文件
├── scripts/                # 构建脚本（多平台交叉编译）
├── docs/                   # 设计文档
├── Dockerfile              # Docker 多阶段构建
├── docker-compose.yml      # Docker Compose 编排
├── Makefile                # 构建命令
└── build.bat               # Windows 构建脚本
```

## License

MIT
