package screenshot

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"makeprofit/internal/browser"
	"makeprofit/internal/config"
	"makeprofit/internal/s3"
	"makeprofit/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Service 截图服务
type Service struct {
	browserService *browser.BrowserService
	s3Client       *s3.Client
	config         *config.Config
	logger         *logrus.Logger
}

// NewService 创建新的截图服务
func NewService(cfg *config.Config) (*Service, error) {
	logger := utils.GetLogger()

	// 创建浏览器服务
	browserService, err := browser.NewBrowserService(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser service: %w", err)
	}

	// 创建S3客户端
	s3Client, err := s3.NewClient(&cfg.S3)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &Service{
		browserService: browserService,
		s3Client:       s3Client,
		config:         cfg,
		logger:         logger,
	}, nil
}

// ScreenshotRequest 截图请求
type ScreenshotRequest struct {
	Symbol    string `json:"symbol" binding:"required"`    // 股票代码，如 "NVDA"
	Market    string `json:"market" binding:"required"`    // 市场，如 "us", "hk", "cn"
	Timeframe string `json:"timeframe" binding:"required"` // 时间框架，如 "1d", "1h"
}

// ScreenshotResponse 截图响应
type ScreenshotResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	CDNURL    string `json:"cdn_url,omitempty"`
	S3URL     string `json:"s3_url,omitempty"`
	Timestamp string `json:"timestamp"`
}

// TakeScreenshot 截取股票K线图
func (s *Service) TakeScreenshot(ctx context.Context, req *ScreenshotRequest) (*ScreenshotResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"symbol":    req.Symbol,
		"market":    req.Market,
		"timeframe": req.Timeframe,
	}).Info("Taking screenshot")

	// 截取截图并上传到S3
	uploadResult, err := s.browserService.TakeScreenshotAndUpload(ctx, req.Symbol, req.Market, req.Timeframe)
	if err != nil {
		s.logger.WithError(err).Error("Failed to take screenshot")
		return &ScreenshotResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to take screenshot: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	s.logger.WithFields(logrus.Fields{
		"symbol":    req.Symbol,
		"market":    req.Market,
		"timeframe": req.Timeframe,
		"cdn_url":   uploadResult.URL,
	}).Info("Screenshot completed successfully")

	return &ScreenshotResponse{
		Success:   true,
		Message:   "Screenshot taken successfully",
		CDNURL:    uploadResult.URL,
		S3URL:     uploadResult.Key, // 这里存储S3 key而不是URL
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// SetupRoutes 设置路由
func (s *Service) SetupRoutes(r *gin.Engine) {
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		stats := s.browserService.GetStats()
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().Format(time.RFC3339),
			"stats":  stats,
		})
	})

	// API路由组
	api := r.Group("/api/v1")
	{
		// 截图API
		api.POST("/screenshot", s.handleScreenshot)
		api.GET("/screenshot/:symbol/:market/:timeframe", s.handleScreenshotGet)

		// 状态监控API
		api.GET("/status", func(c *gin.Context) {
			stats := s.browserService.GetStats()
			taskManager, exists := s.browserService.GetTaskManager()
			var taskStats map[string]interface{}
			if exists {
				taskStats = map[string]interface{}{
					"running_tasks_count": taskManager.GetRunningTasksCount(),
					"running_tasks":       taskManager.GetRunningTasks(),
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"browser_stats": stats,
				"task_stats":    taskStats,
				"timestamp":     time.Now().Format(time.RFC3339),
			})
		})
	}
}

// handleScreenshot POST /api/v1/screenshot
func (s *Service) handleScreenshot(c *gin.Context) {
	var req ScreenshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ScreenshotResponse{
			Success:   false,
			Message:   fmt.Sprintf("Invalid request: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	response, err := s.TakeScreenshot(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ScreenshotResponse{
			Success:   false,
			Message:   fmt.Sprintf("Internal server error: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// handleScreenshotGet GET /api/v1/screenshot/:symbol/:market/:timeframe
func (s *Service) handleScreenshotGet(c *gin.Context) {
	symbol := c.Param("symbol")
	market := c.Param("market")
	timeframe := c.Param("timeframe")

	if symbol == "" || market == "" || timeframe == "" {
		c.JSON(http.StatusBadRequest, ScreenshotResponse{
			Success:   false,
			Message:   "Missing required parameters: symbol, market, timeframe",
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	req := &ScreenshotRequest{
		Symbol:    symbol,
		Market:    market,
		Timeframe: timeframe,
	}

	response, err := s.TakeScreenshot(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ScreenshotResponse{
			Success:   false,
			Message:   fmt.Sprintf("Internal server error: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// Close 关闭服务
func (s *Service) Close() {
	if s.browserService != nil {
		s.browserService.Close()
	}
}
