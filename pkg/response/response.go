package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	apperrors "github.com/xiebiao/bookstore/pkg/errors"
)

// Response 统一响应结构
// 设计说明：
// 1. Code是业务错误码（非HTTP状态码），方便客户端判断错误类型
// 2. Message是用户友好的提示信息
// 3. Data是业务数据，成功时返回，失败时为null
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Success 成功响应（Code=0表示成功）
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应（自动处理AppError）
// 用法：
//
//	err := userService.Register(...)
//	if err != nil {
//	    response.Error(c, err)
//	    return
//	}
func Error(c *gin.Context, err error) {
	// 提取AppError
	appErr := apperrors.GetAppError(err)

	// 记录详细错误到日志（包含内部错误）
	if appErr.Err != nil {
		// TODO: 使用zap日志记录appErr.Err
		// logger.Error("request failed", zap.Error(appErr.Err))
	}

	// 返回用户友好的错误信息
	c.JSON(http.StatusOK, Response{
		Code:    appErr.Code,
		Message: appErr.Message,
		Data:    nil,
	})
}

// ErrorWithCode 自定义错误码和消息
func ErrorWithCode(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// =========================================
// 分页响应结构
// =========================================

// PageData 分页数据封装
type PageData struct {
	List       interface{} `json:"list"`        // 数据列表
	Total      int64       `json:"total"`       // 总记录数
	Page       int         `json:"page"`        // 当前页码
	PageSize   int         `json:"page_size"`   // 每页大小
	TotalPages int         `json:"total_pages"` // 总页数
}

// NewPageData 创建分页数据
func NewPageData(list interface{}, total int64, page, pageSize int) *PageData {
	totalPages := int(total) / pageSize
	if int(total)%pageSize != 0 {
		totalPages++
	}

	return &PageData{
		List:       list,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}
}

// SuccessWithPage 分页成功响应
func SuccessWithPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	Success(c, NewPageData(list, total, page, pageSize))
}
