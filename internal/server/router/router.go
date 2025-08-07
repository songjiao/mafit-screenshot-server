package router

import (
	"makeprofit/internal/server/handler"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine) {
	// 主页面
	r.GET("/", handler.Index)

	// 截图页面
	r.GET("/analysis/:symbolMarket", handler.AnalysisPage)

	// API路由
	api := r.Group("/api")
	{
		// 截图任务
		api.GET("/analysis/:symbolMarket", handler.CreateAnalysis)

		// SSE状态推送
		api.GET("/stream/:taskId", handler.StreamStatus)
	}
}
