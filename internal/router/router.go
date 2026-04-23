package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"grain/config"
	"grain/internal/auth"
	"grain/internal/handler"
	"grain/internal/middleware"
	"grain/internal/repository"
	"grain/internal/server"
	"grain/internal/service"
	"time"
)

// Router 路由管理器
type Router struct {
	engine           *gin.Engine
	db               *gorm.DB
	jwtManager       *auth.JWTManager
	rateLimitManager *server.RateLimiterManager
}

func NewRouter(db *gorm.DB, jwtManager *auth.JWTManager, rateLimitManager *server.RateLimiterManager) *Router {
	// 创建 Gin 引擎
	engine := gin.New()

	return &Router{
		engine:           engine,
		db:               db,
		jwtManager:       jwtManager,
		rateLimitManager: rateLimitManager,
	}
}

func (r *Router) Setup() *gin.Engine {
	//创建路由引擎
	// 全局中间件
	r.registerGlobalMiddlewares()

	// 2. 注册健康检查路由（无需认证）
	r.registerHealthRoutes()

	// 3. 注册 API 路由
	r.registerAPIRoutes()
	return r.engine
}

func (r *Router) registerGlobalMiddlewares() {
	//r.engine.Use(middleware.RecoveryMiddleware())
	//恢复中间件（必须第一个）
	r.engine.Use(gin.Recovery())

	//日志中间件
	r.engine.Use(middleware.LoggerMiddleware())

	//请求ID中间件
	r.engine.Use(middleware.RequestIDMiddleware())

	//CORS中间件
	r.engine.Use(middleware.CorsMiddleware())

	//限流中间件（如果启用）

	if !config.Config.Server.RateLimit.Enabled {
		r.engine.Use(server.RateLimitMiddleware(r.rateLimitManager))
	}

	// 超时中间件
	r.engine.Use(middleware.TimeoutMiddleware(config.Config.Server.ReadTimeout))
}

func (r *Router) registerHealthRoutes() {
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "grain-api",
			"time": gin.H{
				"now": time.Now().Unix(),
			},
		})
	})
	r.engine.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
}

func (r *Router) registerAPIRoutes() {
	userRepo := repository.NewUserRepository(r.db)
	userService := service.NewUserService(userRepo, r.jwtManager)
	userHandler := handler.NewUserHandler(userService)
	// API v1 路由组
	api := r.engine.Group("/api/v1")
	// 公开路由组（无需认证）
	r.registerPublicRoutes(api, userHandler)
}

// registerPublicRoutes 注册公开路由
func (r *Router) registerPublicRoutes(api *gin.RouterGroup, userHandler *handler.UserHandler) {
	public := api.Group("/auth")

	{
		public.POST("/login", userHandler.Login)
		// TODO: public.POST("/register", userHandler.Register)
		// TODO: public.POST("/refresh", userHandler.RefreshToken)
		// TODO: public.POST("/forgot-password", userHandler.ForgotPassword)
		// TODO: public.POST("/reset-password", userHandler.ResetPassword)
	}
}

// registerAuthRoutes 注册需要认证的路由
func (r *Router) registerAuthRoutes(api *gin.RouterGroup, userHandler *handler.UserHandler) {
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(r.jwtManager))
	{
		// 用户相关
		user := protected.Group("/user")
		{
			user.GET("/info", userHandler.GetUserInfo)
			// TODO: user.PUT("/info", userHandler.UpdateUserInfo)
			// TODO: user.POST("/change-password", userHandler.ChangePassword)
			// TODO: user.GET("/profile", userHandler.GetProfile)
		}

		// 通用业务路由（所有登录用户可访问）
		business := protected.Group("/business")
		{
			// TODO: business.GET("/dashboard", dashboardHandler.GetStats)
			// TODO: business.GET("/notifications", notificationHandler.GetList)
			business.GET("")
		}
	}
}

// registerAdminRoutes 注册管理员路由
func (r *Router) registerAdminRoutes(api *gin.RouterGroup, userHandler *handler.UserHandler) {
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(r.jwtManager))
	admin.Use(middleware.RBACMiddleware("admin"))
	{
		// 用户管理
		admin.GET("/users", userHandler.GetUserList)
		// TODO: admin.POST("/user", userHandler.CreateUser)
		// TODO: admin.PUT("/user/:id", userHandler.UpdateUser)
		// TODO: admin.DELETE("/user/:id", userHandler.DeleteUser)
		// TODO: admin.PUT("/user/:id/status", userHandler.UpdateUserStatus)

		// 系统管理
		system := admin.Group("/system")
		{
			system.GET("")
			// TODO: system.GET("/config", systemHandler.GetConfig)
			// TODO: system.PUT("/config", systemHandler.UpdateConfig)
			// TODO: system.GET("/logs", logHandler.GetList)
		}

		// 数据统计
		stats := admin.Group("/stats")
		{
			stats.GET("")
			// TODO: stats.GET("/overview", statsHandler.GetOverview)
			// TODO: stats.GET("/users", statsHandler.GetUserStats)
			// TODO: stats.GET("/requests", statsHandler.GetRequestStats)
		}
	}
}
