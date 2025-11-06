package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/xiebiao/bookstore/services/api-gateway/internal/client"
	"github.com/xiebiao/bookstore/services/api-gateway/internal/dto"
)

// UserHandler 用户相关HTTP处理器
//
// 教学要点：
// 1. HTTP Handler的职责：协议转换（HTTP JSON ↔ gRPC Protobuf）
// 2. 不包含业务逻辑，只做转发
// 3. 错误处理：gRPC错误码 → HTTP状态码
//
// 架构对比：
// Phase 1: HTTP Handler → UseCase → Domain Service → Repository
// Phase 2: HTTP Handler → gRPC Client → user-service
type UserHandler struct {
	userClient *client.UserClient
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userClient *client.UserClient) *UserHandler {
	return &UserHandler{
		userClient: userClient,
	}
}

// Register 用户注册
//
// 教学重点：
// 1. 参数绑定：Gin自动将JSON绑定到结构体并验证
// 2. 协议转换：HTTP Request → gRPC Request → gRPC Response → HTTP Response
// 3. 错误处理：gRPC错误 → HTTP错误
//
// @Summary 用户注册
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param request body dto.RegisterRequest true "注册信息"
// @Success 200 {object} dto.Response{data=dto.RegisterResponse}
// @Router /api/v1/auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	// 步骤1: 参数绑定和验证
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Gin自动验证失败（email格式、密码长度等）
		dto.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 步骤2: 调用gRPC服务
	resp, err := h.userClient.Register(context.Background(), req.Email, req.Password, req.Nickname)
	if err != nil {
		// 步骤3: gRPC错误处理
		h.handleGRPCError(c, err)
		return
	}

	// 步骤4: 返回HTTP响应
	dto.SuccessWithMessage(c, resp.Message, dto.RegisterResponse{
		UserID: resp.UserId,
		Token:  resp.Token,
	})
}

// Login 用户登录
//
// @Summary 用户登录
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "登录信息"
// @Success 200 {object} dto.Response{data=dto.LoginResponse}
// @Router /api/v1/auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	// 步骤1: 参数绑定
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 步骤2: 调用gRPC服务
	resp, err := h.userClient.Login(context.Background(), req.Email, req.Password)
	if err != nil {
		h.handleGRPCError(c, err)
		return
	}

	// 步骤3: 协议转换（gRPC Response → HTTP Response）
	dto.SuccessWithMessage(c, resp.Message, dto.LoginResponse{
		UserID:       resp.UserId,
		Token:        resp.Token,
		RefreshToken: resp.RefreshToken,
		ExpiresIn:    7200, // 2小时，与user-service配置一致
	})
}

// RefreshToken 刷新Token
//
// @Summary 刷新Access Token
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh Token"
// @Success 200 {object} dto.Response{data=dto.RefreshTokenResponse}
// @Router /api/v1/auth/refresh [post]
func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		dto.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	resp, err := h.userClient.RefreshToken(context.Background(), req.RefreshToken)
	if err != nil {
		h.handleGRPCError(c, err)
		return
	}

	dto.SuccessWithMessage(c, resp.Message, dto.RefreshTokenResponse{
		Token:        resp.Token,
		RefreshToken: resp.RefreshToken,
	})
}

// GetUser 获取用户信息
//
// @Summary 获取用户信息
// @Tags 用户管理
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} dto.Response{data=dto.UserInfoResponse}
// @Security BearerAuth
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	// 步骤1: 解析路径参数
	userIDStr := c.Param("id")
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		dto.BadRequest(c, "用户ID格式错误")
		return
	}

	// 步骤2: 调用gRPC服务
	resp, err := h.userClient.GetUser(context.Background(), userID)
	if err != nil {
		h.handleGRPCError(c, err)
		return
	}

	// 步骤3: 返回响应
	dto.SuccessWithMessage(c, resp.Message, dto.UserInfoResponse{
		ID:       resp.User.Id,
		Email:    resp.User.Email,
		Nickname: resp.User.Nickname,
	})
}

// handleGRPCError 处理gRPC错误
//
// 教学重点：
// gRPC错误码 → HTTP状态码映射
//
// gRPC常见错误码：
// - codes.InvalidArgument: 参数错误 → 400
// - codes.Unauthenticated: 未认证 → 401
// - codes.PermissionDenied: 无权限 → 403
// - codes.NotFound: 未找到 → 404
// - codes.Internal: 内部错误 → 500
// - codes.Unavailable: 服务不可用 → 503
func (h *UserHandler) handleGRPCError(c *gin.Context, err error) {
	// 提取gRPC状态码
	st, ok := status.FromError(err)
	if !ok {
		// 不是gRPC错误（网络错误等）
		dto.InternalError(c, "服务调用失败")
		return
	}

	// 根据gRPC错误码返回相应的HTTP错误
	switch st.Code() {
	case codes.InvalidArgument:
		dto.BadRequest(c, st.Message())
	case codes.Unauthenticated:
		dto.Unauthorized(c, st.Message())
	case codes.PermissionDenied:
		dto.Forbidden(c, st.Message())
	case codes.NotFound:
		dto.NotFound(c, st.Message())
	case codes.Unavailable:
		// 服务不可用（user-service宕机）
		dto.Error(c, 503, 50300, "服务暂时不可用，请稍后重试")
	default:
		// 其他错误统一返回500
		dto.InternalError(c, st.Message())
	}
}

// =========================================
// 教学总结：API Gateway Handler设计
// =========================================
//
// 1. 职责单一：
//    - 只做协议转换，不包含业务逻辑
//    - 业务逻辑在后端微服务中
//
// 2. 错误处理：
//    - gRPC错误码映射到HTTP状态码
//    - 统一错误响应格式
//    - 隐藏内部错误细节
//
// 3. 参数验证：
//    - 使用Gin的binding tag自动验证
//    - 减少手动if判断
//
// 4. 性能考虑：
//    - gRPC连接复用（不是每次请求都创建连接）
//    - 超时控制（防止慢请求）
//    - 后续添加：缓存、限流、熔断
//
// 5. 可观测性：
//    - 日志记录（请求ID、耗时）
//    - 监控指标（QPS、错误率）
//    - 链路追踪（后续集成OpenTelemetry）
