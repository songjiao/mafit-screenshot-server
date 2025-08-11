package screenshot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"makeprofit/internal/chartservice"
	"makeprofit/internal/config"
	"makeprofit/internal/s3"
	"makeprofit/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Service 截图服务
type Service struct {
	chartService *chartservice.Client
	s3Client     *s3.Client
	config       *config.Config
	logger       *logrus.Logger
}

// NewService 创建新的截图服务
func NewService(cfg *config.Config) (*Service, error) {
	logger := utils.GetLogger()

	// 创建图表服务客户端
	chartService := chartservice.NewClient(cfg.ChartService.BaseURL)

	// 创建S3客户端
	s3Client, err := s3.NewClient(&cfg.S3)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &Service{
		chartService: chartService,
		s3Client:     s3Client,
		config:       cfg,
		logger:       logger,
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
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	CDNURL     string `json:"cdn_url,omitempty"`
	S3URL      string `json:"s3_url,omitempty"`
	DataCDNURL string `json:"data_cdn_url,omitempty"`
	DataS3URL  string `json:"data_s3_url,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// ScreenshotWithDataResponse 带数据的截图响应
type ScreenshotWithDataResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	CDNURL     string `json:"cdn_url,omitempty"`
	S3URL      string `json:"s3_url,omitempty"`
	DataCDNURL string `json:"data_cdn_url,omitempty"`
	DataS3URL  string `json:"data_s3_url,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// TakeScreenshot 截取股票K线图
func (s *Service) TakeScreenshot(ctx context.Context, req *ScreenshotRequest) (*ScreenshotResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"symbol":    req.Symbol,
		"market":    req.Market,
		"timeframe": req.Timeframe,
	}).Info("Taking screenshot using chart service")

	// 格式化股票代码
	formattedSymbol := s.formatSymbolForMarket(req.Symbol, req.Market)

	// 使用图表服务获取截图
	chartImage, err := s.chartService.TakeScreenshotWithRefresh(ctx, formattedSymbol, req.Timeframe)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get chart image from chart service")
		return &ScreenshotResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to get chart image: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	// 同时获取JSON数据（但不返回给用户，只上传到S3）
	panelData, err := s.chartService.GetPanelData(ctx, formattedSymbol, req.Timeframe)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get panel data, will continue without JSON data")
	}

	// 生成截图文件名
	screenshotFileName := s.generateScreenshotFileName(req.Symbol, req.Market, req.Timeframe)
	if screenshotFileName == "" {
		return &ScreenshotResponse{
			Success:   false,
			Message:   "Failed to generate screenshot filename",
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	// 创建临时文件
	tempDir := os.TempDir()
	screenshotTempFile := filepath.Join(tempDir, screenshotFileName)

	// 写入图片数据到临时文件
	if err := os.WriteFile(screenshotTempFile, chartImage.Data, 0o644); err != nil {
		s.logger.WithError(err).Error("Failed to write image to temp file")
		return &ScreenshotResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to write image to temp file: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	// 确保临时文件被清理
	defer func() {
		if err := os.Remove(screenshotTempFile); err != nil {
			s.logger.WithError(err).Warn("Failed to remove screenshot temp file")
		}
	}()

	// 上传截图到S3
	uploadResult, err := s.s3Client.UploadScreenshot(ctx, screenshotTempFile, req.Symbol, req.Market, req.Timeframe)
	if err != nil {
		s.logger.WithError(err).Error("Failed to upload screenshot to S3")
		return &ScreenshotResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to upload to S3: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	// 如果有JSON数据，上传到S3并返回URL
	var jsonResult *s3.UploadResult
	if panelData != nil && panelData.Success {
		// 生成JSON文件名
		jsonFileName := s.generateJSONFileName(req.Symbol, req.Market, req.Timeframe)
		if jsonFileName != "" {
			jsonTempFile := filepath.Join(tempDir, jsonFileName)

			// 将JSON数据写入临时文件
			jsonData, err := json.Marshal(panelData.Data)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to marshal JSON data")
			} else {
				if err := os.WriteFile(jsonTempFile, jsonData, 0o644); err != nil {
					s.logger.WithError(err).Warn("Failed to write JSON to temp file")
				} else {
					// 确保JSON临时文件被清理
					defer func() {
						if err := os.Remove(jsonTempFile); err != nil {
							s.logger.WithError(err).Warn("Failed to remove JSON temp file")
						}
					}()

					// 上传JSON到S3
					jsonResult, err = s.s3Client.UploadJSONData(ctx, jsonTempFile, req.Symbol, req.Market, req.Timeframe)
					if err != nil {
						s.logger.WithError(err).Warn("Failed to upload JSON data to S3")
					} else {
						s.logger.WithFields(logrus.Fields{
							"symbol":    req.Symbol,
							"market":    req.Market,
							"timeframe": req.Timeframe,
							"json_key":  jsonResult.Key,
						}).Info("JSON data uploaded successfully")
					}
				}
			}
		}
	}

	s.logger.WithFields(logrus.Fields{
		"symbol":    req.Symbol,
		"market":    req.Market,
		"timeframe": req.Timeframe,
		"cdn_url":   uploadResult.URL,
	}).Info("Screenshot completed successfully")

	// 生成CDN URL
	cdnURL := s.generateCDNURL(uploadResult.Key)

	response := &ScreenshotResponse{
		Success:   true,
		Message:   "Screenshot taken successfully",
		CDNURL:    cdnURL,
		S3URL:     uploadResult.Key, // 这里存储S3 key而不是URL
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 如果有JSON数据，添加到响应中
	if jsonResult != nil {
		dataCDNURL := s.generateCDNURL(jsonResult.Key)
		response.DataCDNURL = dataCDNURL
		response.DataS3URL = jsonResult.Key
		s.logger.WithFields(logrus.Fields{
			"symbol":       req.Symbol,
			"market":       req.Market,
			"timeframe":    req.Timeframe,
			"data_cdn_url": dataCDNURL,
		}).Info("JSON data URL added to response")
	}

	return response, nil
}

// TakeScreenshotWithData 截取股票K线图并下载JSON数据
func (s *Service) TakeScreenshotWithData(ctx context.Context, req *ScreenshotRequest) (*ScreenshotWithDataResponse, error) {
	s.logger.WithFields(logrus.Fields{
		"symbol":    req.Symbol,
		"market":    req.Market,
		"timeframe": req.Timeframe,
	}).Info("Taking screenshot with data using chart service")

	// 格式化股票代码
	formattedSymbol := s.formatSymbolForMarket(req.Symbol, req.Market)

	// 1. 获取截图
	chartImage, err := s.chartService.TakeScreenshotWithRefresh(ctx, formattedSymbol, req.Timeframe)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get chart image from chart service")
		return &ScreenshotWithDataResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to get chart image: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	// 2. 获取JSON数据
	panelData, err := s.chartService.GetPanelData(ctx, formattedSymbol, req.Timeframe)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get panel data, will continue without JSON data")
	}

	// 生成截图文件名
	screenshotFileName := s.generateScreenshotFileName(req.Symbol, req.Market, req.Timeframe)
	if screenshotFileName == "" {
		return &ScreenshotWithDataResponse{
			Success:   false,
			Message:   "Failed to generate screenshot filename",
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	// 创建临时文件
	tempDir := os.TempDir()
	screenshotTempFile := filepath.Join(tempDir, screenshotFileName)

	// 写入图片数据到临时文件
	if err := os.WriteFile(screenshotTempFile, chartImage.Data, 0o644); err != nil {
		s.logger.WithError(err).Error("Failed to write image to temp file")
		return &ScreenshotWithDataResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to write image to temp file: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	// 确保临时文件被清理
	defer func() {
		if err := os.Remove(screenshotTempFile); err != nil {
			s.logger.WithError(err).Warn("Failed to remove screenshot temp file")
		}
	}()

	// 上传截图到S3
	screenshotResult, err := s.s3Client.UploadScreenshot(ctx, screenshotTempFile, req.Symbol, req.Market, req.Timeframe)
	if err != nil {
		s.logger.WithError(err).Error("Failed to upload screenshot to S3")
		return &ScreenshotWithDataResponse{
			Success:   false,
			Message:   fmt.Sprintf("Failed to upload screenshot to S3: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		}, nil
	}

	s.logger.WithFields(logrus.Fields{
		"symbol":    req.Symbol,
		"market":    req.Market,
		"timeframe": req.Timeframe,
		"cdn_url":   screenshotResult.URL,
	}).Info("Screenshot with data completed successfully")

	// 生成CDN URL
	cdnURL := s.generateCDNURL(screenshotResult.Key)

	response := &ScreenshotWithDataResponse{
		Success:   true,
		Message:   "Screenshot with data taken successfully",
		CDNURL:    cdnURL,
		S3URL:     screenshotResult.Key, // 这里存储S3 key而不是URL
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// 如果有JSON数据，上传到S3
	if panelData != nil && panelData.Success {
		// 生成JSON文件名
		jsonFileName := s.generateJSONFileName(req.Symbol, req.Market, req.Timeframe)
		if jsonFileName != "" {
			jsonTempFile := filepath.Join(tempDir, jsonFileName)

			// 将JSON数据写入临时文件
			jsonData, err := json.Marshal(panelData.Data)
			if err != nil {
				s.logger.WithError(err).Warn("Failed to marshal JSON data")
			} else {
				if err := os.WriteFile(jsonTempFile, jsonData, 0o644); err != nil {
					s.logger.WithError(err).Warn("Failed to write JSON to temp file")
				} else {
					// 确保JSON临时文件被清理
					defer func() {
						if err := os.Remove(jsonTempFile); err != nil {
							s.logger.WithError(err).Warn("Failed to remove JSON temp file")
						}
					}()

					// 上传JSON到S3
					jsonResult, err := s.s3Client.UploadJSONData(ctx, jsonTempFile, req.Symbol, req.Market, req.Timeframe)
					if err != nil {
						s.logger.WithError(err).Warn("Failed to upload JSON data to S3")
					} else {
						dataCDNURL := s.generateCDNURL(jsonResult.Key)
						response.DataCDNURL = dataCDNURL
						response.DataS3URL = jsonResult.Key
						s.logger.WithFields(logrus.Fields{
							"symbol":       req.Symbol,
							"market":       req.Market,
							"timeframe":    req.Timeframe,
							"data_cdn_url": dataCDNURL,
						}).Info("JSON data uploaded successfully")
					}
				}
			}
		}
	}

	return response, nil
}

// SetupRoutes 设置路由
func (s *Service) SetupRoutes(r *gin.Engine) {
	// 健康检查 - 支持GET和HEAD请求
	r.GET("/health", func(c *gin.Context) {
		// 检查图表服务是否可用
		chartServiceStatus := "healthy"
		if s.chartService != nil {
			// 可以添加图表服务的健康检查逻辑
			chartServiceStatus = "available"
		}

		c.JSON(http.StatusOK, gin.H{
			"status":        "healthy",
			"time":          time.Now().Format(time.RFC3339),
			"chart_service": chartServiceStatus,
		})
	})

	r.HEAD("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// API路由组
	api := r.Group("/api/v1")
	{
		// 截图API
		api.POST("/screenshot", s.handleScreenshot)
		api.GET("/screenshot/:symbol/:market/:timeframe", s.handleScreenshotGet)

		// 带数据的截图API
		api.POST("/screenshot-with-data", s.handleScreenshotWithData)
		api.GET("/screenshot-with-data/:symbol/:market/:timeframe", s.handleScreenshotWithDataGet)

		// 状态监控API
		api.GET("/status", func(c *gin.Context) {
			// 图表服务状态
			chartServiceStatus := "unavailable"
			if s.chartService != nil {
				chartServiceStatus = "available"
			}

			c.JSON(http.StatusOK, gin.H{
				"chart_service_status": chartServiceStatus,
				"chart_service_url":    s.config.ChartService.BaseURL,
				"timestamp":            time.Now().Format(time.RFC3339),
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

// handleScreenshotWithData POST /api/v1/screenshot-with-data
func (s *Service) handleScreenshotWithData(c *gin.Context) {
	var req ScreenshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ScreenshotWithDataResponse{
			Success:   false,
			Message:   fmt.Sprintf("Invalid request: %v", err),
			Timestamp: time.Now().Format(time.RFC3339),
		})
		return
	}

	response, err := s.TakeScreenshotWithData(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ScreenshotWithDataResponse{
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

// handleScreenshotWithDataGet GET /api/v1/screenshot-with-data/:symbol/:market/:timeframe
func (s *Service) handleScreenshotWithDataGet(c *gin.Context) {
	symbol := c.Param("symbol")
	market := c.Param("market")
	timeframe := c.Param("timeframe")

	if symbol == "" || market == "" || timeframe == "" {
		c.JSON(http.StatusBadRequest, ScreenshotWithDataResponse{
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

	response, err := s.TakeScreenshotWithData(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ScreenshotWithDataResponse{
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

// formatSymbolForMarket 根据市场类型格式化股票代码
func (s *Service) formatSymbolForMarket(symbol, market string) string {
	switch market {
	case "hk":
		// 港股需要添加.HK后缀
		if !strings.HasSuffix(symbol, ".HK") {
			return symbol + ".HK"
		}
		return symbol
	case "cn":
		// A股需要添加.SS后缀
		if !strings.HasSuffix(symbol, ".SS") {
			return symbol + ".SS"
		}
		return symbol
	case "us":
		// 美股保持原样
		return symbol
	default:
		return symbol
	}
}

// generateScreenshotFileName 生成截图文件名
func (s *Service) generateScreenshotFileName(symbol, market, timeframe string) string {
	now := time.Now()

	switch timeframe {
	case "1d":
		// 日线：同一天内一支股票只有一张，格式：{symbol}_{market}_1d_{date}.png
		return fmt.Sprintf("%s_%s_1d_%s.png", symbol, market, now.Format("20060102"))
	case "1h":
		// 小时线：{symbol}_{market}_1h_{date}_{hour}.png
		return fmt.Sprintf("%s_%s_1h_%s_%02d.png", symbol, market, now.Format("20060102"), now.Hour())
	case "1wk":
		// 周线：同一周内一支股票只有一张，格式：{symbol}_{market}_1wk_{year}_{week}.png
		year, week := now.ISOWeek()
		return fmt.Sprintf("%s_%s_1wk_%d_%02d.png", symbol, market, year, week)
	default:
		// 其他时间框架使用时间戳（保持向后兼容）
		return fmt.Sprintf("%s_%s_%s_%s.png", symbol, market, timeframe, now.Format("20060102_150405"))
	}
}

// generateJSONFileName 生成JSON文件名
func (s *Service) generateJSONFileName(symbol, market, timeframe string) string {
	now := time.Now()

	switch timeframe {
	case "1d":
		// 日线：同一天内一支股票只有一张，格式：{symbol}_{market}_1d_{date}.json
		return fmt.Sprintf("%s_%s_1d_%s.json", symbol, market, now.Format("20060102"))
	case "1h":
		// 小时线：{symbol}_{market}_1h_{date}_{hour}.json
		return fmt.Sprintf("%s_%s_1h_%s_%02d.json", symbol, market, now.Format("20060102"), now.Hour())
	case "1wk":
		// 周线：同一周内一支股票只有一张，格式：{symbol}_{market}_1wk_{year}_{week}.json
		year, week := now.ISOWeek()
		return fmt.Sprintf("%s_%s_1wk_%d_%02d.json", symbol, market, year, week)
	default:
		// 其他时间框架使用时间戳（保持向后兼容）
		return fmt.Sprintf("%s_%s_%s_%s.json", symbol, market, timeframe, now.Format("20060102_150405"))
	}
}

// generateCDNURL 生成CDN URL
func (s *Service) generateCDNURL(s3Key string) string {
	// 如果CDN配置为空，返回S3 URL
	if s.config.CDN.BaseURL == "" || s.config.CDN.BaseURL == "https://your-cdn-domain.com" {
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
			s.config.S3.Bucket, s.config.S3.Region, s3Key)
	}

	// 从S3 key中提取文件名部分（去掉screenshot/前缀）
	// S3 key格式: screenshot/screenshots/PDD_us_1h_20250808_01.png
	// 我们需要提取: screenshots/PDD_us_1h_20250808_01.png
	fileName := s3Key
	if strings.HasPrefix(s3Key, "screenshot/") {
		fileName = strings.TrimPrefix(s3Key, "screenshot/")
	}

	// 生成CDN URL
	return fmt.Sprintf("%s/%s", s.config.CDN.BaseURL, fileName)
}

// Close 关闭服务
func (s *Service) Close() {
	// 由于不再使用浏览器服务，这里不需要关闭任何资源
	s.logger.Info("Screenshot service closed")
}
