package order

import (
	"fmt"
	"math/rand"
	"time"
)

// GenerateOrderNo 生成订单号
//
// 教学要点：
// 1. 订单号设计原则：
//   - 唯一性：不能重复（通过时间戳 + 随机数保证）
//   - 可读性：包含时间信息，便于查询和统计
//   - 长度适中：20-32位（太短容易冲突，太长不便记忆）
//
// 2. 格式：YYYYMMDDHHMMSS + 6位随机数
//   - 示例：20251106123456789012
//   - 前14位：时间戳（精确到秒）
//   - 后6位：随机数（防止同一秒内重复）
//
// 3. 为什么不用UUID？
//   - UUID：32位十六进制（太长，不便记忆）
//   - 自定义：可以嵌入业务信息（时间、渠道等）
//
// 并发安全：
// - rand需要加锁或使用rand.New(rand.NewSource())
// - 本实现简化为全局rand（Go 1.20+线程安全）
func GenerateOrderNo() string {
	// 时间部分：20251106123456
	now := time.Now()
	timePart := now.Format("20060102150405")

	// 随机数部分：6位数字（100000-999999）
	// 教学要点：
	// rand.Intn(900000) 返回[0, 900000)
	// +100000 得到[100000, 1000000)，即6位数
	randomPart := rand.Intn(900000) + 100000

	// 组合：时间 + 随机数
	orderNo := fmt.Sprintf("%s%d", timePart, randomPart)

	return orderNo
}

// ValidateOrderNo 验证订单号格式
//
// 教学要点：
// 防御性编程：在使用外部输入前进行校验
// - 长度检查：20位
// - 格式检查：全数字
// - 时间检查：时间部分是否合法（可选）
func ValidateOrderNo(orderNo string) bool {
	// 长度检查
	if len(orderNo) != 20 {
		return false
	}

	// 全数字检查
	for _, c := range orderNo {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}
