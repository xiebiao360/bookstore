package errors

import (
	"errors"
	"fmt"
)

// AppError 自定义应用错误
// 设计说明：
// 1. Code用于客户端判断错误类型（不要直接暴露HTTP状态码）
// 2. Message是用户友好的提示信息
// 3. Err是内部错误，仅记录到日志，不返回给客户端（防止泄露敏感信息）
type AppError struct {
	Code    int    `json:"code"`    // 业务错误码
	Message string `json:"message"` // 用户友好的错误提示
	Err     error  `json:"-"`       // 内部错误（不序列化）
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 支持errors.Is和errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建新的AppError
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装系统错误（如数据库错误、网络错误）
// 用途：将底层错误转换为业务错误，隐藏实现细节
func Wrap(err error, message string) *AppError {
	return &AppError{
		Code:    ErrCodeInternal,
		Message: message,
		Err:     err,
	}
}

// Wrapf 格式化包装错误
func Wrapf(err error, format string, args ...interface{}) *AppError {
	return &AppError{
		Code:    ErrCodeInternal,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

// =========================================
// 错误码定义
// =========================================
// 规范：
// - 4xxxx: 客户端错误（参数错误、业务规则校验失败）
// - 5xxxx: 服务端错误（数据库异常、外部服务调用失败）

const (
	// 系统级错误码（50000-50099）
	ErrCodeInternal      = 50000 // 内部错误
	ErrCodeDatabaseError = 50001 // 数据库错误
	ErrCodeRedisError    = 50002 // Redis错误

	// 认证授权错误（40100-40199）
	ErrCodeUnauthorized    = 40100 // 未登录
	ErrCodeInvalidToken    = 40101 // Token无效
	ErrCodeTokenExpired    = 40102 // Token过期
	ErrCodeInvalidPassword = 40103 // 密码错误
	ErrCodeForbidden       = 40104 // 无权限

	// 资源错误（40400-40499）
	ErrCodeNotFound      = 40400 // 资源不存在(通用)
	ErrCodeUserNotFound  = 40401 // 用户不存在
	ErrCodeBookNotFound  = 40402 // 图书不存在
	ErrCodeOrderNotFound = 40403 // 订单不存在

	// 业务规则错误（40000-40099）
	ErrCodeBusinessError      = 40000 // 业务错误(通用)
	ErrCodeInsufficientStock  = 40001 // 库存不足
	ErrCodeInvalidOrderStatus = 40002 // 订单状态非法
	ErrCodeEmailDuplicate     = 40003 // 邮箱已存在
	ErrCodeISBNDuplicate      = 40004 // ISBN已存在
	ErrCodeWeakPassword       = 40005 // 密码强度不足
	ErrCodeDuplicateEntry     = 40009 // 重复记录(通用)

	// 参数错误（40900-40999）
	ErrCodeInvalidParams = 40900 // 参数错误
	ErrCodeBindError     = 40901 // 参数绑定失败
)

// =========================================
// 预定义错误（避免每次都New）
// =========================================

var (
	// 系统错误
	ErrInternal      = New(ErrCodeInternal, "系统内部错误")
	ErrDatabaseError = New(ErrCodeDatabaseError, "数据库错误")
	ErrRedisError    = New(ErrCodeRedisError, "缓存服务错误")

	// 认证授权
	ErrUnauthorized    = New(ErrCodeUnauthorized, "请先登录")
	ErrInvalidToken    = New(ErrCodeInvalidToken, "无效的Token")
	ErrTokenExpired    = New(ErrCodeTokenExpired, "Token已过期")
	ErrInvalidPassword = New(ErrCodeInvalidPassword, "密码错误")
	ErrForbidden       = New(ErrCodeForbidden, "无权限访问")

	// 资源不存在
	ErrUserNotFound  = New(ErrCodeUserNotFound, "用户不存在")
	ErrBookNotFound  = New(ErrCodeBookNotFound, "图书不存在")
	ErrOrderNotFound = New(ErrCodeOrderNotFound, "订单不存在")

	// 业务规则
	ErrInsufficientStock  = New(ErrCodeInsufficientStock, "库存不足")
	ErrInvalidOrderStatus = New(ErrCodeInvalidOrderStatus, "订单状态不允许此操作")
	ErrEmailDuplicate     = New(ErrCodeEmailDuplicate, "邮箱已被注册")
	ErrISBNDuplicate      = New(ErrCodeISBNDuplicate, "ISBN号已存在")
	ErrWeakPassword       = New(ErrCodeWeakPassword, "密码强度不足（需8-20位，包含字母和数字）")

	// 参数错误
	ErrInvalidParams = New(ErrCodeInvalidParams, "参数错误")
	ErrBindError     = New(ErrCodeBindError, "参数格式错误")
)

// =========================================
// 辅助函数
// =========================================

// IsAppError 判断是否为AppError
func IsAppError(err error) bool {
	var appErr *AppError
	return errors.As(err, &appErr)
}

// GetAppError 提取AppError（如果不是AppError则包装成Internal错误）
func GetAppError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Wrap(err, "系统内部错误")
}
