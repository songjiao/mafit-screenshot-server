package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"makeprofit/internal/browser"
	"makeprofit/internal/config"
	"makeprofit/internal/s3"
	"makeprofit/pkg/utils"

	"github.com/sirupsen/logrus"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
	outputDir := flag.String("output", "screenshots", "截图输出目录")
	testS3 := flag.Bool("s3", false, "测试S3上传功能")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置日志
	utils.SetLogLevel(cfg.Logging.Level)
	utils.SetLogFormat(cfg.Logging.Format)
	logger := utils.GetLogger()

	// 创建浏览器池
	pool, err := browser.NewBrowserPool(&cfg.Browser, &cfg.Mafit)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create browser pool")
	}
	defer pool.Close()

	// 创建S3客户端（如果需要）
	var s3Client *s3.Client
	if *testS3 {
		s3Client, err = s3.NewClient(&cfg.S3)
		if err != nil {
			logger.WithError(err).Fatal("Failed to create S3 client")
		}
		logger.Info("S3 client created successfully")
	}

	// 创建任务管理器
	taskManager := browser.NewTaskManager(&cfg.CDN)

	// 创建截图客户端
	screenshotClient := browser.NewScreenshotClient(pool, &cfg.Mafit, &cfg.Browser, s3Client, taskManager)

	// 确保输出目录存在
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		logger.WithError(err).Fatal("Failed to create output directory")
	}

	// 测试截图
	testCases := []struct {
		name      string
		symbol    string
		market    string
		timeframe string
	}{
		{"NVDA日线", "NVDA", "us", "1d"},
		{"NVDA小时线", "NVDA", "us", "1h"},
		{"AAPL日线", "AAPL", "us", "1d"},
		{"AAPL小时线", "AAPL", "us", "1h"},
	}

	if *testS3 {
		// 测试S3上传功能
		logger.Info("Testing S3 upload functionality")
		ctx := context.Background()

		for _, tc := range testCases {
			logger.Infof("Testing S3 upload: %s", tc.name)

			uploadResult, err := screenshotClient.TakeMafitScreenshotAndUpload(ctx, tc.symbol, tc.market, tc.timeframe)
			if err != nil {
				logger.WithError(err).Errorf("Failed to take and upload screenshot for %s", tc.name)
			} else {
				logger.WithFields(logrus.Fields{
					"name": tc.name,
					"url":  uploadResult.URL,
					"key":  uploadResult.Key,
					"size": uploadResult.Size,
				}).Info("Successfully uploaded screenshot to S3")
			}
		}
	} else {
		// 测试本地截图功能
		for _, tc := range testCases {
			outputPath := filepath.Join(*outputDir, fmt.Sprintf("%s_%s_%s.png", tc.symbol, tc.market, tc.timeframe))

			logger.Infof("Testing screenshot: %s", tc.name)

			if err := screenshotClient.TakeMafitScreenshot(tc.symbol, tc.market, tc.timeframe, outputPath); err != nil {
				logger.WithError(err).Errorf("Failed to take screenshot for %s", tc.name)
			} else {
				logger.Infof("Successfully took screenshot: %s", outputPath)
			}
		}
	}

	logger.Info("Screenshot test completed")
}
