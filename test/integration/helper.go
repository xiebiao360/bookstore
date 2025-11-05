package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// 教学说明：测试辅助工具
// 这个文件包含集成测试的通用辅助函数，遵循DRY原则（Don't Repeat Yourself）
// 将重复的代码（HTTP请求、JSON解析）封装成可复用的函数

const (
	// BaseURL API基础URL
	BaseURL = "http://localhost:8080/api/v1"
	// Timeout HTTP请求超时时间
	Timeout = 10 * time.Second
)

// Response 统一响应结构
type Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// RegisterData 注册响应数据
type RegisterData struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

// LoginData 登录响应数据
type LoginData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// BookData 图书响应数据
type BookData struct {
	ID          uint   `json:"id"`
	ISBN        string `json:"isbn"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	Publisher   string `json:"publisher"`
	Price       int64  `json:"price"`
	Stock       int    `json:"stock"`
	Description string `json:"description"`
}

// BookListData 图书列表响应数据
type BookListData struct {
	Items      []BookItem `json:"items"`
	Total      int64      `json:"total"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	TotalPages int        `json:"total_pages"`
}

// BookItem 图书列表项
type BookItem struct {
	ID        uint   `json:"id"`
	ISBN      string `json:"isbn"`
	Title     string `json:"title"`
	Author    string `json:"author"`
	Publisher string `json:"publisher"`
	Price     int64  `json:"price"`
	Stock     int    `json:"stock"`
}

// OrderData 订单响应数据
type OrderData struct {
	OrderID   uint   `json:"order_id"`
	OrderNo   string `json:"order_no"`
	Total     int64  `json:"total"`
	TotalYuan string `json:"total_yuan"`
}

// PostJSON 发送POST请求并解析JSON响应
//
// 教学说明：
// - 使用*testing.T参数，可以在失败时立即终止测试
// - 使用require包进行断言，失败会立即停止（不继续执行）
// - 返回*Response而非error，简化调用方代码
func PostJSON(t *testing.T, url string, data interface{}, token string) *Response {
	jsonData, err := json.Marshal(data)
	require.NoError(t, err, "JSON序列化失败")

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	require.NoError(t, err, "创建HTTP请求失败")

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: Timeout}
	resp, err := client.Do(req)
	require.NoError(t, err, "发送HTTP请求失败")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "读取响应体失败")

	var result Response
	err = json.Unmarshal(body, &result)
	require.NoError(t, err, "解析JSON响应失败: %s", string(body))

	return &result
}

// GetJSON 发送GET请求并解析JSON响应
func GetJSON(t *testing.T, url string, token string) *Response {
	req, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err, "创建HTTP请求失败")

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: Timeout}
	resp, err := client.Do(req)
	require.NoError(t, err, "发送HTTP请求失败")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "读取响应体失败")

	var result Response
	err = json.Unmarshal(body, &result)
	require.NoError(t, err, "解析JSON响应失败: %s", string(body))

	return &result
}

// GenerateTestEmail 生成唯一的测试邮箱
//
// 教学说明：
// 使用时间戳确保邮箱唯一性，避免测试重复运行时邮箱冲突
// 确保邮箱格式正确（包含@和域名）
func GenerateTestEmail(prefix string) string {
	return fmt.Sprintf("%s_%d@test.com", prefix, time.Now().Unix())
}

// GenerateTestISBN 生成唯一的测试ISBN
//
// 教学说明：
// ISBN-13格式：978 + 10位数字
// 使用时间戳的后10位确保唯一性
func GenerateTestISBN() string {
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("978%010d", timestamp%10000000000)
}

// RegisterTestUser 注册测试用户并返回Token
//
// 教学说明：
// 这是一个"高阶"辅助函数，封装了注册+登录的完整流程
// 简化了测试代码，让测试更关注业务逻辑而非基础设施
func RegisterTestUser(t *testing.T, nickname string) (email string, token string) {
	// 1. 注册
	email = GenerateTestEmail(nickname)
	registerReq := map[string]string{
		"email":    email,
		"password": "Test1234",
		"nickname": nickname,
	}

	registerResp := PostJSON(t, BaseURL+"/users/register", registerReq, "")
	require.Equal(t, 0, registerResp.Code, "注册失败: %s", registerResp.Message)

	// 2. 登录
	loginReq := map[string]string{
		"email":    email,
		"password": "Test1234",
	}

	loginResp := PostJSON(t, BaseURL+"/users/login", loginReq, "")
	require.Equal(t, 0, loginResp.Code, "登录失败: %s", loginResp.Message)

	var loginData LoginData
	err := json.Unmarshal(loginResp.Data, &loginData)
	require.NoError(t, err, "解析登录响应失败")

	return email, loginData.AccessToken
}

// PublishTestBook 上架测试图书并返回图书ID
//
// 教学说明：
// 封装了图书上架流程，返回bookID供后续测试使用
func PublishTestBook(t *testing.T, token string, title string, stock int) uint {
	isbn := GenerateTestISBN()
	bookReq := map[string]interface{}{
		"title":       title,
		"author":      "测试作者",
		"isbn":        isbn,
		"publisher":   "测试出版社",
		"price":       8900, // 89.00元
		"stock":       stock,
		"description": "集成测试用图书",
	}

	bookResp := PostJSON(t, BaseURL+"/books", bookReq, token)
	require.Equal(t, 0, bookResp.Code, "图书上架失败: %s", bookResp.Message)

	var bookData BookData
	err := json.Unmarshal(bookResp.Data, &bookData)
	require.NoError(t, err, "解析图书响应失败")

	return bookData.ID
}
