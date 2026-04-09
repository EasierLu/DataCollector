package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware JWT 认证中间件
// 从 Authorization: Bearer <token> 头获取 token 并验证
// 如果请求头没有 token，则尝试从 URL 查询参数 token 获取（支持 WebSocket 连接）
// 验证成功后将 Claims 信息存入 gin.Context：
//   c.Set("user_id", claims.UserID)
//   c.Set("username", claims.Username)
//   c.Set("role", claims.Role)
// 验证失败返回 401
func JWTAuthMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 优先从请求头获取 Authorization
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			// 解析 Bearer token
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				tokenString = parts[1]
			}
		}

		// 如果请求头没有 token，尝试从 WebSocket 子协议获取
		if tokenString == "" {
			if proto := c.GetHeader("Sec-WebSocket-Protocol"); proto != "" {
				for _, p := range strings.Split(proto, ",") {
					p = strings.TrimSpace(p)
					if strings.HasPrefix(p, "access_token.") {
						tokenString = strings.TrimPrefix(p, "access_token.")
						break
					}
				}
			}
		}

		if tokenString == "" {
			model.SendError(c, http.StatusUnauthorized, model.CodeInvalidJWT, "缺少认证信息")
			c.Abort()
			return
		}

		// 验证 token
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				model.SendError(c, http.StatusUnauthorized, model.CodeTokenExpired, "Token 已过期")
			} else {
				model.SendError(c, http.StatusUnauthorized, model.CodeInvalidJWT, "无效的 Token")
			}
			c.Abort()
			return
		}

		// 将用户信息存入 context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RequireRole RBAC 角色检查中间件
// 检查 context 中的 role 是否在允许的角色列表中
// 不在则返回 403
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		if !exists {
			model.SendError(c, http.StatusForbidden, model.CodePermissionDenied, "未获取到用户角色信息")
			c.Abort()
			return
		}

		userRole, ok := roleValue.(string)
		if !ok {
			model.SendError(c, http.StatusForbidden, model.CodePermissionDenied, "用户角色信息格式错误")
			c.Abort()
			return
		}

		// 检查角色是否在允许列表中
		for _, allowedRole := range roles {
			if userRole == allowedRole {
				c.Next()
				return
			}
		}

		model.SendError(c, http.StatusForbidden, model.CodePermissionDenied, "权限不足")
		c.Abort()
	}
}

// InitChecker 初始化状态检查函数类型
type InitChecker func() bool

// SetupCheckMiddleware 检查系统是否已初始化的中间件
// 需要传入一个检查函数来查询初始化状态
// 如果未初始化，对管理页面请求重定向到 /setup
func SetupCheckMiddleware(checker InitChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查系统是否已初始化
		if checker() {
			// 已初始化，继续处理请求
			c.Next()
			return
		}

		// 未初始化，检查是否是管理页面请求
		path := c.Request.URL.Path

		// 允许访问 /setup 相关路径
		if strings.HasPrefix(path, "/setup") || strings.HasPrefix(path, "/api/v1/setup") || strings.HasPrefix(path, "/api/setup") {
			c.Next()
			return
		}

		// 允许访问静态资源（CSS、JS、图片等）
		if strings.HasPrefix(path, "/static/") ||
			strings.HasPrefix(path, "/assets/") ||
			strings.HasSuffix(path, ".css") ||
			strings.HasSuffix(path, ".js") ||
			strings.HasSuffix(path, ".png") ||
			strings.HasSuffix(path, ".jpg") ||
			strings.HasSuffix(path, ".jpeg") ||
			strings.HasSuffix(path, ".gif") ||
			strings.HasSuffix(path, ".svg") ||
			strings.HasSuffix(path, ".ico") {
			c.Next()
			return
		}

		// API 请求返回 JSON 错误
		if strings.HasPrefix(path, "/api/") {
			model.SendError(c, http.StatusServiceUnavailable, model.CodeInitFailed, "系统未初始化，请先完成初始化")
			c.Abort()
			return
		}

		// 页面请求重定向到 /setup
		c.Redirect(http.StatusFound, "/setup")
		c.Abort()
	}
}
