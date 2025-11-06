package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Logger 请求日志中间件
//
// 教学要点：
// 1. 记录每个请求的基本信息（方法、路径、耗时、状态码）
// 2. 生成唯一的请求ID（Trace ID），便于链路追踪
// 3. 结构化日志输出
//
// DO（正确做法）：
// - 记录请求ID，方便排查问题
// - 记录耗时，发现慢请求
// - 记录客户端IP（注意代理情况）
//
// DON'T（错误做法）：
// - 记录敏感信息（密码、Token）
// - 记录完整的请求体（可能很大，影响性能）
// - 阻塞主流程（日志应该异步写入）
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 步骤1: 生成请求ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// 步骤2: 记录开始时间
		startTime := time.Now()

		// 步骤3: 处理请求
		c.Next()

		// 步骤4: 记录请求信息
		endTime := time.Now()
		latency := endTime.Sub(startTime)

		// 提取客户端IP
		// 教学说明：
		// - X-Forwarded-For: 经过代理时的真实IP
		// - X-Real-IP: Nginx等代理设置的真实IP
		// - RemoteAddr: 直连时的IP
		clientIP := c.ClientIP()

		// 提取错误信息（如果有）
		var errMsg string
		if len(c.Errors) > 0 {
			errMsg = c.Errors.String()
		}

		// 结构化日志输出
		// 教学重点：
		// 生产环境应使用zap/logrus等结构化日志库
		// 这里简化为fmt输出，便于教学
		logFormat := "[GIN] %s | %3d | %13v | %15s | %-7s %s"

		// 根据状态码使用不同颜色（终端输出）
		statusColor := getStatusColor(c.Writer.Status())
		methodColor := getMethodColor(c.Request.Method)
		resetColor := "\033[0m"

		fmt.Printf(
			statusColor+logFormat+resetColor+" %s\n",
			endTime.Format("2006/01/02 - 15:04:05"),
			c.Writer.Status(),
			latency,
			clientIP,
			methodColor+c.Request.Method+resetColor,
			c.Request.URL.Path,
			errMsg,
		)

		// 记录慢请求警告
		if latency > 3*time.Second {
			fmt.Printf("[WARN] Slow request: %s %s took %v\n",
				c.Request.Method,
				c.Request.URL.Path,
				latency,
			)
		}
	}
}

// getStatusColor 根据HTTP状态码返回颜色
func getStatusColor(statusCode int) string {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return "\033[32m" // 绿色（成功）
	case statusCode >= 300 && statusCode < 400:
		return "\033[36m" // 青色（重定向）
	case statusCode >= 400 && statusCode < 500:
		return "\033[33m" // 黄色（客户端错误）
	default:
		return "\033[31m" // 红色（服务器错误）
	}
}

// getMethodColor 根据HTTP方法返回颜色
func getMethodColor(method string) string {
	switch method {
	case "GET":
		return "\033[34m" // 蓝色
	case "POST":
		return "\033[36m" // 青色
	case "PUT":
		return "\033[33m" // 黄色
	case "DELETE":
		return "\033[31m" // 红色
	case "PATCH":
		return "\033[32m" // 绿色
	case "HEAD":
		return "\033[35m" // 紫色
	case "OPTIONS":
		return "\033[37m" // 白色
	default:
		return "\033[0m" // 默认
	}
}

// =========================================
// 教学总结：日志中间件最佳实践
// =========================================
//
// 1. 请求ID（Trace ID）：
//    - 唯一标识每个请求
//    - 便于分布式追踪（跨服务传递）
//    - 排查问题时根据请求ID查询完整链路
//
// 2. 记录内容：
//    ✅ 应该记录：
//      - 请求方法、路径
//      - 状态码
//      - 耗时
//      - 客户端IP
//      - 错误信息
//
//    ❌ 不应该记录：
//      - 密码、Token等敏感信息
//      - 完整请求体（可能包含大文件）
//      - 个人隐私信息（需脱敏）
//
// 3. 性能考虑：
//    - 日志应该异步写入（不阻塞请求）
//    - 使用结构化日志（便于检索）
//    - 日志分级（debug/info/warn/error）
//    - 日志轮转（防止磁盘占满）
//
// 4. 生产环境升级：
//    - 使用zap/logrus替代fmt
//    - 日志输出到文件/ELK/Loki
//    - 集成OpenTelemetry实现分布式追踪
//    - 添加业务指标（QPS、错误率）
