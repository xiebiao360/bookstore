package order

import (
	"fmt"
	"math/rand"
	"time"
)

// GenerateOrderNo 生成订单号
// 教学要点:订单号设计原则
// 1. 全局唯一(避免冲突)
// 2. 时间有序(便于分库分表)
// 3. 不可预测(防止恶意遍历)
//
// 格式:ORD + 时间戳(秒) + 6位随机数
// 示例:ORD1699248000123456
//
// 生产环境推荐:
// - 雪花算法(Snowflake):分布式唯一ID
// - UUID:简单但无序
// - 数据库自增ID:单点瓶颈
func GenerateOrderNo() string {
	timestamp := time.Now().Unix()
	random := rand.Intn(1000000) // 6位随机数
	return fmt.Sprintf("ORD%d%06d", timestamp, random)
}
