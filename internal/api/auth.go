package api

import (
	"net/http"

	"github.com/datacollector/datacollector/internal/auth"
	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	store      storage.DataStore
	jwtManager *auth.JWTManager
}

// NewAuthHandler 创建新的认证处理器
func NewAuthHandler(store storage.DataStore, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		store:      store,
		jwtManager: jwtManager,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
}

// Login 用户登录
// POST /api/v1/admin/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// 获取用户
	user, err := h.store.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		model.SendError(c, http.StatusUnauthorized, model.CodeLoginFailed, "")
		return
	}

	// 验证密码
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		model.SendError(c, http.StatusUnauthorized, model.CodeLoginFailed, "")
		return
	}

	// 检查用户状态
	if user.Status == 0 {
		model.SendError(c, http.StatusForbidden, model.CodePermissionDenied, "")
		return
	}

	// 生成 JWT
	token, expiresIn, err := h.jwtManager.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to generate token")
		return
	}

	model.SendSuccess(c, LoginResponse{
		Token:     token,
		ExpiresIn: expiresIn,
	})
}

// RefreshTokenResponse 刷新 Token 响应
type RefreshTokenResponse struct {
	Token     string `json:"token"`
	ExpiresIn int64  `json:"expires_in"`
}

// RefreshToken 刷新 JWT Token
// POST /api/v1/admin/refresh-token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	// 从请求头获取当前 token
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidJWT, "missing Authorization header")
		return
	}

	// 解析 Bearer token
	parts := make([]string, 0)
	for i, part := range splitAuthHeader(authHeader) {
		if i < 2 {
			parts = append(parts, part)
		}
	}
	if len(parts) != 2 || parts[0] != "Bearer" {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidJWT, "invalid Authorization format")
		return
	}

	tokenString := parts[1]

	// 刷新 token
	newToken, expiresIn, err := h.jwtManager.RefreshToken(tokenString)
	if err != nil {
		if err.Error() == "token expired" {
			model.SendError(c, http.StatusUnauthorized, model.CodeTokenExpired, "")
		} else if err.Error() == "token can only be refreshed when less than 2 hours remaining" {
			model.SendError(c, http.StatusBadRequest, model.CodeInvalidJWT, err.Error())
		} else {
			model.SendError(c, http.StatusUnauthorized, model.CodeInvalidJWT, err.Error())
		}
		return
	}

	model.SendSuccess(c, RefreshTokenResponse{
		Token:     newToken,
		ExpiresIn: expiresIn,
	})
}

// ChangePassword 修改密码
// POST /api/v1/admin/change-password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// 从 JWT 中获取当前用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidJWT, "")
		return
	}

	// 获取用户信息
	user, err := h.store.GetUserByID(c.Request.Context(), userID.(int64))
	if err != nil || user == nil {
		model.SendError(c, http.StatusInternalServerError, model.CodePasswordChangeFail, "用户不存在")
		return
	}

	// 验证旧密码
	if !auth.CheckPassword(req.OldPassword, user.PasswordHash) {
		model.SendError(c, http.StatusBadRequest, model.CodeOldPasswordWrong, "")
		return
	}

	// 加密新密码
	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodePasswordChangeFail, "密码加密失败")
		return
	}

	// 更新密码
	user.PasswordHash = newHash
	if err := h.store.UpdateUser(c.Request.Context(), user); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodePasswordChangeFail, "更新密码失败")
		return
	}

	model.SendSuccess(c, nil)
}

// splitAuthHeader 分割 Authorization 头
func splitAuthHeader(header string) []string {
	result := make([]string, 0)
	current := ""
	for _, ch := range header {
		if ch == ' ' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
