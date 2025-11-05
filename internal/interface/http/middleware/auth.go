package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/xiebiao/bookstore/internal/infrastructure/persistence/redis"
	"github.com/xiebiao/bookstore/pkg/jwt"
	"github.com/xiebiao/bookstore/pkg/response"
)

// AuthMiddleware JWT认证中间件
// 设计说明：
// 1. 从Header提取Token
// 2. 验证Token有效性
// 3. 检查Token黑名单
// 4. 将用户信息注入Context
type AuthMiddleware struct {
	jwtManager   *jwt.Manager
	sessionStore *redis.SessionStore
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(jwtManager *jwt.Manager, sessionStore *redis.SessionStore) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager:   jwtManager,
		sessionStore: sessionStore,
	}
}

// RequireAuth 要求登录
// 使用方式：
//
//	authorized := r.Group("/api/v1")
//	authorized.Use(authMiddleware.RequireAuth())
//	authorized.GET("/profile", handler.GetProfile)
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从Header提取Token
		// 格式：Authorization: Bearer <token>
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.ErrorWithCode(c, 40100, "请先登录")
			c.Abort()
			return
		}

		// 2. 解析Token格式
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.ErrorWithCode(c, 40101, "Token格式错误")
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 3. 检查Token是否在黑名单中（用户已登出或Token被强制失效）
		isBlacklisted, err := m.sessionStore.IsInBlacklist(c.Request.Context(), tokenString)
		if err != nil {
			response.ErrorWithCode(c, 50000, "验证Token失败")
			c.Abort()
			return
		}
		if isBlacklisted {
			response.ErrorWithCode(c, 40102, "Token已失效，请重新登录")
			c.Abort()
			return
		}

		// 4. 验证Token并解析Claims
		claims, err := m.jwtManager.ParseToken(tokenString)
		if err != nil {
			response.Error(c, err) // 自动处理ErrTokenExpired、ErrInvalidToken
			c.Abort()
			return
		}

		// 5. 将用户信息注入到Context（后续Handler可以使用）
		// 学习要点：使用Context传递请求级别的数据
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("nickname", claims.Nickname)

		// 6. 继续处理请求
		c.Next()
	}
}

// OptionalAuth 可选登录
// 说明：如果有Token则验证，没有则继续（用于某些公开+登录都能访问的接口）
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 没有Token，继续处理（作为匿名用户）
			c.Next()
			return
		}

		// 有Token，验证逻辑与RequireAuth相同
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString := parts[1]
			claims, err := m.jwtManager.ParseToken(tokenString)
			if err == nil {
				// Token有效，注入用户信息
				c.Set("user_id", claims.UserID)
				c.Set("email", claims.Email)
				c.Set("nickname", claims.Nickname)
			}
		}

		c.Next()
	}
}

// =========================================
// Context辅助函数（供Handler使用）
// =========================================

// GetUserID 从Context获取当前登录用户ID
// 使用示例：
//
//	userID := middleware.GetUserID(c)
//	if userID == 0 {
//	    // 未登录
//	}
func GetUserID(c *gin.Context) uint {
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(uint); ok {
			return uid
		}
	}
	return 0
}

// GetEmail 从Context获取当前登录用户邮箱
func GetEmail(c *gin.Context) string {
	if email, exists := c.Get("email"); exists {
		if e, ok := email.(string); ok {
			return e
		}
	}
	return ""
}

// MustGetUserID 从Context获取用户ID（如果不存在则panic）
// 说明：用于已经通过RequireAuth中间件的Handler
func MustGetUserID(c *gin.Context) uint {
	userID := GetUserID(c)
	if userID == 0 {
		panic("user_id not found in context")
	}
	return userID
}

// =========================================
// 学习要点总结
// =========================================
//
// 1. 中间件执行顺序
//    r.Use(Logger())        // 1. 日志中间件
//    r.Use(Recovery())      // 2. Recovery中间件
//    r.Use(Auth())          // 3. 认证中间件
//    r.GET("/api", handler) // 4. 业务Handler
//
// 2. c.Abort() vs c.Next()
//    - c.Abort(): 终止后续Handler执行（用于鉴权失败）
//    - c.Next(): 继续执行后续Handler
//
// 3. Context传递数据
//    - c.Set("key", value): 写入数据
//    - c.Get("key"): 读取数据
//    - 数据仅在当前请求的生命周期内有效
//
// 4. 安全建议
//    - 始终检查Token黑名单（防止已登出Token继续使用）
//    - Token泄露后可以通过黑名单强制失效
//    - 敏感操作需要二次验证（如修改密码需要输入旧密码）
