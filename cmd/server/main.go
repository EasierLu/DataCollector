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

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/datacollector/datacollector/internal/auth"
	"github.com/datacollector/datacollector/internal/collector"
	"github.com/datacollector/datacollector/internal/config"
	"github.com/datacollector/datacollector/internal/monitor"
	"github.com/datacollector/datacollector/internal/server"
	"github.com/datacollector/datacollector/internal/storage"
)

const version = "1.0.0"

func main() {
	// 1. 初始化日志（slog，JSON格式，根据环境选择输出）
	logger := initLogger()
	logger.Info("starting DataCollector server", "version", version)

	// 2. 加载配置
	cfg := loadConfig(logger)
	if cfg == nil {
		logger.Error("failed to load configuration, exiting")
		os.Exit(1)
	}

	// 3. 校验 JWT 密钥
	validateJWTSecret(cfg, logger)

	// 4. 创建数据目录和日志目录
	ensureDirectories(cfg, logger)

	// 4. 配置日志轮转（如果输出到文件，使用 lumberjack）
	if cfg.Log.Output == "file" {
		logger = initFileLogger(cfg.Log)
	}

	// 5. 初始化数据库存储
	ctx := context.Background()
	store, err := storage.NewDataStore(cfg)
	if err != nil {
		logger.Error("failed to create datastore", "error", err)
		os.Exit(1)
	}

	// 执行数据库迁移
	if err := store.Init(ctx); err != nil {
		logger.Error("failed to initialize database", "error", err)
		os.Exit(1)
	}

	// 启动自检
	if err := store.Ping(ctx); err != nil {
		logger.Error("database ping failed", "error", err)
		os.Exit(1)
	}
	logger.Info("database connection established")

	// 6. 初始化 JWT Manager
	jwtManager := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.Expiration)
	logger.Info("JWT manager initialized")

	// 7. 初始化 WebSocket Hub
	hub := monitor.NewWebSocketHub(logger, cfg.Collector.AllowedOrigins)
	go hub.Run()
	logger.Info("WebSocket hub started")

	// 8. 初始化统计聚合器
	aggregator := monitor.NewAggregator(store, hub, logger)
	go aggregator.Start(ctx)
	logger.Info("statistics aggregator started")

	// 9. 初始化数据处理器
	processor := collector.NewProcessor(store, aggregator.EventChannel())
	logger.Info("data processor initialized")

	// 10. 创建并配置 HTTP Server
	srv := server.NewServer(cfg, store, jwtManager, processor, hub, logger)
	srv.Setup()
	logger.Info("HTTP server configured")

	// 11. 启动 HTTP 服务（goroutine）
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

	// 12. 信号监听（SIGTERM, SIGINT）
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	logger.Info("shutdown signal received, starting graceful shutdown...")

	// 13. 优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭 HTTP Server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP server shutdown error", "error", err)
	}

	// 停止聚合器（flush 最后的统计数据）
	aggregator.Stop()
	logger.Info("statistics aggregator stopped")

	// 关闭数据库连接
	if err := store.Close(); err != nil {
		logger.Error("database close error", "error", err)
	}

	logger.Info("shutdown complete")
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

// loadConfig 加载配置
func loadConfig(logger *slog.Logger) *config.Config {
	cfgPath := "configs/config.yaml"

	// 尝试从配置文件加载
	cfg, err := config.Load(cfgPath)
	if err != nil {
		logger.Warn("failed to load config file, using default config", "path", cfgPath, "error", err)
		cfg = config.DefaultConfig()
	} else {
		logger.Info("configuration loaded from file", "path", cfgPath)
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
