package grpc_client

import (
	"context"
	"fmt"
	"log"
	"time"

	catalogv1 "github.com/xiebiao/bookstore/proto/catalogv1"
	inventoryv1 "github.com/xiebiao/bookstore/proto/inventoryv1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// InventoryClient inventory-service客户端
//
// 教学要点：
// 1. 为什么需要封装gRPC客户端？
//   - 连接管理：统一创建和关闭连接
//   - 超时控制：每个请求都设置超时
//   - 错误处理：统一的错误格式转换
//   - 重试逻辑：失败时自动重试（Phase 3）
//
// 2. 客户端生命周期：
//   - 应用启动时创建（长连接）
//   - 应用关闭时Close（释放资源）
//   - 不要每次请求都创建新连接（性能差）
type InventoryClient struct {
	conn   *grpc.ClientConn
	client inventoryv1.InventoryServiceClient
}

// NewInventoryClient 创建inventory-service客户端
//
// 教学要点：
// 1. grpc.Dial参数：
//   - target：服务地址（"localhost:9004"）
//   - options：连接选项
//   - WithTransportCredentials：TLS配置
//   - insecure.NewCredentials()：不使用TLS（开发环境）
//   - credentials.NewTLS()：使用TLS（生产环境）
//   - WithBlock：阻塞直到连接成功（可选）
//   - 不使用：Dial立即返回，后台异步连接
//   - 使用：Dial阻塞，连接失败则返回错误
//
// 2. 连接池：
//   - gRPC内部维护连接池（HTTP/2多路复用）
//   - 单个conn可以支持多个并发请求
//   - 通常一个服务只需要一个conn
//
// DO vs DON'T:
// ❌ DON'T: 每次RPC调用都Dial新连接
//
//	conn, _ := grpc.Dial(...)
//	client := inventoryv1.NewInventoryServiceClient(conn)
//	client.GetStock(...)
//	conn.Close() // 连接建立和销毁开销大
//
// ✅ DO: 应用启动时创建一次，复用连接
//
//	inventoryClient := NewInventoryClient(addr)
//	defer inventoryClient.Close()
func NewInventoryClient(addr string) (*InventoryClient, error) {
	// Dial选项
	opts := []grpc.DialOption{
		// 不使用TLS（开发环境）
		// 生产环境应该使用TLS证书
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// 设置默认超时（可选）
		// grpc.WithTimeout(5 * time.Second),
	}

	// 创建连接
	// 教学要点：
	// Dial是异步的，不会阻塞等待连接成功
	// - 好处：启动快
	// - 坏处：第一次RPC可能失败（连接未建立）
	// - 解决：使用grpc.WithBlock()或重试机制
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("连接inventory-service失败: %w", err)
	}

	// 创建客户端stub
	client := inventoryv1.NewInventoryServiceClient(conn)

	log.Printf("✅ inventory-service客户端已创建: %s", addr)

	return &InventoryClient{
		conn:   conn,
		client: client,
	}, nil
}

// Close 关闭连接
func (c *InventoryClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// DeductStock 扣减库存
//
// 教学要点：
// 1. Context超时控制：
//   - WithTimeout：设置请求超时
//   - defer cancel()：确保context释放（防止泄漏）
//
// 2. 为什么需要超时？
//   - 防止下游服务hang住
//   - 快速失败，释放资源
//   - 保护整个调用链
//
// 3. 超时时间设置：
//   - 库存扣减：5秒（Redis Lua脚本快）
//   - 支付：10秒（可能调用第三方）
//   - 查询：3秒
//
// 分布式事务：
// - 扣减库存是Saga的第一步
// - 失败时需要补偿（不需要释放，因为未扣减成功）
// - 成功后如果订单创建失败，需要调用ReleaseStock
func (c *InventoryClient) DeductStock(
	ctx context.Context,
	bookID uint,
	quantity int,
	orderID uint,
	timeout time.Duration,
) (*inventoryv1.DeductStockResponse, error) {
	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 发起RPC调用
	resp, err := c.client.DeductStock(ctx, &inventoryv1.DeductStockRequest{
		BookId:   uint64(bookID),
		Quantity: int32(quantity),
		OrderId:  uint64(orderID),
	})

	if err != nil {
		return nil, fmt.Errorf("扣减库存RPC调用失败: %w", err)
	}

	return resp, nil
}

// ReleaseStock 释放库存（补偿操作）
//
// 教学要点：
// 补偿场景：
// 1. 扣减库存成功，但订单创建失败
// 2. 扣减库存成功，但支付失败
// 3. 用户取消订单
// 4. 订单超时未支付
//
// 幂等性：
// - ReleaseStock是幂等操作（inventory-service的Lua脚本保证）
// - 多次调用不会重复释放
func (c *InventoryClient) ReleaseStock(
	ctx context.Context,
	bookID uint,
	quantity int,
	orderID uint,
	timeout time.Duration,
) (*inventoryv1.ReleaseStockResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := c.client.ReleaseStock(ctx, &inventoryv1.ReleaseStockRequest{
		BookId:   uint64(bookID),
		Quantity: int32(quantity),
		OrderId:  uint64(orderID),
	})

	if err != nil {
		return nil, fmt.Errorf("释放库存RPC调用失败: %w", err)
	}

	return resp, nil
}

// BatchGetStock 批量查询库存
//
// 教学要点：
// 为什么需要批量查询？
// - 创建订单前需要检查所有商品的库存
// - 单个查询：N次RPC调用（N个商品）
// - 批量查询：1次RPC调用
// - 性能提升：减少网络往返（RTT）
func (c *InventoryClient) BatchGetStock(
	ctx context.Context,
	bookIDs []uint,
	timeout time.Duration,
) (*inventoryv1.BatchGetStockResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 转换类型：uint → uint64
	ids := make([]uint64, len(bookIDs))
	for i, id := range bookIDs {
		ids[i] = uint64(id)
	}

	resp, err := c.client.BatchGetStock(ctx, &inventoryv1.BatchGetStockRequest{
		BookIds: ids,
	})

	if err != nil {
		return nil, fmt.Errorf("批量查询库存RPC调用失败: %w", err)
	}

	return resp, nil
}

// ============================================================
// CatalogClient - catalog-service客户端
// ============================================================

type CatalogClient struct {
	conn   *grpc.ClientConn
	client catalogv1.CatalogServiceClient
}

// NewCatalogClient 创建catalog-service客户端
func NewCatalogClient(addr string) (*CatalogClient, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("连接catalog-service失败: %w", err)
	}

	client := catalogv1.NewCatalogServiceClient(conn)

	log.Printf("✅ catalog-service客户端已创建: %s", addr)

	return &CatalogClient{
		conn:   conn,
		client: client,
	}, nil
}

// Close 关闭连接
func (c *CatalogClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetBook 查询图书详情
//
// 教学要点：
// 为什么需要查询图书？
// 1. 验证book_id是否存在
// 2. 获取图书价格（计算订单金额）
// 3. 获取图书标题（存储到OrderItem冗余字段）
//
// 数据冗余 vs 实时查询：
// - 冗余：OrderItem存储book_title
//   - 优点：查询订单时无需跨服务调用
//   - 缺点：图书改名后历史订单显示旧名称
//
// - 实时查询：每次查询订单都调用catalog-service
//   - 优点：数据永远最新
//   - 缺点：性能差，依赖catalog-service可用性
//
// 微服务设计原则：适度冗余，避免级联查询
func (c *CatalogClient) GetBook(
	ctx context.Context,
	bookID uint,
	timeout time.Duration,
) (*catalogv1.GetBookResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resp, err := c.client.GetBook(ctx, &catalogv1.GetBookRequest{
		BookId: uint64(bookID),
	})

	if err != nil {
		return nil, fmt.Errorf("查询图书RPC调用失败: %w", err)
	}

	return resp, nil
}

// BatchGetBooks 批量查询图书
//
// 教学要点：
// 用途：
// - 创建订单时批量获取图书信息
// - 一次RPC调用获取所有商品的价格和标题
func (c *CatalogClient) BatchGetBooks(
	ctx context.Context,
	bookIDs []uint,
	timeout time.Duration,
) (*catalogv1.BatchGetBooksResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ids := make([]uint64, len(bookIDs))
	for i, id := range bookIDs {
		ids[i] = uint64(id)
	}

	resp, err := c.client.BatchGetBooks(ctx, &catalogv1.BatchGetBooksRequest{
		BookIds: ids,
	})

	if err != nil {
		return nil, fmt.Errorf("批量查询图书RPC调用失败: %w", err)
	}

	return resp, nil
}
