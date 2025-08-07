package middleware

import (
	"net/http"

	"makeprofit/pkg/utils"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger := utils.GetLogger()

		if err, ok := recovered.(string); ok {
			logger.WithField("error", err).Error("Panic recovered")
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
		})
	})
}
