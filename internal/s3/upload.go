package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sirupsen/logrus"
)

// UploadResult 上传结果
type UploadResult struct {
	URL      string    `json:"url"`
	Key      string    `json:"key"`
	Size     int64     `json:"size"`
	Uploaded time.Time `json:"uploaded"`
}

// UploadFile 上传文件到S3
func (c *Client) UploadFile(ctx context.Context, localPath, s3Key string) (*UploadResult, error) {
	// 打开本地文件
	file, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", localPath, err)
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// 构建完整的S3 key
	fullKey := filepath.Join(c.config.ImagePrefix, s3Key)

	// 获取文件扩展名以确定Content-Type
	ext := filepath.Ext(localPath)
	contentType := "application/octet-stream" // 默认类型

	switch ext {
	case ".png":
		contentType = "image/png"
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	case ".svg":
		contentType = "image/svg+xml"
	}

	// 创建带超时的上下文
	uploadCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	// 上传文件到S3
	_, err = c.s3Client.PutObject(uploadCtx, &s3.PutObjectInput{
		Bucket:      aws.String(c.config.Bucket),
		Key:         aws.String(fullKey),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload file to S3: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"local_path":   localPath,
		"s3_key":       fullKey,
		"size":         fileInfo.Size(),
		"content_type": contentType,
	}).Info("File uploaded to S3 successfully")

	// 构建S3 URL
	s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		c.config.Bucket, c.config.Region, fullKey)

	return &UploadResult{
		URL:      s3URL,
		Key:      fullKey,
		Size:     fileInfo.Size(),
		Uploaded: time.Now(),
	}, nil
}

// UploadScreenshot 上传截图文件
func (c *Client) UploadScreenshot(ctx context.Context, localPath, symbol, market, timeframe string) (*UploadResult, error) {
	// 根据时间框架生成文件名
	var s3Key string
	now := time.Now()

	switch timeframe {
	case "1d":
		// 日线：同一天内一支股票只有一张，格式：{symbol}_{market}_1d_{date}.png
		s3Key = fmt.Sprintf("screenshots/%s_%s_1d_%s.png", symbol, market, now.Format("20060102"))
	case "1h":
		// 小时线：根据市场开市时间生成，格式：{symbol}_{market}_1h_{date}_{hour}.png
		s3Key = fmt.Sprintf("screenshots/%s_%s_1h_%s_%02d.png", symbol, market, now.Format("20060102"), now.Hour())
	case "1wk":
		// 周线：同一周内一支股票只有一张，格式：{symbol}_{market}_1wk_{year}_{week}.png
		year, week := now.ISOWeek()
		s3Key = fmt.Sprintf("screenshots/%s_%s_1wk_%d_%02d.png", symbol, market, year, week)
	default:
		// 其他时间框架使用时间戳（保持向后兼容）
		timestamp := now.Format("20060102_150405")
		s3Key = fmt.Sprintf("screenshots/%s_%s_%s_%s.png", symbol, market, timeframe, timestamp)
	}

	return c.UploadFile(ctx, localPath, s3Key)
}

// UploadJSONData 上传JSON数据文件
func (c *Client) UploadJSONData(ctx context.Context, localPath, symbol, market, timeframe string) (*UploadResult, error) {
	// 根据时间框架生成文件名
	var s3Key string
	now := time.Now()

	switch timeframe {
	case "1d":
		// 日线：同一天内一支股票只有一张，格式：{symbol}_{market}_1d_{date}.json
		s3Key = fmt.Sprintf("data/%s_%s_1d_%s.json", symbol, market, now.Format("20060102"))
	case "1h":
		// 小时线：根据市场开市时间生成，格式：{symbol}_{market}_1h_{date}_{hour}.json
		s3Key = fmt.Sprintf("data/%s_%s_1h_%s_%02d.json", symbol, market, now.Format("20060102"), now.Hour())
	case "1wk":
		// 周线：同一周内一支股票只有一张，格式：{symbol}_{market}_1wk_{year}_{week}.json
		year, week := now.ISOWeek()
		s3Key = fmt.Sprintf("data/%s_%s_1wk_%d_%02d.json", symbol, market, year, week)
	default:
		// 其他时间框架使用时间戳（保持向后兼容）
		timestamp := now.Format("20060102_150405")
		s3Key = fmt.Sprintf("data/%s_%s_%s_%s.json", symbol, market, timeframe, timestamp)
	}

	return c.UploadFile(ctx, localPath, s3Key)
}

// UploadReader 上传Reader内容到S3
func (c *Client) UploadReader(ctx context.Context, reader io.Reader, s3Key, contentType string) (*UploadResult, error) {
	// 构建完整的S3 key
	fullKey := filepath.Join(c.config.ImagePrefix, s3Key)

	// 上传内容到S3
	_, err := c.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(c.config.Bucket),
		Key:         aws.String(fullKey),
		Body:        reader,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upload content to S3: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"s3_key":       fullKey,
		"content_type": contentType,
	}).Info("Content uploaded to S3 successfully")

	// 构建S3 URL
	s3URL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		c.config.Bucket, c.config.Region, fullKey)

	return &UploadResult{
		URL:      s3URL,
		Key:      fullKey,
		Size:     0, // 无法确定大小
		Uploaded: time.Now(),
	}, nil
}
