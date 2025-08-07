package s3

import (
	"context"
	"fmt"

	"makeprofit/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sirupsen/logrus"
)

// Client S3客户端
type Client struct {
	s3Client *s3.Client
	config   *config.S3Config
	logger   *logrus.Logger
}

// NewClient 创建新的S3客户端
func NewClient(cfg *config.S3Config) (*Client, error) {
	logger := logrus.New()

	// 加载AWS配置
	var err error
	var awsCfg aws.Config

	// 如果配置文件中提供了凭证，使用它们
	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		awsCfg, err = awsconfig.LoadDefaultConfig(context.TODO(),
			awsconfig.WithRegion(cfg.Region),
			awsconfig.WithCredentialsProvider(credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     cfg.AccessKeyID,
					SecretAccessKey: cfg.SecretAccessKey,
				},
			}),
		)
	} else {
		// 否则使用默认配置（从环境变量或~/.aws/credentials读取）
		awsCfg, err = awsconfig.LoadDefaultConfig(context.TODO(),
			awsconfig.WithRegion(cfg.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 创建S3客户端，添加超时配置
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// 设置更长的超时时间
		o.ClientLogMode = 0 // 禁用客户端日志以减少噪音
	})

	return &Client{
		s3Client: s3Client,
		config:   cfg,
		logger:   logger,
	}, nil
}

// GetConfig 获取S3配置
func (c *Client) GetConfig() *config.S3Config {
	return c.config
}
