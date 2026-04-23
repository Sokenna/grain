package server

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"grain/config"
	"log/slog"
	"net/http"
)

// Server 服务器结构
type Server struct {
	engine *gin.Engine
	server *http.Server
	config *config.ServerConfig
}

// New 创建新的服务器实例
func New(cfg *config.AppConfig, router *gin.Engine) *Server {
	//设置Gin模式
	switch cfg.Server.Mode {
	case "debug":
		gin.SetMode(gin.DebugMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.ReleaseMode)
	}

	srv := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}
	return &Server{
		engine: router,
		server: srv,
		config: &cfg.Server,
	}
}

// Run 启动服务器
func (s *Server) Run() error {
	slog.Info("服务器启动",
		"address", s.server.Addr,
		"mode", s.config.Mode,
		"rate_limit_enabled", s.config.RateLimit.Enabled,
	)
	//TLS支持
	if s.config.TLS.Enabled {
		//http.ListenAndServeTLS(s.server.Addr, s.config.TLS.CertFile, s.config.TLS.KeyFile, s.engine)
		slog.Info("启用 TLS",
			"cert_file", s.config.TLS.CertFile,
			"key_file", s.config.TLS.KeyFile,
		)
		return s.server.ListenAndServeTLS(s.config.TLS.CertFile, s.config.TLS.KeyFile)
	}
	//return http.ListenAndServe(s.server.Addr, s.engine)
	return s.server.ListenAndServe()
}

// Shutdown 优雅关闭
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("服务器正在关闭...")
	return s.server.Shutdown(ctx)
}

// Engine 获取 Gin 引擎
func (s *Server) Engine() *gin.Engine {
	return s.engine
}
