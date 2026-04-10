package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/datacollector/datacollector/internal/api"
	"github.com/datacollector/datacollector/internal/auth"
	"github.com/datacollector/datacollector/internal/collector"
	"github.com/datacollector/datacollector/internal/config"
	"github.com/datacollector/datacollector/internal/monitor"
	"github.com/datacollector/datacollector/internal/server"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/datacollector/datacollector/internal/webhook"
)

const (
	version    = "1.0.0"
	configPath = "config.yaml"
)

func main() {
	logger := initLogger()
	logger.Info("starting DataCollector server", "version", version)

	// 全局信号监听
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	for {
		cfg := loadConfig(logger)
		if cfg == nil {
			logger.Error("failed to load configuration, exiting")
			os.Exit(1)
		}

		restartChan := make(chan struct{}, 1)

		var shouldRestart bool
		if !cfg.Initialized {
			logger.Info("system not initialized, starting setup-only mode")
			shouldRestart = runSetupOnly(cfg, logger, restartChan, quit)
		} else {
			logger.Info("system initialized, starting full mode")
			shouldRestart = runFull(cfg, logger, restartChan, quit)
		}

		if !shouldRestart {
			break
		}
		logger.Info("server restarting...")
	}

	logger.Info("shutdown complete")
}

// runSetupOnly 未初始化模式，返回 true 表示需要重启
func runSetupOnly(cfg *config.Config, logger *slog.Logger, restartChan chan struct{}, quit chan os.Signal) bool {
	gin.SetMode(cfg.Server.Mode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	// 未初始化模式：非 setup/静态资源请求重定向到 /setup
	engine.Use(auth.SetupCheckMiddleware(func() bool { return false }))

	// 注册 setup 路由
	api.RegisterSetupRoutes(engine, cfg, configPath, restartChan)

	// 提供 SPA 静态资源
	server.ServeSPA(engine, logger)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: engine,
	}

	go func() {
		logger.Info("setup HTTP server starting", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	var isRestart bool
	select {
	case <-restartChan:
		isRestart = true
		logger.Info("restart signal received from setup handler")
	case <-quit:
		isRestart = false
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	return isRestart
}

// runFull 已初始化模式，返回 true 表示需要重启
func runFull(cfg *config.Config, logger *slog.Logger, restartChan chan struct{}, quit chan os.Signal) bool {
	// 校验 JWT 密钥
	validateJWTSecret(cfg, logger)

	// 创建数据目录和日志目录
	ensureDirectories(cfg, logger)

	// 配置日志轮转
	if cfg.Log.Output == "file" {
		logger = initFileLogger(cfg.Log)
	}

	// 初始化数据库存储
	ctx := context.Background()
	store, err := storage.NewDataStore(cfg)
	if err != nil {
		logger.Error("failed to create datastore", "error", err)
		os.Exit(1)
	}
	if err := store.Init(ctx); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}
	if err := store.Ping(ctx); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	logger.Info("database connection established")

	// 初始化 JWT Manager
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Expiration)
	logger.Info("JWT manager initialized")

	// 初始化 WebSocket Hub
	hub := monitor.NewWebSocketHub(logger, cfg.Collector.AllowedOrigins)
	go hub.Run()
	logger.Info("WebSocket hub started")

	// 初始化统计聚合器
	aggregator := monitor.NewAggregator(store, hub, logger)
	go aggregator.Start(ctx)
	logger.Info("statistics aggregator started")

	// 初始化 Webhook 分发器
	webhookDispatcher := webhook.NewDispatcher(1000)
	webhookDispatcher.Start()
	logger.Info("webhook dispatcher started")

	// 初始化数据处理器
	processor := collector.NewProcessor(store, aggregator.EventChannel(), webhookDispatcher.EventChan())
	logger.Info("data processor initialized")

	// 创建并配置 HTTP Server
	srv := server.NewServer(cfg, store, jwtManager, processor, hub, logger)
	srv.Setup(configPath, restartChan)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: srv.Engine(),
	}

	go func() {
		logger.Info("HTTP server starting", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	var isRestart bool
	select {
	case <-restartChan:
		isRestart = true
		logger.Info("restart signal received from handler")
	case <-quit:
		isRestart = false
		logger.Info("shutdown signal received, starting graceful shutdown...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	aggregator.Stop()
	logger.Info("statistics aggregator stopped")

	webhookDispatcher.Stop()
	logger.Info("webhook dispatcher stopped")

	if err := store.Close(); err != nil {
		logger.Error("database close error", "error", err)
	}

	return isRestart
}

// initLogger 初始化日志（默认输出到 stdout）
func initLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// initFileLogger 初始化文件日志（带轮转）
func initFileLogger(cfg config.LogConfig) *slog.Logger {
	writer := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize, // MB
		MaxAge:     cfg.MaxAge,  // days
		MaxBackups: 3,
		Compress:   true,
	}

	level := parseLogLevel(cfg.Level)

	return slog.New(slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: level,
	}))
}

// loadConfig 加载配置，文件不存在时自动生成默认配置
func loadConfig(logger *slog.Logger) *config.Config {
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error("failed to parse config file", "path", configPath, "error", err)
		return nil
	}

	if cfg == nil {
		// config.yaml 不存在，尝试从旧路径 configs/config.yaml 迁移
		legacyPath := "configs/config.yaml"
		legacyCfg, legacyErr := config.Load(legacyPath)
		if legacyErr != nil {
			logger.Warn("failed to parse legacy config file", "path", legacyPath, "error", legacyErr)
		}
		if legacyCfg != nil {
			// 从旧配置迁移，默认设为已初始化（因为旧版本的数据库已存在）
			cfg = legacyCfg
			cfg.Initialized = true
			if err := cfg.Save(configPath); err != nil {
				logger.Error("failed to migrate config file", "error", err)
				return nil
			}
			logger.Info("migrated config from legacy path", "from", legacyPath, "to", configPath)
		} else {
			// 首次启动，生成默认配置
			cfg = config.DefaultConfig()
			if err := cfg.Save(configPath); err != nil {
				logger.Error("failed to write default config file", "path", configPath, "error", err)
				return nil
			}
			logger.Info("default config file created", "path", configPath)
		}
	} else {
		logger.Info("configuration loaded from file", "path", configPath)
	}

	return cfg
}

// ensureDirectories 确保数据目录和日志目录存在
func ensureDirectories(cfg *config.Config, logger *slog.Logger) {
	dirs := []string{
		"./data",
		"./logs",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Error("failed to create directory", "path", dir, "error", err)
			os.Exit(1)
		}
	}
}

const defaultJWTSecret = "change-me-to-a-secure-random-string"

// validateJWTSecret checks if the JWT secret is still the insecure default value.
// In release mode it refuses to start; in debug mode it generates a random ephemeral secret.
func validateJWTSecret(cfg *config.Config, logger *slog.Logger) {
	if cfg.JWT.Secret != defaultJWTSecret {
		return
	}

	if cfg.Server.Mode == "release" {
		logger.Error("JWT secret is still the default value — refusing to start in release mode. " +
			"Set jwt.secret in config.yaml or JWT_SECRET environment variable.")
		os.Exit(1)
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		logger.Error("failed to generate random JWT secret", "error", err)
		os.Exit(1)
	}
	cfg.JWT.Secret = hex.EncodeToString(b)
	logger.Warn("JWT secret was default — generated random ephemeral secret (set jwt.secret for production)")
}

// parseLogLevel 解析日志级别
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
