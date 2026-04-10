package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// limiterEntry 包装令牌桶及其最后访问时间
type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
	rps      float64
	burst    int
}

// RateLimiter 基于令牌桶算法的限流器
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
}

// NewRateLimiter 创建新的限流器
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*limiterEntry),
	}
	go rl.cleanupRoutine()
	return rl
}

// Allow 判断给定 key 是否允许通过（令牌桶算法）
// rps: 每秒允许的请求数, burst: 突发量上限
func (rl *RateLimiter) Allow(key string, rps float64, burst int) bool {
	rl.mu.Lock()
	entry, exists := rl.limiters[key]
	if !exists || entry.rps != rps || entry.burst != burst {
		// 创建新桶或参数已变更时重新创建
		limiter := rate.NewLimiter(rate.Limit(rps), burst)
		entry = &limiterEntry{
			limiter:  limiter,
			lastSeen: time.Now(),
			rps:      rps,
			burst:    burst,
		}
		rl.limiters[key] = entry
	} else {
		entry.lastSeen = time.Now()
	}
	allowed := entry.limiter.Allow()
	rl.mu.Unlock()

	return allowed
}

// cleanupRoutine 定期清理 5 分钟内无访问的桶
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cutoff := time.Now().Add(-5 * time.Minute)
	for key, entry := range rl.limiters {
		if entry.lastSeen.Before(cutoff) {
			delete(rl.limiters, key)
		}
	}
}

// RateLimitConfigProvider 动态获取限流配置的函数类型
type RateLimitConfigProvider func(ctx context.Context) (rps float64, burst int)

// IPRateLimitMiddleware 按 IP 限流（支持动态配置）
func (rl *RateLimiter) IPRateLimitMiddleware(provider RateLimitConfigProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		rps, burst := provider(c.Request.Context())
		if rps <= 0 {
			// 未配置或配置无效，跳过限流
			c.Next()
			return
		}

		clientIP := c.ClientIP()
		key := "ip:" + clientIP

		if !rl.Allow(key, rps, burst) {
			model.SendError(c, http.StatusTooManyRequests, model.CodeRateLimitExceeded, "请求频率超限，请稍后再试")
			c.Abort()
			return
		}

		c.Next()
	}
}
