package api

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/datacollector/datacollector/internal/auth"
	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

const (
	maxLoginFailures   = 5
	loginLockoutWindow = 15 * time.Minute
)

// loginAttempt 记录登录失败
type loginAttempt struct {
	failures int
	lockedAt time.Time
}

// AuthHandler 认证处理器
type AuthHandler struct {
	store      storage.DataStore
	jwtManager *auth.JWTManager

	mu       sync.Mutex
	attempts map[string]*loginAttempt // username -> attempt
}

// NewAuthHandler 创建新的认证处理器
func NewAuthHandler(store storage.DataStore, jwtManager *auth.JWTManager) *AuthHandler {
	return &AuthHandler{
		store:      store,
		jwtManager: jwtManager,
		attempts:   make(map[string]*loginAttempt),
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

// isAccountLocked 检查账户是否被锁定
func (h *AuthHandler) isAccountLocked(username string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	attempt, ok := h.attempts[username]
	if !ok {
		return false
	}
	if attempt.failures >= maxLoginFailures {
		if time.Since(attempt.lockedAt) < loginLockoutWindow {
			return true
		}
		// 锁定窗口已过，重置
		delete(h.attempts, username)
	}
	return false
}

// recordLoginFailure 记录登录失败
func (h *AuthHandler) recordLoginFailure(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	attempt, ok := h.attempts[username]
	if !ok {
		attempt = &loginAttempt{}
		h.attempts[username] = attempt
	}
	attempt.failures++
	if attempt.failures >= maxLoginFailures {
		attempt.lockedAt = time.Now()
	}
}

// clearLoginFailures 清除登录失败记录
func (h *AuthHandler) clearLoginFailures(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.attempts, username)
}

// Login 用户登录
// POST /api/v1/admin/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// 检查账户是否被锁定
	if h.isAccountLocked(req.Username) {
		model.SendError(c, http.StatusTooManyRequests, model.CodeRateLimitExceeded, "账户已被锁定，请稍后再试")
		return
	}

	// 获取用户
	user, err := h.store.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		h.recordLoginFailure(req.Username)
		model.SendError(c, http.StatusUnauthorized, model.CodeLoginFailed, "")
		return
	}

	// 验证密码
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		h.recordLoginFailure(req.Username)
		model.SendError(c, http.StatusUnauthorized, model.CodeLoginFailed, "")
		return
	}

	// 检查用户状态
	if user.Status == 0 {
		model.SendError(c, http.StatusForbidden, model.CodePermissionDenied, "")
		return
	}

	// 登录成功，清除失败记录
	h.clearLoginFailures(req.Username)

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
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidJWT, "invalid Authorization format")
		return
	}

	tokenString := parts[1]

	// 刷新 token
	newToken, expiresIn, err := h.jwtManager.RefreshToken(tokenString)
	if err != nil {
		if errors.Is(err, auth.ErrTokenExpired) {
			model.SendError(c, http.StatusUnauthorized, model.CodeTokenExpired, "")
		} else if errors.Is(err, auth.ErrRefreshTooEarly) {
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
	if err != nil {
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
