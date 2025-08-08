package browser

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"makeprofit/internal/config"
	"makeprofit/internal/s3"
	"makeprofit/pkg/utils"

	"github.com/sirupsen/logrus"
)

// BrowserService 浏览器服务
type BrowserService struct {
	browserPool      *BrowserPool
	screenshotClient *ScreenshotClient
	config           *config.Config
	s3Client         *s3.Client
}

// NewBrowserService 创建新的浏览器服务
func NewBrowserService(cfg *config.Config) (*BrowserService, error) {
	// 创建浏览器池
	browserPool, err := NewBrowserPool(&cfg.Browser, &cfg.Mafit)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser pool: %w", err)
	}

	// 创建S3客户端
	s3Client, err := s3.NewClient(&cfg.S3)
	if err != nil {
		browserPool.Close()
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	screenshotClient := NewScreenshotClient(browserPool, &cfg.Mafit, &cfg.Browser, s3Client)

	return &BrowserService{
		browserPool:      browserPool,
		screenshotClient: screenshotClient,
		config:           cfg,
		s3Client:         s3Client,
	}, nil
}

// TakeStockScreenshots 截取股票K线图
func (bs *BrowserService) TakeStockScreenshots(symbol, market string) (dailyPath, hourlyPath string, err error) {
	logger := utils.GetLogger()

	// 生成时间戳
	timestamp := time.Now().Format("2006010215") // yyyyMMddHH

	// 生成文件路径
	dailyPath = filepath.Join("temp", "screenshots", symbol, market, "1d", fmt.Sprintf("%s.png", timestamp))
	hourlyPath = filepath.Join("temp", "screenshots", symbol, market, "1h", fmt.Sprintf("%s.png", timestamp))

	logger.Infof("Taking screenshots for %s.%s", symbol, market)

	// 截取日线图
	logger.Info("Taking daily chart screenshot")
	if err := bs.screenshotClient.TakeMafitScreenshot(symbol, market, "1d", dailyPath); err != nil {
		logger.WithError(err).Error("Failed to take daily chart screenshot")
		return "", "", fmt.Errorf("failed to take daily chart screenshot: %w", err)
	}

	// 截取小时图
	logger.Info("Taking hourly chart screenshot")
	if err := bs.screenshotClient.TakeMafitScreenshot(symbol, market, "1h", hourlyPath); err != nil {
		logger.WithError(err).Error("Failed to take hourly chart screenshot")
		return "", "", fmt.Errorf("failed to take hourly chart screenshot: %w", err)
	}

	logger.Infof("Successfully took screenshots: %s, %s", dailyPath, hourlyPath)
	return dailyPath, hourlyPath, nil
}

// TakeStockScreenshotsAndUpload 截取股票K线图并上传到S3
func (bs *BrowserService) TakeStockScreenshotsAndUpload(ctx context.Context, symbol, market string) (*s3.UploadResult, *s3.UploadResult, error) {
	logger := utils.GetLogger()

	logger.Infof("Taking screenshots and uploading for %s.%s", symbol, market)

	// 截取日线图并上传
	logger.Info("Taking daily chart screenshot and uploading")
	dailyResult, err := bs.screenshotClient.TakeMafitScreenshotAndUpload(ctx, symbol, market, "1d")
	if err != nil {
		logger.WithError(err).Error("Failed to take and upload daily chart screenshot")
		return nil, nil, fmt.Errorf("failed to take and upload daily chart screenshot: %w", err)
	}

	// 截取小时图并上传
	logger.Info("Taking hourly chart screenshot and uploading")
	hourlyResult, err := bs.screenshotClient.TakeMafitScreenshotAndUpload(ctx, symbol, market, "1h")
	if err != nil {
		logger.WithError(err).Error("Failed to take and upload hourly chart screenshot")
		return nil, nil, fmt.Errorf("failed to take and upload hourly chart screenshot: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"symbol":     symbol,
		"market":     market,
		"daily_url":  dailyResult.URL,
		"hourly_url": hourlyResult.URL,
		"daily_key":  dailyResult.Key,
		"hourly_key": hourlyResult.Key,
	}).Info("Successfully took and uploaded screenshots")

	return dailyResult, hourlyResult, nil
}

// Close 关闭浏览器服务
func (bs *BrowserService) Close() {
	if bs.browserPool != nil {
		bs.browserPool.Close()
	}
}

// GetStats 获取浏览器统计信息
func (bs *BrowserService) GetStats() map[string]interface{} {
	if bs.browserPool != nil {
		return bs.browserPool.GetStats()
	}
	return map[string]interface{}{
		"error": "Browser pool not initialized",
	}
}



// TakeScreenshotAndUpload 截取截图并上传到S3
func (bs *BrowserService) TakeScreenshotAndUpload(ctx context.Context, symbol, market, timeframe string) (*s3.UploadResult, error) {
	return bs.screenshotClient.TakeMafitScreenshotAndUpload(ctx, symbol, market, timeframe)
}
