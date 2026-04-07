package api

import (
	"net/http"
	"time"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct {
	store     storage.DataStore
	startTime time.Time
	version   string
}

// NewHealthHandler 创建新的健康检查处理器
func NewHealthHandler(store storage.DataStore, version string) *HealthHandler {
	return &HealthHandler{
		store:     store,
		startTime: time.Now(),
		version:   version,
	}
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status   string `json:"status"`
	Version  string `json:"version"`
	Uptime   string `json:"uptime"`
	Database string `json:"database"`
}

// HealthCheck 健康检查
// GET /api/v1/health
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	// Ping 数据库检查连接
	err := h.store.Ping(c.Request.Context())

	if err != nil {
		// 数据库连接失败返回 503
		c.JSON(http.StatusServiceUnavailable, model.Response{
			Code:    model.CodeSystemUnhealthy,
			Message: model.GetErrorMessage(model.CodeSystemUnhealthy),
			Data: gin.H{
				"status":   "unhealthy",
				"database": "disconnected",
			},
		})
		return
	}

	// 计算运行时间
	uptime := time.Since(h.startTime)

	model.SendSuccess(c, HealthResponse{
		Status:   "healthy",
		Version:  h.version,
		Uptime:   uptime.String(),
		Database: "connected",
	})
}
