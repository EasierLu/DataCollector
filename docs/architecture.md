# DataCollector 系统架构设计

## 1. 文档定位

本文档描述 DataCollector 系统的整体技术架构，包括模块划分、交互关系、数据流向、部署拓扑及关键技术决策。详细的功能需求、API 接口和数据库模型请参考《需求文档》。

---

## 2. 系统概述

DataCollector 是一个基于 Go + Gin 的 HTTP/HTTPS 数据收集系统，核心目标是提供一个轻量、易部署的通用数据采集平台。系统设计遵循以下原则：

- **零依赖部署**：单二进制文件即可运行，默认 SQLite 数据库无需额外安装
- **模块内聚、接口清晰**：各核心模块职责单一，通过明确的内部接口交互
- **多数据库适配**：抽象存储层，支持 SQLite / PostgreSQL / MySQL 无缝切换
- **安全优先**：Token 哈希存储、JWT 认证、RBAC 权限控制

---

## 3. 技术选型与理由

| 技术 | 选型 | 理由 |
|------|------|------|
| 后端语言 | Go 1.21+ | 编译为原生二进制，跨平台交叉编译方便，并发性能优异 |
| Web 框架 | Gin | Go 生态中最成熟的 HTTP 框架，中间件机制灵活，社区活跃 |
| 默认数据库 | SQLite | 嵌入式，零配置，适合小型部署和开发环境 |
| 生产数据库 | PostgreSQL / MySQL | 满足高并发和大数据量场景 |
| 前端方案 | HTML + TailwindCSS + Alpine.js | 轻量级服务端渲染，无需构建工具链，嵌入二进制分发 |
| 实时通信 | WebSocket / SSE | 仪表盘实时数据推送 |
| 配置管理 | YAML + 环境变量 | 开发用配置文件，容器化环境用环境变量覆盖 |

---

## 4. 系统架构总览

```
                                    ┌─────────────────────────────────────┐
                                    │          DataCollector Server       │
                                    │                                     │
  ┌──────────┐  POST /api/v1/collect/:source_id  │  ┌───────────┐   ┌──────────────┐  │
  │ 数据提交  │ ──────────────────────────────── │─▶│ Collector  │──▶│   Storage    │  │
  │  客户端   │   X-Data-Token                   │  │  数据采集   │   │  数据存储层   │  │
  └──────────┘                      │  └───────────┘   │              │  │
                                    │        │         │  ┌────────┐  │  │
                                    │        ▼         │  │ SQLite │  │  │
                                    │  ┌───────────┐   │  ├────────┤  │  │
  ┌──────────┐   /admin/*           │  │   Auth     │   │  │ PgSQL  │  │  │
  │ 管理员   │ ─────────────────── │─▶│  鉴权系统   │   │  ├────────┤  │  │
  │ 浏览器   │   JWT Bearer         │  └───────────┘   │  │ MySQL  │  │  │
  └──────────┘                      │        │         │  └────────┘  │  │
                                    │        ▼         └──────────────┘  │
                                    │  ┌───────────┐   ┌──────────────┐  │
                                    │  │   Admin    │──▶│   Monitor    │  │
                                    │  │  管理后台   │   │  实时监控     │  │
                                    │  └───────────┘   └──────────────┘  │
                                    │                         │          │
                                    │                    WebSocket/SSE   │
                                    └─────────────────────────┼──────────┘
                                                              │
                                                              ▼
                                                      ┌──────────────┐
                                                      │  仪表盘推送   │
                                                      └──────────────┘
```

---

## 5. 核心模块设计

### 5.1 数据采集模块 (Collector)

职责：接收外部数据提交请求，执行验证、限流和持久化。

请求处理流程：

```
HTTP Request
    │
    ▼
Rate Limiter（按 Token + IP 双维度限流）
    │
    ▼
Token Authenticator（X-Data-Token 验证，哈希比对）
    │
    ▼
Schema Validator（根据数据源 schema_config 动态验证字段）
    │
    ▼
Data Persister（写入 data_records 表 + 更新统计计数）
    │
    ▼
HTTP Response
```

关键设计决策：

- Token 验证采用 SHA-256 哈希比对，数据库不存储明文，提交时实时计算哈希后查库。为减少数据库压力，验证通过的 Token 在内存中缓存（TTL 5 分钟）。
- 限流器使用滑动窗口算法（`golang.org/x/time/rate` 或自实现），支持按 Token 和按 IP 两个维度独立限流。
- 数据验证规则从数据源的 `schema_config` 字段动态加载，支持的验证类型包括：required、type（string/number/email/url）、max_length、min_length、pattern（正则）。

### 5.2 鉴权模块 (Auth)

职责：管理后台的用户认证和权限控制。

两套独立认证机制：

- **管理后台认证 (JWT)**：用户登录后签发 JWT Token，有效期 24 小时。Token 通过 `Authorization: Bearer` 头传递。中间件在路由层统一拦截和校验，支持 RBAC（admin/user 两级角色）。
- **数据采集认证 (Data Token)**：独立于 JWT 体系。Token 格式 `dt_` + 32 位随机字符，仅在生成时展示明文，数据库只存储 SHA-256 哈希。详见《需求文档》附录 A。

密码存储使用 bcrypt（cost factor = 12），JWT 签名密钥通过配置文件或环境变量注入。

### 5.3 管理后台模块 (Admin)

职责：提供 Web 管理界面，数据源配置、数据查询、Token 管理、系统设置。

采用**服务端渲染**方案（Go template + TailwindCSS + Alpine.js），所有前端资源通过 `go:embed` 嵌入二进制文件。这一决策的核心考量是保持"单文件分发"的部署简洁性，避免引入前端构建工具链。

页面交互使用 Alpine.js 处理动态行为（如表单验证、弹窗确认），图表使用 Chart.js 或类似轻量图表库渲染。

### 5.4 数据存储模块 (Storage)

职责：数据库连接管理、数据读写抽象、迁移管理。

存储层通过接口抽象，上层代码不直接依赖具体数据库驱动：

```go
type DataStore interface {
    CreateRecord(ctx context.Context, record *DataRecord) (int64, error)
    QueryRecords(ctx context.Context, filter RecordFilter) (*PageResult, error)
    GetSourceByID(ctx context.Context, id int64) (*DataSource, error)
    // ... 其他方法
}
```

数据库适配策略：

- **SQLite**：默认选项，数据文件存储在 `./data/` 目录。JSON 字段以 TEXT 类型存储，应用层负责序列化/反序列化。
- **PostgreSQL / MySQL**：使用原生 JSON/JSONB 类型，利用数据库层 JSON 查询能力。
- **迁移管理**：使用内嵌迁移文件（`go:embed`），启动时自动检测并执行增量迁移。

### 5.5 实时监控模块 (Monitor)

职责：数据流入统计、系统指标收集、仪表盘实时推送。

监控模块维护一个内存中的统计聚合器，数据采集模块每次成功写入后通过 channel 通知监控模块更新计数。聚合器定期（每分钟）将统计数据持久化到 statistics 表，同时通过 WebSocket 向已连接的仪表盘客户端推送实时更新。

---

## 6. 请求生命周期

一个数据采集请求的完整处理链路：

```
客户端 HTTP 请求
    │
    ▼
Gin Engine
    │
    ├──▶ Logger 中间件（记录请求日志、分配 trace_id）
    │
    ├──▶ Recovery 中间件（panic 恢复）
    │
    ├──▶ CORS 中间件（跨域检查）
    │
    ├──▶ Body Size 中间件（请求体大小限制，默认 1MB）
    │
    ├──▶ Rate Limit 中间件（IP 维度限流）
    │
    ▼
Collector Handler
    │
    ├──▶ Token 认证（X-Data-Token 哈希比对）
    │
    ├──▶ 数据源查找（根据 source_id 加载配置）
    │
    ├──▶ Schema 验证（字段类型、必填、长度等）
    │
    ├──▶ 数据持久化（写入 data_records）
    │
    ├──▶ 统计更新（通知 Monitor 模块）
    │
    ▼
返回响应
```

---

## 7. 部署架构

### 7.1 单机部署（默认）

最简单的部署方式，适合小团队和个人使用：

```
┌──────────────────────────────┐
│      单台服务器 / 本地机器     │
│                              │
│  ┌────────────────────────┐  │
│  │  DataCollector 进程     │  │
│  │  (内嵌 Web 前端)       │  │
│  │  监听 :8080            │  │
│  └───────────┬────────────┘  │
│              │               │
│  ┌───────────▼────────────┐  │
│  │  ./data/datacollector.db │  │
│  │  (SQLite 数据文件)      │  │
│  └────────────────────────┘  │
└──────────────────────────────┘
```

### 7.2 容器化部署

适合需要标准化运维流程的场景：

```
┌─────────────────────────────────────────────┐
│                 Docker Host                  │
│                                             │
│  ┌─────────────────┐  ┌─────────────────┐  │
│  │  DataCollector   │  │  PostgreSQL     │  │
│  │  容器            │──│  容器           │  │
│  │  :8080          │  │  :5432          │  │
│  └─────────────────┘  └─────────────────┘  │
│          │                     │            │
│          ▼                     ▼            │
│  ┌─────────────┐      ┌─────────────┐      │
│  │ volume:logs │      │ volume:data │      │
│  └─────────────┘      └─────────────┘      │
└─────────────────────────────────────────────┘
```

### 7.3 生产高可用部署（扩展方案）

当单实例无法满足性能需求时，可以水平扩展：

```
                    ┌──────────────┐
                    │  Nginx / LB  │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │ Instance │ │ Instance │ │ Instance │
        │    #1    │ │    #2    │ │    #3    │
        └────┬─────┘ └────┬─────┘ └────┬─────┘
             │             │             │
             └─────────────┼─────────────┘
                           ▼
                    ┌──────────────┐
                    │  PostgreSQL  │
                    │  (共享数据库) │
                    └──────────────┘
```

> 注意：水平扩展时必须使用 PostgreSQL 或 MySQL，不能使用 SQLite。JWT 签名密钥需在所有实例间保持一致（通过环境变量或配置中心下发）。

---

## 8. 项目目录结构

```
DataCollector/
├── cmd/
│   └── server/              # 主程序入口
│       └── main.go          # 启动逻辑、信号处理、优雅关闭
├── internal/                # 内部包，不对外暴露
│   ├── api/                 # HTTP Handler 层
│   │   ├── setup.go         # 系统初始化接口
│   │   ├── collector.go     # 数据采集接口
│   │   ├── admin.go         # 管理后台接口
│   │   ├── health.go        # 健康检查接口
│   │   └── auth.go          # 认证接口
│   ├── auth/                # 鉴权逻辑
│   │   ├── jwt.go           # JWT 签发与验证
│   │   └── middleware.go    # 认证中间件
│   ├── collector/           # 数据采集核心逻辑
│   │   ├── validator.go     # Schema 动态验证
│   │   └── processor.go     # 数据处理与持久化
│   ├── config/              # 配置管理
│   │   └── config.go        # 配置加载（YAML + 环境变量）
│   ├── storage/             # 数据存储抽象层
│   │   ├── interface.go     # DataStore 接口定义
│   │   ├── sqlite.go        # SQLite 实现
│   │   ├── postgres.go      # PostgreSQL 实现
│   │   ├── mysql.go         # MySQL 实现
│   │   └── migrations/      # 数据库迁移文件（go:embed）
│   ├── middleware/          # Gin 中间件
│   │   ├── cors.go          # CORS 跨域
│   │   ├── ratelimit.go     # 限流
│   │   ├── bodysize.go      # 请求体大小限制
│   │   └── logger.go        # 请求日志
│   ├── monitor/             # 监控统计
│   │   ├── aggregator.go    # 统计聚合器
│   │   └── websocket.go     # WebSocket 推送
│   └── web/                 # 管理后台前端资源
│       ├── embed.go         # go:embed 静态资源
│       ├── static/          # CSS, JS, 图片
│       └── templates/       # Go HTML 模板
├── configs/
│   └── config.yaml          # 默认配置文件
├── data/                    # SQLite 数据目录（运行时生成）
├── logs/                    # 日志目录（运行时生成）
├── scripts/
│   ├── build.sh             # 多平台交叉编译脚本
│   └── release.sh           # 发布脚本
├── Dockerfile               # 多阶段构建
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

与需求文档中项目结构的主要差异说明：原 `internal/database/` 重命名为 `internal/storage/`，更准确地反映其"存储抽象层"的职责；新增 `internal/api/health.go` 对应健康检查端点；将 `internal/collector/storage.go` 重命名为 `processor.go`，避免与 storage 包混淆。

---

## 9. 关键技术决策记录

### 决策 1：前端方案选择服务端渲染

**背景**：需要为管理后台提供 Web 界面。

**备选方案**：(A) React/Vue SPA + API 分离；(B) Go template + TailwindCSS + Alpine.js 服务端渲染。

**决策**：选择方案 B。

**理由**：管理后台交互复杂度低，服务端渲染可将所有前端资源通过 `go:embed` 嵌入单个二进制文件，保持零依赖部署。避免引入 Node.js 构建工具链和前后端分离带来的部署复杂度。

**风险**：如果未来管理后台交互复杂度显著提升（如拖拽式表单构建器），可能需要迁移到 SPA 方案。

### 决策 2：Token 哈希存储而非加密存储

**背景**：数据采集 Token 需要安全存储。

**决策**：使用 SHA-256 单向哈希，不使用可逆加密。

**理由**：Data Token 的用途类似于 API Key，只需验证"持有者是否拥有正确的 Token"，不需要还原原文。哈希存储即使数据库被攻破，攻击者也无法反推出有效 Token。这与 GitHub Personal Access Token 的安全模型一致。

### 决策 3：SQLite 作为默认数据库

**背景**：系统需要"零配置启动"。

**决策**：默认使用 SQLite，同时保留 PostgreSQL/MySQL 适配。

**理由**：SQLite 不需要独立的数据库服务进程，数据存储在单个文件中，完美契合"下载即用"的产品目标。对于数据量较小（百万级以下记录）的场景，SQLite 的读写性能足够。当数据量增长或需要多实例部署时，用户可以通过初始化向导切换到 PostgreSQL/MySQL。

**约束**：SQLite 不支持并发写入，高并发写入场景下需要应用层串行化（通过 channel 或 mutex）。

---

**文档版本**: 1.0  
**更新日期**: 2026-04-07  
**关联文档**: 需求文档 v1.5
