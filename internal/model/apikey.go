package model

import (
	"strings"
	"time"
)

// API Key 权限常量
const (
	PermissionQuery = "query" // 数据查询权限
)

// AllPermissions 所有可用权限列表（新增权限在此追加即可）
var AllPermissions = []string{
	PermissionQuery,
}

// ApiKey 独立 API Key 模型（用于数据查询等操作，与数据上报 Token 无关）
type ApiKey struct {
	ID          int64      `json:"id"`
	KeyHash     string     `json:"-"` // SHA-256 哈希，不暴露
	Name        string     `json:"name"`
	Permissions string     `json:"permissions"` // 逗号分隔的权限列表，如 "query" 或 "query,export"
	ExpiresAt   *time.Time `json:"expires_at"`  // nil = 永不过期
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedBy   int64      `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

// HasPermission 检查 API Key 是否拥有指定权限
func (k *ApiKey) HasPermission(perm string) bool {
	for _, p := range strings.Split(k.Permissions, ",") {
		if strings.TrimSpace(p) == perm {
			return true
		}
	}
	return false
}
