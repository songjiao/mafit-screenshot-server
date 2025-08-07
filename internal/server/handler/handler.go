package handler

import (
	"net/http"
	"time"

	"makeprofit/internal/browser"
	"makeprofit/internal/config"
	"makeprofit/pkg/models"
	"makeprofit/pkg/utils"

	"github.com/gin-gonic/gin"
)

var cfg *config.Config
var browserService *browser.BrowserService

// InitHandler 初始化处理器
func InitHandler(config *config.Config, bs *browser.BrowserService) {
	cfg = config
	browserService = bs
}

// Index 主页面
func Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "股票截图服务",
	})
}

// AnalysisPage 截图页面
func AnalysisPage(c *gin.Context) {
	symbolMarket := c.Param("symbolMarket")

	c.HTML(http.StatusOK, "analysis.html", gin.H{
		"title":        "股票截图",
		"symbolMarket": symbolMarket,
	})
}

// CreateAnalysis 创建截图任务
func CreateAnalysis(c *gin.Context) {
	symbolMarket := c.Param("symbolMarket")

	// 解析股票代码和市场
	symbol, market, err := utils.ParseSymbolMarket(symbolMarket)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    "INVALID_SYMBOL_MARKET",
			Message: "无效的股票代码和市场格式",
			Details: err.Error(),
		})
		return
	}

	// 生成任务ID
	taskID := utils.GenerateTaskKey(symbol, market)

	// 创建响应
	response := models.AnalysisResponse{
		TaskID:              taskID,
		Status:              "pending",
		CreatedAt:           utils.FormatDateTime(utils.EstimateCompletionTime()),
		EstimatedCompletion: utils.FormatDateTime(utils.EstimateCompletionTime()),
		ResultURL:           cfg.CDN.BaseURL + "/" + cfg.CDN.ResultPath + "/" + symbolMarket + "/" + taskID[len(taskID)-10:],
	}

	c.JSON(http.StatusOK, response)
}

// StreamStatus SSE状态推送
func StreamStatus(c *gin.Context) {
	taskID := c.Param("taskId")

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// 创建SSE连接
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SSE not supported"})
		return
	}

	// 模拟状态推送
	statuses := []models.TaskStatus{
		{TaskID: taskID, Status: "processing", Message: "准备开始...", Percentage: 0},
		{TaskID: taskID, Status: "processing", Message: "启动浏览器...", Percentage: 20},
		{TaskID: taskID, Status: "processing", Message: "开始截图日线图...", Percentage: 40},
		{TaskID: taskID, Status: "processing", Message: "开始截图小时图...", Percentage: 60},
		{TaskID: taskID, Status: "processing", Message: "截图完成，上传到S3...", Percentage: 80},
		{TaskID: taskID, Status: "completed", Message: "任务完成", Percentage: 100},
	}

	for i, status := range statuses {
		data := "data: " + status.ToJSON() + "\n\n"
		c.Writer.WriteString(data)
		flusher.Flush()

		if status.Status == "completed" {
			break
		}

		// 模拟处理时间
		if i < len(statuses)-1 {
			time.Sleep(2 * time.Second)
		}
	}
}
