package config

import (
	"fmt"
	"net/url"
	"time"

	"github.com/spf13/viper"
)

// Config 全局配置结构
// 设计说明：使用Viper管理配置，支持YAML文件、环境变量覆盖、配置热重载
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"` // debug | release | test
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	Charset         string        `mapstructure:"charset"`
	ParseTime       bool          `mapstructure:"parse_time"`
	Loc             string        `mapstructure:"loc"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

// DSN 生成MySQL连接字符串
// 格式：user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
// 注意：loc参数需要URL编码（Asia/Shanghai → Asia%2FShanghai）
func (d DatabaseConfig) DSN() string {
	// URL编码loc参数
	loc := url.QueryEscape(d.Loc)
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.Charset, d.ParseTime, loc)
}

type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// Addr 返回Redis地址
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

type JWTConfig struct {
	Secret             string        `mapstructure:"secret"`
	AccessTokenExpire  time.Duration `mapstructure:"access_token_expire"`
	RefreshTokenExpire time.Duration `mapstructure:"refresh_token_expire"`
}

type LogConfig struct {
	Level        string `mapstructure:"level"`  // debug | info | warn | error
	Format       string `mapstructure:"format"` // console | json
	Output       string `mapstructure:"output"` // stdout | stderr | /path/to/file
	EnableCaller bool   `mapstructure:"enable_caller"`
}

// Load 加载配置文件
// 支持：
// 1. 默认加载config/config.yaml
// 2. 通过环境变量BOOKSTORE_ENV指定环境（如config.prod.yaml）
// 3. 环境变量覆盖（如BOOKSTORE_DATABASE_PASSWORD）
func Load() (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	// 环境特定配置（如config.prod.yaml）
	if env := viper.GetString("env"); env != "" {
		v.SetConfigName("config." + env)
	}

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 环境变量绑定（自动转换，如BOOKSTORE_DATABASE_PASSWORD → database.password）
	v.SetEnvPrefix("BOOKSTORE")
	v.AutomaticEnv()

	// 解析到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 配置验证
	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validate 配置校验
func validate(cfg *Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("无效的服务端口: %d", cfg.Server.Port)
	}

	if cfg.JWT.Secret == "your-secret-key-change-in-production" && cfg.Server.Mode == "release" {
		return fmt.Errorf("生产环境必须修改JWT密钥")
	}

	return nil
}
