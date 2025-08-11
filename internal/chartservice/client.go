package chartservice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"makeprofit/pkg/utils"

	"github.com/sirupsen/logrus"
)

// Client 本地图表服务客户端
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewClient 创建新的本地图表服务客户端
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: utils.GetLogger(),
	}
}

// PanelData 面板数据响应
type PanelData struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// ChartImage 图表图片响应
type ChartImage struct {
	Data []byte
	Type string
}

// RefreshResponse 刷新响应
type RefreshResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// SaveChartRequest 保存图表请求
type SaveChartRequest struct {
	FilePath string `json:"filepath"`
}

// GetPanelData 获取静态面板数据
func (c *Client) GetPanelData(ctx context.Context, symbol, duration string) (*PanelData, error) {
	url := fmt.Sprintf("%s/kline/panel/%s/%s", c.baseURL, symbol, duration)

	c.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"duration": duration,
		"url":      url,
	}).Info("Getting panel data")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 尝试解析为JSON数组（图表服务返回的是数组格式）
	var dataArray interface{}
	if err := json.Unmarshal(body, &dataArray); err != nil {
		return nil, fmt.Errorf("failed to decode response as JSON: %w", err)
	}

	// 构造PanelData响应
	panelData := &PanelData{
		Success: true,
		Data:    dataArray,
		Message: "Data retrieved successfully",
	}

	c.logger.WithFields(logrus.Fields{
		"symbol":    symbol,
		"duration":  duration,
		"data_type": fmt.Sprintf("%T", dataArray),
	}).Info("Panel data retrieved successfully")

	return panelData, nil
}

// RefreshKlineData 刷新K线数据
func (c *Client) RefreshKlineData(ctx context.Context, symbol, duration string) (*RefreshResponse, error) {
	url := fmt.Sprintf("%s/kline/refresh/%s/%s", c.baseURL, symbol, duration)

	c.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"duration": duration,
		"url":      url,
	}).Info("Refreshing kline data")

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var refreshResp RefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&refreshResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &refreshResp, nil
}

// GetChartImage 获取图表图片
func (c *Client) GetChartImage(ctx context.Context, symbol, duration string) (*ChartImage, error) {
	url := fmt.Sprintf("%s/kline/chart/%s/%s", c.baseURL, symbol, duration)

	c.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"duration": duration,
		"url":      url,
	}).Info("Getting chart image")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 读取图片数据
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// 获取Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/png" // 默认假设是PNG
	}

	return &ChartImage{
		Data: imageData,
		Type: contentType,
	}, nil
}

// SaveChartImage 保存图表图片到指定路径
func (c *Client) SaveChartImage(ctx context.Context, symbol, duration, filepath string) (*RefreshResponse, error) {
	url := fmt.Sprintf("%s/kline/chart/%s/%s", c.baseURL, symbol, duration)

	c.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"duration": duration,
		"filepath": filepath,
		"url":      url,
	}).Info("Saving chart image")

	// 准备请求数据
	requestData := SaveChartRequest{
		FilePath: filepath,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var saveResp RefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&saveResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &saveResp, nil
}

// TakeScreenshotWithRefresh 先刷新K线数据，然后获取图表图片
func (c *Client) TakeScreenshotWithRefresh(ctx context.Context, symbol, duration string) (*ChartImage, error) {
	c.logger.WithFields(logrus.Fields{
		"symbol":   symbol,
		"duration": duration,
	}).Info("Taking screenshot with refresh")

	// 1. 先刷新K线数据
	refreshResp, err := c.RefreshKlineData(ctx, symbol, duration)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to refresh kline data, will continue with current data")
	} else {
		c.logger.WithFields(logrus.Fields{
			"symbol":   symbol,
			"duration": duration,
			"success":  refreshResp.Success,
			"message":  refreshResp.Message,
		}).Info("Kline data refresh completed")
	}

	// 2. 获取图表图片
	chartImage, err := c.GetChartImage(ctx, symbol, duration)
	if err != nil {
		return nil, fmt.Errorf("failed to get chart image: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"symbol":       symbol,
		"duration":     duration,
		"image_size":   len(chartImage.Data),
		"content_type": chartImage.Type,
	}).Info("Chart image retrieved successfully")

	return chartImage, nil
}
