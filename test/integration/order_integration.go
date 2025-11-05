package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	baseURL = "http://localhost:8080/api/v1"
)

// 测试响应结构
type Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type RegisterData struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

type LoginData struct {
	AccessToken string `json:"access_token"`
}

type BookData struct {
	ID uint `json:"id"`
}

type OrderData struct {
	OrderID   uint   `json:"order_id"`
	OrderNo   string `json:"order_no"`
	Total     int64  `json:"total"`
	TotalYuan string `json:"total_yuan"`
}

func main() {
	fmt.Println("================================================")
	fmt.Println("订单模块集成测试")
	fmt.Println("================================================")

	// 步骤1: 注册测试用户
	fmt.Println("\n➜ 步骤1: 注册测试用户")
	email := fmt.Sprintf("buyer_%d@test.com", time.Now().Unix())
	registerReq := map[string]string{
		"email":    email,
		"password": "Test1234",
		"nickname": "测试买家",
	}

	registerResp, err := postJSON(baseURL+"/users/register", registerReq, "")
	if err != nil {
		fmt.Printf("✗ 注册失败: %v\n", err)
		return
	}

	if registerResp.Code != 0 {
		fmt.Printf("✗ 注册失败: %s\n", registerResp.Message)
		return
	}

	fmt.Printf("✓ 注册成功\n")

	// 步骤2: 登录获取Token
	fmt.Println("\n➜ 步骤2: 用户登录获取Token")
	loginReq := map[string]string{
		"email":    email,
		"password": "Test1234",
	}

	loginResp, err := postJSON(baseURL+"/users/login", loginReq, "")
	if err != nil {
		fmt.Printf("✗ 登录失败: %v\n", err)
		return
	}

	if loginResp.Code != 0 {
		fmt.Printf("✗ 登录失败: %s\n", loginResp.Message)
		return
	}

	var loginData LoginData
	json.Unmarshal(loginResp.Data, &loginData)
	token := loginData.AccessToken

	fmt.Printf("✓ 登录成功，Token: %s...\n", token[:20])

	// 步骤3: 上架测试图书（库存10本）
	fmt.Println("\n➜ 步骤3: 上架测试图书（库存10本）")
	// 使用时间戳生成唯一的ISBN-13 (13位数字)
	// 格式: 978 + 7位出版社号 + 3位随机数 = 13位
	timestamp := time.Now().Unix()
	isbn := fmt.Sprintf("9787115428%03d", timestamp%1000)
	fmt.Printf("生成的ISBN: %s (长度: %d)\n", isbn, len(isbn))
	bookReq := map[string]interface{}{
		"title":       "《Go并发编程实战》",
		"author":      "测试作者",
		"isbn":        isbn,
		"publisher":   "测试出版社",
		"price":       8900,
		"stock":       10,
		"description": "测试用图书",
	}

	bookResp, err := postJSON(baseURL+"/books", bookReq, token)
	if err != nil {
		fmt.Printf("✗ 图书上架失败: %v\n", err)
		return
	}

	if bookResp.Code != 0 {
		fmt.Printf("✗ 图书上架失败: %s\n", bookResp.Message)
		return
	}

	var bookData BookData
	json.Unmarshal(bookResp.Data, &bookData)
	bookID := bookData.ID

	fmt.Printf("✓ 图书上架成功，图书ID: %d\n", bookID)

	// 测试场景1: 正常下单
	fmt.Println("\n================================================")
	fmt.Println("测试场景1: 正常下单（购买3本）")
	fmt.Println("================================================")

	orderReq1 := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"book_id":  bookID,
				"quantity": 3,
			},
		},
	}

	orderResp1, err := postJSON(baseURL+"/orders", orderReq1, token)
	if err != nil {
		fmt.Printf("✗ 下单失败: %v\n", err)
		return
	}

	if orderResp1.Code != 0 {
		fmt.Printf("✗ 下单失败: %s\n", orderResp1.Message)
		return
	}

	var orderData1 OrderData
	json.Unmarshal(orderResp1.Data, &orderData1)

	fmt.Printf("✓ 下单成功，订单ID: %d，订单号: %s\n", orderData1.OrderID, orderData1.OrderNo)
	fmt.Printf("  订单金额: %s元 (预期: 267.00元)\n", orderData1.TotalYuan)
	fmt.Println("  预期结果：剩余库存 = 10 - 3 = 7本")

	// 测试场景2: 库存不足
	fmt.Println("\n================================================")
	fmt.Println("测试场景2: 库存不足（购买8本，剩余7本）")
	fmt.Println("================================================")

	orderReq2 := map[string]interface{}{
		"items": []map[string]interface{}{
			{
				"book_id":  bookID,
				"quantity": 8,
			},
		},
	}

	orderResp2, err := postJSON(baseURL+"/orders", orderReq2, token)
	if err != nil {
		fmt.Printf("✗ 请求失败: %v\n", err)
		return
	}

	if orderResp2.Code != 0 {
		fmt.Printf("✓ 正确返回库存不足错误: %s\n", orderResp2.Message)
	} else {
		fmt.Printf("✗ 未正确处理库存不足场景\n")
		return
	}

	// 测试场景3: 并发下单（验证防超卖）
	fmt.Println("\n================================================")
	fmt.Println("测试场景3: 并发下单（10个用户同时抢购剩余7本）")
	fmt.Println("================================================")

	fmt.Println("创建10个并发请求，每个购买1本...")

	var wg sync.WaitGroup
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			orderReq := map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"book_id":  bookID,
						"quantity": 1,
					},
				},
			}

			resp, err := postJSON(baseURL+"/orders", orderReq, token)

			mu.Lock()
			if err == nil && resp.Code == 0 {
				successCount++
			} else {
				failCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	fmt.Println("\n并发测试结果：")
	fmt.Printf("  - 成功下单: %d 个\n", successCount)
	fmt.Printf("  - 失败下单: %d 个\n", failCount)

	// 验证结果
	if successCount == 7 && failCount == 3 {
		fmt.Println("\n✓ 防超卖机制测试通过！成功订单数 = 剩余库存数")
		fmt.Println("✓ 悲观锁(SELECT FOR UPDATE)有效防止了超卖")
	} else if successCount > 7 {
		fmt.Printf("\n✗ 防超卖机制失败！出现超卖情况\n")
		fmt.Printf("✗ 成功订单数(%d) > 剩余库存(7)\n", successCount)
		return
	} else {
		fmt.Printf("\n⚠ 并发测试结果异常\n")
		fmt.Printf("⚠ 预期成功7个，实际成功%d个\n", successCount)
	}

	fmt.Println("\n================================================")
	fmt.Println("集成测试完成")
	fmt.Println("================================================")
	fmt.Println("\n教学要点总结：")
	fmt.Println("1. 正常下单流程：锁库存 → 创建订单 → 扣库存")
	fmt.Println("2. 库存不足校验：在事务内进行，保证一致性")
	fmt.Println("3. 并发防超卖：使用SELECT FOR UPDATE悲观锁")
	fmt.Println("   - 多个事务同时执行时，只有一个能获得锁")
	fmt.Println("   - 其他事务等待，直到前一个事务提交或回滚")
	fmt.Println("   - 保证库存扣减的原子性和一致性")
}

func postJSON(url string, data interface{}, token string) (*Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result Response
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	return &result, nil
}
