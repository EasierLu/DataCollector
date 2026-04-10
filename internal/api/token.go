package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// TokenHandler Token 管理处理器
type TokenHandler struct {
	store storage.DataStore
}

// NewTokenHandler 创建新的 Token 管理处理器
func NewTokenHandler(store storage.DataStore) *TokenHandler {
	return &TokenHandler{
		store: store,
	}
}

// CreateTokenRequest 创建 Token 请求
type CreateTokenRequest struct {
	Name      string     `json:"name" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// UpdateTokenStatusRequest 更新 Token 状态请求
type UpdateTokenStatusRequest struct {
	Status int `json:"status" binding:"required,oneof=0 1"`
}

// CreateTokenResponse 创建 Token 响应（包含明文 token）
type CreateTokenResponse struct {
	ID        int64      `json:"id"`
	Token     string     `json:"token"`
	Name      string     `json:"name"`
	Status    int        `json:"status"`
	ExpiresAt *time.Time `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

// generateRandomToken 生成随机 token（256 bits）
func generateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "dt_" + hex.EncodeToString(bytes), nil
}

// hashToken 计算 token 的 HMAC-SHA256 哈希
func hashToken(token string) string {
	return hmacSHA256(token)
}

// CreateToken 创建 Token
// POST /api/v1/admin/sources/:id/tokens
func (h *TokenHandler) CreateToken(c *gin.Context) {
	sourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid source id")
		return
	}

	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// 从 context 获取 user_id
	userID, exists := c.Get("user_id")
	if !exists {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidJWT, "user_id not found in context")
		return
	}

	// 生成 token 明文
	plainToken, err := generateRandomToken()
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to generate token")
		return
	}

	// 计算哈希
	tokenHash := hashToken(plainToken)

	token := &model.DataToken{
		SourceID:  sourceID,
		TokenHash: tokenHash,
		Name:      req.Name,
		Status:    1,
		ExpiresAt: req.ExpiresAt,
		CreatedBy: userID.(int64),
	}

	id, err := h.store.CreateToken(c.Request.Context(), token)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	// 返回响应（包含明文 token，仅此一次可见）
	model.SendSuccess(c, CreateTokenResponse{
		ID:        id,
		Token:     plainToken,
		Name:      token.Name,
		Status:    token.Status,
		ExpiresAt: token.ExpiresAt,
		CreatedAt: token.CreatedAt,
	})
}

// ListTokens 获取 Token 列表
// GET /api/v1/admin/sources/:id/tokens
func (h *TokenHandler) ListTokens(c *gin.Context) {
	sourceID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid source id")
		return
	}

	tokens, err := h.store.ListTokensBySourceID(c.Request.Context(), sourceID)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	// 不返回 token 明文或哈希值，只返回元信息
	model.SendSuccess(c, tokens)
}

// UpdateTokenStatus 更新 Token 状态
// PUT /api/v1/admin/tokens/:id/status
func (h *TokenHandler) UpdateTokenStatus(c *gin.Context) {
	tokenID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid token id")
		return
	}

	var req UpdateTokenStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	if err := h.store.UpdateTokenStatus(c.Request.Context(), tokenID, req.Status); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "token status updated successfully"})
}

// DeleteToken 删除 Token
// DELETE /api/v1/admin/tokens/:id
func (h *TokenHandler) DeleteToken(c *gin.Context) {
	tokenID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid token id")
		return
	}

	if err := h.store.DeleteToken(c.Request.Context(), tokenID); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "token deleted successfully"})
}
