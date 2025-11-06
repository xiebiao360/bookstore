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

	inventoryv1 "github.com/xiebiao/bookstore/proto/inventoryv1"
	"github.com/xiebiao/bookstore/services/inventory-service/internal/grpc/handler"
	"github.com/xiebiao/bookstore/services/inventory-service/internal/infrastructure/config"
	"github.com/xiebiao/bookstore/services/inventory-service/internal/infrastructure/persistence/mysql"
	redisStore "github.com/xiebiao/bookstore/services/inventory-service/internal/infrastructure/persistence/redis"
)

// main inventory-service主程序
//
// 教学要点：
// 1. 双存储架构启动流程
//   - MySQL：持久化存储
//   - Redis：高性能缓存 + Lua脚本
//
// 2. Lua脚本预加载
//   - 启动时加载脚本到Redis
//   - 后续使用EVALSHA调用（性能优化）
//
// 3. 优雅关闭
//   - 停止接受新请求
//   - 等待现有请求完成
//   - 关闭数据库连接
func main() {
	// 步骤1：加载配置
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 步骤2：初始化MySQL连接
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

	// 步骤4：创建Redis库存存储并预加载Lua脚本
	inventoryStore := redisStore.NewInventoryStore(redisClient)

	// 教学要点：预加载Lua脚本到Redis
	// 好处：后续使用EVALSHA调用，减少网络传输
	if err := inventoryStore.LoadScripts(ctx); err != nil {
		log.Fatalf("加载Lua脚本失败: %v", err)
	}

	log.Println("✅ Lua脚本预加载成功")

	// 步骤5：创建仓储实例
	inventoryRepo := mysql.NewInventoryRepository(db)
	logRepo := mysql.NewLogRepository(db)

	// 步骤6：创建gRPC Handler
	inventoryHandler := handler.NewInventoryServiceServer(inventoryRepo, logRepo, inventoryStore)

	// 步骤7：创建gRPC服务器
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(10*1024*1024), // 10MB
		grpc.MaxSendMsgSize(10*1024*1024),
	)

	// 注册服务
	inventoryv1.RegisterInventoryServiceServer(grpcServer, inventoryHandler)

	// 注册反射服务（用于grpcurl调试）
	reflection.Register(grpcServer)

	// 步骤8：启动gRPC服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
	}

	// 在goroutine中启动服务器
	go func() {
		log.Printf("🚀 inventory-service 启动成功，监听端口: %s", addr)
		log.Printf("📊 高并发库存扣减已启用（Redis + Lua脚本）")
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("gRPC服务器启动失败: %v", err)
		}
	}()

	// 步骤9：优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("📴 收到关闭信号，开始优雅关闭...")

	// 停止gRPC服务器
	grpcServer.GracefulStop()

	log.Println("✅ inventory-service 已安全关闭")
}
