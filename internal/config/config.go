package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	S3           S3Config           `mapstructure:"s3"`
	CDN          CDNConfig          `mapstructure:"cdn"`
	ChartService ChartServiceConfig `mapstructure:"chart_service"`
	Logging      LoggingConfig      `mapstructure:"logging"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type S3Config struct {
	Region          string `mapstructure:"region"`
	Bucket          string `mapstructure:"bucket"`
	ImagePrefix     string `mapstructure:"image_prefix"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
}

type CDNConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

type ChartServiceConfig struct {
	BaseURL string `mapstructure:"base_url"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

var globalConfig *Config

func Load(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 启用环境变量支持
	viper.AutomaticEnv()

	// 绑定环境变量到配置键
	viper.BindEnv("s3.region", "AWS_REGION")
	viper.BindEnv("s3.bucket", "AWS_S3_BUCKET")
	viper.BindEnv("s3.access_key_id", "AWS_ACCESS_KEY_ID")
	viper.BindEnv("s3.secret_access_key", "AWS_SECRET_ACCESS_KEY")
	viper.BindEnv("cdn.base_url", "CDN_BASE_URL")
	viper.BindEnv("chart_service.base_url", "CHART_SERVICE_BASE_URL")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	globalConfig = &config
	return &config, nil
}

func Get() *Config {
	return globalConfig
}
