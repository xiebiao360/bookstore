package payment

import (
	"fmt"
	"math/rand"
	"time"
)

// GeneratePaymentNo 生成支付流水号
//
// 教学要点：
// 格式：PAY + YYYYMMDDHHMMSS + 6位随机数
// 示例：PAY20251106123456789012
func GeneratePaymentNo() string {
	now := time.Now()
	timePart := now.Format("20060102150405")
	randomPart := rand.Intn(900000) + 100000
	return fmt.Sprintf("PAY%s%d", timePart, randomPart)
}
