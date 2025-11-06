package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiebiao/bookstore/services/api-gateway/internal/client"
	"github.com/xiebiao/bookstore/services/api-gateway/internal/dto"
)

// Auth JWT鉴权中间件
//
// 教学要点：
// 1. Gateway层统一鉴权（避免每个服务重复实现）
// 2. 从Header提取Token → 调用user-service验证 → 注入用户信息到Context
// 3. 区分公开接口和需要鉴权的接口
//
// 工作流程：
// 1. 提取Authorization header
// 2. 调用user-service的ValidateToken验证
// 3. 验证通过：将用户信息注入Context，继续处理请求
// 4. 验证失败：返回401错误，中断请求
//
// DO（正确做法）：
// - Gateway统一鉴权（单一职责）
// - 将用户信息注入Context（后续Handler可使用）
// - 公开接口不使用此中间件
//
// DON'T（错误做法）：
// - 在每个Handler中重复验证Token
// - 直接解析JWT（应该调用user-service，保证一致性）
// - 所有接口都强制鉴权（注册、登录不需要）
func Auth(userClient *client.UserClient) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 步骤1: 提取Authorization header
		// 格式：Authorization: Bearer <token>
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			dto.Unauthorized(c, "缺少Authorization header")
			c.Abort()
			return
		}

		// 步骤2: 解析Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			dto.Unauthorized(c, "Authorization格式错误，应为：Bearer <token>")
			c.Abort()
			return
		}

		token := parts[1]
		if token == "" {
			dto.Unauthorized(c, "Token不能为空")
			c.Abort()
			return
		}

		// 步骤3: 调用user-service验证Token
		// 教学重点：
		// Gateway不自己解析JWT，而是调用user-service验证
		// 好处：
		// 1. 保证Gateway和user-service的逻辑一致
		// 2. user-service可以检查黑名单（登出后的Token）
		// 3. 集中管理Token验证逻辑
		resp, err := userClient.ValidateToken(c.Request.Context(), token)
		if err != nil {
			dto.Unauthorized(c, "Token验证失败: "+err.Error())
			c.Abort()
			return
		}

		// 步骤4: 检查Token是否有效
		if !resp.Valid {
			dto.Unauthorized(c, "Token无效或已过期")
			c.Abort()
			return
		}

		// 步骤5: 注入用户信息到Context
		// 后续的Handler可以通过c.Get("user_id")获取
		c.Set("user_id", resp.UserId)
		c.Set("user_email", resp.Email)

		// 步骤6: 继续处理请求
		c.Next()
	}
}

// GetUserID 从Context中获取用户ID
//
// 使用示例：
//
//	func (h *OrderHandler) CreateOrder(c *gin.Context) {
//	    userID := middleware.GetUserID(c)
//	    // 创建订单...
//	}
func GetUserID(c *gin.Context) uint64 {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0
	}

	if id, ok := userID.(uint64); ok {
		return id
	}

	return 0
}

// GetUserEmail 从Context中获取用户邮箱
func GetUserEmail(c *gin.Context) string {
	email, exists := c.Get("user_email")
	if !exists {
		return ""
	}

	if e, ok := email.(string); ok {
		return e
	}

	return ""
}

// =========================================
// 教学总结：Gateway鉴权设计
// =========================================
//
// 1. 为什么在Gateway鉴权？
//    - 单一职责：Gateway统一处理认证
//    - 避免重复：后端服务不需要重复验证Token
//    - 安全性：统一的安全策略
//    - 性能：Gateway可以缓存验证结果
//
// 2. 为什么调用user-service验证而不是自己解析JWT？
//    优点：
//    - 逻辑一致：避免Gateway和user-service的JWT解析逻辑不一致
//    - 黑名单检查：user-service可以检查Token是否在黑名单中（用户登出）
//    - 集中管理：Token验证规则统一在user-service管理
//
//    缺点：
//    - 性能开销：每次请求都要调用user-service（RPC调用）
//
//    优化方案：
//    - 缓存验证结果（Redis，TTL=1分钟）
//    - 减少RPC调用次数
//
// 3. 公开接口 vs 需要鉴权的接口：
//    公开接口（不使用Auth中间件）：
//    - POST /api/v1/auth/register
//    - POST /api/v1/auth/login
//    - POST /api/v1/auth/refresh
//    - GET  /api/v1/books（浏览图书）
//
//    需要鉴权的接口（使用Auth中间件）：
//    - GET  /api/v1/users/:id
//    - POST /api/v1/orders（下单）
//    - POST /api/v1/books（上架图书）
//
// 4. 用户信息注入Context：
//    - 方便后续Handler使用
//    - 避免重复解析Token
//    - 类型安全（提供GetUserID辅助函数）
//
// 5. 后续优化方向：
//    - 添加Token缓存（减少RPC调用）
//    - 支持多种认证方式（JWT、OAuth2）
//    - 添加权限控制（RBAC）
//    - 限流（防止暴力破解）
