package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	appuser "github.com/xiebiao/bookstore/internal/application/user"
	"github.com/xiebiao/bookstore/internal/domain/user"
	"github.com/xiebiao/bookstore/internal/infrastructure/config"
	"github.com/xiebiao/bookstore/internal/infrastructure/persistence/mysql"
	"github.com/xiebiao/bookstore/internal/interface/http/handler"
	"github.com/xiebiao/bookstore/pkg/response"
)

// main 主程序入口
// 当前版本：Phase 1 - Week 1 - 用户注册功能
// 说明：手动依赖注入（Week 3会引入Wire自动生成）
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

	// 2. 初始化数据库连接
	db, err := mysql.NewDB(cfg)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 3. 依赖注入（手动组装）
	// 学习要点：这是经典的依赖注入模式
	// Repository ← Service ← UseCase ← Handler
	userRepo := mysql.NewUserRepository(db)
	userService := user.NewService(userRepo)
	registerUseCase := appuser.NewRegisterUseCase(userService)
	userHandler := handler.NewUserHandler(registerUseCase)

	// 4. 初始化Gin引擎
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// 5. 注册路由
	registerRoutes(r, userHandler)

	// 6. 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("\n🚀 服务启动成功！\n")
	fmt.Printf("   访问地址: http://localhost%s\n", addr)
	fmt.Printf("   健康检查: http://localhost%s/ping\n", addr)
	fmt.Printf("   用户注册: POST http://localhost%s/api/v1/users/register\n", addr)
	fmt.Printf("\n按Ctrl+C停止服务\n\n")

	if err := r.Run(addr); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}

// registerRoutes 注册路由
func registerRoutes(r *gin.Engine, userHandler *handler.UserHandler) {
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
		// 用户模块
		users := v1.Group("/users")
		{
			users.POST("/register", userHandler.Register) // ✅ 已实现
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
