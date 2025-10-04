package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github-reports/internal/api"
	"github-reports/internal/config"

	"github.com/gin-gonic/gin"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置无效: %v", err)
	}

	log.Println("配置加载成功")

	// 设置 HTTP 服务器
	router := gin.Default()

	// 创建 API 处理器
	handler := api.NewHandler(cfg)

	// 注册路由
	v1 := router.Group("/api/v1")
	{
		// 健康检查 - 无需认证
		v1.GET("/health", handler.Health)

		// Webhook - 需要认证
		v1.POST("/webhook", handler.AuthMiddleware(), handler.Webhook)
	}

	// 设置 HTTP 服务器
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// 在 goroutine 中启动服务器
	go func() {
		log.Printf("在 %s 启动 HTTP 服务器", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务器...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("服务器强制关闭: %v", err)
	}

	log.Println("服务器已退出")
}
