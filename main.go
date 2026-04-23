/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"grain/cmd"
	"grain/config"
	"grain/internal/auth"
	"grain/internal/router"
	"grain/internal/server"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cmd.Execute()

	jwtManager := auth.NewJWTManager(&config.Config.Auth.JWT)
	// 4. 创建限流管理器
	rateLimitManager := server.NewRateLimiterManager(&config.Config.Server.RateLimit)

	routerManager := router.NewRouter(config.DB, jwtManager, rateLimitManager)
	engine := routerManager.Setup()
	// 创建服务器
	srv := server.New(&config.Config, engine)
	go func() {
		if err := srv.Run(); err != nil && err != http.ErrServerClosed {
			slog.Error("服务器启动失败", "error", err)
			os.Exit(1)
		}
	}()
	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("收到关闭信号，开始优雅关闭...")

	ctx, cancel := context.WithTimeout(context.Background(), config.Config.Server.GracefulShutdown.Timeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("服务器关闭失败", "error", err)
	}
	// 关闭数据库连接
	if sqlDB, err := config.DB.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			slog.Error("关闭数据库连接失败", "error", err)
		}
	}

	slog.Info("服务器已关闭")
}
