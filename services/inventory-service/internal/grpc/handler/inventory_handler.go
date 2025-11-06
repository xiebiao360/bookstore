package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/xiebiao/bookstore/proto/inventoryv1"
	"github.com/xiebiao/bookstore/services/inventory-service/internal/domain/inventory"
	"github.com/xiebiao/bookstore/services/inventory-service/internal/infrastructure/persistence/redis"
)

// InventoryServiceServer 库存服务gRPC实现
//
// 教学要点：
// 1. 双存储架构
//   - Redis：实时库存（高性能，TPS > 10000）
//   - MySQL：持久化存储（高可靠，用于对账）
//
// 2. 读写分离策略
//   - 读操作：优先Redis，fallback到MySQL
//   - 写操作：先Redis（快速响应），异步MySQL（持久化）
//
// 3. 数据一致性
//   - 最终一致性（Redis → MySQL定时同步）
//   - 对账机制（MySQL为准）
type InventoryServiceServer struct {
	inventoryv1.UnimplementedInventoryServiceServer
	repo       inventory.Repository    // MySQL仓储
	logRepo    inventory.LogRepository // 日志仓储
	redisStore *redis.InventoryStore   // Redis存储
}

// NewInventoryServiceServer 创建gRPC服务实例
func NewInventoryServiceServer(
	repo inventory.Repository,
	logRepo inventory.LogRepository,
	redisStore *redis.InventoryStore,
) *InventoryServiceServer {
	return &InventoryServiceServer{
		repo:       repo,
		logRepo:    logRepo,
		redisStore: redisStore,
	}
}

// GetStock 查询库存
//
// 教学要点：读操作优先Redis
// 1. 先查Redis（快速）
// 2. Redis未命中，查MySQL
// 3. 将MySQL数据写入Redis（预热）
func (s *InventoryServiceServer) GetStock(ctx context.Context, req *inventoryv1.GetStockRequest) (*inventoryv1.GetStockResponse, error) {
	bookID := uint(req.BookId)

	// 步骤1：先查Redis
	stock, err := s.redisStore.GetStock(ctx, bookID)
	if err == nil {
		return &inventoryv1.GetStockResponse{
			Code:    0,
			Message: "success",
			BookId:  req.BookId,
			Stock:   int32(stock),
		}, nil
	}

	// 步骤2：Redis查询失败，查MySQL
	inv, err := s.repo.GetByBookID(ctx, bookID)
	if err != nil {
		if errors.Is(err, inventory.ErrInventoryNotFound) {
			return &inventoryv1.GetStockResponse{
				Code:    40401,
				Message: "库存记录不存在",
				BookId:  req.BookId,
				Stock:   0,
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "查询库存失败: %v", err)
	}

	// 步骤3：将MySQL数据写入Redis（预热）
	go func() {
		_ = s.redisStore.SetStock(context.Background(), bookID, inv.Stock)
	}()

	return &inventoryv1.GetStockResponse{
		Code:    0,
		Message: "success",
		BookId:  req.BookId,
		Stock:   int32(inv.Stock),
	}, nil
}

// BatchGetStock 批量查询库存
func (s *InventoryServiceServer) BatchGetStock(ctx context.Context, req *inventoryv1.BatchGetStockRequest) (*inventoryv1.BatchGetStockResponse, error) {
	if len(req.BookIds) == 0 {
		return &inventoryv1.BatchGetStockResponse{
			Code:    0,
			Message: "success",
			Stocks:  []*inventoryv1.StockInfo{},
		}, nil
	}

	// 转换ID
	bookIDs := make([]uint, len(req.BookIds))
	for i, id := range req.BookIds {
		bookIDs[i] = uint(id)
	}

	// 批量查询Redis
	stockMap, err := s.redisStore.BatchGetStock(ctx, bookIDs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "批量查询库存失败: %v", err)
	}

	// 转换结果
	stocks := make([]*inventoryv1.StockInfo, 0, len(stockMap))
	for bookID, stock := range stockMap {
		stocks = append(stocks, &inventoryv1.StockInfo{
			BookId: uint64(bookID),
			Stock:  int32(stock),
		})
	}

	return &inventoryv1.BatchGetStockResponse{
		Code:    0,
		Message: "success",
		Stocks:  stocks,
	}, nil
}

// DeductStock 扣减库存
//
// 教学要点：
// 1. 写操作优先Redis（高性能）
// 2. Lua脚本保证原子性和幂等性
// 3. 异步同步到MySQL（持久化）
//
// 返回码：
// 0: 成功
// 1: 库存不足
// 2: 重复扣减（幂等性）
func (s *InventoryServiceServer) DeductStock(ctx context.Context, req *inventoryv1.DeductStockRequest) (*inventoryv1.DeductStockResponse, error) {
	bookID := uint(req.BookId)
	quantity := int(req.Quantity)
	orderID := uint(req.OrderId)

	// 参数验证
	if quantity <= 0 {
		return &inventoryv1.DeductStockResponse{
			Code:    40001,
			Message: "扣减数量必须大于0",
		}, nil
	}

	// 步骤1：Redis扣减（Lua脚本）
	code, err := s.redisStore.DeductStock(ctx, bookID, quantity, orderID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "扣减库存失败: %v", err)
	}

	// 步骤2：处理扣减结果
	switch code {
	case 0:
		// 库存不足
		return &inventoryv1.DeductStockResponse{
			Code:    40100,
			Message: "库存不足",
		}, nil

	case 1:
		// 扣减成功
		// 步骤3：异步同步到MySQL
		go func() {
			if err := s.repo.DeductStock(context.Background(), bookID, quantity, orderID); err != nil {
				// 同步失败记录日志（生产环境应接入告警）
				// logger.Error("sync to mysql failed", zap.Error(err))
			}
		}()

		// 查询剩余库存
		remainingStock, _ := s.redisStore.GetStock(ctx, bookID)

		return &inventoryv1.DeductStockResponse{
			Code:           0,
			Message:        "扣减成功",
			RemainingStock: int32(remainingStock),
		}, nil

	case 2:
		// 重复扣减（幂等性）
		remainingStock, _ := s.redisStore.GetStock(ctx, bookID)
		return &inventoryv1.DeductStockResponse{
			Code:           0,
			Message:        "订单已处理（幂等性）",
			RemainingStock: int32(remainingStock),
		}, nil

	default:
		return nil, status.Errorf(codes.Internal, "未知的扣减结果: %d", code)
	}
}

// ReleaseStock 释放库存
func (s *InventoryServiceServer) ReleaseStock(ctx context.Context, req *inventoryv1.ReleaseStockRequest) (*inventoryv1.ReleaseStockResponse, error) {
	bookID := uint(req.BookId)
	quantity := int(req.Quantity)
	orderID := uint(req.OrderId)
	reason := req.Reason

	// 步骤1：Redis释放（Lua脚本）
	code, err := s.redisStore.ReleaseStock(ctx, bookID, quantity, orderID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "释放库存失败: %v", err)
	}

	// 步骤2：处理释放结果
	switch code {
	case 0:
		// 订单未扣减，无需释放
		return &inventoryv1.ReleaseStockResponse{
			Code:    40001,
			Message: "订单未扣减库存，无需释放",
		}, nil

	case 1:
		// 释放成功
		// 步骤3：异步同步到MySQL
		go func() {
			if err := s.repo.ReleaseStock(context.Background(), bookID, quantity, orderID, reason); err != nil {
				// logger.Error("sync release to mysql failed", zap.Error(err))
			}
		}()

		currentStock, _ := s.redisStore.GetStock(ctx, bookID)

		return &inventoryv1.ReleaseStockResponse{
			Code:         0,
			Message:      "释放成功",
			CurrentStock: int32(currentStock),
		}, nil

	case 2:
		// 重复释放（幂等性）
		currentStock, _ := s.redisStore.GetStock(ctx, bookID)
		return &inventoryv1.ReleaseStockResponse{
			Code:         0,
			Message:      "订单已释放（幂等性）",
			CurrentStock: int32(currentStock),
		}, nil

	default:
		return nil, status.Errorf(codes.Internal, "未知的释放结果: %d", code)
	}
}

// RestockInventory 补充库存
func (s *InventoryServiceServer) RestockInventory(ctx context.Context, req *inventoryv1.RestockInventoryRequest) (*inventoryv1.RestockInventoryResponse, error) {
	bookID := uint(req.BookId)
	quantity := int(req.Quantity)

	if quantity <= 0 {
		return &inventoryv1.RestockInventoryResponse{
			Code:    40001,
			Message: "补充数量必须大于0",
		}, nil
	}

	// Redis补货
	newStock, err := s.redisStore.RestockInventory(ctx, bookID, quantity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "补充库存失败: %v", err)
	}

	// 异步同步到MySQL
	go func() {
		if err := s.repo.RestockInventory(context.Background(), bookID, quantity); err != nil {
			// logger.Error("sync restock to mysql failed", zap.Error(err))
		}
	}()

	return &inventoryv1.RestockInventoryResponse{
		Code:         0,
		Message:      "补充成功",
		CurrentStock: int32(newStock),
	}, nil
}

// GetInventoryLogs 获取库存变更日志
func (s *InventoryServiceServer) GetInventoryLogs(ctx context.Context, req *inventoryv1.GetInventoryLogsRequest) (*inventoryv1.GetInventoryLogsResponse, error) {
	bookID := uint(req.BookId)
	page := int(req.Page)
	pageSize := int(req.PageSize)

	// 查询日志
	logs, total, err := s.logRepo.ListByBookID(ctx, bookID, page, pageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询库存日志失败: %v", err)
	}

	// 转换结果
	pbLogs := make([]*inventoryv1.InventoryLog, len(logs))
	for i, log := range logs {
		pbLogs[i] = &inventoryv1.InventoryLog{
			Id:          uint64(log.ID),
			BookId:      uint64(log.BookID),
			ChangeType:  string(log.ChangeType),
			Quantity:    int32(log.Quantity),
			BeforeStock: int32(log.BeforeStock),
			AfterStock:  int32(log.AfterStock),
			OrderId:     uint64(log.OrderID),
			CreatedAt:   log.CreatedAt.Unix(),
		}
	}

	return &inventoryv1.GetInventoryLogsResponse{
		Code:    0,
		Message: "success",
		Logs:    pbLogs,
		Total:   uint32(total),
	}, nil
}
