package handler

import (
	"context"
	"fmt"
	"log"
	"time"

	orderv1 "github.com/xiebiao/bookstore/proto/orderv1"
	"github.com/xiebiao/bookstore/services/order-service/internal/domain/order"
	"github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/config"
	"github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/grpc_client"
	redisStore "github.com/xiebiao/bookstore/services/order-service/internal/infrastructure/persistence/redis"
)

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

func (s *OrderServiceServer) CreateOrder(ctx context.Context, req *orderv1.CreateOrderRequest) (*orderv1.CreateOrderResponse, error) {
	if err := s.validateCreateOrderRequest(req); err != nil {
		return &orderv1.CreateOrderResponse{Code: 40000, Message: err.Error()}, nil
	}

	var orderItems []order.OrderItem
	var total int64

	for _, item := range req.Items {
		bookResp, err := s.catalogClient.GetBook(ctx, uint(item.BookId), s.cfg.GetServiceTimeout("catalog"))
		if err != nil || bookResp.Code != 0 {
			return &orderv1.CreateOrderResponse{Code: 40400, Message: "图书不存在"}, nil
		}

		orderItem := order.OrderItem{
			BookID:    uint(item.BookId),
			BookTitle: bookResp.Book.Title,
			Quantity:  int(item.Quantity),
			Price:     bookResp.Book.Price,
		}
		orderItems = append(orderItems, orderItem)
		total += int64(orderItem.Quantity) * orderItem.Price
	}

	var deductedBooks []uint
	for _, item := range req.Items {
		resp, err := s.inventoryClient.DeductStock(ctx, uint(item.BookId), int(item.Quantity), 0, s.cfg.GetServiceTimeout("inventory"))
		if err != nil || resp.Code != 0 {
			s.compensateDeductStock(ctx, deductedBooks, req.UserId)
			return &orderv1.CreateOrderResponse{Code: 40100, Message: "库存不足"}, nil
		}
		deductedBooks = append(deductedBooks, uint(item.BookId))
	}

	orderEntity := &order.Order{
		OrderNo: order.GenerateOrderNo(),
		UserID:  uint(req.UserId),
		Status:  order.OrderStatusPending,
		Total:   total,
		Items:   orderItems,
	}

	if err := s.repo.Create(ctx, orderEntity); err != nil {
		s.compensateDeductStock(ctx, deductedBooks, req.UserId)
		return &orderv1.CreateOrderResponse{Code: 50002, Message: "创建订单失败"}, nil
	}

	expireAt := time.Now().Add(time.Duration(s.cfg.Order.PaymentTimeout) * time.Minute)
	s.cache.SetPendingOrder(ctx, orderEntity.ID, expireAt)
	log.Printf("✅ 订单创建成功: %s", orderEntity.OrderNo)

	return &orderv1.CreateOrderResponse{Code: 0, Message: "订单创建成功", OrderNo: orderEntity.OrderNo, OrderId: uint64(orderEntity.ID), Total: orderEntity.Total}, nil
}

func (s *OrderServiceServer) compensateDeductStock(ctx context.Context, bookIDs []uint, userID uint64) {
	for _, bookID := range bookIDs {
		s.inventoryClient.ReleaseStock(ctx, bookID, 1, 0, s.cfg.GetServiceTimeout("inventory"))
	}
}

func (s *OrderServiceServer) validateCreateOrderRequest(req *orderv1.CreateOrderRequest) error {
	if req.UserId == 0 {
		return fmt.Errorf("用户ID不能为空")
	}
	if len(req.Items) == 0 {
		return order.ErrEmptyOrderItems
	}
	if len(req.Items) > s.cfg.Order.MaxItemsPerOrder {
		return fmt.Errorf("单个订单最多%d种商品", s.cfg.Order.MaxItemsPerOrder)
	}
	for _, item := range req.Items {
		if item.BookId == 0 {
			return fmt.Errorf("商品ID不能为空")
		}
		if item.Quantity <= 0 || item.Quantity > int32(s.cfg.Order.MaxQuantityPerItem) {
			return fmt.Errorf("商品数量必须在1-%d之间", s.cfg.Order.MaxQuantityPerItem)
		}
	}
	return nil
}

func (s *OrderServiceServer) GetOrder(ctx context.Context, req *orderv1.GetOrderRequest) (*orderv1.GetOrderResponse, error) {
	orderID := uint(req.OrderId)
	cached, _ := s.cache.GetOrder(ctx, orderID)
	if cached != "" {
		var o order.Order
		if err := redisStore.UnmarshalOrder(cached, &o); err == nil {
			return &orderv1.GetOrderResponse{Code: 0, Message: "success", Order: s.convertOrderToProto(&o)}, nil
		}
	}

	o, err := s.repo.FindByID(ctx, orderID)
	if err != nil {
		if order.IsNotFoundError(err) {
			return &orderv1.GetOrderResponse{Code: 40400, Message: "订单不存在"}, nil
		}
		return &orderv1.GetOrderResponse{Code: 50003, Message: "查询订单失败"}, nil
	}

	go func() {
		orderJSON, _ := redisStore.MarshalOrder(o)
		s.cache.SetOrder(context.Background(), orderID, orderJSON, 5*time.Minute)
	}()

	return &orderv1.GetOrderResponse{Code: 0, Message: "success", Order: s.convertOrderToProto(o)}, nil
}

func (s *OrderServiceServer) UpdateOrderStatus(ctx context.Context, req *orderv1.UpdateOrderStatusRequest) (*orderv1.UpdateOrderStatusResponse, error) {
	return &orderv1.UpdateOrderStatusResponse{Code: 0, Message: "待实现"}, nil
}

func (s *OrderServiceServer) ListUserOrders(ctx context.Context, req *orderv1.ListUserOrdersRequest) (*orderv1.ListUserOrdersResponse, error) {
	return &orderv1.ListUserOrdersResponse{Code: 0, Message: "待实现"}, nil
}

func (s *OrderServiceServer) CancelOrder(ctx context.Context, req *orderv1.CancelOrderRequest) (*orderv1.CancelOrderResponse, error) {
	return &orderv1.CancelOrderResponse{Code: 0, Message: "待实现"}, nil
}

func (s *OrderServiceServer) convertOrderToProto(o *order.Order) *orderv1.Order {
	items := make([]*orderv1.OrderItemDetail, len(o.Items))
	for i, item := range o.Items {
		items[i] = &orderv1.OrderItemDetail{
			Id:        uint64(item.ID),
			OrderId:   uint64(item.OrderID),
			BookId:    uint64(item.BookID),
			BookTitle: item.BookTitle,
			Quantity:  int32(item.Quantity),
			Price:     item.Price,
		}
	}
	return &orderv1.Order{
		Id:        uint64(o.ID),
		OrderNo:   o.OrderNo,
		UserId:    uint64(o.UserID),
		Total:     o.Total,
		Status:    int32(o.Status),
		Items:     items,
		CreatedAt: o.CreatedAt.Unix(),
		UpdatedAt: o.UpdatedAt.Unix(),
	}
}
