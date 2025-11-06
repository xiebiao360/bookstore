package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config API Gateway配置结构
//
// 教学要点：
// 1. Gateway配置包含HTTP服务器 + gRPC客户端配置
// 2. JWT配置必须与user-service保持一致（验证Token）
// 3. 为服务发现（Consul）预留扩展点
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	GRPC   GRPCConfig   `mapstructure:"grpc"`
	JWT    JWTConfig    `mapstructure:"jwt"`
	Log    LogConfig    `mapstructure:"log"`
	CORS   CORSConfig   `mapstructure:"cors"`
}

// ServerConfig HTTP服务器配置
type ServerConfig struct {
	Name         string `mapstructure:"name"`
	Version      string `mapstructure:"version"`
	HTTPPort     int    `mapstructure:"http_port"`
	Mode         string `mapstructure:"mode"`          // gin mode
	ReadTimeout  int    `mapstructure:"read_timeout"`  // 秒
	WriteTimeout int    `mapstructure:"write_timeout"` // 秒
}

// GRPCConfig gRPC客户端配置
//
// 教学说明：
// Phase 2 Week 5: 使用直连模式（host:port）
// Phase 2 Week 6: 将升级为服务发现模式（consul://service-name）
type GRPCConfig struct {
	UserService ServiceConfig `mapstructure:"user_service"`
	// 后续添加其他服务
	// CatalogService ServiceConfig `mapstructure:"catalog_service"`
	// OrderService   ServiceConfig `mapstructure:"order_service"`
}

// ServiceConfig 单个gRPC服务配置
type ServiceConfig struct {
	Addr    string `mapstructure:"addr"`    // 服务地址
	Timeout int    `mapstructure:"timeout"` // 调用超时（秒）
}

// GetTimeout 获取超时Duration
func (s *ServiceConfig) GetTimeout() time.Duration {
	if s.Timeout <= 0 {
		return 5 * time.Second // 默认5秒
	}
	return time.Duration(s.Timeout) * time.Second
}

// JWTConfig JWT配置
//
// 教学重点：
// Gateway需要验证Token，所以secret必须与user-service一致
type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	AccessTokenExpire  string `mapstructure:"access_token_expire"`
	RefreshTokenExpire string `mapstructure:"refresh_token_expire"`
}

// GetAccessTokenDuration 获取Access Token有效期
func (j *JWTConfig) GetAccessTokenDuration() time.Duration {
	d, err := time.ParseDuration(j.AccessTokenExpire)
	if err != nil {
		return 2 * time.Hour // 默认2小时
	}
	return d
}

// GetRefreshTokenDuration 获取Refresh Token有效期
func (j *JWTConfig) GetRefreshTokenDuration() time.Duration {
	d, err := time.ParseDuration(j.RefreshTokenExpire)
	if err != nil {
		return 168 * time.Hour // 默认7天
	}
	return d
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`  // debug/info/warn/error
	Format string `mapstructure:"format"` // console/json
	Output string `mapstructure:"output"` // stdout/文件路径
}

// CORSConfig 跨域配置
type CORSConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowOrigins     []string `mapstructure:"allow_origins"`
	AllowMethods     []string `mapstructure:"allow_methods"`
	AllowHeaders     []string `mapstructure:"allow_headers"`
	ExposeHeaders    []string `mapstructure:"expose_headers"`
	AllowCredentials bool     `mapstructure:"allow_credentials"`
	MaxAge           int      `mapstructure:"max_age"` // 秒
}

// Load 加载配置文件
//
// 教学要点：
// 1. 使用Viper加载YAML配置
// 2. 支持环境变量覆盖（GATEWAY_SERVER_HTTP_PORT）
// 3. 配置验证（必填项检查）
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 步骤1: 设置配置文件
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// 步骤2: 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 步骤3: 支持环境变量覆盖
	// 例如：GATEWAY_SERVER_HTTP_PORT=9090
	v.SetEnvPrefix("GATEWAY")
	v.AutomaticEnv()

	// 步骤4: 解析到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 步骤5: 配置验证
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("配置验证失败: %w", err)
	}

	return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 必填项检查
	if c.Server.HTTPPort <= 0 {
		return fmt.Errorf("server.http_port 必须大于0")
	}

	if c.GRPC.UserService.Addr == "" {
		return fmt.Errorf("grpc.user_service.addr 不能为空")
	}

	if c.JWT.Secret == "" || c.JWT.Secret == "your-256-bit-secret-key-change-in-production" {
		// 生产环境警告
		if c.Server.Mode == "release" {
			return fmt.Errorf("生产环境必须修改JWT secret")
		}
	}

	return nil
}

// =========================================
// 教学总结：配置管理最佳实践
// =========================================
//
// 1. 配置分层：
//    - 默认值在结构体中定义
//    - YAML文件覆盖默认值
//    - 环境变量覆盖YAML（生产环境密码）
//
// 2. 配置验证：
//    - 启动时验证，快速失败（Fail Fast）
//    - 避免运行时才发现配置错误
//
// 3. 类型安全：
//    - 使用结构体而非map[string]interface{}
//    - 编译期类型检查
//
// 4. 扩展性：
//    - 预留字段（注释掉的catalog_service）
//    - 便于后续添加新服务
