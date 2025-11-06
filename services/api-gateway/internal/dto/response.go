package dto

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
//
// 教学要点：
// 1. 统一响应格式（前端更容易处理）
// 2. Code字段：0成功，非0失败
// 3. Message字段：给用户的提示信息
// 4. Data字段：业务数据（成功时返回）
type Response struct {
	Code    int         `json:"code"`           // 业务状态码：0成功，非0失败
	Message string      `json:"message"`        // 提示信息
	Data    interface{} `json:"data,omitempty"` // 业务数据（可选）
}

// Success 成功响应
//
// 使用示例：
//
//	dto.Success(c, gin.H{"user_id": 1, "token": "xxx"})
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMessage 自定义消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
//
// 教学说明：
// HTTP状态码和业务状态码分离：
// - HTTP状态码：400/401/403/500等（协议层）
// - 业务状态码：40101（密码错误）、40401（用户不存在）等
//
// 使用示例：
//
//	dto.Error(c, http.StatusBadRequest, 40001, "参数错误")
func Error(c *gin.Context, httpCode int, bizCode int, message string) {
	c.JSON(httpCode, Response{
		Code:    bizCode,
		Message: message,
	})
}

// BadRequest 400错误
func BadRequest(c *gin.Context, message string) {
	Error(c, http.StatusBadRequest, 40000, message)
}

// Unauthorized 401错误（未登录）
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, 40100, message)
}

// Forbidden 403错误（无权限）
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, 40300, message)
}

// NotFound 404错误
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, 40400, message)
}

// InternalError 500错误
func InternalError(c *gin.Context, message string) {
	Error(c, http.StatusInternalServerError, 50000, message)
}

// =========================================
// 请求DTO定义
// =========================================

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=20"`
	Nickname string `json:"nickname" binding:"required,min=2,max=50"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshTokenRequest 刷新Token请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// =========================================
// 响应DTO定义
// =========================================

// RegisterResponse 注册响应
type RegisterResponse struct {
	UserID uint64 `json:"user_id"`
	Token  string `json:"token,omitempty"` // 可选：注册后直接登录
}

// LoginResponse 登录响应
type LoginResponse struct {
	UserID       uint64 `json:"user_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Access Token过期时间（秒）
}

// RefreshTokenResponse 刷新Token响应
type RefreshTokenResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// UserInfoResponse 用户信息响应
type UserInfoResponse struct {
	ID       uint64 `json:"id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

// =========================================
// 教学总结：API响应设计最佳实践
// =========================================
//
// 1. 统一响应格式：
//    {
//      "code": 0,
//      "message": "success",
//      "data": {...}
//    }
//    - 前端只需判断code是否为0
//    - message可直接显示给用户
//
// 2. HTTP状态码 vs 业务状态码：
//    - HTTP状态码：协议层（200/400/500）
//    - 业务状态码：业务层（40101密码错误、40401用户不存在）
//    - 好处：业务错误码可以更精细
//
// 3. 参数验证：
//    - 使用binding tag（required、email、min、max）
//    - Gin自动验证，失败返回400
//    - 减少手动if err != nil代码
//
// 4. 敏感信息：
//    - 不返回password字段
//    - Token只在登录/刷新时返回
//    - 错误信息不泄露内部实现细节
