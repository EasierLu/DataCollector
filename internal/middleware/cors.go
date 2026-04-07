package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware 返回 CORS 中间件
// allowedOrigins: 允许的源列表，["*"] 表示允许所有
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查是否允许所有源
		allowAll := false
		for _, o := range allowedOrigins {
			if o == "*" {
				allowAll = true
				break
			}
		}

		// 设置 Access-Control-Allow-Origin
		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else {
			// 检查请求的 Origin 是否在允许列表中
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					c.Header("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		// 设置其他 CORS 头
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Data-Token")
		c.Header("Access-Control-Max-Age", "86400")

		// 处理预检请求 (OPTIONS)
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
