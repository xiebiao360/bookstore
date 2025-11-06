package book

import "errors"

// 领域错误定义
//
// 教学要点：
// 1. 错误定义在领域层，供所有上层使用
// 2. 使用errors.New创建哨兵错误（sentinel error）
// 3. 可以用errors.Is判断错误类型
//
// 错误码规范（与HTTP状态码映射）：
// - 400xx：客户端错误（参数错误、业务规则违反）
// - 404xx：资源不存在
// - 500xx：服务器错误

var (
	// ISBN相关错误
	ErrISBNRequired = errors.New("ISBN不能为空")
	ErrInvalidISBN  = errors.New("ISBN格式不正确")
	ErrISBNDup      = errors.New("ISBN已存在")

	// 标题相关错误
	ErrTitleRequired = errors.New("书名不能为空")

	// 作者相关错误
	ErrAuthorRequired = errors.New("作者不能为空")

	// 价格相关错误
	ErrInvalidPrice = errors.New("价格必须大于0")

	// 图书不存在
	ErrBookNotFound = errors.New("图书不存在")
)
