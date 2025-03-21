package main

import (
	"context"
	"fmt"
	"github.com/ninesun/ossmanager-backend/pkg/api"
	"github.com/ninesun/ossmanager-backend/pkg/config"
	"github.com/ninesun/ossmanager-backend/pkg/db"
	"github.com/ninesun/ossmanager-backend/pkg/function"
	"github.com/ninesun/ossmanager-backend/pkg/logger"
	"github.com/ninesun/ossmanager-backend/pkg/oss/factory"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("configs")
	if err != nil {
		panic(fmt.Sprintf("加载配置失败: %v", err))
	}

	// 初始化日志系统
	if err := logger.Init(&cfg.Log); err != nil {
		panic(fmt.Sprintf("初始化日志系统失败: %v", err))
	}
	defer logger.Sync()

	logger.Info("OSS管理系统后端服务启动中...")
	logger.Info("配置加载成功", zap.String("env", cfg.App.Env))

	// 初始化数据库
	if err := db.Init(&cfg.Database); err != nil {
		logger.Fatal("初始化数据库失败", zap.Error(err))
	}
	logger.Info("数据库初始化成功")

	// 创建存储服务工厂
	storageFactory := factory.NewStorageFactory(&cfg.OSS)

	// 创建MD5计算器
	md5Calculator := function.NewMD5Calculator(storageFactory, cfg.App.Workers)
	logger.Info("MD5计算器初始化成功", zap.Int("workers", cfg.App.Workers))

	// 设置路由
	router := api.SetupRouter(storageFactory, md5Calculator)

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.App.Host, cfg.App.Port),
		Handler: router,
	}

	// 优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 启动HTTP服务器
	go func() {
		logger.Info("HTTP服务器启动成功", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP服务器启动失败", zap.Error(err))
		}
	}()

	// 等待退出信号
	<-quit
	logger.Info("正在关闭服务器...")

	// 关闭MD5计算器
	md5Calculator.Stop()
	logger.Info("MD5计算器已关闭")

	// 设置关闭超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("服务器关闭异常", zap.Error(err))
	}

	// 关闭数据库连接
	if err := db.Close(); err != nil {
		logger.Error("关闭数据库连接失败", zap.Error(err))
	}

	logger.Info("服务器已安全关闭")
} 