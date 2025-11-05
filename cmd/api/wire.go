//go:build wireinject
// +build wireinject

// Wire依赖注入配置文件
//
// 教学说明：
// 1. Wire是Google开发的编译期依赖注入工具
// 2. 与运行时反射注入（如Spring的@Autowired）不同，Wire在编译期生成代码
// 3. 优势：零运行时开销、类型安全、编译期检测循环依赖
//
// Wire工作流程：
// Step 1: 编写wire.go（本文件），定义Providers和Injector
// Step 2: 运行 `wire gen ./cmd/api`
// Step 3: Wire生成wire_gen.go，包含完整的依赖创建代码
// Step 4: main.go调用wire_gen.go中的InitializeApp()
//
// 核心概念：
// - Provider: 提供依赖的构造函数（如NewUserRepository）
// - Injector: 声明最终要构造的目标类型（如*gin.Engine）
// - wire.Build(): 告诉Wire如何组装依赖链

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	goredis "github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	appbook "github.com/xiebiao/bookstore/internal/application/book"
	apporder "github.com/xiebiao/bookstore/internal/application/order"
	appuser "github.com/xiebiao/bookstore/internal/application/user"
	"github.com/xiebiao/bookstore/internal/domain/book"
	"github.com/xiebiao/bookstore/internal/domain/user"
	"github.com/xiebiao/bookstore/internal/infrastructure/config"
	"github.com/xiebiao/bookstore/internal/infrastructure/persistence/mysql"
	"github.com/xiebiao/bookstore/internal/infrastructure/persistence/redis"
	"github.com/xiebiao/bookstore/internal/interface/http/handler"
	"github.com/xiebiao/bookstore/internal/interface/http/middleware"
	"github.com/xiebiao/bookstore/pkg/jwt"
)

// ========================================
// Wire Provider Sets (依赖分组)
// ========================================
// 教学说明：
// ProviderSet 将相关的 Provider 分组，便于管理和复用
// 例如：基础设施层的所有Provider放在一起

// infrastructureSet 基础设施层依赖
// 包含：配置加载、数据库连接、Redis连接
var infrastructureSet = wire.NewSet(
	config.Load,     // 加载配置文件
	mysql.NewDB,     // 创建MySQL连接
	redis.NewClient, // 创建Redis连接
)

// repositorySet 仓储层依赖
// 包含：所有Repository的构造函数
var repositorySet = wire.NewSet(
	mysql.NewUserRepository,  // 用户仓储
	mysql.NewBookRepository,  // 图书仓储
	mysql.NewOrderRepository, // 订单仓储
	mysql.NewTxManager,       // 事务管理器
)

// domainSet 领域层依赖
// 包含：所有领域服务的构造函数
var domainSet = wire.NewSet(
	user.NewService, // 用户领域服务
	book.NewService, // 图书领域服务
)

// applicationSet 应用层依赖
// 包含：所有Use Case的构造函数
var applicationSet = wire.NewSet(
	appuser.NewRegisterUseCase,     // 用户注册用例
	appuser.NewLoginUseCase,        // 用户登录用例
	appbook.NewPublishBookUseCase,  // 图书上架用例
	appbook.NewListBooksUseCase,    // 图书列表用例
	apporder.NewCreateOrderUseCase, // 创建订单用例
)

// middlewareSet 中间件依赖
// 包含：JWT管理器、认证中间件
var middlewareSet = wire.NewSet(
	provideJWTManager,            // JWT管理器（需要从config提取参数）
	provideSessionStore,          // Session存储（需要从Redis创建）
	middleware.NewAuthMiddleware, // 认证中间件
)

// handlerSet HTTP处理器依赖
// 包含：所有Handler的构造函数
var handlerSet = wire.NewSet(
	handler.NewUserHandler,  // 用户处理器
	handler.NewBookHandler,  // 图书处理器
	handler.NewOrderHandler, // 订单处理器
)

// ========================================
// Custom Providers (自定义Provider)
// ========================================
// 教学说明：
// 有些依赖的构造函数参数不是直接的类型，需要从Config中提取
// 这时需要编写自定义Provider函数

// provideJWTManager 从配置创建JWT管理器
// 教学要点：config.Config 包含多个字段，但jwt.NewManager只需要JWT相关的配置
// Wire无法自动知道如何从Config提取参数，所以需要手动编写Provider
func provideJWTManager(cfg *config.Config) *jwt.Manager {
	return jwt.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpire,
		cfg.JWT.RefreshTokenExpire,
	)
}

// provideSessionStore 从Redis客户端创建Session存储
// 教学要点：redis.NewSessionStore需要*goredis.Client参数
// Wire会自动注入redis.NewClient()的返回值
func provideSessionStore(client *goredis.Client) *redis.SessionStore {
	return redis.NewSessionStore(client)
}

// provideGinEngine 创建并配置Gin引擎
// 教学要点：
// 1. Gin引擎需要注册所有路由
// 2. 路由注册需要所有的Handler和Middleware
// 3. Wire会自动注入这些依赖
// 4. 这里直接在函数内注册路由，避免与main.go中的registerRoutes函数冲突
func provideGinEngine(
	cfg *config.Config,
	userHandler *handler.UserHandler,
	bookHandler *handler.BookHandler,
	orderHandler *handler.OrderHandler,
	authMiddleware *middleware.AuthMiddleware,
) *gin.Engine {
	// 设置运行模式
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 注册路由
	// 健康检查
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
			"status":  "healthy",
		})
	})

	// Swagger文档路由
	// 教学说明：
	// - ginSwagger.WrapHandler: Swagger UI的HTTP处理器
	// - swaggerFiles.Handler: 提供swagger.json等静态文件
	// - 访问 http://localhost:8080/swagger/index.html 查看API文档
	// - 生产环境建议禁用Swagger或添加访问控制
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API路由组
	v1 := r.Group("/api/v1")
	{
		// 用户模块（公开接口）
		users := v1.Group("/users")
		{
			users.POST("/register", userHandler.Register)
			users.POST("/login", userHandler.Login)
		}

		// 需要认证的路由
		authorized := v1.Group("")
		authorized.Use(authMiddleware.RequireAuth())
		{
			// 个人信息
			authorized.GET("/profile", func(c *gin.Context) {
				userID := middleware.GetUserID(c)
				email := middleware.GetEmail(c)
				c.JSON(200, gin.H{
					"user_id": userID,
					"email":   email,
					"message": "这是需要登录才能访问的接口",
				})
			})
		}

		// 图书模块
		books := v1.Group("/books")
		{
			// 公开接口
			books.GET("", bookHandler.ListBooks)

			// 需要登录
			books.POST("", authMiddleware.RequireAuth(), bookHandler.PublishBook)
		}

		// 订单模块（需要登录）
		orders := v1.Group("/orders")
		orders.Use(authMiddleware.RequireAuth())
		{
			orders.POST("", orderHandler.CreateOrder)
		}
	}

	return r
}

// ========================================
// Wire Injector (依赖注入器)
// ========================================
// 教学说明：
// InitializeApp是Wire的入口函数（Injector）
//
// wire.Build() 告诉Wire需要哪些Provider来构建*gin.Engine
// Wire会自动分析依赖关系：
//
// 依赖链示例：
// *gin.Engine 需要 → *handler.UserHandler
// *handler.UserHandler 需要 → *appuser.RegisterUseCase
// *appuser.RegisterUseCase 需要 → *user.Service
// *user.Service 需要 → user.Repository
// user.Repository 需要 → *gorm.DB
// *gorm.DB 需要 → *config.Config
//
// Wire会按正确的顺序调用所有构造函数

// InitializeApp 初始化整个应用
// 返回：配置好的Gin引擎
// 错误：如果任何依赖创建失败
//
// 教学说明：
// Wire Injector函数的返回值有限制：
// - 第一个返回值：要构造的目标类型（*gin.Engine）
// - 第二个返回值（可选）：只能是error或cleanup函数
// - 不能返回多个业务对象，如果需要Config可以在provideGinEngine中处理
func InitializeApp() (*gin.Engine, error) {
	// wire.Build 的参数是所有的 Provider
	// Wire会在编译期分析依赖关系，生成初始化代码
	wire.Build(
		// 基础设施层
		infrastructureSet,

		// 仓储层
		repositorySet,

		// 领域层
		domainSet,

		// 应用层
		applicationSet,

		// 中间件层
		middlewareSet,

		// 接口层
		handlerSet,

		// Gin引擎
		provideGinEngine,
	)

	// 返回值类型必须与wire.Build的最终产出一致
	// Wire会在wire_gen.go中生成实际的初始化代码
	// 这里的返回值是占位符，实际运行时会被wire_gen.go替代
	return nil, nil
}
