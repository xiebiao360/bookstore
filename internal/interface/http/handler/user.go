package handler

import (
	"github.com/gin-gonic/gin"

	appuser "github.com/xiebiao/bookstore/internal/application/user"
	"github.com/xiebiao/bookstore/internal/interface/http/dto"
	"github.com/xiebiao/bookstore/pkg/response"
)

// UserHandler 用户HTTP处理器
// 设计说明：
// 1. Handler只负责HTTP相关的事情：解析请求、调用应用层、返回响应
// 2. 不包含业务逻辑（业务逻辑在domain和application层）
// 3. 使用依赖注入，便于测试
type UserHandler struct {
	registerUseCase *appuser.RegisterUseCase
	loginUseCase    *appuser.LoginUseCase
}

// NewUserHandler 创建用户处理器
func NewUserHandler(
	registerUseCase *appuser.RegisterUseCase,
	loginUseCase *appuser.LoginUseCase,
) *UserHandler {
	return &UserHandler{
		registerUseCase: registerUseCase,
		loginUseCase:    loginUseCase,
	}
}

// Register 用户注册
// @Summary      用户注册
// @Description  创建新用户账号
// @Tags         用户模块
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterRequest true "注册信息"
// @Success      200 {object} response.Response{data=dto.UserResponse} "注册成功"
// @Failure      400 {object} response.Response "参数错误"
// @Failure      409 {object} response.Response "邮箱已存在"
// @Router       /users/register [post]
//
// 教学说明：Swagger注释详解
//   - @Summary: 接口简短描述（显示在接口列表）
//   - @Description: 接口详细描述
//   - @Tags: 接口分组标签（方便在Swagger UI中分类）
//   - @Accept: 请求Content-Type
//   - @Produce: 响应Content-Type
//   - @Param: 参数定义
//     格式: name paramType dataType required comment
//   - name: 参数名（这里是request，表示请求体）
//   - paramType: body（请求体）| query（查询参数）| path（路径参数）| header（请求头）
//   - dataType: 数据类型（dto.RegisterRequest）
//   - required: 是否必填（true/false）
//   - comment: 参数说明
//   - @Success: 成功响应
//     格式: httpCode dataType comment
//     {data=dto.UserResponse} 表示data字段的类型
//   - @Failure: 失败响应
//   - @Router: 路由定义（path + httpMethod）
//     注意：路径是相对于@BasePath的，所以是/users/register而不是/api/v1/users/register
func (h *UserHandler) Register(c *gin.Context) {
	// 1. 绑定并验证参数
	// 学习要点：Gin的ShouldBindJSON会自动校验binding tag
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 参数验证失败（如邮箱格式错误、密码长度不足）
		response.ErrorWithCode(c, 40900, "参数错误: "+err.Error())
		return
	}

	// 2. 调用应用层用例
	// 学习要点：Handler不直接调用domain层，而是通过application层
	result, err := h.registerUseCase.Execute(c.Request.Context(), appuser.RegisterRequest{
		Email:    req.Email,
		Password: req.Password,
		Nickname: req.Nickname,
	})

	if err != nil {
		// 业务错误（如邮箱已存在、密码强度不足）
		response.Error(c, err)
		return
	}

	// 3. 返回成功响应
	// 将application层的DTO转换为HTTP层的DTO
	response.Success(c, &dto.UserResponse{
		ID:       result.ID,
		Email:    result.Email,
		Nickname: result.Nickname,
	})
}

// Login 用户登录
// @Summary      用户登录
// @Description  验证邮箱密码，返回JWT Token
// @Tags         用户模块
// @Accept       json
// @Produce      json
// @Param        request body dto.LoginRequest true "登录信息"
// @Success      200 {object} response.Response{data=dto.LoginResponse} "登录成功，返回access_token和refresh_token"
// @Failure      400 {object} response.Response "参数错误"
// @Failure      401 {object} response.Response "邮箱或密码错误"
// @Failure      404 {object} response.Response "用户不存在"
// @Router       /users/login [post]
//
// 教学说明：JWT认证流程
// 1. 客户端发送邮箱+密码
// 2. 服务端验证密码（bcrypt对比哈希值）
// 3. 验证成功后生成JWT Token：
//   - Access Token: 有效期2小时，用于API认证
//   - Refresh Token: 有效期7天，用于刷新Access Token
//
// 4. 将Session信息存储到Redis（用于登出功能）
// 5. 返回Token给客户端
// 6. 客户端后续请求携带Token: Authorization: Bearer <token>
func (h *UserHandler) Login(c *gin.Context) {
	// 1. 绑定并验证参数
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithCode(c, 40900, "参数错误: "+err.Error())
		return
	}

	// 2. 调用登录用例
	result, err := h.loginUseCase.Execute(c.Request.Context(), appuser.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})

	if err != nil {
		// 登录失败（邮箱不存在或密码错误）
		response.Error(c, err)
		return
	}

	// 3. 返回成功响应（包含Token）
	response.Success(c, &dto.LoginResponse{
		User: dto.UserInfo{
			ID:       result.User.ID,
			Email:    result.User.Email,
			Nickname: result.User.Nickname,
		},
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
	})
}

// =========================================
// 学习要点总结
// =========================================
//
// 1. 为什么需要多个DTO？
//    - HTTP层DTO（dto.RegisterRequest）：包含验证tag，服务于HTTP协议
//    - 应用层DTO（appuser.RegisterRequest）：纯数据结构，服务于用例
//    - 领域实体（user.User）：包含业务逻辑，不应暴露给外部
//
// 2. 参数验证的三层防护：
//    - HTTP层：binding tag校验（格式、长度）
//    - 领域服务：业务规则校验（密码强度、邮箱唯一性）
//    - 数据库：约束校验（UNIQUE索引、NOT NULL）
//
// 3. 错误处理：
//    - 参数绑定失败：返回40900（客户端参数错误）
//    - 业务错误：由response.Error()自动处理AppError
//    - 系统错误：包装为50000，记录详细日志
