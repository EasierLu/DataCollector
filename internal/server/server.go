package server

import (
	"context"
	"html/template"
	"log/slog"
	"net/http"

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

	// 注册静态资源路由
	s.registerStaticRoutes()

	// 注册页面路由
	s.RegisterPageRoutes()

	// 注册 API 路由
	api.RegisterRoutes(s.engine, s.store, s.config, s.jwtManager, s.processor, s.rateLimiter)

	// 注册 WebSocket 路由（需要 JWT 认证）
	s.engine.GET("/api/v1/admin/ws/monitor",
		auth.JWTAuthMiddleware(s.jwtManager),
		s.hub.HandleWebSocket,
	)
}

// Engine 返回 gin.Engine 供 http.Server 使用
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// registerStaticRoutes 注册静态资源路由
func (s *Server) registerStaticRoutes() {
	// 静态资源路由
	staticFS := http.FS(web.StaticFS)
	fileServer := http.FileServer(staticFS)

	s.engine.GET("/static/*filepath", func(c *gin.Context) {
		c.Request.URL.Path = "/static/" + c.Param("filepath")
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}

// RegisterPageRoutes 注册前端页面路由
func (s *Server) RegisterPageRoutes() {
	// 加载 HTML 模板
	tmpl, err := template.ParseFS(web.TemplateFS, "templates/*.html")
	if err != nil {
		s.logger.Error("failed to parse templates", "error", err)
		return
	}
	s.engine.SetHTMLTemplate(tmpl)

	// 页面路由映射
	// GET / -> 重定向到 /dashboard
	s.engine.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "/dashboard")
	})

	// GET /login -> 渲染 login.html
	s.engine.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"version": version,
		})
	})

	// GET /setup -> 渲染 setup.html
	s.engine.GET("/setup", func(c *gin.Context) {
		c.HTML(http.StatusOK, "setup.html", gin.H{
			"version": version,
		})
	})

	// GET /dashboard -> 渲染 dashboard.html
	s.engine.GET("/dashboard", func(c *gin.Context) {
		c.HTML(http.StatusOK, "dashboard.html", gin.H{
			"version": version,
		})
	})

	// GET /sources -> 渲染 sources.html
	s.engine.GET("/sources", func(c *gin.Context) {
		c.HTML(http.StatusOK, "sources.html", gin.H{
			"version": version,
		})
	})

	// GET /sources/:id -> 渲染 source_detail.html
	s.engine.GET("/sources/:id", func(c *gin.Context) {
		c.HTML(http.StatusOK, "source_detail.html", gin.H{
			"version": version,
		})
	})

	// GET /data -> 渲染 data.html
	s.engine.GET("/data", func(c *gin.Context) {
		c.HTML(http.StatusOK, "data.html", gin.H{
			"version": version,
		})
	})

	// GET /settings -> 渲染 settings.html
	s.engine.GET("/settings", func(c *gin.Context) {
		c.HTML(http.StatusOK, "settings.html", gin.H{
			"version": version,
		})
	})
}

// version 变量需要在 routes.go 中定义
var version string

// SetVersion 设置版本号（由 main.go 调用）
func SetVersion(v string) {
	version = v
}
