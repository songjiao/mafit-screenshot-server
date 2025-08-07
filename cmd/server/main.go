package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"

	"makeprofit/internal/config"
	"makeprofit/internal/server"
	"makeprofit/pkg/utils"
)

func main() {
	// 解析命令行参数
	configPath := flag.String("config", "configs/config.yaml", "配置文件路径")
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

	// 创建服务器
	srv := server.NewServer(cfg)
	if srv == nil {
		log.Fatal("Failed to create server")
	}

	// 启动服务器
	addr := cfg.Server.Host + ":" + strconv.Itoa(cfg.Server.Port)
	logger.Infof("Starting server on %s", addr)

	if err := http.ListenAndServe(addr, srv.Router); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
