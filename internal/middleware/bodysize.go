package middleware

import (
	"net/http"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/gin-gonic/gin"
)

// BodySizeLimitMiddleware 限制请求体大小
// maxSize: 最大字节数
func BodySizeLimitMiddleware(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 使用 MaxBytesReader 限制请求体大小
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

// MaxBytesErrorHandler 处理请求体过大错误
// 需要在路由中配合 Recovery 中间件使用，或者在错误处理中检查
func MaxBytesErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 检查是否有请求体过大的错误
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				if err.Err != nil {
					// 检查是否是请求体过大错误
					if err.Err.Error() == "http: request body too large" {
						model.SendError(c, http.StatusRequestEntityTooLarge, model.CodeValidationFailed, "请求体过大")
						return
					}
				}
			}
		}
	}
}
