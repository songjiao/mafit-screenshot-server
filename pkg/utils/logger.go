package utils

import (
	"os"

	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()

	// 设置日志格式
	logger.SetFormatter(&logrus.JSONFormatter{})

	// 设置日志级别
	logger.SetLevel(logrus.InfoLevel)

	// 设置输出
	logger.SetOutput(os.Stdout)
}

// GetLogger 获取日志实例
func GetLogger() *logrus.Logger {
	return logger
}

// SetLogLevel 设置日志级别
func SetLogLevel(level string) {
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "info":
		logger.SetLevel(logrus.InfoLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}
}

// SetLogFormat 设置日志格式
func SetLogFormat(format string) {
	switch format {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{})
	default:
		logger.SetFormatter(&logrus.JSONFormatter{})
	}
}
