package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	userapp "github.com/xiebiao/bookstore/internal/application/user"
	userdomain "github.com/xiebiao/bookstore/internal/domain/user"
	mysqlrepo "github.com/xiebiao/bookstore/internal/infrastructure/persistence/mysql"
	redisstore "github.com/xiebiao/bookstore/internal/infrastructure/persistence/redis"
	"github.com/xiebiao/bookstore/pkg/jwt"
	pb "github.com/xiebiao/bookstore/proto/user/v1"
	"github.com/xiebiao/bookstore/services/user-service/internal/config"
	"github.com/xiebiao/bookstore/services/user-service/internal/grpc/handler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================
// 教学说明：gRPC微服务启动流程
// ============================================================
//
// Phase 1 vs Phase 2 启动流程对比：
//
// Phase 1 (HTTP服务):
// 1. 加载配置
// 2. 初始化数据库/Redis
// 3. 依赖注入（Wire）
// 4. 启动Gin服务器（HTTP端口8080）
// 5. 优雅关闭
//
// Phase 2 (gRPC服务):
// 1. 加载配置
// 2. 初始化数据库/Redis（复用Phase 1代码）
// 3. 依赖注入（手动注入）
// 4. 创建gRPC服务器
// 5. 注册gRPC服务
// 6. 启动gRPC服务器（gRPC端口9001）
// 7. 优雅关闭
//
// ============================================================

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

	// 步骤2: 初始化数据库
	db, err := initDatabase(&cfg.Database)
	if err != nil {
		log.Fatalf("❌ 数据库初始化失败: %v", err)
	}
	fmt.Println("✓ 数据库连接成功")

	// 自动迁移
	if err := db.AutoMigrate(&userdomain.User{}); err != nil {
		log.Fatalf("❌ 数据库迁移失败: %v", err)
	}
	fmt.Println("✓ 数据库表结构同步完成")

	// 步骤3: 初始化Redis
	redisClient := initRedis(&cfg.Redis)
	fmt.Println("✓ Redis连接成功")

	// 步骤4: 依赖注入
	// 教学说明：
	// Phase 1: UseCase模式
	// Repository → DomainService → UseCase → Handler

	userRepo := mysqlrepo.NewUserRepository(db)
	sessionStore := redisstore.NewSessionStore(redisClient)

	// Domain Service
	userDomainService := userdomain.NewService(userRepo)

	// JWT Manager
	jwtManager := jwt.NewManager(
		cfg.JWT.Secret,
		cfg.JWT.GetAccessTokenDuration(),
		cfg.JWT.GetRefreshTokenDuration(),
	)

	// UseCases
	registerUC := userapp.NewRegisterUseCase(userDomainService)
	loginUC := userapp.NewLoginUseCase(userDomainService, jwtManager, sessionStore)
	logoutUC := userapp.NewLogoutUseCase(sessionStore)

	// gRPC Handler
	// 教学说明：
	// Phase 2新增依赖：jwtManager、sessionStore、userDomainService
	// 用于实现ValidateToken、GetUser、RefreshToken三个方法
	userGRPCHandler := handler.NewUserServiceServer(
		registerUC,
		loginUC,
		logoutUC,
		jwtManager,        // 用于ValidateToken和RefreshToken
		sessionStore,      // 用于检查Token黑名单和会话状态
		userDomainService, // 用于GetUser直接查询用户信息
	)

	// 步骤5: 创建gRPC服务器
	grpcServer := grpc.NewServer()

	// 步骤6: 注册gRPC服务
	pb.RegisterUserServiceServer(grpcServer, userGRPCHandler)
	reflection.Register(grpcServer)
	fmt.Println("✓ gRPC服务已注册")

	// 步骤7: 启动gRPC服务器
	addr := fmt.Sprintf(":%d", cfg.Server.GRPCPort)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("❌ 监听端口失败: %v", err)
	}

	go func() {
		fmt.Printf("🚀 gRPC服务器启动成功: %s\n", addr)
		fmt.Println("\n使用以下命令测试：")
		fmt.Printf("  grpcurl -plaintext localhost:%d list\n", cfg.Server.GRPCPort)
		fmt.Printf("  grpcurl -plaintext localhost:%d user.v1.UserService/Register\n", cfg.Server.GRPCPort)
		fmt.Println()

		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("❌ gRPC服务器启动失败: %v", err)
		}
	}()

	// 步骤8: 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n⏳ 正在优雅关闭服务...")

	_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	grpcServer.GracefulStop()
	fmt.Println("✓ gRPC服务器已关闭")

	sqlDB, _ := db.DB()
	sqlDB.Close()
	fmt.Println("✓ 数据库连接已关闭")

	redisClient.Close()
	fmt.Println("✓ Redis连接已关闭")

	fmt.Println("👋 服务已完全关闭")
}

// initDatabase 初始化数据库连接
func initDatabase(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(mysql.Open(cfg.GetDSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	return db, nil
}

// initRedis 初始化Redis连接
func initRedis(cfg *config.RedisConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis连接失败: %v", err)
	}

	return client
}
