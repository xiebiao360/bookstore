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

	"github.com/go-redis/redis/v8"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	catalogv1 "github.com/xiebiao/bookstore/proto/catalogv1"
	"github.com/xiebiao/bookstore/services/catalog-service/internal/grpc/handler"
	"github.com/xiebiao/bookstore/services/catalog-service/internal/infrastructure/config"
	"github.com/xiebiao/bookstore/services/catalog-service/internal/infrastructure/persistence/mysql"
	redisStore "github.com/xiebiao/bookstore/services/catalog-service/internal/infrastructure/persistence/redis"
)

// main catalog-service主程序
//
// 教学要点：
// 1. 微服务启动流程
//
//   - 加载配置
//
//   - 初始化基础设施（数据库、Redis）
//
//   - 创建gRPC服务
//
//   - 优雅关闭
//
//     2. Phase 1 vs Phase 2 对比
//     Phase 1: HTTP服务器（Gin）
//     Phase 2: gRPC服务器
//
//     3. 依赖注入（手动实现，Week 7会引入Wire）
//     配置 → 数据库 → 仓储 → 缓存 → Handler → gRPC Server
func main() {
	// 步骤1：加载配置
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 步骤2：初始化数据库连接
	db, err := mysql.NewDB(&cfg.Database)
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	log.Println("✅ 数据库连接成功")

	// 步骤3：初始化Redis连接
	redisClient := redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	})

	defer redisClient.Close()

	// 测试Redis连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis连接失败: %v", err)
	}

	log.Println("✅ Redis连接成功")

	// 步骤4：创建仓储和缓存实例
	bookRepo := mysql.NewBookRepository(db)
	cacheStore := redisStore.NewCacheStore(
		redisClient,
		cfg.Cache.GetListTTL(),
		cfg.Cache.GetDetailTTL(),
		cfg.Cache.GetSearchTTL(),
	)

	// 步骤5：创建gRPC Handler
	catalogHandler := handler.NewCatalogServiceServer(bookRepo, cacheStore)

	// 步骤6：创建gRPC服务器
	grpcServer := grpc.NewServer(
		// 教学要点：gRPC服务器选项
		// 1. MaxRecvMsgSize：最大接收消息大小（默认4MB）
		// 2. MaxSendMsgSize：最大发送消息大小（默认无限制）
		// 3. ConnectionTimeout：连接超时
		grpc.MaxRecvMsgSize(10*1024*1024), // 10MB
		grpc.MaxSendMsgSize(10*1024*1024),
	)

	// 注册服务
	catalogv1.RegisterCatalogServiceServer(grpcServer, catalogHandler)

	// 注册反射服务（用于grpcurl调试）
	// 教学要点：
	// - 开发环境启用反射（便于调试）
	// - 生产环境可以禁用（安全性）
	reflection.Register(grpcServer)

	// 步骤7：启动gRPC服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
	}

	// 在goroutine中启动服务器
	go func() {
		log.Printf("🚀 catalog-service 启动成功，监听端口: %s", addr)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC服务器启动失败: %v", err)
		}
	}()

	// 步骤8：优雅关闭
	// 教学要点：
	// 1. 监听系统信号（SIGINT、SIGTERM）
	// 2. 收到信号后停止接受新请求
	// 3. 等待现有请求处理完成
	// 4. 关闭数据库连接
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("📴 收到关闭信号，开始优雅关闭...")

	// 停止gRPC服务器（等待现有请求完成）
	grpcServer.GracefulStop()

	log.Println("✅ catalog-service 已安全关闭")
}
