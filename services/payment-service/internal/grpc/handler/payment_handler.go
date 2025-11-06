package handler

import (
	"context"
	"log"
	"math/rand"

	paymentv1 "github.com/xiebiao/bookstore/proto/paymentv1"
	"github.com/xiebiao/bookstore/services/payment-service/internal/domain/payment"
)

type PaymentServiceServer struct {
	paymentv1.UnimplementedPaymentServiceServer
	repo payment.Repository
}

func NewPaymentServiceServer(repo payment.Repository) *PaymentServiceServer {
	return &PaymentServiceServer{repo: repo}
}

func (s *PaymentServiceServer) Pay(ctx context.Context, req *paymentv1.PayRequest) (*paymentv1.PayResponse, error) {
	existing, _ := s.repo.FindByOrderID(ctx, uint(req.OrderId))
	if existing != nil && existing.Status == payment.PaymentStatusPaid {
		return &paymentv1.PayResponse{Code: 0, Message: "订单已支付", PaymentNo: existing.PaymentNo}, nil
	}

	isSuccess := rand.Intn(100) < 70
	p := &payment.Payment{
		PaymentNo:     payment.GeneratePaymentNo(),
		OrderID:       uint(req.OrderId),
		Amount:        req.Amount,
		PaymentMethod: req.PaymentMethod,
	}

	if isSuccess {
		p.Status = payment.PaymentStatusPaid
		p.ThirdPartyNo = "MOCK" + p.PaymentNo
		s.repo.Create(ctx, p)
		log.Printf("✅ 支付成功: %s", p.PaymentNo)
		return &paymentv1.PayResponse{Code: 0, Message: "支付成功", PaymentNo: p.PaymentNo, ThirdPartyNo: p.ThirdPartyNo}, nil
	} else {
		p.Status = payment.PaymentStatusFailed
		s.repo.Create(ctx, p)
		return &paymentv1.PayResponse{Code: 1, Message: "支付失败（Mock）"}, nil
	}
}

func (s *PaymentServiceServer) GetPaymentStatus(ctx context.Context, req *paymentv1.GetPaymentStatusRequest) (*paymentv1.GetPaymentStatusResponse, error) {
	p, err := s.repo.FindByOrderID(ctx, uint(req.OrderId))
	if err != nil {
		return &paymentv1.GetPaymentStatusResponse{Code: 40400, Message: "支付记录不存在"}, nil
	}
	return &paymentv1.GetPaymentStatusResponse{
		Code: 0,
		Payment: &paymentv1.Payment{
			Id:            uint64(p.ID),
			PaymentNo:     p.PaymentNo,
			OrderId:       uint64(p.OrderID),
			Amount:        p.Amount,
			Status:        int32(p.Status),
			PaymentMethod: p.PaymentMethod,
			CreatedAt:     p.CreatedAt.Unix(),
		},
	}, nil
}

func (s *PaymentServiceServer) Refund(ctx context.Context, req *paymentv1.RefundRequest) (*paymentv1.RefundResponse, error) {
	p, err := s.repo.FindByOrderID(ctx, uint(req.OrderId))
	if err != nil {
		return &paymentv1.RefundResponse{Code: 40400, Message: "支付记录不存在"}, nil
	}
	if !p.CanRefund() {
		return &paymentv1.RefundResponse{Code: 40000, Message: "支付不可退款"}, nil
	}
	p.UpdateStatus(payment.PaymentStatusRefunded)
	s.repo.Update(ctx, p)
	return &paymentv1.RefundResponse{Code: 0, Message: "退款成功", RefundNo: "REF" + p.PaymentNo}, nil
}
