package middleware

import (
	"time"

	"makeprofit/pkg/utils"

	"github.com/gin-gonic/gin"
)

func Logger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger := utils.GetLogger()

		logger.WithFields(map[string]interface{}{
			"timestamp": param.TimeStamp.Format(time.RFC3339),
			"status":    param.StatusCode,
			"latency":   param.Latency,
			"client_ip": param.ClientIP,
			"method":    param.Method,
			"path":      param.Path,
			"error":     param.ErrorMessage,
		}).Info("HTTP Request")

		return ""
	})
}
