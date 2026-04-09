package api

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// 全局限流配置的默认值
const (
	DefaultRateLimitPerIP         = 200
	DefaultRateLimitPerIPBurst    = 50
	DefaultRateLimitPerToken      = 100
	DefaultRateLimitPerTokenBurst = 20
)

// SettingsHandler 系统设置处理器
type SettingsHandler struct {
	store storage.DataStore
}

// NewSettingsHandler 创建新的设置处理器
func NewSettingsHandler(store storage.DataStore) *SettingsHandler {
	return &SettingsHandler{store: store}
}

// RateLimitSettings 限流配置响应/请求结构
type RateLimitSettings struct {
	RateLimitPerIP         int `json:"rate_limit_per_ip"`
	RateLimitPerIPBurst    int `json:"rate_limit_per_ip_burst"`
	RateLimitPerToken      int `json:"rate_limit_per_token"`
	RateLimitPerTokenBurst int `json:"rate_limit_per_token_burst"`
}

// GetRateLimitSettings 获取全局限流配置
// GET /api/v1/admin/settings/rate-limit
func (h *SettingsHandler) GetRateLimitSettings(c *gin.Context) {
	ctx := c.Request.Context()
	settings := h.loadRateLimitSettings(ctx)
	model.SendSuccess(c, settings)
}

// UpdateRateLimitSettings 更新全局限流配置
// PUT /api/v1/admin/settings/rate-limit
func (h *SettingsHandler) UpdateRateLimitSettings(c *gin.Context) {
	var req RateLimitSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// 参数校验
	if req.RateLimitPerIP < 0 || req.RateLimitPerIPBurst < 0 ||
		req.RateLimitPerToken < 0 || req.RateLimitPerTokenBurst < 0 {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "rate limit values must be non-negative")
		return
	}

	ctx := c.Request.Context()

	pairs := map[string]int{
		"rate_limit_per_ip":          req.RateLimitPerIP,
		"rate_limit_per_ip_burst":    req.RateLimitPerIPBurst,
		"rate_limit_per_token":       req.RateLimitPerToken,
		"rate_limit_per_token_burst": req.RateLimitPerTokenBurst,
	}

	for key, val := range pairs {
		if err := h.store.SetConfig(ctx, key, strconv.Itoa(val)); err != nil {
			model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to save config: "+err.Error())
			return
		}
	}

	InvalidateRateLimitCache()

	model.SendSuccess(c, req)
}

// loadRateLimitSettings 从数据库加载限流配置
func (h *SettingsHandler) loadRateLimitSettings(ctx context.Context) RateLimitSettings {
	return LoadRateLimitSettings(ctx, h.store)
}

var (
	rateLimitCacheMu   sync.RWMutex
	rateLimitCacheVal  *RateLimitSettings
	rateLimitCacheTime time.Time
	rateLimitCacheTTL  = 30 * time.Second
)

// InvalidateRateLimitCache clears the cached rate limit settings.
func InvalidateRateLimitCache() {
	rateLimitCacheMu.Lock()
	rateLimitCacheVal = nil
	rateLimitCacheMu.Unlock()
}

// LoadRateLimitSettings 从 store 加载全局限流配置（带缓存，供外部使用）
func LoadRateLimitSettings(ctx context.Context, store storage.DataStore) RateLimitSettings {
	rateLimitCacheMu.RLock()
	if rateLimitCacheVal != nil && time.Since(rateLimitCacheTime) < rateLimitCacheTTL {
		cached := *rateLimitCacheVal
		rateLimitCacheMu.RUnlock()
		return cached
	}
	rateLimitCacheMu.RUnlock()

	settings := loadRateLimitSettingsFromDB(ctx, store)

	rateLimitCacheMu.Lock()
	rateLimitCacheVal = &settings
	rateLimitCacheTime = time.Now()
	rateLimitCacheMu.Unlock()

	return settings
}

func loadRateLimitSettingsFromDB(ctx context.Context, store storage.DataStore) RateLimitSettings {
	settings := RateLimitSettings{
		RateLimitPerIP:         DefaultRateLimitPerIP,
		RateLimitPerIPBurst:    DefaultRateLimitPerIPBurst,
		RateLimitPerToken:      DefaultRateLimitPerToken,
		RateLimitPerTokenBurst: DefaultRateLimitPerTokenBurst,
	}

	if v, err := store.GetConfig(ctx, "rate_limit_per_ip"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			settings.RateLimitPerIP = n
		}
	}
	if v, err := store.GetConfig(ctx, "rate_limit_per_ip_burst"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			settings.RateLimitPerIPBurst = n
		}
	}
	if v, err := store.GetConfig(ctx, "rate_limit_per_token"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			settings.RateLimitPerToken = n
		}
	}
	if v, err := store.GetConfig(ctx, "rate_limit_per_token_burst"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			settings.RateLimitPerTokenBurst = n
		}
	}

	return settings
}
