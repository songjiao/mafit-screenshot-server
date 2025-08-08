package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server  ServerConfig  `mapstructure:"server"`
	Browser BrowserConfig `mapstructure:"browser"`
	S3      S3Config      `mapstructure:"s3"`
	CDN     CDNConfig     `mapstructure:"cdn"`
	Mafit   MafitConfig   `mapstructure:"mafit"`
	Logging LoggingConfig `mapstructure:"logging"`
}

type ServerConfig struct {
	Port         int           `mapstructure:"port"`
	Host         string        `mapstructure:"host"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type BrowserConfig struct {
	Headless     bool          `mapstructure:"headless"`
	Timeout      time.Duration `mapstructure:"timeout"`
	PoolSize     int           `mapstructure:"pool_size"`      // 浏览器池大小
	UserAgent    string        `mapstructure:"user_agent"`
	
	// 性能优化配置
	MaxConcurrentPages int `mapstructure:"max_concurrent_pages"` // 每个浏览器最大并发页面数
	RendererProcesses  int `mapstructure:"renderer_processes"`   // 渲染进程数
	WebGLContexts      int `mapstructure:"webgl_contexts"`       // WebGL上下文数
	MemoryLimit        int `mapstructure:"memory_limit"`         // 内存限制(MB)
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



type MafitConfig struct {
	BaseURL        string `mapstructure:"base_url"`
	JWTAccessToken string `mapstructure:"jwt_access_token"`
	SidebarSheet   string `mapstructure:"sidebar_sheet"`
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
	viper.BindEnv("browser.pool_size", "BROWSER_POOL_SIZE")
	viper.BindEnv("browser.timeout", "BROWSER_TIMEOUT")
	viper.BindEnv("browser.user_agent", "BROWSER_USER_AGENT")
	viper.BindEnv("browser.max_concurrent_pages", "BROWSER_MAX_CONCURRENT")
	viper.BindEnv("browser.renderer_processes", "BROWSER_RENDERER_PROCESSES")
	viper.BindEnv("browser.webgl_contexts", "BROWSER_WEBGL_CONTEXTS")
	viper.BindEnv("browser.memory_limit", "BROWSER_MEMORY_LIMIT")
	viper.BindEnv("s3.region", "AWS_REGION")
	viper.BindEnv("s3.bucket", "AWS_S3_BUCKET")
	viper.BindEnv("s3.access_key_id", "AWS_ACCESS_KEY_ID")
	viper.BindEnv("s3.secret_access_key", "AWS_SECRET_ACCESS_KEY")
	viper.BindEnv("cdn.base_url", "CDN_BASE_URL")
	viper.BindEnv("mafit.base_url", "MAFIT_BASE_URL")
	viper.BindEnv("mafit.jwt_access_token", "MAFIT_JWT_TOKEN")
	viper.BindEnv("mafit.sidebar_sheet", "MAFIT_SIDEBAR_SHEET")

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
