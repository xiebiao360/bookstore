package order

import (
	apperrors "github.com/xiebiao/bookstore/pkg/errors"
)

// 订单领域错误定义
var (
	// ErrOrderNotFound 订单不存在
	ErrOrderNotFound = apperrors.New(apperrors.ErrCodeOrderNotFound, "订单不存在")

	// ErrInvalidStatusTransition 非法的状态转换
	ErrInvalidStatusTransition = apperrors.New(apperrors.ErrCodeBusinessError, "订单状态不允许此操作")

	// ErrOrderNoGenerate 订单号生成失败
	ErrOrderNoGenerate = apperrors.New(apperrors.ErrCodeInternal, "订单号生成失败")

	// ErrInvalidOrderItems 订单明细不合法
	ErrInvalidOrderItems = apperrors.New(apperrors.ErrCodeInvalidParams, "订单明细不能为空")

	// ErrInvalidQuantity 购买数量不合法
	ErrInvalidQuantity = apperrors.New(apperrors.ErrCodeInvalidParams, "购买数量必须大于0")
)
