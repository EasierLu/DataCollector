package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
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
	store       storage.DataStore // 已初始化模式下非 nil，未初始化模式下为 nil
	config      *config.Config
	configPath  string // 配置文件路径，用于回写
	jwtManager  *auth.JWTManager
	restartChan chan<- struct{} // 重启信号 channel
}

// NewSetupHandler 创建新的初始化处理器
func NewSetupHandler(store storage.DataStore, cfg *config.Config, configPath string, jwtManager *auth.JWTManager, restartChan chan<- struct{}) *SetupHandler {
	return &SetupHandler{
		store:       store,
		config:      cfg,
		configPath:  configPath,
		jwtManager:  jwtManager,
		restartChan: restartChan,
	}
}

// CheckStatusRequest 检查初始化状态响应
type CheckStatusResponse struct {
	Initialized bool `json:"initialized"`
}

// CheckStatus 检查系统初始化状态
// GET /api/v1/setup/status
func (h *SetupHandler) CheckStatus(c *gin.Context) {
	model.SendSuccess(c, CheckStatusResponse{Initialized: h.config.Initialized})
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
	if h.config.Initialized {
		model.SendError(c, http.StatusForbidden, model.CodeAlreadyInitialized, "")
		return
	}

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

	if req.Port < 1 || req.Port > 65535 {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid port number")
		return
	}

	if req.Host == "" || req.DBName == "" || req.User == "" {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "host, dbname and user are required")
		return
	}

	// 构建连接字符串
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable connect_timeout=5",
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
	if h.config.Initialized {
		model.SendError(c, http.StatusBadRequest, model.CodeAlreadyInitialized, "")
		return
	}

	// 2. 更新配置中的数据库配置
	h.config.Database.Driver = req.Database.Driver
	if req.Database.Driver == "sqlite" {
		h.config.Database.SQLite = req.Database.SQLite
	} else if req.Database.Driver == "postgres" {
		h.config.Database.Postgres = req.Database.Postgres
	}

	// 3. 更新服务器配置
	h.config.Server.Port = req.Server.Port

	// 4. 用新配置创建数据库存储
	newStore, err := storage.NewDataStore(h.config)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to create datastore: "+err.Error())
		return
	}
	defer newStore.Close()

	// 5. 执行数据库迁移建表
	ctx := c.Request.Context()
	if err := newStore.Init(ctx); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to initialize database: "+err.Error())
		return
	}

	// 6. 验证数据库连接
	if err := newStore.Ping(ctx); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to ping database: "+err.Error())
		return
	}

	// 7. 加密管理员密码
	passwordHash, err := auth.HashPassword(req.Admin.Password)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to hash password: "+err.Error())
		return
	}

	// 8. 创建管理员用户（如果已存在则更新密码）
	existingUser, err := newStore.GetUserByUsername(ctx, req.Admin.Username)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to check existing user: "+err.Error())
		return
	}
	if existingUser != nil {
		existingUser.PasswordHash = passwordHash
		if err := newStore.UpdateUser(ctx, existingUser); err != nil {
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
		_, err = newStore.CreateUser(ctx, adminUser)
		if err != nil {
			model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to create admin user: "+err.Error())
			return
		}
	}

	// 9. 设置 initialized 并写回配置文件
	h.config.Initialized = true
	if err := h.config.Save(h.configPath); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to save config: "+err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "initialization successful, server is restarting..."})

	// 响应发送后触发重启
	go func() {
		time.Sleep(500 * time.Millisecond)
		h.restartChan <- struct{}{}
	}()
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

	// 将 initialized 设为 false 并写回配置文件
	h.config.Initialized = false
	if err := h.config.Save(h.configPath); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInitFailed, "failed to save config: "+err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "system reinitialized, server is restarting..."})

	// 响应发送后触发重启
	go func() {
		time.Sleep(500 * time.Millisecond)
		h.restartChan <- struct{}{}
	}()
}
