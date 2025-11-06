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

	orderv1 "github.com/xiebiao/bookstore/proto/orderv1"
	"github.com/xiebiao/bookstore/services/order-service/internal/domain/order"
	"github.com/xiebiao/bookstore/services/order-service/internal/grpc/handler"
	"github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/config"
	"github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/grpc_client"
	"github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/persistence/mysql"
	redisStore "github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/persistence/redis"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 1. 加载配置
	cfg := config.Load("./config/config.yaml")
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	// 2. 初始化MySQL
	db := mysql.InitDB(&cfg.Database)
	defer mysql.Close(db)

	// 3. 初始化Redis
	redisClient := redisStore.InitRedis(&cfg.Redis)
	defer redisClient.Close()

	// 4. 初始化gRPC客户端（下游服务）
	inventoryClient, err := grpc_client.NewInventoryClient(cfg.GetServiceAddr("inventory"))
	if err != nil {
		log.Fatalf("创建inventory客户端失败: %v", err)
	}
	defer inventoryClient.Close()

	catalogClient, err := grpc_client.NewCatalogClient(cfg.GetServiceAddr("catalog"))
	if err != nil {
		log.Fatalf("创建catalog客户端失败: %v", err)
	}
	defer catalogClient.Close()

	// 5. 创建仓储和缓存
	orderRepo := mysql.NewOrderRepository(db)
	orderCache := redisStore.NewOrderCache(redisClient)

	// 6. 创建gRPC服务
	grpcServer := grpc.NewServer()
	orderService := handler.NewOrderServiceServer(
		orderRepo,
		orderCache,
		inventoryClient,
		catalogClient,
		cfg,
	)
	orderv1.RegisterOrderServiceServer(grpcServer, orderService)

	// 启用反射（便于grpcurl调试）
	reflection.Register(grpcServer)

	// 7. 启动定时任务（订单超时取消）
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go startOrderTimeoutTask(ctx, orderRepo, orderCache, inventoryClient, cfg)

	// 8. 启动gRPC服务器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
	if err != nil {
		log.Fatalf("监听端口失败: %v", err)
	}

	log.Printf("🚀 order-service 启动成功，监听端口: :%d", cfg.Server.Port)
	log.Printf("📋 订单超时时间: %d分钟", cfg.Order.PaymentTimeout)

	// 9. 优雅关闭
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC服务启动失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务...")
	grpcServer.GracefulStop()
	log.Println("✅ 服务已关闭")
}

// startOrderTimeoutTask 启动订单超时取消定时任务
//
// 教学要点：
// 1. 定时任务设计：
//   - 每分钟扫描一次Redis ZSet
//   - 查询过期的订单（score <= 当前时间）
//   - 批量取消订单并释放库存
//
// 2. 分布式锁（可选）：
//   - 多实例部署时需要分布式锁（防止重复处理）
//   - 使用Redis SETNX实现
//   - Phase 2简化为单实例
//
// 3. 容错处理：
//   - 单个订单取消失败不影响其他订单
//   - 失败的订单下次继续处理
func startOrderTimeoutTask(
	ctx context.Context,
	repo order.Repository,
	cache redisStore.OrderCache,
	inventoryClient *grpc_client.InventoryClient,
	cfg *config.Config,
) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Println("📅 订单超时取消任务已启动")

	for {
		select {
		case <-ctx.Done():
			log.Println("订单超时任务已停止")
			return
		case <-ticker.C:
			// 执行超时检查
			expiredOrders, err := cache.GetExpiredOrders(ctx, 100)
			if err != nil {
				log.Printf("查询超时订单失败: %v", err)
				continue
			}

			if len(expiredOrders) == 0 {
				continue
			}

			log.Printf("发现%d个超时订单，开始自动取消", len(expiredOrders))

			for _, orderID := range expiredOrders {
				if err := cancelExpiredOrder(ctx, orderID, repo, cache, inventoryClient, cfg); err != nil {
					log.Printf("取消订单失败 (order_id=%d): %v", orderID, err)
				} else {
					log.Printf("✅ 订单已自动取消 (order_id=%d)", orderID)
				}
			}
		}
	}
}

// cancelExpiredOrder 取消超时订单
func cancelExpiredOrder(
	ctx context.Context,
	orderID uint,
	repo order.Repository,
	cache redisStore.OrderCache,
	inventoryClient *grpc_client.InventoryClient,
	cfg *config.Config,
) error {
	// 1. 查询订单
	o, err := repo.FindByID(ctx, orderID)
	if err != nil {
		return err
	}

	// 2. 检查状态（只取消待支付订单）
	if o.Status != order.OrderStatusPending {
		// 已支付或已取消，从待支付队列移除
		cache.RemovePendingOrder(ctx, orderID)
		return nil
	}

	// 3. 更新订单状态为已取消
	if err := o.UpdateStatus(order.OrderStatusCancelled); err != nil {
		return err
	}

	if err := repo.Update(ctx, o); err != nil {
		return err
	}

	// 4. 释放库存
	for _, item := range o.Items {
		_, err := inventoryClient.ReleaseStock(
			ctx,
			item.BookID,
			item.Quantity,
			o.ID,
			cfg.GetServiceTimeout("inventory"),
		)
		if err != nil {
			log.Printf("释放库存失败 (book_id=%d): %v", item.BookID, err)
		}
	}

	// 5. 从待支付队列移除
	cache.RemovePendingOrder(ctx, orderID)

	// 6. 删除缓存
	cache.DeleteOrder(ctx, orderID)

	return nil
}
