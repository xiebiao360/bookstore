package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/xiebiao/bookstore/services/api-gateway/internal/client"
	"github.com/xiebiao/bookstore/services/api-gateway/internal/config"
	"github.com/xiebiao/bookstore/services/api-gateway/internal/handler"
	"github.com/xiebiao/bookstore/services/api-gateway/internal/middleware"
)

// main API Gateway启动入口
//
// 教学要点：
// 1. Gateway作为HTTP入口，转发请求到后端gRPC服务
// 2. 依赖注入：配置 → gRPC客户端 → Handler → 路由
// 3. 优雅关闭：捕获信号，关闭连接
//
// 架构层次：
// HTTP请求 → Gin Router → Middleware → Handler → gRPC Client → Backend Service
func main() {
	// 步骤1: 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	fmt.Printf("🚀 启动 %s v%s\n", cfg.Server.Name, cfg.Server.Version)

	// 步骤2: 初始化gRPC客户端
	userClient, err := client.NewUserClient(cfg.GRPC.UserService)
	if err != nil {
		log.Fatalf("❌ 初始化user-service客户端失败: %v", err)
	}
	defer userClient.Close()
	fmt.Println("✓ user-service客户端连接成功")

	// 后续添加其他服务客户端：
	// catalogClient, _ := client.NewCatalogClient(cfg.GRPC.CatalogService)
	// orderClient, _ := client.NewOrderClient(cfg.GRPC.OrderService)

	// 步骤3: 初始化Handler
	userHandler := handler.NewUserHandler(userClient)

	// 步骤4: 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	// 步骤5: 创建Gin引擎
	router := gin.New()

	// 步骤6: 注册全局中间件
	// 教学说明：
	// 中间件执行顺序：Logger → Recovery → CORS → 路由匹配 → Auth（如果有） → Handler
	router.Use(middleware.Logger())       // 请求日志
	router.Use(gin.Recovery())            // Panic恢复
	router.Use(middleware.CORS(cfg.CORS)) // 跨域处理

	// 步骤7: 注册路由
	// 教学重点：
	// 1. 公开路由（不需要鉴权）
	// 2. 受保护路由（需要Auth中间件鉴权）
	setupRoutes(router, userHandler, userClient)

	// 步骤8: 创建HTTP服务器
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 步骤9: 启动HTTP服务器（goroutine）
	go func() {
		fmt.Printf("🚀 API Gateway启动成功: http://localhost:%d\n", cfg.Server.HTTPPort)
		fmt.Println("\n📖 API端点：")
		fmt.Println("  POST /api/v1/auth/register   - 用户注册")
		fmt.Println("  POST /api/v1/auth/login      - 用户登录")
		fmt.Println("  POST /api/v1/auth/refresh    - 刷新Token")
		fmt.Println("  GET  /api/v1/users/:id       - 获取用户信息（需要鉴权）")
		fmt.Println("  GET  /health                 - 健康检查")
		fmt.Println()

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ HTTP服务器启动失败: %v", err)
		}
	}()

	// 步骤10: 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n⏳ 正在优雅关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("❌ 服务器强制关闭:", err)
	}

	fmt.Println("✓ HTTP服务器已关闭")
	fmt.Println("✓ gRPC客户端已关闭")
	fmt.Println("👋 服务已完全关闭")
}

// setupRoutes 设置路由
//
// 教学要点：
// 1. 路由分组：按功能模块分组（auth、users、books、orders）
// 2. 中间件应用：公开路由 vs 受保护路由
// 3. RESTful设计：统一的API风格
func setupRoutes(router *gin.Engine, userHandler *handler.UserHandler, userClient *client.UserClient) {
	// 健康检查（无需鉴权）
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "api-gateway",
			"version": "1.0.0",
		})
	})

	// API v1路由组
	v1 := router.Group("/api/v1")
	{
		// 认证路由（公开，无需鉴权）
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)    // 注册
			auth.POST("/login", userHandler.Login)          // 登录
			auth.POST("/refresh", userHandler.RefreshToken) // 刷新Token
		}

		// 用户路由（需要鉴权）
		users := v1.Group("/users")
		users.Use(middleware.Auth(userClient)) // 应用Auth中间件
		{
			users.GET("/:id", userHandler.GetUser) // 获取用户信息
		}

		// 后续添加其他路由组：
		// books := v1.Group("/books")
		// {
		//     books.GET("", bookHandler.List)       // 列表（公开）
		//     books.POST("", middleware.Auth(userClient), bookHandler.Create) // 上架（需要鉴权）
		// }
		//
		// orders := v1.Group("/orders")
		// orders.Use(middleware.Auth(userClient)) // 所有订单接口都需要鉴权
		// {
		//     orders.POST("", orderHandler.Create)
		//     orders.GET("/:id", orderHandler.GetByID)
		// }
	}
}

// =========================================
// 教学总结：API Gateway启动流程
// =========================================
//
// 1. 依赖注入顺序：
//    配置 → gRPC客户端 → Handler → Router
//    - 每一层只依赖上一层
//    - 便于单元测试（Mock依赖）
//
// 2. 中间件应用：
//    - 全局中间件：Logger、Recovery、CORS
//    - 路由组中间件：Auth（只应用于需要鉴权的路由）
//    - 执行顺序：从外到内
//
// 3. 路由设计：
//    RESTful风格：
//    - GET /api/v1/users/:id      （获取资源）
//    - POST /api/v1/users          （创建资源）
//    - PUT /api/v1/users/:id       （更新资源）
//    - DELETE /api/v1/users/:id    （删除资源）
//
// 4. 优雅关闭：
//    - 捕获SIGINT/SIGTERM信号
//    - 停止接收新请求
//    - 等待现有请求处理完成（最多10秒）
//    - 关闭gRPC连接
//
// 5. 健康检查：
//    - GET /health
//    - Kubernetes liveness/readiness probe
//    - 负载均衡器健康检查
//
// 6. 后续扩展：
//    - 添加Swagger文档（swaggo/swag）
//    - 添加限流中间件（rate limiting）
//    - 添加监控指标（Prometheus）
//    - 集成服务发现（Consul）
