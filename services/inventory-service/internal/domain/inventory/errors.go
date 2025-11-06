package inventory

import "errors"

// 领域错误定义
//
// 教学要点：
// 1. 库存错误分类
//    - 参数错误（40xxx）
//    - 业务错误（库存不足、库存为负）
//    - 系统错误（50xxx）
//
// 2. 错误码设计规范
//    - 40001-40099：参数错误
//    - 40100-40199：库存不足相关
//    - 50001-50099：系统错误

var (
	// 参数错误
	ErrInvalidBookID       = errors.New("无效的图书ID")
	ErrInvalidQuantity     = errors.New("无效的扣减数量")
	ErrNegativeStock       = errors.New("库存不能为负数")
	ErrNegativeLockedStock = errors.New("锁定库存不能为负数")

	// 业务错误
	ErrInsufficientStock       = errors.New("库存不足")
	ErrInsufficientLockedStock = errors.New("锁定库存不足")
	ErrInventoryNotFound       = errors.New("库存记录不存在")
	ErrInconsistentTotalStock  = errors.New("总库存不一致")

	// 幂等性错误
	ErrDuplicateDeduction = errors.New("重复扣减（订单已处理）")
	ErrDuplicateRelease   = errors.New("重复释放（订单已处理）")
)
