package server

import (
	"fmt"
	"makeprofit/internal/browser"
	"makeprofit/internal/config"
	"makeprofit/internal/server/handler"
	"makeprofit/internal/server/middleware"
	"makeprofit/internal/server/router"

	"github.com/gin-gonic/gin"
)

type Server struct {
	Config         *config.Config
	Router         *gin.Engine
	BrowserService *browser.BrowserService
}

func NewServer(cfg *config.Config) *Server {
	// 设置Gin模式
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin引擎
	engine := gin.New()

	// 添加中间件
	engine.Use(middleware.Logger())
	engine.Use(middleware.Recovery())
	engine.Use(middleware.CORS())

	// 设置静态文件服务
	engine.Static("/static", cfg.Server.StaticPath)
	engine.LoadHTMLGlob(cfg.Server.TemplatePath + "/*")

	// 创建浏览器服务
	browserService, err := browser.NewBrowserService(cfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to create browser service: %v", err))
	}

	// 初始化处理器
	handler.InitHandler(cfg, browserService)

	// 设置路由
	router.SetupRoutes(engine)

	return &Server{
		Config:         cfg,
		Router:         engine,
		BrowserService: browserService,
	}
}
