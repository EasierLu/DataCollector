# DataCollector API 文档

> 基础路径：`/api/v1`  
> 版本：1.0.0

---

## 通用说明

### 响应格式

所有接口均返回统一 JSON 格式：

```json
{
  "code": 0,
  "message": "成功",
  "data": {}
}
```

`code` 为 `0` 表示成功，非零为错误码。

### 认证方式

| 方式 | 请求头 | 适用范围 |
|------|--------|----------|
| JWT Token | `Authorization: Bearer <token>` | 管理后台接口 (`/admin/*`) |
| Data Token | `X-Data-Token: dt_<hex>` | 数据采集接口 (`/collect/*`) |

### 速率限制

数据采集接口启用了基于 IP 和 Token 的滑动窗口限流，超限返回 `429` 状态码。

---

## 健康检查

### GET /health

检查系统运行状态。**无需认证。**

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "uptime": "1h30m45s",
    "database": "connected"
  }
}
```

---

## 系统初始化

### GET /setup/status

查询系统是否已初始化。**无需认证。**

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "initialized": true
  }
}
```

### POST /setup/test-db

测试数据库连接。**无需认证。**

**请求体：**

```json
{
  "driver": "postgres",
  "host": "localhost",
  "port": 5432,
  "user": "admin",
  "password": "secret",
  "dbname": "datacollector"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| driver | string | 是 | `postgres` 或 `sqlite` |
| host | string | postgres 时必填 | 数据库主机 |
| port | integer | postgres 时必填 | 数据库端口 |
| user | string | postgres 时必填 | 用户名 |
| password | string | postgres 时必填 | 密码 |
| dbname | string | postgres 时必填 | 数据库名 |

### POST /setup/init

初始化系统（数据库、管理员账户）。**无需认证。**

**请求体：**

```json
{
  "database": {
    "driver": "postgres",
    "postgres": {
      "host": "localhost",
      "port": 5432,
      "user": "admin",
      "password": "secret",
      "dbname": "datacollector"
    }
  },
  "server": {
    "port": 8080
  },
  "admin": {
    "username": "admin",
    "password": "password123"
  }
}
```

> `admin.password` 最少 6 个字符。系统已初始化时返回错误码 `5002`。

### POST /setup/reinit

重新初始化系统。**需要 JWT 认证 + admin 角色。**

**请求体：**

```json
{
  "confirm": "REINITIALIZE"
}
```

> `confirm` 字段必须为精确字符串 `REINITIALIZE`。操作完成后需要重启服务器。

---

## 认证

### POST /admin/login

管理员登录，获取 JWT Token。

**请求体：**

```json
{
  "username": "admin",
  "password": "password123"
}
```

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 86400
  }
}
```

> `expires_in` 单位为秒（默认 24 小时）。将 `token` 放入后续请求的 `Authorization: Bearer <token>` 头中。

### POST /admin/refresh-token

刷新即将过期的 JWT Token。**需要 JWT 认证。**

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 86400
  }
}
```

> 仅当 Token 剩余有效期不足 2 小时时才允许刷新。

---

## 数据源管理

### GET /admin/sources

获取数据源列表。**需要 JWT 认证。**

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| page | integer | 1 | 页码 |
| size | integer | 10 | 每页数量 (1-100) |

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "total": 5,
    "list": [
      {
        "id": 1,
        "name": "User Events",
        "description": "用户活动追踪",
        "schema_config": {
          "fields": [
            {
              "name": "email",
              "type": "email",
              "required": true,
              "max_length": 255
            }
          ]
        },
        "status": 1,
        "created_by": 1,
        "created_at": "2024-01-15T10:30:00Z",
        "updated_at": "2024-01-15T10:30:00Z",
        "token_count": 3
      }
    ]
  }
}
```

### POST /admin/sources

创建数据源。**需要 JWT 认证。**

**请求体：**

```json
{
  "name": "User Events",
  "description": "用户活动追踪",
  "schema_config": {
    "fields": [
      {
        "name": "email",
        "type": "email",
        "required": true,
        "max_length": 255
      },
      {
        "name": "age",
        "type": "integer",
        "required": false
      }
    ]
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 数据源名称 |
| description | string | 否 | 描述 |
| schema_config | object | 否 | 数据校验规则 |

**Schema 字段类型：** `string`, `number`, `email`, `url`, `boolean`, `date`, `datetime`, `integer`, `float`, `array`, `object`

**字段校验选项：** `required` (bool), `max_length` (int), `min_length` (int), `pattern` (regex)

### PUT /admin/sources/{id}

更新数据源。**需要 JWT 认证。**

请求体同创建接口。路径参数 `id` 为数据源 ID。

### DELETE /admin/sources/{id}

删除数据源。**需要 JWT 认证。**

路径参数 `id` 为数据源 ID。

---

## Token 管理

### POST /admin/sources/{id}/tokens

为指定数据源创建 Data Token。**需要 JWT 认证。**

**请求体：**

```json
{
  "name": "Production Token",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | Token 名称 |
| expires_at | string | 否 | 过期时间 (RFC3339)，不填则永不过期 |

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "id": 42,
    "token": "dt_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
    "name": "Production Token",
    "status": 1,
    "expires_at": "2024-12-31T23:59:59Z",
    "created_at": "2024-01-15T10:30:00Z"
  }
}
```

> **Token 仅在创建时返回明文，之后无法再次获取。** 格式为 `dt_` + 32 位十六进制字符，数据库中以 SHA-256 哈希存储。

### GET /admin/sources/{id}/tokens

获取指定数据源的 Token 列表。**需要 JWT 认证。**

### PUT /admin/tokens/{id}/status

更新 Token 状态。**需要 JWT 认证。**

**请求体：**

```json
{
  "status": 0
}
```

> `status`: `1` = 启用，`0` = 禁用。

### DELETE /admin/tokens/{id}

删除 Token。**需要 JWT 认证。**

---

## 数据采集

### POST /collect/{source_id}

提交单条数据记录。**需要 Data Token 认证。** 受速率限制。

**请求头：**

```
X-Data-Token: dt_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6
Content-Type: application/json
```

**请求体：** 根据数据源的 `schema_config` 提交对应字段。

```json
{
  "email": "user@example.com",
  "action": "login"
}
```

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "record_id": 12345
  }
}
```

**校验错误响应 (400)：**

```json
{
  "code": 1002,
  "message": "数据验证失败",
  "errors": {
    "email": "invalid email format",
    "action": "max_length exceeded (max: 50)"
  }
}
```

### POST /collect/{source_id}/batch

批量提交数据记录。**需要 Data Token 认证。** 受速率限制。

**请求体：**

```json
{
  "records": [
    { "email": "user1@example.com", "action": "login" },
    { "email": "user2@example.com", "action": "signup" }
  ]
}
```

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "total": 2,
    "succeeded": 2,
    "failed": 0,
    "record_ids": [12345, 12346]
  }
}
```

> 支持部分成功：即使部分记录校验失败，成功的记录仍会被保存。

---

## 数据管理

### GET /admin/data

查询数据记录。**需要 JWT 认证。**

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| source_id | integer | - | 按数据源筛选 |
| start_date | string | - | 开始日期 (YYYY-MM-DD) |
| end_date | string | - | 结束日期 (YYYY-MM-DD) |
| page | integer | 1 | 页码 |
| size | integer | 20 | 每页数量 (1-100) |

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "total": 150,
    "list": [
      {
        "id": 12345,
        "source_id": 1,
        "token_id": 42,
        "data": {
          "email": "user@example.com",
          "action": "login"
        },
        "ip_address": "192.168.1.1",
        "user_agent": "Mozilla/5.0 ...",
        "created_at": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

### DELETE /admin/data/{id}

删除单条数据记录。**需要 JWT 认证。**

### POST /admin/data/batch-delete

批量删除数据记录。**需要 JWT 认证。**

**请求体：**

```json
{
  "ids": [12345, 12346, 12347]
}
```

### GET /admin/data/export

导出数据。**需要 JWT 认证。**

**查询参数：**

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| source_id | integer | - | 按数据源筛选 |
| start_date | string | - | 开始日期 |
| end_date | string | - | 结束日期 |
| format | string | csv | 导出格式：`csv` 或 `json` |

> 响应为文件下载，`Content-Disposition` 头包含文件名。

---

## 仪表盘

### GET /admin/dashboard

获取仪表盘统计数据。**需要 JWT 认证。**

**响应：**

```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "today_count": 145,
    "week_count": 890,
    "month_count": 3250,
    "total_sources": 5,
    "recent_records": [
      {
        "id": 12345,
        "source_id": 1,
        "data": { "email": "user@example.com" },
        "ip_address": "192.168.1.1",
        "created_at": "2024-01-15T10:30:00Z"
      }
    ]
  }
}
```

---

## 错误码参考

| 错误码 | HTTP 状态 | 含义 |
|--------|-----------|------|
| 0 | 200 | 成功 |
| 1000 | 401 | Data Token 无效或缺失 |
| 1001 | 403 | Data Token 已禁用 |
| 1002 | 400 | 数据校验失败 |
| 1003 | 429 | 超出速率限制 |
| 2000 | 401 | 登录失败（用户名或密码错误） |
| 2001 | 401 | JWT Token 已过期 |
| 2002 | 403 | 权限不足 |
| 2003 | 401 | JWT Token 无效 |
| 3000 | 404 | 数据源不存在 |
| 3001 | 500 | 创建数据源失败 |
| 3002 | 500 | 更新数据源失败 |
| 3003 | 500 | 删除数据源失败 |
| 4000 | 400 | 查询参数错误 |
| 4001 | 500 | 数据导出失败 |
| 5000 | 503 | 系统状态异常 |
| 5001 | 500 | 系统初始化失败 |
| 5002 | 400 | 系统已初始化 |
| 9000 | 400 | 缺少必要参数 |
| 9001 | 500 | 内部服务器错误 |
