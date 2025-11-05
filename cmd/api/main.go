package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/xiebiao/bookstore/internal/infrastructure/config"
	"github.com/xiebiao/bookstore/pkg/response"
)

// main 主程序入口
// 当前版本：Phase 1脚手架验证版本
// 说明：验证配置加载、Web框架、Docker环境是否正常
func main() {
	// 1. 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	fmt.Printf("✓ 配置加载成功\n")
	fmt.Printf("  - 服务端口: %d\n", cfg.Server.Port)
	fmt.Printf("  - 运行模式: %s\n", cfg.Server.Mode)
	fmt.Printf("  - 数据库: %s:%d/%s\n", cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	fmt.Printf("  - Redis: %s\n", cfg.Redis.Addr())

	// 2. 初始化Gin引擎
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 3. 注册路由
	registerRoutes(r)

	// 4. 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("\n🚀 服务启动成功！\n")
	fmt.Printf("   访问地址: http://localhost%s\n", addr)
	fmt.Printf("   健康检查: http://localhost%s/ping\n", addr)
	fmt.Printf("\n按Ctrl+C停止服务\n\n")

	if err := r.Run(addr); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}

// registerRoutes 注册路由
func registerRoutes(r *gin.Engine) {
	// 健康检查
	r.GET("/ping", func(c *gin.Context) {
		response.Success(c, gin.H{
			"message": "pong",
			"status":  "healthy",
		})
	})

	// API路由组
	v1 := r.Group("/api/v1")
	{
		// 用户模块（后续实现）
		users := v1.Group("/users")
		{
			users.POST("/register", func(c *gin.Context) {
				response.ErrorWithCode(c, 50000, "用户注册功能正在开发中...")
			})
			users.POST("/login", func(c *gin.Context) {
				response.ErrorWithCode(c, 50000, "用户登录功能正在开发中...")
			})
		}

		// 图书模块（后续实现）
		books := v1.Group("/books")
		{
			books.GET("", func(c *gin.Context) {
				response.ErrorWithCode(c, 50000, "图书列表功能正在开发中...")
			})
			books.POST("", func(c *gin.Context) {
				response.ErrorWithCode(c, 50000, "图书上架功能正在开发中...")
			})
		}

		// 订单模块（后续实现）
		orders := v1.Group("/orders")
		{
			orders.POST("", func(c *gin.Context) {
				response.ErrorWithCode(c, 50000, "订单创建功能正在开发中...")
			})
		}
	}
}
