package server

import (
	"context"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/datacollector/datacollector/internal/api"
	"github.com/datacollector/datacollector/internal/auth"
	"github.com/datacollector/datacollector/internal/collector"
	"github.com/datacollector/datacollector/internal/config"
	"github.com/datacollector/datacollector/internal/middleware"
	"github.com/datacollector/datacollector/internal/monitor"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/datacollector/datacollector/internal/web"
)

// Server HTTP Server 封装
type Server struct {
	engine     *gin.Engine
	config     *config.Config
	store      storage.DataStore
	jwtManager *auth.JWTManager
	processor  *collector.Processor
	hub        *monitor.WebSocketHub
	logger     *slog.Logger
	rateLimiter *middleware.RateLimiter
}

// NewServer 创建新的 HTTP Server
func NewServer(
	cfg *config.Config,
	store storage.DataStore,
	jwtManager *auth.JWTManager,
	processor *collector.Processor,
	hub *monitor.WebSocketHub,
	logger *slog.Logger,
) *Server {
	return &Server{
		config:      cfg,
		store:       store,
		jwtManager:  jwtManager,
		processor:   processor,
		hub:         hub,
		logger:      logger,
		rateLimiter: middleware.NewRateLimiter(),
	}
}

// Setup 配置 Gin Engine，注册全局中间件和路由
func (s *Server) Setup() {
	// 设置 gin mode (debug/release)
	gin.SetMode(s.config.Server.Mode)

	// 创建 gin engine
	s.engine = gin.New()

	// 注册全局中间件
	s.engine.Use(gin.Recovery())
	s.engine.Use(middleware.RequestLoggerMiddleware(s.logger))
	s.engine.Use(middleware.CORSMiddleware(s.config.Collector.AllowedOrigins))
	s.engine.Use(middleware.BodySizeLimitMiddleware(s.config.Collector.MaxBodySize))
	s.engine.Use(middleware.MaxBytesErrorHandler())

	// 初始化状态检查中间件
	initChecker := func() bool {
		val, err := s.store.GetConfig(context.Background(), "initialized")
		return err == nil && val == "true"
	}
	s.engine.Use(auth.SetupCheckMiddleware(initChecker))

	// 注册 API 路由
	api.RegisterRoutes(s.engine, s.store, s.config, s.jwtManager, s.processor, s.rateLimiter)

	// 注册 WebSocket 路由（需要 JWT 认证）
	s.engine.GET("/api/v1/admin/ws/monitor",
		auth.JWTAuthMiddleware(s.jwtManager),
		s.hub.HandleWebSocket,
	)

	// 注册 SPA 静态资源和 fallback 路由
	s.serveSPA()
}

// Engine 返回 gin.Engine 供 http.Server 使用
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// serveSPA 提供 Vue SPA 静态资源服务，并处理前端路由 fallback
func (s *Server) serveSPA() {
	// 从 embed.FS 中获取 dist 子目录
	distFS, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		s.logger.Error("failed to get dist sub filesystem", "error", err)
		return
	}

	// 读取 index.html 内容用于 SPA fallback
	indexHTML, err := fs.ReadFile(distFS, "index.html")
	if err != nil {
		s.logger.Error("failed to read index.html", "error", err)
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	// 使用 NoRoute 处理所有未匹配的路由
	s.engine.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		// API 路由返回 404 JSON
		if strings.HasPrefix(path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    404,
				"message": "API not found",
			})
			return
		}

		// 尝试作为静态文件提供（JS、CSS、图片等）
		filePath := strings.TrimPrefix(path, "/")
		if filePath != "" {
			if f, err := distFS.Open(filePath); err == nil {
				f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}

		// 其他所有路由返回 index.html（SPA 前端路由）
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}
