package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	apperrors "github.com/xiebiao/bookstore/pkg/errors"
)

// Manager JWT管理器
// 设计说明：
// 1. 使用双Token机制：Access Token（短期）+ Refresh Token（长期）
// 2. Access Token用于API鉴权，有效期短（2小时）
// 3. Refresh Token用于刷新Access Token，有效期长（7天）
type Manager struct {
	secret             string        // JWT签名密钥
	accessTokenExpire  time.Duration // Access Token有效期
	refreshTokenExpire time.Duration // Refresh Token有效期
}

// NewManager 创建JWT管理器
func NewManager(secret string, accessTokenExpire, refreshTokenExpire time.Duration) *Manager {
	return &Manager{
		secret:             secret,
		accessTokenExpire:  accessTokenExpire,
		refreshTokenExpire: refreshTokenExpire,
	}
}

// Claims 自定义JWT Claims
// 学习要点：
// 1. 嵌入jwt.RegisteredClaims获取标准字段（exp、iat、nbf等）
// 2. 添加自定义字段（UserID、Email）
type Claims struct {
	UserID   uint   `json:"user_id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	jwt.RegisteredClaims
}

// TokenPair Token对（Access + Refresh）
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Access Token过期时间（秒）
}

// GenerateToken 生成Token对
// 参数：
// - userID: 用户ID
// - email: 用户邮箱
// - nickname: 用户昵称
func (m *Manager) GenerateToken(userID uint, email, nickname string) (*TokenPair, error) {
	now := time.Now()

	// 1. 生成Access Token
	accessClaims := Claims{
		UserID:   userID,
		Email:    email,
		Nickname: nickname,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "bookstore",
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessTokenString, err := accessToken.SignedString([]byte(m.secret))
	if err != nil {
		return nil, apperrors.Wrap(err, "生成Access Token失败")
	}

	// 2. 生成Refresh Token（只包含UserID，减少payload大小）
	refreshClaims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "bookstore",
			Subject:   fmt.Sprintf("%d", userID),
		},
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshTokenString, err := refreshToken.SignedString([]byte(m.secret))
	if err != nil {
		return nil, apperrors.Wrap(err, "生成Refresh Token失败")
	}

	return &TokenPair{
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenString,
		ExpiresIn:    int64(m.accessTokenExpire.Seconds()),
	}, nil
}

// ParseToken 解析并验证Token
// 学习要点：
// 1. 验证签名（防止伪造）
// 2. 验证过期时间（exp）
// 3. 验证生效时间（nbf）
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非法的签名算法: %v", token.Header["alg"])
		}
		return []byte(m.secret), nil
	})

	if err != nil {
		// 判断具体的错误类型
		if err == jwt.ErrTokenExpired {
			return nil, apperrors.ErrTokenExpired
		}
		return nil, apperrors.ErrInvalidToken
	}

	// 提取Claims
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, apperrors.ErrInvalidToken
}

// RefreshAccessToken 使用Refresh Token刷新Access Token
// 业务流程：
// 1. 验证Refresh Token有效性
// 2. 提取UserID
// 3. 生成新的Access Token
func (m *Manager) RefreshAccessToken(refreshToken string) (string, error) {
	// 1. 解析Refresh Token
	claims, err := m.ParseToken(refreshToken)
	if err != nil {
		return "", err
	}

	// 2. 生成新的Access Token
	now := time.Now()
	newClaims := Claims{
		UserID:   claims.UserID,
		Email:    claims.Email,
		Nickname: claims.Nickname,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "bookstore",
			Subject:   fmt.Sprintf("%d", claims.UserID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	tokenString, err := token.SignedString([]byte(m.secret))
	if err != nil {
		return "", apperrors.Wrap(err, "刷新Token失败")
	}

	return tokenString, nil
}

// =========================================
// 学习要点总结
// =========================================
//
// 1. 为什么使用双Token机制？
//    - Access Token短期有效（2小时），减少泄露风险
//    - Refresh Token长期有效（7天），减少用户频繁登录
//    - 如果Access Token泄露，2小时后自动失效
//    - Refresh Token可以存储在HttpOnly Cookie中，更安全
//
// 2. JWT的结构
//    Header.Payload.Signature
//    - Header: {"alg":"HS256","typ":"JWT"}
//    - Payload: {"user_id":1,"email":"test@example.com","exp":1699999999}
//    - Signature: HMAC-SHA256(base64(header) + "." + base64(payload), secret)
//
// 3. JWT的优缺点
//    优点：
//    - 无状态（服务端不需要存储session）
//    - 跨域友好（可以跨服务验证）
//    - 可扩展（可以添加自定义字段）
//    缺点：
//    - 无法主动失效（需要配合黑名单机制）
//    - Token较大（Base64编码后约200-300字节）
//
// 4. 安全建议
//    - secret必须足够复杂（建议32位以上随机字符串）
//    - 生产环境必须使用HTTPS（防止Token被截获）
//    - 敏感操作需要二次验证（如修改密码、支付）
//    - 可以加入设备指纹、IP限制等增强安全性
