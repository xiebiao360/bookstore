package payment

import (
	"context"
)

// Repository 支付仓储接口
type Repository interface {
	Create(ctx context.Context, payment *Payment) error
	FindByID(ctx context.Context, id uint) (*Payment, error)
	FindByOrderID(ctx context.Context, orderID uint) (*Payment, error)
	Update(ctx context.Context, payment *Payment) error
}
