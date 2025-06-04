package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// ConfigV2 优化后的配置结构
type ConfigV2 struct {
	App      AppConfigV2      `yaml:"app" validate:"required"`
	Database DatabaseConfigV2 `yaml:"database" validate:"required"`
	Redis    RedisConfigV2    `yaml:"redis" validate:"required"`
	Logger   LoggerConfigV2   `yaml:"logger" validate:"required"`
	Security SecurityConfigV2 `yaml:"security" validate:"required"`
	API      APIConfigV2      `yaml:"api" validate:"required"`
}

// AppConfigV2 应用配置
type AppConfigV2 struct {
	Name        string        `yaml:"name" validate:"required" env:"APP_NAME"`
	Version     string        `yaml:"version" validate:"required" env:"APP_VERSION"`
	Environment string        `yaml:"environment" validate:"required,oneof=development staging production" env:"APP_ENV"`
	Port        int           `yaml:"port" validate:"required,min=1,max=65535" env:"APP_PORT"`
	Host        string        `yaml:"host" validate:"required" env:"APP_HOST"`
	Debug       bool          `yaml:"debug" env:"APP_DEBUG"`
	Timeout     time.Duration `yaml:"timeout" validate:"required" env:"APP_TIMEOUT"`
}

// DatabaseConfigV2 数据库配置
type DatabaseConfigV2 struct {
	Host            string        `yaml:"host" validate:"required" env:"DB_HOST"`
	Port            int           `yaml:"port" validate:"required,min=1,max=65535" env:"DB_PORT"`
	User            string        `yaml:"user" validate:"required" env:"DB_USER"`
	Password        string        `yaml:"password" validate:"required" env:"DB_PASSWORD"`
	Name            string        `yaml:"name" validate:"required" env:"DB_NAME"`
	Charset         string        `yaml:"charset" validate:"required" env:"DB_CHARSET"`
	MaxIdleConns    int           `yaml:"max_idle_conns" validate:"min=1" env:"DB_MAX_IDLE_CONNS"`
	MaxOpenConns    int           `yaml:"max_open_conns" validate:"min=1" env:"DB_MAX_OPEN_CONNS"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" validate:"required" env:"DB_CONN_MAX_LIFETIME"`
	SSLMode         string        `yaml:"ssl_mode" validate:"oneof=disable require verify-ca verify-full" env:"DB_SSL_MODE"`
}

// RedisConfigV2 Redis配置
type RedisConfigV2 struct {
	Host         string        `yaml:"host" validate:"required" env:"REDIS_HOST"`
	Port         int           `yaml:"port" validate:"required,min=1,max=65535" env:"REDIS_PORT"`
	Password     string        `yaml:"password" env:"REDIS_PASSWORD"`
	DB           int           `yaml:"db" validate:"min=0,max=15" env:"REDIS_DB"`
	PoolSize     int           `yaml:"pool_size" validate:"min=1" env:"REDIS_POOL_SIZE"`
	MinIdleConns int           `yaml:"min_idle_conns" validate:"min=0" env:"REDIS_MIN_IDLE_CONNS"`
	DialTimeout  time.Duration `yaml:"dial_timeout" validate:"required" env:"REDIS_DIAL_TIMEOUT"`
	ReadTimeout  time.Duration `yaml:"read_timeout" validate:"required" env:"REDIS_READ_TIMEOUT"`
	WriteTimeout time.Duration `yaml:"write_timeout" validate:"required" env:"REDIS_WRITE_TIMEOUT"`
}

// LoggerConfigV2 日志配置
type LoggerConfigV2 struct {
	Level      string `yaml:"level" validate:"required,oneof=debug info warn error" env:"LOG_LEVEL"`
	Format     string `yaml:"format" validate:"required,oneof=json console" env:"LOG_FORMAT"`
	Output     string `yaml:"output" validate:"required" env:"LOG_OUTPUT"`
	MaxSize    int    `yaml:"max_size" validate:"min=1" env:"LOG_MAX_SIZE"`
	MaxBackups int    `yaml:"max_backups" validate:"min=0" env:"LOG_MAX_BACKUPS"`
	MaxAge     int    `yaml:"max_age" validate:"min=1" env:"LOG_MAX_AGE"`
	Compress   bool   `yaml:"compress" env:"LOG_COMPRESS"`
}

// SecurityConfigV2 安全配置
type SecurityConfigV2 struct {
	JWT JWTConfigV2 `yaml:"jwt" validate:"required"`
}

// JWTConfigV2 JWT配置
type JWTConfigV2 struct {
	Secret     string        `yaml:"secret" validate:"required,min=32" env:"JWT_SECRET"`
	Expiration time.Duration `yaml:"expiration" validate:"required" env:"JWT_EXPIRATION"`
	Issuer     string        `yaml:"issuer" validate:"required" env:"JWT_ISSUER"`
}

// APIConfigV2 API配置
type APIConfigV2 struct {
	RateLimit RateLimitConfigV2 `yaml:"rate_limit" validate:"required"`
	CORS      CORSConfigV2      `yaml:"cors" validate:"required"`
}

// RateLimitConfigV2 限流配置
type RateLimitConfigV2 struct {
	Enabled bool          `yaml:"enabled" env:"RATE_LIMIT_ENABLED"`
	RPS     int           `yaml:"rps" validate:"min=1" env:"RATE_LIMIT_RPS"`
	Burst   int           `yaml:"burst" validate:"min=1" env:"RATE_LIMIT_BURST"`
	Window  time.Duration `yaml:"window" validate:"required" env:"RATE_LIMIT_WINDOW"`
}

// CORSConfigV2 CORS配置
type CORSConfigV2 struct {
	AllowOrigins     []string `yaml:"allow_origins" validate:"required" env:"CORS_ALLOW_ORIGINS"`
	AllowMethods     []string `yaml:"allow_methods" validate:"required" env:"CORS_ALLOW_METHODS"`
	AllowHeaders     []string `yaml:"allow_headers" validate:"required" env:"CORS_ALLOW_HEADERS"`
	ExposeHeaders    []string `yaml:"expose_headers" env:"CORS_EXPOSE_HEADERS"`
	AllowCredentials bool     `yaml:"allow_credentials" env:"CORS_ALLOW_CREDENTIALS"`
	MaxAge           int      `yaml:"max_age" validate:"min=0" env:"CORS_MAX_AGE"`
}

// LoadConfigV2 加载优化后的配置
func LoadConfigV2(configPath string) (*ConfigV2, error) {
	config := &ConfigV2{}

	// 读取配置文件
	if err := loadFromFile(config, configPath); err != nil {
		return nil, fmt.Errorf("failed to load config from file: %w", err)
	}

	// 从环境变量覆盖配置
	if err := loadFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load config from env: %w", err)
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

// loadFromFile 从文件加载配置
func loadFromFile(config *ConfigV2, configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// loadFromEnv 从环境变量加载配置
func loadFromEnv(config interface{}) error {
	return setEnvValues(config)
}

// setEnvValues 递归设置环境变量值
func setEnvValues(v interface{}) error {
	// 这里需要使用反射来遍历结构体字段并设置环境变量值
	// 为了简化，这里只是一个示例实现
	// 实际实现需要更复杂的反射逻辑
	return nil
}

// validateConfig 验证配置
func validateConfig(config *ConfigV2) error {
	validate := validator.New()
	return validate.Struct(config)
}

// GetEnvString 获取字符串环境变量
func GetEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt 获取整数环境变量
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool 获取布尔环境变量
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetEnvDuration 获取时间间隔环境变量
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetEnvStringSlice 获取字符串切片环境变量
func GetEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// IsDevelopment 是否为开发环境
func (c *ConfigV2) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsProduction 是否为生产环境
func (c *ConfigV2) IsProduction() bool {
	return c.App.Environment == "production"
}

// GetDSN 获取数据库连接字符串
func (c *ConfigV2) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Name,
		c.Database.Charset,
	)
}

// GetRedisAddr 获取Redis地址
func (c *ConfigV2) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}
