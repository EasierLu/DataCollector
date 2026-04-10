package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// ApiKeyHandler API Key 管理处理器
type ApiKeyHandler struct {
	store storage.DataStore
}

// NewApiKeyHandler 创建新的 API Key 管理处理器
func NewApiKeyHandler(store storage.DataStore) *ApiKeyHandler {
	return &ApiKeyHandler{store: store}
}

// CreateApiKeyRequest 创建 API Key 请求
type CreateApiKeyRequest struct {
	Name        string     `json:"name" binding:"required"`
	Permissions []string   `json:"permissions" binding:"required,min=1"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// CreateApiKeyResponse 创建 API Key 响应（包含明文 key，仅此一次可见）
type CreateApiKeyResponse struct {
	ID          int64      `json:"id"`
	Key         string     `json:"key"`
	Name        string     `json:"name"`
	Permissions string     `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// UpdateApiKeyPermissionsRequest 更新 API Key 权限请求
type UpdateApiKeyPermissionsRequest struct {
	Permissions []string `json:"permissions" binding:"required,min=1"`
}

// generateApiKey 生成随机 API Key 明文
func generateApiKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "ak_" + hex.EncodeToString(bytes), nil
}

// hashApiKey 计算 API Key 的 HMAC-SHA256 哈希
func hashApiKey(key string) string {
	return hmacSHA256(key)
}

// CreateApiKey 创建 API Key
// POST /api/v1/admin/settings/api-keys
func (h *ApiKeyHandler) CreateApiKey(c *gin.Context) {
	var req CreateApiKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidJWT, "user_id not found in context")
		return
	}

	plainKey, err := generateApiKey()
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to generate API Key")
		return
	}

	// 验证权限值是否合法
	for _, p := range req.Permissions {
		valid := false
		for _, ap := range model.AllPermissions {
			if p == ap {
				valid = true
				break
			}
		}
		if !valid {
			model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "无效的权限: "+p)
			return
		}
	}

	keyHash := hashApiKey(plainKey)
	permissions := strings.Join(req.Permissions, ",")

	apiKey := &model.ApiKey{
		KeyHash:     keyHash,
		Name:        req.Name,
		Permissions: permissions,
		ExpiresAt:   req.ExpiresAt,
		CreatedBy:   userID.(int64),
	}

	id, err := h.store.CreateApiKey(c.Request.Context(), apiKey)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	model.SendSuccess(c, CreateApiKeyResponse{
		ID:          id,
		Key:         plainKey,
		Name:        apiKey.Name,
		Permissions: apiKey.Permissions,
		ExpiresAt:   apiKey.ExpiresAt,
		CreatedAt:   apiKey.CreatedAt,
	})
}

// ListApiKeys 获取 API Key 列表
// GET /api/v1/admin/settings/api-keys
func (h *ApiKeyHandler) ListApiKeys(c *gin.Context) {
	keys, err := h.store.ListApiKeys(c.Request.Context())
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	if keys == nil {
		keys = make([]*model.ApiKey, 0)
	}

	model.SendSuccess(c, keys)
}

// UpdateApiKeyPermissions 更新 API Key 权限
// PUT /api/v1/admin/settings/api-keys/:id/permissions
func (h *ApiKeyHandler) UpdateApiKeyPermissions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid API Key id")
		return
	}

	var req UpdateApiKeyPermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// 验证权限值是否合法
	for _, p := range req.Permissions {
		valid := false
		for _, ap := range model.AllPermissions {
			if p == ap {
				valid = true
				break
			}
		}
		if !valid {
			model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "无效的权限: "+p)
			return
		}
	}

	permissions := strings.Join(req.Permissions, ",")
	if err := h.store.UpdateApiKeyPermissions(c.Request.Context(), id, permissions); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "API Key permissions updated successfully"})
}

// ListPermissions 获取所有可用权限列表
// GET /api/v1/admin/settings/api-keys/permissions
func (h *ApiKeyHandler) ListPermissions(c *gin.Context) {
	model.SendSuccess(c, model.AllPermissions)
}

// DeleteApiKey 删除 API Key
// DELETE /api/v1/admin/settings/api-keys/:id
func (h *ApiKeyHandler) DeleteApiKey(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid API Key id")
		return
	}

	if err := h.store.DeleteApiKey(c.Request.Context(), id); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "API Key deleted successfully"})
}
