// Package mq 提供基于RabbitMQ的消息发布/订阅功能
//
// 核心概念（RabbitMQ）：
// 1. Producer（生产者）：发送消息到Exchange
// 2. Exchange（交换机）：路由消息到Queue
// 3. Queue（队列）：存储消息，等待消费
// 4. Consumer（消费者）：从Queue接收消息
// 5. Binding（绑定）：Exchange和Queue的路由规则
//
// Exchange类型：
// - Direct：根据routing_key精确匹配
// - Topic：根据routing_key模式匹配（支持通配符）
// - Fanout：广播到所有绑定的Queue
//
// 教学要点：
// - 理解消息队列的异步解耦作用
// - 掌握事件驱动架构的设计模式
// - 学习消息可靠性保证（持久化、确认机制）
package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// Publisher 消息发布者
type Publisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string // Exchange名称
}

// NewPublisher 创建消息发布者
//
// 参数：
//
//	url: RabbitMQ连接URL（如 amqp://user:pass@localhost:5672/）
//	exchange: Exchange名称
//	exchangeType: Exchange类型（direct/topic/fanout）
//
// 示例：
//
//	publisher, err := NewPublisher(
//	    "amqp://admin:admin123@localhost:5672/",
//	    "bookstore.events",    // Exchange名称
//	    "topic",               // Topic类型，支持通配符
//	)
func NewPublisher(url, exchange, exchangeType string) (*Publisher, error) {
	// 1. 连接RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("连接RabbitMQ失败: %w", err)
	}

	// 2. 创建Channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("创建Channel失败: %w", err)
	}

	// 3. 声明Exchange
	//
	// 参数说明：
	// - Durable: true表示持久化（RabbitMQ重启后Exchange不会丢失）
	// - AutoDelete: false表示不自动删除
	// - Internal: false表示可以由生产者直接发送消息
	// - NoWait: false表示等待服务器确认
	err = channel.ExchangeDeclare(
		exchange,     // Exchange名称
		exchangeType, // Exchange类型
		true,         // Durable（持久化）
		false,        // AutoDelete
		false,        // Internal
		false,        // NoWait
		nil,          // Arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("声明Exchange失败: %w", err)
	}

	log.Printf("✅ 消息发布者已创建: Exchange=%s, Type=%s", exchange, exchangeType)

	return &Publisher{
		conn:     conn,
		channel:  channel,
		exchange: exchange,
	}, nil
}

// Publish 发布消息
//
// 参数：
//
//	routingKey: 路由键（用于匹配Queue）
//	message: 消息内容（会被序列化为JSON）
//
// 示例：
//
//	err := publisher.Publish("order.created", OrderCreatedEvent{
//	    OrderID: 123,
//	    UserID:  456,
//	})
//
// 教学要点：
// - 消息持久化：DeliveryMode=2（确保RabbitMQ重启后消息不丢失）
// - ContentType：application/json（便于跨语言）
// - Timestamp：记录消息发送时间（便于调试）
func (p *Publisher) Publish(routingKey string, message interface{}) error {
	// 1. 序列化消息为JSON
	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("消息序列化失败: %w", err)
	}

	// 2. 发布消息
	err = p.channel.PublishWithContext(
		context.Background(),
		p.exchange, // Exchange
		routingKey, // Routing Key
		false,      // Mandatory（找不到Queue时是否返回消息）
		false,      // Immediate（消费者不可达时是否返回消息）
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent, // 消息持久化
			Timestamp:    time.Now(),
		},
	)

	if err != nil {
		return fmt.Errorf("发布消息失败: %w", err)
	}

	log.Printf("📤 消息已发布: RoutingKey=%s, Body=%s", routingKey, string(body))
	return nil
}

// Close 关闭发布者
func (p *Publisher) Close() error {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
	return nil
}

// Consumer 消息消费者
type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   string // Queue名称
}

// NewConsumer 创建消息消费者
//
// 参数：
//
//	url: RabbitMQ连接URL
//	exchange: Exchange名称
//	exchangeType: Exchange类型
//	queue: Queue名称（如 order.notification）
//	routingKeys: 订阅的路由键列表（支持通配符，如 order.*）
//
// 示例：
//
//	consumer, err := NewConsumer(
//	    "amqp://admin:admin123@localhost:5672/",
//	    "bookstore.events",
//	    "topic",
//	    "order.notification",         // Queue名称
//	    []string{"order.*"},          // 订阅所有order.开头的事件
//	)
func NewConsumer(url, exchange, exchangeType, queue string, routingKeys []string) (*Consumer, error) {
	// 1. 连接RabbitMQ
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("连接RabbitMQ失败: %w", err)
	}

	// 2. 创建Channel
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("创建Channel失败: %w", err)
	}

	// 3. 声明Exchange（与Publisher保持一致）
	err = channel.ExchangeDeclare(
		exchange,
		exchangeType,
		true,  // Durable
		false, // AutoDelete
		false, // Internal
		false, // NoWait
		nil,
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("声明Exchange失败: %w", err)
	}

	// 4. 声明Queue
	//
	// 参数说明：
	// - Durable: true表示持久化
	// - AutoDelete: false表示没有消费者时不自动删除
	// - Exclusive: false表示允许多个消费者
	q, err := channel.QueueDeclare(
		queue, // Queue名称
		true,  // Durable
		false, // AutoDelete
		false, // Exclusive
		false, // NoWait
		nil,   // Arguments
	)
	if err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("声明Queue失败: %w", err)
	}

	// 5. 绑定Queue到Exchange
	//
	// Topic Exchange支持通配符：
	// - * 匹配一个单词（如 order.* 匹配 order.created, order.paid）
	// - # 匹配零个或多个单词（如 order.# 匹配 order.created, order.payment.success）
	for _, routingKey := range routingKeys {
		err = channel.QueueBind(
			q.Name,     // Queue名称
			routingKey, // Routing Key（支持通配符）
			exchange,   // Exchange名称
			false,      // NoWait
			nil,        // Arguments
		)
		if err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("绑定Queue失败: %w", err)
		}
	}

	log.Printf("✅ 消息消费者已创建: Queue=%s, RoutingKeys=%v", queue, routingKeys)

	return &Consumer{
		conn:    conn,
		channel: channel,
		queue:   q.Name,
	}, nil
}

// Consume 开始消费消息
//
// 参数：
//
//	handler: 消息处理函数
//
// 示例：
//
//	err := consumer.Consume(func(body []byte) error {
//	    var event OrderCreatedEvent
//	    if err := json.Unmarshal(body, &event); err != nil {
//	        return err
//	    }
//	    // 处理事件：发送邮件
//	    sendEmail(event.UserID, "订单创建成功")
//	    return nil
//	})
//
// 教学要点：
// - AutoAck: false（手动确认，确保消息处理成功后才从队列删除）
// - 失败重试：handler返回错误时，消息会被Nack（重新入队）
// - 优雅退出：监听ctx.Done()，收到信号时停止消费
func (c *Consumer) Consume(ctx context.Context, handler func([]byte) error) error {
	// 1. 设置Qos（Quality of Service）
	//
	// PrefetchCount: 1表示每次只取1条消息（处理完才取下一条）
	// 好处：负载均衡（多个消费者时，工作量平均分配）
	err := c.channel.Qos(
		1,     // PrefetchCount
		0,     // PrefetchSize
		false, // Global
	)
	if err != nil {
		return fmt.Errorf("设置Qos失败: %w", err)
	}

	// 2. 开始消费
	msgs, err := c.channel.Consume(
		c.queue, // Queue名称
		"",      // Consumer标签（空表示自动生成）
		false,   // AutoAck（false表示手动确认）
		false,   // Exclusive
		false,   // NoLocal
		false,   // NoWait
		nil,     // Arguments
	)
	if err != nil {
		return fmt.Errorf("开始消费失败: %w", err)
	}

	log.Printf("📥 开始消费消息: Queue=%s", c.queue)

	// 3. 处理消息
	for {
		select {
		case <-ctx.Done():
			// 收到退出信号
			log.Printf("🛑 消费者退出: Queue=%s", c.queue)
			return nil

		case msg, ok := <-msgs:
			if !ok {
				// Channel关闭
				return fmt.Errorf("消息Channel已关闭")
			}

			log.Printf("📬 收到消息: RoutingKey=%s, Body=%s", msg.RoutingKey, string(msg.Body))

			// 处理消息
			err := handler(msg.Body)
			if err != nil {
				// 处理失败，Nack（重新入队）
				log.Printf("❌ 消息处理失败: %v, 消息将重新入队", err)
				msg.Nack(false, true) // Requeue=true
			} else {
				// 处理成功，Ack（确认）
				msg.Ack(false)
				log.Printf("✅ 消息处理成功")
			}
		}
	}
}

// Close 关闭消费者
func (c *Consumer) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	return nil
}

// ==================== DO/DON'T 对比 ====================

// ❌ DON'T: 同步调用（阻塞主流程）
//
// 问题场景：
// func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
//     // 创建订单
//     order := &Order{...}
//     db.Create(order)
//
//     // 同步发送邮件（阻塞3秒）
//     sendEmail(order.UserID, "订单创建成功") // 如果邮件服务慢，用户要等3秒
//
//     return nil
// }
//
// 后果：
// 1. 用户体验差（等待时间长）
// 2. 邮件服务故障会导致订单创建失败
// 3. 无法横向扩展（邮件发送和订单创建在同一进程）

// ✅ DO: 异步发布事件（快速响应）
//
// func CreateOrder(ctx context.Context, req *CreateOrderRequest) error {
//     // 1. 创建订单
//     order := &Order{...}
//     db.Create(order)
//
//     // 2. 发布事件（异步，<1ms）
//     publisher.Publish("order.created", OrderCreatedEvent{
//         OrderID: order.ID,
//         UserID:  order.UserID,
//     })
//
//     // 3. 立即返回（不等待邮件发送）
//     return nil
// }
//
// // 单独的消费者进程处理事件
// func main() {
//     consumer.Consume(ctx, func(body []byte) error {
//         var event OrderCreatedEvent
//         json.Unmarshal(body, &event)
//
//         // 发送邮件（慢操作，不影响订单创建）
//         sendEmail(event.UserID, "订单创建成功")
//         return nil
//     })
// }
//
// 优点：
// 1. 快速响应（订单创建<10ms）
// 2. 解耦（邮件服务故障不影响订单）
// 3. 可扩展（启动多个消费者进程）
// 4. 削峰填谷（邮件慢慢发，不影响用户）

// ==================== 教学总结 ====================
//
// 消息队列的核心价值：
// 1. **异步解耦**：生产者和消费者独立部署、独立扩展
// 2. **削峰填谷**：高峰期消息堆积，低峰期慢慢处理
// 3. **最终一致性**：订单立即创建，邮件稍后发送（用户可接受）
// 4. **可靠性**：消息持久化，消费失败自动重试
//
// 适用场景：
// - ✅ 异步通知（邮件、短信、推送）
// - ✅ 日志收集（应用日志 → ELK）
// - ✅ 数据同步（Redis → MySQL）
// - ✅ 流量削峰（秒杀场景）
//
// 不适用场景：
// - ❌ 同步查询（用户查询订单详情，需要立即返回）
// - ❌ 强一致性（支付扣款，必须立即确认）
