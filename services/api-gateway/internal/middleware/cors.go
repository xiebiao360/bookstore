package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/xiebiao/bookstore/services/api-gateway/internal/config"
)

// CORS 跨域资源共享中间件
//
// 教学要点：
// 1. CORS解决浏览器跨域请求问题
// 2. 预检请求（OPTIONS）的处理
// 3. 允许的域名、方法、头部配置
//
// 什么是CORS？
// - 浏览器的同源策略：协议+域名+端口必须相同
// - 跨域请求需要服务端返回CORS头部
// - 例如：前端http://localhost:3000 → 后端http://localhost:8080
//
// DO（正确做法）：
// - 生产环境：配置具体的允许域名
// - 开发环境：可以使用"*"（所有域名）
//
// DON'T（错误做法）：
// - 生产环境使用"*"（安全风险）
// - allow_credentials=true时不能使用"*"
func CORS(cfg config.CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果未启用CORS，直接跳过
		if !cfg.Enabled {
			c.Next()
			return
		}

		// 获取请求的Origin
		origin := c.Request.Header.Get("Origin")

		// 检查Origin是否在允许列表中
		allowed := false
		for _, allowOrigin := range cfg.AllowOrigins {
			if allowOrigin == "*" || allowOrigin == origin {
				c.Header("Access-Control-Allow-Origin", allowOrigin)
				allowed = true
				break
			}
		}

		if !allowed && origin != "" {
			// Origin不在允许列表中
			c.AbortWithStatus(403)
			return
		}

		// 设置允许的方法
		c.Header("Access-Control-Allow-Methods", joinStrings(cfg.AllowMethods, ", "))

		// 设置允许的头部
		c.Header("Access-Control-Allow-Headers", joinStrings(cfg.AllowHeaders, ", "))

		// 设置暴露的头部
		if len(cfg.ExposeHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", joinStrings(cfg.ExposeHeaders, ", "))
		}

		// 是否允许携带认证信息（Cookie、Authorization header）
		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		// 预检请求缓存时间
		if cfg.MaxAge > 0 {
			c.Header("Access-Control-Max-Age", string(rune(cfg.MaxAge)))
		}

		// 处理预检请求（OPTIONS）
		// 浏览器在发送跨域请求前，会先发送OPTIONS请求询问是否允许
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// joinStrings 连接字符串数组
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// =========================================
// 教学总结：CORS工作原理
// =========================================
//
// 1. 简单请求（不触发预检）：
//    - 方法：GET、POST、HEAD
//    - 头部：Accept、Accept-Language、Content-Language、Content-Type（仅限application/x-www-form-urlencoded、multipart/form-data、text/plain）
//    - 浏览器直接发送请求，服务端返回Access-Control-Allow-Origin
//
// 2. 预检请求（OPTIONS）：
//    - 非简单请求（如PUT、DELETE、自定义头部）
//    - 浏览器先发送OPTIONS询问
//    - 服务端返回允许的方法、头部
//    - 浏览器再发送实际请求
//
// 3. 携带认证信息：
//    - withCredentials=true（前端设置）
//    - Access-Control-Allow-Credentials=true（后端设置）
//    - Access-Control-Allow-Origin不能是"*"（必须指定具体域名）
//
// 4. 安全建议：
//    - 生产环境不要使用"*"
//    - 配置具体的允许域名列表
//    - 设置合理的MaxAge（减少预检请求）
