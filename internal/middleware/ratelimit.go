package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/gin-gonic/gin"
)

// RateLimiter 限流器
type RateLimiter struct {
	// 内部使用 map + sync.RWMutex 实现滑动窗口
	// key: 标识符（token 值或 IP）
	// value: 请求时间戳列表
	records map[string][]time.Time
	mu      sync.RWMutex
	window  time.Duration
}

// NewRateLimiter 创建新的限流器
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		records: make(map[string][]time.Time),
		window:  time.Minute,
	}
	// 启动定期清理 goroutine
	go rl.cleanupRoutine()
	return rl
}

// cleanupRoutine 定期清理过期的记录
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup 清理所有过期的记录
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	for key, timestamps := range rl.records {
		// 过滤掉过期的记录
		var valid []time.Time
		for _, ts := range timestamps {
			if ts.After(cutoff) {
				valid = append(valid, ts)
			}
		}

		if len(valid) == 0 {
			delete(rl.records, key)
		} else {
			rl.records[key] = valid
		}
	}
}

// isAllowed 检查是否允许请求（滑动窗口算法）
func (rl *RateLimiter) isAllowed(key string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// 获取该 key 的历史记录
	timestamps := rl.records[key]

	// 过滤掉过期的记录
	var valid []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			valid = append(valid, ts)
		}
	}

	// 检查是否超过限制
	if len(valid) >= limit {
		rl.records[key] = valid
		return false
	}

	// 添加当前请求时间戳
	valid = append(valid, now)
	rl.records[key] = valid

	return true
}

// IPRateLimitMiddleware 按 IP 限流
// limit: 每分钟允许的最大请求数
func (rl *RateLimiter) IPRateLimitMiddleware(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !rl.isAllowed(clientIP, limit) {
			model.SendError(c, http.StatusTooManyRequests, model.CodeRateLimitExceeded, "请求频率超限，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}

// TokenRateLimitMiddleware 按 Data Token 限流
// limit: 每分钟允许的最大请求数
// 从 X-Data-Token 头获取 token 作为限流 key
func (rl *RateLimiter) TokenRateLimitMiddleware(limit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Data-Token")
		if token == "" {
			model.SendError(c, http.StatusBadRequest, model.CodeInvalidToken, "缺少 X-Data-Token 请求头")
			c.Abort()
			return
		}

		if !rl.isAllowed(token, limit) {
			model.SendError(c, http.StatusTooManyRequests, model.CodeRateLimitExceeded, "请求频率超限，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}
