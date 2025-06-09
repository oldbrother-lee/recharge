package configs

import (
	"fmt"

	"github.com/spf13/viper"
)

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	DB     DBConfig     `mapstructure:"database"`
	JWT    JWTConfig    `mapstructure:"jwt"`
	Log    LogConfig    `mapstructure:"log"`
	Task   TaskConfig   `mapstructure:"task"`
	API    APIConfig    `mapstructure:"api"`
	Redis  RedisConfig  `mapstructure:"redis"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"dbname"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
	Expire int    `mapstructure:"expire"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

type TaskConfig struct {
	Interval      int `mapstructure:"interval"`
	MaxRetries    int `mapstructure:"max_retries"`
	RetryDelay    int `mapstructure:"retry_delay"`
	MaxConcurrent int `mapstructure:"max_concurrent"`
}

type APIConfig struct {
	Key     string `mapstructure:"key"`
	UserID  string `mapstructure:"user_id"`
	BaseURL string `mapstructure:"base_url"`
}

var config *Config

// LoadConfig 从指定路径加载配置文件
func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func GetConfig() *Config {
	if config == nil {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath("./configs")

		// 设置环境变量前缀
		viper.SetEnvPrefix("RECHARGE")
		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err != nil {
			panic(fmt.Errorf("fatal error config file: %w", err))
		}

		if err := viper.Unmarshal(&config); err != nil {
			panic(fmt.Errorf("fatal error config file: %w", err))
		}
	}

	return config
}

// Init 初始化配置
func Init(env string) error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	// 设置环境变量前缀
	viper.SetEnvPrefix("RECHARGE")
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	if err := viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}
