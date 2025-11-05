package main

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"

	appbook "github.com/xiebiao/bookstore/internal/application/book"
	appuser "github.com/xiebiao/bookstore/internal/application/user"
	"github.com/xiebiao/bookstore/internal/domain/book"
	"github.com/xiebiao/bookstore/internal/domain/user"
	"github.com/xiebiao/bookstore/internal/infrastructure/config"
	"github.com/xiebiao/bookstore/internal/infrastructure/persistence/mysql"
	"github.com/xiebiao/bookstore/internal/infrastructure/persistence/redis"
	"github.com/xiebiao/bookstore/internal/interface/http/handler"
	"github.com/xiebiao/bookstore/internal/interface/http/middleware"
	"github.com/xiebiao/bookstore/pkg/jwt"
	"github.com/xiebiao/bookstore/pkg/response"
)

// main 主程序入口
// 当前版本：Phase 1 - Week 2 Day 8-9 - 图书上架功能
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

	// 3. 初始化Redis连接
	redisClient, err := redis.NewClient(cfg)
	if err != nil {
		log.Fatalf("初始化Redis失败: %v", err)
	}

	// 4. 依赖注入（手动组装）
	// 学习要点：依赖注入链
	// Repository ← Service ← UseCase ← Handler

	// 基础设施层
	userRepo := mysql.NewUserRepository(db)
	bookRepo := mysql.NewBookRepository(db) // 图书仓储
	sessionStore := redis.NewSessionStore(redisClient)
	jwtManager := jwt.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpire,
		cfg.JWT.RefreshTokenExpire,
	)

	// 领域层
	userService := user.NewService(userRepo)
	bookService := book.NewService(bookRepo) // 图书领域服务

	// 应用层
	registerUseCase := appuser.NewRegisterUseCase(userService)
	loginUseCase := appuser.NewLoginUseCase(userService, jwtManager, sessionStore)
	publishBookUseCase := appbook.NewPublishBookUseCase(bookService) // 图书上架用例

	// 接口层
	userHandler := handler.NewUserHandler(registerUseCase, loginUseCase)
	bookHandler := handler.NewBookHandler(publishBookUseCase) // 图书处理器
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, sessionStore)

	// 5. 初始化Gin引擎
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// 6. 注册路由
	registerRoutes(r, userHandler, bookHandler, authMiddleware)

	// 7. 启动服务
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	fmt.Printf("\n🚀 服务启动成功！\n")
	fmt.Printf("   访问地址: http://localhost%s\n", addr)
	fmt.Printf("   健康检查: http://localhost%s/ping\n", addr)
	fmt.Printf("   用户注册: POST http://localhost%s/api/v1/users/register\n", addr)
	fmt.Printf("   用户登录: POST http://localhost%s/api/v1/users/login\n", addr)
	fmt.Printf("   图书上架: POST http://localhost%s/api/v1/books (需要登录)\n", addr)
	fmt.Printf("\n按Ctrl+C停止服务\n\n")

	if err := r.Run(addr); err != nil {
		log.Fatalf("启动服务失败: %v", err)
	}
}

// registerRoutes 注册路由
func registerRoutes(r *gin.Engine, userHandler *handler.UserHandler, bookHandler *handler.BookHandler, authMiddleware *middleware.AuthMiddleware) {
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
		// 用户模块（公开接口，不需要登录）
		users := v1.Group("/users")
		{
			users.POST("/register", userHandler.Register) // ✅ 注册
			users.POST("/login", userHandler.Login)       // ✅ 登录
		}

		// 需要认证的路由（示例）
		authorized := v1.Group("")
		authorized.Use(authMiddleware.RequireAuth()) // 应用认证中间件
		{
			// 用户个人信息（需要登录）
			authorized.GET("/profile", func(c *gin.Context) {
				// 从Context获取当前登录用户信息
				userID := middleware.GetUserID(c)
				email := middleware.GetEmail(c)

				response.Success(c, gin.H{
					"user_id": userID,
					"email":   email,
					"message": "这是需要登录才能访问的接口",
				})
			})
		}

		// 图书模块
		books := v1.Group("/books")
		{
			// 查询图书列表(公开接口,不需要登录)
			books.GET("", func(c *gin.Context) {
				response.ErrorWithCode(c, 50000, "图书列表功能正在开发中(Week 2 Day 10-11)...")
			})

			// 上架图书(需要登录)
			books.POST("", authMiddleware.RequireAuth(), bookHandler.PublishBook) // ✅ 图书上架
		}

		// 订单模块（后续实现）
		orders := v1.Group("/orders")
		orders.Use(authMiddleware.RequireAuth()) // 订单相关都需要登录
		{
			orders.POST("", func(c *gin.Context) {
				response.ErrorWithCode(c, 50000, "订单创建功能正在开发中...")
			})
		}
	}
}
