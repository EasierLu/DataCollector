package api

import (
	"github.com/datacollector/datacollector/internal/auth"
	"github.com/datacollector/datacollector/internal/collector"
	"github.com/datacollector/datacollector/internal/config"
	"github.com/datacollector/datacollector/internal/middleware"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有 API 路由
// 这个函数由 server 包调用，在这里集中注册所有路由
func RegisterRoutes(
	r *gin.Engine,
	store storage.DataStore,
	cfg *config.Config,
	jwtManager *auth.JWTManager,
	processor *collector.Processor,
	rateLimiter *middleware.RateLimiter,
) {
	// 创建处理器实例
	authHandler := NewAuthHandler(store, jwtManager)
	dashboardHandler := NewDashboardHandler(store)
	sourceHandler := NewSourceHandler(store)
	tokenHandler := NewTokenHandler(store)
	dataHandler := NewDataHandler(store)
	exportHandler := NewExportHandler(store)
	healthHandler := NewHealthHandler(store, "1.0.0")
	setupHandler := NewSetupHandler(store, cfg, jwtManager)
	collectorHandler := NewCollectorHandler(store, processor)

	// API v1 路由组
	apiV1 := r.Group("/api/v1")
	{
		// 健康检查 - 无认证
		apiV1.GET("/health", healthHandler.HealthCheck)

		// 初始化相关路由 - 无认证
		setup := apiV1.Group("/setup")
		{
			setup.GET("/status", setupHandler.CheckStatus)
			setup.POST("/test-db", setupHandler.TestDatabase)
			setup.POST("/init", setupHandler.Initialize)
		}

		// 数据采集路由 - 使用 Data Token 认证（由 collector 中间件处理）
		// 应用 IP 限流和 Token 限流中间件
		collect := apiV1.Group("/collect")
		collect.Use(rateLimiter.IPRateLimitMiddleware(cfg.Collector.RateLimitPerIP))
		collect.Use(rateLimiter.TokenRateLimitMiddleware(cfg.Collector.RateLimitPerToken))
		{
			collect.POST("/:source_id", collectorHandler.CollectData)
			collect.POST("/:source_id/batch", collectorHandler.CollectBatchData)
		}

		// 管理后台路由
		admin := apiV1.Group("/admin")
		{
			// 登录接口 - 不需要 JWT 认证
			admin.POST("/login", authHandler.Login)

			// 需要 JWT 认证的路由
			adminAuth := admin.Group("")
			adminAuth.Use(auth.JWTAuthMiddleware(jwtManager))
			{
				// Token 刷新
				adminAuth.POST("/refresh-token", authHandler.RefreshToken)

				// 仪表盘
				adminAuth.GET("/dashboard", dashboardHandler.GetDashboard)

				// 数据源管理
				sources := adminAuth.Group("/sources")
				{
					sources.GET("", sourceHandler.ListSources)
					sources.POST("", sourceHandler.CreateSource)
					sources.PUT("/:id", sourceHandler.UpdateSource)
					sources.DELETE("/:id", sourceHandler.DeleteSource)

					// Token 管理（嵌套在数据源下）
					sources.POST("/:id/tokens", tokenHandler.CreateToken)
					sources.GET("/:id/tokens", tokenHandler.ListTokens)
				}

				// Token 管理（独立路由）
				tokens := adminAuth.Group("/tokens")
				{
					tokens.PUT("/:id/status", tokenHandler.UpdateTokenStatus)
					tokens.DELETE("/:id", tokenHandler.DeleteToken)
				}

				// 数据管理
				data := adminAuth.Group("/data")
				{
					data.GET("", dataHandler.QueryData)
					data.DELETE("/:id", dataHandler.DeleteRecord)
					data.POST("/batch-delete", dataHandler.BatchDeleteRecords)
				}

				// 数据导出
				adminAuth.GET("/data/export", exportHandler.ExportData)
			}
		}

		// 重新初始化路由 - 需要 JWT 认证 + admin 角色
		reinit := apiV1.Group("/setup/reinit")
		reinit.Use(auth.JWTAuthMiddleware(jwtManager))
		reinit.Use(auth.RequireRole("admin"))
		{
			reinit.POST("", setupHandler.Reinitialize)
		}
	}
}
