package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/xiebiao/bookstore/pkg/saga"
	orderv1 "github.com/xiebiao/bookstore/proto/orderv1"
	"github.com/xiebiao/bookstore/services/order-service/internal/domain/order"
	"github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/config"
	"github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/grpc_client"
	redisStore "github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/persistence/redis"
)

// OrderServiceServer gRPC服务实现
type OrderServiceServer struct {
	orderv1.UnimplementedOrderServiceServer
	repo            order.Repository
	cache           redisStore.OrderCache
	inventoryClient *grpc_client.InventoryClient
	catalogClient   *grpc_client.CatalogClient
	cfg             *config.Config
}

func NewOrderServiceServer(
	repo order.Repository,
	cache redisStore.OrderCache,
	inventoryClient *grpc_client.InventoryClient,
	catalogClient *grpc_client.CatalogClient,
	cfg *config.Config,
) *OrderServiceServer {
	return &OrderServiceServer{
		repo:            repo,
		cache:           cache,
		inventoryClient: inventoryClient,
		catalogClient:   catalogClient,
		cfg:             cfg,
	}
}

// CreateOrder 创建订单（使用Saga框架重构版）
//
// 重构说明：
// - 旧实现：手写补偿逻辑，代码分散，难以维护
// - 新实现：使用pkg/saga框架，步骤清晰，补偿自动化
//
// Saga流程：
// 1. 查询图书信息（catalog-service）
// 2. 扣减库存（inventory-service）
// 3. 创建订单（order-service）
// 4. 添加到待支付队列（Redis）
//
// 补偿流程（任一步骤失败）：
// - 如果步骤3失败：释放步骤2扣减的库存
// - 如果步骤4失败：取消步骤3创建的订单 + 释放步骤2的库存
//
// 教学要点：
// - Saga步骤拆分粒度：每个步骤应该是原子操作
// - 补偿幂等性：使用订单ID作为幂等键
// - 超时控制：整体超时30秒（可配置）
func (s *OrderServiceServer) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	// 1. 参数校验
	if err := s.validateCreateOrderRequest(req); err != nil {
		return &orderv1.CreateOrderResponse{Code: 40000, Message: err.Error()}, nil
	}

	// 2. 准备Saga上下文数据
	sagaCtx := &CreateOrderSagaContext{
		userID:          uint(req.UserId),
		items:           req.Items,
		orderItems:      make([]order.OrderItem, 0),
		deductedBookIDs: make([]uint, 0),
		total:           0,
		orderEntity:     nil,
	}

	// 3. 构建Saga流程
	orderSaga := s.buildCreateOrderSaga(sagaCtx)

	// 4. 执行Saga
	if err := orderSaga.Execute(ctx); err != nil {
		log.Printf("❌ 订单Saga执行失败: %v", err)
		return &orderv1.CreateOrderResponse{
			Code:    50000,
			Message: fmt.Sprintf("订单创建失败: %v", err),
		}, nil
	}

	// 5. 返回成功响应
	log.Printf("✅ 订单创建成功: %s", sagaCtx.orderEntity.OrderNo)
	return &orderv1.CreateOrderResponse{
		Code:    0,
		Message: "订单创建成功",
		OrderNo: sagaCtx.orderEntity.OrderNo,
		OrderId: uint64(sagaCtx.orderEntity.ID),
		Total:   sagaCtx.orderEntity.Total,
	}, nil
}

// CreateOrderSagaContext Saga执行上下文（存储中间状态）
//
// 为什么需要上下文？
// - 步骤之间需要传递数据（如订单ID、扣减的库存）
// - 补偿操作需要访问正向操作的结果
//
// 设计要点：
// - 使用结构体封装，避免全局变量
// - 字段可导出，便于测试
type CreateOrderSagaContext struct {
	userID          uint
	items           []*orderv1.OrderItem
	orderItems      []order.OrderItem // 查询图书后构建的订单明细
	deductedBookIDs []uint            // 已扣减库存的图书ID（用于补偿）
	total           int64             // 订单总金额
	orderEntity     *order.Order      // 创建的订单实体
}

// buildCreateOrderSaga 构建创建订单的Saga流程
//
// 教学要点：
// - 使用闭包捕获sagaCtx，避免全局变量
// - 每个步骤独立，便于单元测试
// - 补偿操作与正向操作对应
func (s *OrderServiceServer) buildCreateOrderSaga(sagaCtx *CreateOrderSagaContext) *saga.Saga {
	orderSaga := saga.NewSaga(30 * time.Second)

	// ==================== 步骤1：查询图书信息 ====================
	orderSaga.AddStep("查询图书信息",
		// 正向操作：查询catalog-service获取图书信息
		func(ctx context.Context) error {
			for _, item := range sagaCtx.items {
				bookResp, err := s.catalogClient.GetBook(
					ctx,
					uint(item.BookId),
					s.cfg.GetServiceTimeout("catalog"),
				)
				if err != nil || bookResp.Code != 0 {
					return fmt.Errorf("图书[%d]不存在", item.BookId)
				}

				// 构建订单明细（冗余存储图书信息）
				orderItem := order.OrderItem{
					BookID:    uint(item.BookId),
					BookTitle: bookResp.Book.Title, // 冗余存储，避免后续查询
					Quantity:  int(item.Quantity),
					Price:     bookResp.Book.Price,
				}
				sagaCtx.orderItems = append(sagaCtx.orderItems, orderItem)
				sagaCtx.total += int64(orderItem.Quantity) * orderItem.Price
			}
			return nil
		},
		// 补偿操作：查询操作无需补偿
		nil,
	)

	// ==================== 步骤2：扣减库存 ====================
	orderSaga.AddStep("扣减库存",
		// 正向操作：调用inventory-service扣减库存
		func(ctx context.Context) error {
			for _, item := range sagaCtx.items {
				resp, err := s.inventoryClient.DeductStock(
					ctx,
					uint(item.BookId),
					int(item.Quantity),
					0, // reference_id暂时为0，后续可改为订单ID
					s.cfg.GetServiceTimeout("inventory"),
				)
				if err != nil || resp.Code != 0 {
					return fmt.Errorf("库存不足[图书:%d]", item.BookId)
				}

				// 记录已扣减的图书ID（用于补偿）
				sagaCtx.deductedBookIDs = append(sagaCtx.deductedBookIDs, uint(item.BookId))
			}
			return nil
		},
		// 补偿操作：释放已扣减的库存
		//
		// 幂等性设计：
		// - inventory-service的ReleaseStock内部应实现幂等
		// - 建议使用订单ID作为幂等键
		func(ctx context.Context) error {
			for _, bookID := range sagaCtx.deductedBookIDs {
				// 查找对应的数量
				var quantity int
				for _, item := range sagaCtx.items {
					if uint(item.BookId) == bookID {
						quantity = int(item.Quantity)
						break
					}
				}

				_, err := s.inventoryClient.ReleaseStock(
					ctx,
					bookID,
					quantity,
					0, // reference_id
					s.cfg.GetServiceTimeout("inventory"),
				)
				if err != nil {
					log.Printf("⚠️ 释放库存失败[图书:%d]: %v", bookID, err)
					// 继续执行后续补偿，不中断
				}
			}
			return nil
		},
	)

	// ==================== 步骤3：创建订单 ====================
	orderSaga.AddStep("创建订单",
		// 正向操作：创建订单记录
		func(ctx context.Context) error {
			sagaCtx.orderEntity = &order.Order{
				OrderNo: order.GenerateOrderNo(),
				UserID:  sagaCtx.userID,
				Status:  order.OrderStatusPending,
				Total:   sagaCtx.total,
				Items:   sagaCtx.orderItems,
			}

			if err := s.repo.Create(ctx, sagaCtx.orderEntity); err != nil {
				return fmt.Errorf("创建订单失败: %w", err)
			}
			return nil
		},
		// 补偿操作：取消订单
		//
		// 设计选择：
		// - 方案1：删除订单记录（简单但丢失审计信息）
		// - 方案2：更新订单状态为CANCELLED（保留审计信息）✅
		func(ctx context.Context) error {
			if sagaCtx.orderEntity != nil && sagaCtx.orderEntity.ID > 0 {
				// 更新订单状态为已取消
				if err := sagaCtx.orderEntity.UpdateStatus(order.OrderStatusCancelled); err != nil {
					return err
				}
				if err := s.repo.Update(ctx, sagaCtx.orderEntity); err != nil {
					log.Printf("⚠️ 取消订单失败[订单:%s]: %v", sagaCtx.orderEntity.OrderNo, err)
				}
			}
			return nil
		},
	)

	// ==================== 步骤4：添加到待支付队列 ====================
	orderSaga.AddStep("添加到待支付队列",
		// 正向操作：将订单加入Redis ZSet（15分钟后过期）
		func(ctx context.Context) error {
			expireAt := time.Now().Add(time.Duration(s.cfg.Order.PaymentTimeout) * time.Minute)
			if err := s.cache.SetPendingOrder(ctx, sagaCtx.orderEntity.ID, expireAt); err != nil {
				return fmt.Errorf("添加到待支付队列失败: %w", err)
			}
			return nil
		},
		// 补偿操作：从待支付队列移除
		func(ctx context.Context) error {
			if sagaCtx.orderEntity != nil && sagaCtx.orderEntity.ID > 0 {
				if err := s.cache.RemovePendingOrder(ctx, sagaCtx.orderEntity.ID); err != nil {
					log.Printf("⚠️ 从待支付队列移除失败[订单:%s]: %v", sagaCtx.orderEntity.OrderNo, err)
				}
			}
			return nil
		},
	)

	return orderSaga
}

// validateCreateOrderRequest 校验创建订单请求
func (s *OrderServiceServer) validateCreateOrderRequest(req *orderv1.CreateOrderRequest) error {
	if req.UserId == 0 {
		return fmt.Errorf("用户ID不能为空")
	}
	if len(req.Items) == 0 {
		return fmt.Errorf("订单明细不能为空")
	}
	for _, item := range req.Items {
		if item.BookId == 0 {
			return fmt.Errorf("图书ID不能为空")
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("数量必须大于0")
		}
	}
	return nil
}

// ==================== DO/DON'T 对比 ====================

// ❌ DON'T: 手写补偿逻辑，代码分散
//
// 问题：
// 1. 补偿逻辑与业务逻辑混在一起，难以维护
// 2. 容易遗漏补偿步骤
// 3. 难以测试（无法Mock单个步骤）
//
// func (s *OrderServiceServer) CreateOrder_Old(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
//     // 查询图书
//     for _, item := range req.Items {
//         book, err := s.catalogClient.GetBook(ctx, item.BookId)
//         if err != nil {
//             return nil, err
//         }
//     }
//
//     // 扣减库存
//     var deductedBooks []uint
//     for _, item := range req.Items {
//         if err := s.inventoryClient.DeductStock(ctx, item.BookId, item.Quantity); err != nil {
//             // ⚠️ 手写补偿逻辑
//             for _, bookID := range deductedBooks {
//                 s.inventoryClient.ReleaseStock(ctx, bookID, 1)
//             }
//             return nil, err
//         }
//         deductedBooks = append(deductedBooks, item.BookId)
//     }
//
//     // 创建订单
//     order := &Order{...}
//     if err := s.repo.Create(ctx, order); err != nil {
//         // ⚠️ 再次手写补偿逻辑（代码重复）
//         for _, bookID := range deductedBooks {
//             s.inventoryClient.ReleaseStock(ctx, bookID, 1)
//         }
//         return nil, err
//     }
//
//     return &CreateOrderResponse{OrderNo: order.OrderNo}, nil
// }

// ✅ DO: 使用Saga框架，步骤清晰
//
// 优点：
// 1. 步骤定义集中，易于理解和维护
// 2. 补偿逻辑与正向操作配对，不易遗漏
// 3. 每个步骤可独立测试
// 4. 支持超时控制、故障恢复等高级特性
//
// func (s *OrderServiceServer) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*CreateOrderResponse, error) {
//     saga := saga.NewSaga(30 * time.Second)
//
//     saga.AddStep("查询图书", queryBooks, nil)
//     saga.AddStep("扣减库存", deductStock, releaseStock)
//     saga.AddStep("创建订单", createOrder, cancelOrder)
//
//     if err := saga.Execute(ctx); err != nil {
//         return nil, err
//     }
//
//     return &CreateOrderResponse{OrderNo: order.OrderNo}, nil
// }

// ==================== 其他gRPC方法保持不变 ====================

func (s *OrderServiceServer) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	if req.OrderId == 0 {
		return &orderv1.GetOrderResponse{Code: 40000, Message: "订单ID不能为空"}, nil
	}

	orderEntity, err := s.repo.FindByID(ctx, uint(req.OrderId))
	if err != nil {
		return &orderv1.GetOrderResponse{Code: 40400, Message: "订单不存在"}, nil
	}

	items := make([]*orderv1.OrderItemDetail, 0, len(orderEntity.Items))
	for _, item := range orderEntity.Items {
		items = append(items, &orderv1.OrderItemDetail{
			Id:        uint64(item.ID),
			OrderId:   uint64(item.OrderID),
			BookId:    uint64(item.BookID),
			BookTitle: item.BookTitle,
			Quantity:  int32(item.Quantity),
			Price:     item.Price,
		})
	}

	return &orderv1.GetOrderResponse{
		Code: 0,
		Order: &orderv1.Order{
			Id:      uint64(orderEntity.ID),
			OrderNo: orderEntity.OrderNo,
			UserId:  uint64(orderEntity.UserID),
			Total:   orderEntity.Total,
			Status:  int32(orderEntity.Status),
			Items:   items,
		},
	}, nil
}
