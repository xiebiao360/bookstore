package mq

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

// TestOrderEvent 测试事件结构
type TestOrderEvent struct {
	OrderID uint   `json:"order_id"`
	UserID  uint   `json:"user_id"`
	Action  string `json:"action"`
}

// TestPublisher_Publish 测试发布消息
func TestPublisher_Publish(t *testing.T) {
	// 创建发布者
	publisher, err := NewPublisher(
		"amqp://admin:admin123@localhost:5672/",
		"bookstore.test.events",
		"topic",
	)
	if err != nil {
		t.Fatalf("创建Publisher失败: %v", err)
	}
	defer publisher.Close()

	// 发布消息
	event := TestOrderEvent{
		OrderID: 123,
		UserID:  456,
		Action:  "created",
	}

	err = publisher.Publish("order.created", event)
	if err != nil {
		t.Fatalf("发布消息失败: %v", err)
	}

	t.Log("✅ 消息发布成功")
}

// TestConsumer_Consume 测试消费消息
func TestConsumer_Consume(t *testing.T) {
	// 创建消费者
	consumer, err := NewConsumer(
		"amqp://admin:admin123@localhost:5672/",
		"bookstore.test.events",
		"topic",
		"test.order.queue",
		[]string{"order.*"}, // 订阅所有order.开头的事件
	)
	if err != nil {
		t.Fatalf("创建Consumer失败: %v", err)
	}
	defer consumer.Close()

	// 先发布一条消息
	publisher, err := NewPublisher(
		"amqp://admin:admin123@localhost:5672/",
		"bookstore.test.events",
		"topic",
	)
	if err != nil {
		t.Fatalf("创建Publisher失败: %v", err)
	}
	defer publisher.Close()

	event := TestOrderEvent{
		OrderID: 789,
		UserID:  101,
		Action:  "paid",
	}
	publisher.Publish("order.paid", event)

	// 消费消息
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	received := false
	go func() {
		consumer.Consume(ctx, func(body []byte) error {
			var receivedEvent TestOrderEvent
			if err := json.Unmarshal(body, &receivedEvent); err != nil {
				return err
			}

			t.Logf("📬 收到事件: %+v", receivedEvent)

			if receivedEvent.OrderID == 789 && receivedEvent.Action == "paid" {
				received = true
				cancel() // 收到预期消息，停止消费
			}

			return nil
		})
	}()

	// 等待消费完成
	<-ctx.Done()

	if !received {
		t.Error("未收到预期的消息")
	} else {
		t.Log("✅ 消息消费成功")
	}
}

// TestPubSub_Integration 集成测试：发布订阅完整流程
func TestPubSub_Integration(t *testing.T) {
	// 创建发布者
	publisher, err := NewPublisher(
		"amqp://admin:admin123@localhost:5672/",
		"bookstore.test.events",
		"topic",
	)
	if err != nil {
		t.Fatalf("创建Publisher失败: %v", err)
	}
	defer publisher.Close()

	// 创建消费者
	consumer, err := NewConsumer(
		"amqp://admin:admin123@localhost:5672/",
		"bookstore.test.events",
		"topic",
		"test.integration.queue",
		[]string{"order.*"},
	)
	if err != nil {
		t.Fatalf("创建Consumer失败: %v", err)
	}
	defer consumer.Close()

	// 启动消费者
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	receivedEvents := make([]string, 0)

	go func() {
		consumer.Consume(ctx, func(body []byte) error {
			var event TestOrderEvent
			json.Unmarshal(body, &event)

			receivedEvents = append(receivedEvents, event.Action)
			t.Logf("📬 收到事件: %s", event.Action)

			if len(receivedEvents) >= 3 {
				cancel() // 收到3条消息，停止
			}

			return nil
		})
	}()

	// 等待消费者启动
	time.Sleep(1 * time.Second)

	// 发布3条消息
	events := []string{"created", "paid", "shipped"}
	for i, action := range events {
		err := publisher.Publish("order."+action, TestOrderEvent{
			OrderID: uint(i + 1),
			UserID:  100,
			Action:  action,
		})
		if err != nil {
			t.Errorf("发布消息失败: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 等待消费完成
	<-ctx.Done()

	// 验证
	if len(receivedEvents) != 3 {
		t.Errorf("期望收到3条消息，实际收到%d条", len(receivedEvents))
	}

	t.Logf("✅ 集成测试通过，收到事件: %v", receivedEvents)
}
