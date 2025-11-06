package payment

import "errors"

var (
	ErrPaymentNotFound      = errors.New("支付记录不存在")
	ErrDuplicatePayment     = errors.New("订单已支付")
	ErrPaymentNotRefundable = errors.New("支付不可退款")
	ErrInvalidAmount        = errors.New("支付金额异常")
)
