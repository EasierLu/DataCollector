package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/datacollector/datacollector/internal/auth"
	"github.com/datacollector/datacollector/internal/config"
	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// SetupHandler 系统初始化处理器
type SetupHandler struct {
	store      storage.DataStore
	config     *config.Config
	jwtManager *auth.JWTManager
}

// NewSetupHandler 创建新的初始化处理器
func NewSetupHandler(store storage.DataStore, cfg *config.Config, jwtManager *auth.JWTManager) *SetupHandler {
	return &SetupHandler{
		store:      store,
		config:     cfg,
		jwtManager: jwtManager,
	}
}

// CheckStatusRequest 检查初始化状态响应
type CheckStatusResponse struct {
	Initialized bool `json:"initialized"`
}

// CheckStatus 检查系统初始化状态
// GET /api/v1/setup/status
func (h *SetupHandler) CheckStatus(c *gin.Context) {
	initialized := false
	value, err := h.store.GetConfig(c.Request.Context(), "initialized")
	if err == nil && value == "true" {
		initialized = true
	}

	model.SendSuccess(c, CheckStatusResponse{Initialized: initialized})
}

// TestDatabaseRequest 测试数据库连接请求
type TestDatabaseRequest struct {
	Driver   string `json:"driver" binding:"required"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
}

// TestDatabase 测试数据库连接
// POST /api/v1/setup/test-db
func (h *SetupHandler) TestDatabase(c *gin.Context) {
	var req TestDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// SQLite 不需要测试连接
	if req.Driver == "sqlite" {
		model.SendSuccess(c, gin.H{"message": "SQLite does not require connection test"})
		return
	}

	// 只支持 postgres
	if req.Driver != "postgres" {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "unsupported database driver")
		return
	}

	// 构建连接字符串
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		req.Host, req.Port, req.User, req.Password, req.DBName)

	// 尝试连接
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		model.SendError(c, http.StatusOK, model.CodeInitFailed, "connection failed: "+err.Error())
		return
	}
	defer db.Close()

	// 测试 Ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		model.SendError(c, http.StatusOK, model.CodeInitFailed, "connection failed: "+err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "connection successful"})
}

// InitializeRequest 初始化系统请求
type InitializeRequest struct {
	Database DatabaseConfigInit `json:"database" binding:"required"`
	Server   ServerConfigInit   `json:"server" binding:"required"`
	Admin    AdminConfigInit    `json:"admin" binding:"required"`
}

// DatabaseConfigInit 数据库初始化配置
type DatabaseConfigInit struct {
	Driver   string                `json:"driver" binding:"required"`
	SQLite   config.SQLiteConfig   `json:"sqlite"`
	Postgres config.PostgresConfig `json:"postgres"`
}

// ServerConfigInit 服务器初始化配置
type ServerConfigInit struct {
	Port int `json:"port" binding:"required"`
}

// AdminConfigInit 管理员初始化配置
type AdminConfigInit struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

// Initialize 初始化系统
// POST /api/v1/setup/init
func (h *SetupHandler) Initialize(c *gin.Context) {
	var req InitializeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// 1. 检查是否已初始化
	value, err := h.store.GetConfig(c.Request.Context(), "initialized")
	if err == nil && value == "true" {
		model.SendError(c, http.StatusBadRequest, model.CodeAlreadyInitialized, "")
		return
	}

	// 2. 更新数据库配置
	h.config.Database.Driver = req.Database.Driver
	if req.Database.Driver == "sqlite" {
		h.config.Database.SQLite = req.Database.SQLite
	} else if req.Database.Driver == "postgres" {
		h.config.Database.Postgres = req.Database.Postgres
	}

	// 3. 更新服务器配置
	h.config.Server.Port = req.Server.Port

	// 4. 加密管理员密码
	passwordHash, err := auth.HashPassword(req.Admin.Password)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to hash password: "+err.Error())
		return
	}

	// 5. 创建管理员用户（如果已存在则更新密码）
	existingUser, _ := h.store.GetUserByUsername(c.Request.Context(), req.Admin.Username)
	if existingUser != nil {
		// 用户已存在，更新密码
		existingUser.PasswordHash = passwordHash
		if err := h.store.UpdateUser(c.Request.Context(), existingUser); err != nil {
			model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to update admin user: "+err.Error())
			return
		}
	} else {
		adminUser := &model.User{
			Username:     req.Admin.Username,
			PasswordHash: passwordHash,
			Role:         "admin",
			Status:       1,
		}
		_, err = h.store.CreateUser(c.Request.Context(), adminUser)
		if err != nil {
			model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to create admin user: "+err.Error())
			return
		}
	}

	// 6. 设置系统配置
	ctx := c.Request.Context()
	if err := h.store.SetConfig(ctx, "initialized", "true"); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to set initialized config: "+err.Error())
		return
	}
	if err := h.store.SetConfig(ctx, "db_driver", req.Database.Driver); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to set db_driver config: "+err.Error())
		return
	}
	if err := h.store.SetConfig(ctx, "server_port", strconv.Itoa(req.Server.Port)); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to set server_port config: "+err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "initialization successful"})
}

// ReinitializeRequest 重新初始化请求
type ReinitializeRequest struct {
	Confirm string `json:"confirm" binding:"required"`
}

// Reinitialize 重新初始化系统
// POST /api/v1/setup/reinit (需要 JWT 认证)
func (h *SetupHandler) Reinitialize(c *gin.Context) {
	// 从上下文中获取用户信息（由 JWT 中间件设置）
	role, exists := c.Get("role")
	if !exists || role != "admin" {
		model.SendError(c, http.StatusForbidden, model.CodePermissionDenied, "")
		return
	}

	var req ReinitializeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// 验证确认字符串
	if req.Confirm != "REINITIALIZE" {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid confirmation string")
		return
	}

	// 执行重新初始化：清除所有数据
	ctx := c.Request.Context()
	if err := h.store.ResetAllData(ctx); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to clear data: "+err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "system reinitialized, please restart the server"})
}

// RegisterRoutes 注册初始化相关路由
func (h *SetupHandler) RegisterRoutes(r *gin.RouterGroup) {
	setup := r.Group("/setup")
	{
		setup.GET("/status", h.CheckStatus)
		setup.POST("/test-db", h.TestDatabase)
		setup.POST("/init", h.Initialize)
		// reinit 需要 JWT 认证，在外部注册时挂载到需要认证的路由组
	}
}

// RegisterReinitRoute 注册重新初始化路由（需要认证）
func (h *SetupHandler) RegisterReinitRoute(r *gin.RouterGroup) {
	r.POST("/reinit", h.Reinitialize)
}
