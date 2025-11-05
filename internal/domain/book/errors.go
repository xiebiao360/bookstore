package book

import (
	apperrors "github.com/xiebiao/bookstore/pkg/errors"
)

// 图书领域错误定义
var (
	// ErrBookNotFound 图书不存在
	ErrBookNotFound = apperrors.New(apperrors.ErrCodeNotFound, "图书不存在")

	// ErrISBNDuplicate ISBN已存在
	ErrISBNDuplicate = apperrors.New(apperrors.ErrCodeDuplicateEntry, "ISBN号已存在")

	// ErrInvalidPrice 无效的价格
	ErrInvalidPrice = apperrors.New(apperrors.ErrCodeInvalidParams, "价格必须大于0")

	// ErrInvalidStock 无效的库存
	ErrInvalidStock = apperrors.New(apperrors.ErrCodeInvalidParams, "库存不能为负数")

	// ErrInvalidQuantity 无效的数量
	ErrInvalidQuantity = apperrors.New(apperrors.ErrCodeInvalidParams, "数量必须大于0")

	// ErrInsufficientStock 库存不足
	ErrInsufficientStock = apperrors.New(apperrors.ErrCodeBusinessError, "库存不足")

	// ErrInvalidISBN ISBN格式不正确
	ErrInvalidISBN = apperrors.New(apperrors.ErrCodeInvalidParams, "ISBN格式不正确")

	// ErrUnauthorized 无权操作此图书
	ErrUnauthorized = apperrors.New(apperrors.ErrCodeUnauthorized, "无权操作此图书")
)
