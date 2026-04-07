package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequestLoggerMiddleware 记录每个 HTTP 请求的结构化日志
// 记录字段：trace_id, method, path, status, latency, client_ip, user_agent
func RequestLoggerMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成唯一的 trace_id
		traceID := uuid.New().String()

		// 将 trace_id 存入 context
		c.Set("trace_id", traceID)

		// 记录请求开始时间
		start := time.Now()

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取请求信息
		method := c.Request.Method
		path := c.Request.URL.Path
		status := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// 构建日志属性
		attrs := []slog.Attr{
			slog.String("trace_id", traceID),
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("latency", latency),
			slog.String("client_ip", clientIP),
			slog.String("user_agent", userAgent),
		}

		// 如果有错误，记录错误信息
		if len(c.Errors) > 0 {
			errMsgs := make([]string, 0, len(c.Errors))
			for _, err := range c.Errors {
				errMsgs = append(errMsgs, err.Error())
			}
			attrs = append(attrs, slog.Any("errors", errMsgs))
		}

		// 根据状态码选择日志级别
		if status >= 500 {
			logger.LogAttrs(c.Request.Context(), slog.LevelError, "HTTP request completed", attrs...)
		} else if status >= 400 {
			logger.LogAttrs(c.Request.Context(), slog.LevelWarn, "HTTP request completed", attrs...)
		} else {
			logger.LogAttrs(c.Request.Context(), slog.LevelInfo, "HTTP request completed", attrs...)
		}
	}
}
