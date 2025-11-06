package mysql

import (
	"context"
	"errors"
	"fmt"

	"github.com/xiebiao/bookstore/services/payment-service/internal/domain/payment"
	"gorm.io/gorm"
)

type paymentRepository struct {
	db *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) payment.Repository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) Create(ctx context.Context, p *payment.Payment) error {
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		return fmt.Errorf("创建支付记录失败: %w", err)
	}
	return nil
}

func (r *paymentRepository) FindByID(ctx context.Context, id uint) (*payment.Payment, error) {
	var p payment.Payment
	err := r.db.WithContext(ctx).First(&p, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, payment.ErrPaymentNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) FindByOrderID(ctx context.Context, orderID uint) (*payment.Payment, error) {
	var p payment.Payment
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, payment.ErrPaymentNotFound
		}
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepository) Update(ctx context.Context, p *payment.Payment) error {
	result := r.db.WithContext(ctx).Save(p)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return payment.ErrPaymentNotFound
	}
	return nil
}
