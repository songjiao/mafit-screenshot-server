package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"makeprofit/internal/config"
	"makeprofit/internal/s3"
	"makeprofit/pkg/utils"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
)

// ScreenshotClient 截图客户端
type ScreenshotClient struct {
	browserPool   *BrowserPool
	config        *config.MafitConfig
	browserConfig *config.BrowserConfig
	s3Client      *s3.Client
	taskManager   *TaskManager
}

// NewScreenshotClient 创建新的截图客户端
func NewScreenshotClient(browserPool *BrowserPool, cfg *config.MafitConfig, browserCfg *config.BrowserConfig, s3Client *s3.Client, taskManager *TaskManager) *ScreenshotClient {
	return &ScreenshotClient{
		browserPool:   browserPool,
		config:        cfg,
		browserConfig: browserCfg,
		s3Client:      s3Client,
		taskManager:   taskManager,
	}
}

// TakeScreenshot 方法已被删除，使用 TakeScreenshotWithViewport 替代

// TakeScreenshotWithViewport 截取指定视口大小的截图
func (sc *ScreenshotClient) TakeScreenshotWithViewport(url, outputPath string, width, height int) error {
	logger := utils.GetLogger()

	// 从浏览器池获取浏览器实例
	pooledBrowser, err := sc.browserPool.GetBrowser(context.Background())
	if err != nil {
		logger.WithError(err).Error("Failed to get browser from pool")
		return fmt.Errorf("failed to get browser from pool: %w", err)
	}
	defer sc.browserPool.ReturnBrowser(pooledBrowser)

	// 使用池中的页面
	page := pooledBrowser.Page

	// 设置请求头
	sc.setRequestHeaders(page)

	// 设置页面超时
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 导航到目标页面
	logger.Infof("Navigating to: %s", url)
	if err := page.Context(ctx).Navigate(url); err != nil {
		logger.WithError(err).Error("Failed to navigate to page")
		return fmt.Errorf("failed to navigate to page: %w", err)
	}

	// 等待页面基本加载
	logger.Info("Waiting for page to load...")
	if err := page.WaitLoad(); err != nil {
		logger.WithError(err).Warn("Page load timeout, continuing anyway")
	}

	// 检查当前页面URL
	logger.Info("Checking current page URL...")
	currentURL, err := page.Eval(`() => window.location.href`)
	if err == nil {
		logger.Infof("Current page URL: %s", currentURL.Value.String())

		// 检查是否被重定向到登录页面
		if strings.Contains(currentURL.Value.String(), "/login") {
			logger.Error("Page was redirected to login page")
			return fmt.Errorf("page was redirected to login page, authentication may have failed")
		}
	}

	// 点击刷新按钮并等待K线图加载完成
	logger.Info("Clicking refresh button and waiting for chart data to load...")
	if err := sc.clickRefreshAndWaitForLoad(page); err != nil {
		logger.WithError(err).Error("Failed to refresh chart data")
		return fmt.Errorf("failed to refresh chart data: %w", err)
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		logger.WithError(err).Error("Failed to create output directory")
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 截取全屏截图
	logger.Infof("Taking screenshot with viewport: %s", outputPath)
	page.MustScreenshot(outputPath)

	// 截取全屏截图
	logger.Infof("Taking screenshot with viewport: %s", outputPath)
	page.MustScreenshot(outputPath)

	logger.Infof("Viewport screenshot saved: %s", outputPath)
	return nil
}

// TakeMafitScreenshot 截取mafit.fun的K线图
func (sc *ScreenshotClient) TakeMafitScreenshot(symbol, market, timeframe, outputPath string) error {
	// 根据市场类型格式化股票代码
	formattedSymbol := sc.formatSymbolForMarket(symbol, market)

	// 构建URL
	url := fmt.Sprintf("%s/apps/quote/folder/%s/%s/%s",
		sc.config.BaseURL, market, formattedSymbol, timeframe)

	logger := utils.GetLogger()
	logger.Infof("Taking mafit screenshot: %s", url)

	// 使用1920x1080的视口大小
	return sc.TakeScreenshotWithViewport(url, outputPath, 1920, 1080)
}

// formatSymbolForMarket 根据市场类型格式化股票代码
func (sc *ScreenshotClient) formatSymbolForMarket(symbol, market string) string {
	switch market {
	case "hk":
		// 港股需要添加.HK后缀
		if !strings.HasSuffix(symbol, ".HK") {
			return symbol + ".HK"
		}
		return symbol
	case "us":
		// 美股保持原样
		return symbol
	case "cn":
		// A股保持原样
		return symbol
	default:
		return symbol
	}
}

// TakeMafitScreenshotAndUpload 截取mafit.fun的K线图并上传到S3
func (sc *ScreenshotClient) TakeMafitScreenshotAndUpload(ctx context.Context, symbol, market, timeframe string) (*s3.UploadResult, error) {
	logger := utils.GetLogger()

	// 生成任务键和CDN URL
	taskKey := sc.taskManager.generateTaskKey(symbol, market, timeframe)
	cdnURL := sc.taskManager.generateCDNURL(symbol, market, timeframe)

	// 检查CDN上是否已存在图片
	if sc.taskManager.checkCDNExists(cdnURL) {
		logger.WithFields(logrus.Fields{
			"symbol":    symbol,
			"market":    market,
			"timeframe": timeframe,
			"cdn_url":   cdnURL,
		}).Info("Screenshot already exists on CDN")

		// 返回CDN URL作为结果
		return &s3.UploadResult{
			URL:      cdnURL,
			Key:      "", // CDN URL，不需要S3 key
			Size:     0,  // 未知大小
			Uploaded: time.Now(),
		}, nil
	}

	// 检查是否有相同的任务正在运行
	if sc.taskManager.isTaskRunning(taskKey) {
		logger.WithFields(logrus.Fields{
			"symbol":    symbol,
			"market":    market,
			"timeframe": timeframe,
		}).Info("Task is already running, waiting for completion")

		// 等待任务完成
		if err := sc.taskManager.waitForTask(taskKey, 5*time.Minute); err != nil {
			return nil, fmt.Errorf("wait for task completion failed: %w", err)
		}

		// 再次检查CDN
		if sc.taskManager.checkCDNExists(cdnURL) {
			logger.Info("Screenshot was uploaded by another task")
			return &s3.UploadResult{
				URL:      cdnURL,
				Key:      "",
				Size:     0,
				Uploaded: time.Now(),
			}, nil
		}
	}

	// 检查任务是否已完成
	if sc.taskManager.isTaskCompleted(taskKey) {
		logger.WithFields(logrus.Fields{
			"symbol":    symbol,
			"market":    market,
			"timeframe": timeframe,
		}).Info("Task is already completed, checking CDN")

		// 检查CDN是否存在
		if sc.taskManager.checkCDNExists(cdnURL) {
			logger.Info("Screenshot was already completed by another task")
			return &s3.UploadResult{
				URL:      cdnURL,
				Key:      "",
				Size:     0,
				Uploaded: time.Now(),
			}, nil
		}
	}

	// 开始新任务
	_, err := sc.taskManager.startTask(taskKey)
	if err != nil {
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	// 确保任务完成后清理（无论成功还是失败）
	defer func() {
		if r := recover(); r != nil {
			// 发生panic时也要清理任务
			sc.taskManager.failTask(taskKey)
			panic(r) // 重新抛出panic
		} else {
			// 正常完成时清理任务
			sc.taskManager.completeTask(taskKey)
		}
	}()

	// 构建URL
	url := fmt.Sprintf("%s/apps/quote/folder/%s/%s/%s",
		sc.config.BaseURL, market, symbol, timeframe)

	logger.Infof("Taking mafit screenshot and uploading: %s", url)

	// 生成文件名
	fileName := sc.generateScreenshotFileName(symbol, market, timeframe)
	if fileName == "" {
		return nil, fmt.Errorf("failed to generate filename for %s %s %s", symbol, market, timeframe)
	}

	// 创建临时文件路径
	tempDir := os.TempDir()
	tempFile := filepath.Join(tempDir, fileName)

	// 截取截图
	if err := sc.TakeScreenshotWithViewport(url, tempFile, 1920, 1080); err != nil {
		// 截图失败时标记任务失败
		sc.taskManager.failTask(taskKey)
		return nil, fmt.Errorf("failed to take screenshot: %w", err)
	}

	// 确保临时文件被清理
	defer func() {
		if err := os.Remove(tempFile); err != nil {
			logger.WithError(err).Warn("Failed to remove temp file")
		}
	}()

	// 检查S3客户端是否可用
	if sc.s3Client == nil {
		sc.taskManager.failTask(taskKey)
		return nil, fmt.Errorf("S3 client not configured")
	}

	// 上传到S3
	uploadResult, err := sc.s3Client.UploadScreenshot(ctx, tempFile, symbol, market, timeframe)
	if err != nil {
		sc.taskManager.failTask(taskKey)
		return nil, fmt.Errorf("failed to upload screenshot to S3: %w", err)
	}

	// 将S3 URL转换为CDN URL
	cdnResult := &s3.UploadResult{
		URL:      cdnURL,
		Key:      uploadResult.Key,
		Size:     uploadResult.Size,
		Uploaded: uploadResult.Uploaded,
	}

	logger.WithFields(logrus.Fields{
		"symbol":    symbol,
		"market":    market,
		"timeframe": timeframe,
		"s3_url":    uploadResult.URL,
		"cdn_url":   cdnURL,
		"s3_key":    uploadResult.Key,
	}).Info("Screenshot uploaded to S3 and available on CDN")

	return cdnResult, nil
}

// setRequestHeaders 设置请求头
func (sc *ScreenshotClient) setRequestHeaders(page *rod.Page) {
	// 通过JavaScript设置User-Agent和请求头
	page.MustEvalOnNewDocument(`
		// 设置User-Agent
		Object.defineProperty(navigator, 'userAgent', {
			get: () => '` + sc.browserConfig.UserAgent + `',
		});
		
		// 隐藏webdriver属性
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined,
		});
		
		// 设置语言偏好
		Object.defineProperty(navigator, 'language', {
			get: () => 'zh-CN',
		});
		
		Object.defineProperty(navigator, 'languages', {
			get: () => ['zh-CN', 'zh', 'en'],
		});
		
		// 拦截所有请求并添加请求头
		const originalFetch = window.fetch;
		window.fetch = function(url, options = {}) {
			options.headers = {
				...options.headers,
				'Upgrade-Insecure-Requests': '1',
				'Accept-Language': 'zh-CN,zh;q=0.9,en;q=0.8',
				'sec-ch-ua': '"Not)A;Brand";v="8", "Chromium";v="138", "Google Chrome";v="138"',
				'sec-ch-ua-mobile': '?0',
				'sec-ch-ua-platform': '"Windows"'
			};
			return originalFetch(url, options);
		};
	`)
}

// setLocalStorage 设置localStorage
func (sc *ScreenshotClient) setLocalStorage(page *rod.Page) {
	logger := utils.GetLogger()

	// 检查配置值是否为空或占位符
	if sc.config.JWTAccessToken == "" ||
		sc.config.JWTAccessToken == "your_jwt_access_token_here" ||
		sc.config.SidebarSheet == "" ||
		sc.config.SidebarSheet == "your_sidebar_sheet_here" {
		logger.Warn("JWT access token or sidebar sheet not configured, skipping localStorage setup")
		return
	}

	// 等待页面完全加载
	time.Sleep(2 * time.Second)

	// 使用简单的JavaScript代码设置localStorage
	// 直接使用字符串，避免JSON序列化问题
	_, err := page.Eval(fmt.Sprintf("localStorage.setItem('jwt_access_token', '%s');", sc.config.JWTAccessToken))
	if err != nil {
		logger.WithError(err).Warn("Failed to set jwt_access_token")
	} else {
		logger.Debug("Successfully set jwt_access_token")
	}

	_, err = page.Eval(fmt.Sprintf("localStorage.setItem('sidebarSheet', '%s');", sc.config.SidebarSheet))
	if err != nil {
		logger.WithError(err).Warn("Failed to set sidebarSheet")
	} else {
		logger.Debug("Successfully set sidebarSheet")
	}

	logger.Info("localStorage setup completed")
}

// generateScreenshotFileName 生成截图文件名
func (sc *ScreenshotClient) generateScreenshotFileName(symbol, market, timeframe string) string {
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

// 旧的refreshPageData和waitForRefreshComplete方法已被新的clickRefreshAndWaitForLoad方法替代

// clickRefreshAndWaitForLoad 点击刷新按钮并等待K线图加载完成
func (sc *ScreenshotClient) clickRefreshAndWaitForLoad(page *rod.Page) error {
	logger := utils.GetLogger()

	// 查找刷新按钮 - 使用多种选择器尝试
	logger.Info("Looking for refresh button...")

	// 首先尝试指定的选择器
	simpleSelectors := []string{
		"#\\:rk\\:", // 用户指定的选择器
		"button",
		"button[class*='refresh']",
		"button[class*='Refresh']",
		"button:contains('刷新')",
		"button:contains('Refresh')",
	}

	var refreshButton *rod.Element
	var err error

	// 尝试简单选择器
	for _, selector := range simpleSelectors {
		logger.Debugf("Trying simple selector: %s", selector)
		refreshButton, err = page.Element(selector)
		if err == nil && refreshButton != nil {
			text, _ := refreshButton.Text()
			logger.Infof("Found button with selector '%s', text: '%s'", selector, text)

			// 如果按钮文本包含"刷新"，就使用这个按钮
			if strings.Contains(text, "刷新") || strings.Contains(text, "Refresh") {
				logger.Infof("Found refresh button with selector: %s", selector)
				break
			}
		}
	}

	// 如果还是找不到
	if refreshButton == nil {
		logger.Warn("Refresh button not found with any selector, continuing without refresh...")
		// 不返回错误，继续截图流程
		return nil
	}

	// 获取按钮详细信息用于调试
	buttonText, _ := refreshButton.Text()
	buttonHTML, _ := refreshButton.HTML()
	buttonClass, _ := refreshButton.Attribute("class")
	logger.Infof("Selected button details: text='%s', class='%s', html='%s'", buttonText, buttonClass, buttonHTML)

	// 检查按钮是否可见和可点击
	if !refreshButton.MustVisible() {
		logger.Error("Refresh button is not visible")
		return fmt.Errorf("refresh button is not visible")
	}

	// 获取按钮初始状态
	initialText, _ := refreshButton.Text()
	logger.Infof("Found refresh button with text: %s", initialText)

	// 点击刷新按钮
	logger.Info("Clicking refresh button...")
	if err := refreshButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
		logger.WithError(err).Error("Failed to click refresh button")
		return fmt.Errorf("failed to click refresh button: %w", err)
	}

	// 等待按钮变为disable状态（开始加载）
	logger.Info("Waiting for button to become disabled (loading starts)...")
	time.Sleep(1 * time.Second)

	// 等待按钮重新变为enabled状态（加载完成）
	logger.Info("Waiting for button to become enabled (loading completes)...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 记录初始时间，用于计算加载时间
	loadStartTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for chart data to load")
		default:
			// 检查按钮是否处于disable状态
			isDisabled, _ := refreshButton.Eval(`() => {
				const button = this;
				return button.disabled === true;
			}`)

			// 如果按钮不再处于disable状态，说明加载完成
			if isDisabled == nil || !isDisabled.Value.Bool() {
				loadTime := time.Since(loadStartTime)
				logger.Infof("Chart data loading completed in %v", loadTime)

				// 刷新按钮重新变为enabled状态后，等待1秒确保数据稳定
				logger.Info("Refresh button became enabled, waiting 1s for data stabilization...")
				time.Sleep(1 * time.Second)

				return nil
			}

			loadTime := time.Since(loadStartTime)
			logger.Debugf("Still loading after %v", loadTime)
			time.Sleep(500 * time.Millisecond) // 每500ms检查一次
		}
	}
}
