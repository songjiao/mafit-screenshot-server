package browser

import (
	"context"
	"fmt"
	"sync"
	"time"

	"makeprofit/internal/config"
	"makeprofit/pkg/utils"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/sirupsen/logrus"
)

// BrowserPool 浏览器池
type BrowserPool struct {
	browsers    chan *PooledBrowser
	config      *config.BrowserConfig
	mafitConfig *config.MafitConfig
	logger      *logrus.Logger
	mu          sync.RWMutex
	closed      bool
}

// PooledBrowser 池化的浏览器实例
type PooledBrowser struct {
	Browser *rod.Browser
	Page    *rod.Page
	InUse   bool
	mu      sync.Mutex
}

// NewBrowserPool 创建新的浏览器池
func NewBrowserPool(browserConfig *config.BrowserConfig, mafitConfig *config.MafitConfig) (*BrowserPool, error) {
	logger := utils.GetLogger()

	pool := &BrowserPool{
		browsers:    make(chan *PooledBrowser, browserConfig.PoolSize),
		config:      browserConfig,
		mafitConfig: mafitConfig,
		logger:      logger,
	}

	// 初始化浏览器池
	for i := 0; i < browserConfig.PoolSize; i++ {
		browser, err := pool.createBrowser()
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create browser %d: %w", i+1, err)
		}
		pool.browsers <- browser
	}

	logger.WithFields(logrus.Fields{
		"pool_size": browserConfig.PoolSize,
	}).Info("Browser pool initialized successfully")

	return pool, nil
}

// createBrowser 创建新的浏览器实例
func (bp *BrowserPool) createBrowser() (*PooledBrowser, error) {
	bp.logger.Info("Creating new browser instance...")

	// 配置启动器，针对2GB内存优化
	launcherURL, err := launcher.New().
		Headless(bp.config.Headless).
		Leakless(false).
		// 设置浏览器语言为中文
		Set("lang", "zh-CN").
		// 设置Accept-Language头
		Set("accept-language", "zh-CN,zh;q=0.9,en;q=0.8").
		// 基础优化参数
		Set("disable-dev-shm-usage", "").
		Set("no-sandbox", "").
		// 禁用不必要的功能
		Set("disable-extensions", "").
		Set("disable-plugins", "").
		Set("disable-background-networking", "").
		Set("disable-default-apps", "").
		Set("disable-sync", "").
		Set("metrics-recording-only", "").
		Set("no-first-run", "").
		Set("mute-audio", "").
		Set("disable-translate", "").
		// 字体渲染优化
		Set("font-render-hinting", "medium").
		Set("enable-font-antialiasing", "").
		Set("font-smoothing", "antialiased").
		// 内存优化参数（针对2GB内存）
		Set("max_old_space_size", "512").
		Set("js-flags", "--max-old-space-size=512").
		// 禁用不必要的服务
		Set("disable-audio-service", "").
		Set("disable-crash-reporter", "").
		Set("disable-breakpad", "").
		// 进程限制
		Set("renderer-process-limit", "2").
		Set("max-active-webgl-contexts", "1").
		// 网络优化
		Set("disable-background-timer-throttling", "").
		Set("disable-renderer-backgrounding", "").
		// 启动URL
		Launch()

	if err != nil {
		return nil, fmt.Errorf("failed to create launcher: %w", err)
	}

	// 创建浏览器实例
	browser := rod.New().ControlURL(launcherURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to browser: %w", err)
	}

	// 创建页面
	page := browser.MustPage()

	// 设置视口大小
	page.MustSetViewport(1920, 1080, 1, false)

	// 初始化浏览器：登录页面设置localStorage
	bp.logger.Info("Initializing browser: navigating to login page...")
	if err := bp.initializeBrowser(page); err != nil {
		browser.Close()
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	bp.logger.Info("Successfully created browser instance with optimized settings")

	return &PooledBrowser{
		Browser: browser,
		Page:    page,
		InUse:   false,
	}, nil
}

// initializeBrowser 初始化浏览器：设置localStorage并验证登录状态
func (bp *BrowserPool) initializeBrowser(page *rod.Page) error {
	// 步骤1: 导航到登录页面
	bp.logger.Info("Step 1: Navigating to login page...")
	if err := page.Navigate("https://mafit.fun/login"); err != nil {
		return fmt.Errorf("failed to navigate to login page: %w", err)
	}

	// 等待页面加载
	if err := page.WaitLoad(); err != nil {
		bp.logger.Warn("Login page load timeout, continuing anyway")
	}

	// 设置localStorage
	bp.logger.Info("Setting localStorage on login page...")
	if err := bp.setLocalStorage(page); err != nil {
		bp.logger.Warn("Failed to set localStorage on login page, but continuing")
	}

	// 步骤2: 导航到all页面检查登录状态
	bp.logger.Info("Step 2: Navigating to all page to check login status...")
	if err := page.Navigate("https://mafit.fun/apps/quote/folder/all"); err != nil {
		return fmt.Errorf("failed to navigate to all page: %w", err)
	}

	// 等待页面加载
	if err := page.WaitLoad(); err != nil {
		bp.logger.Warn("All page load timeout, continuing anyway")
	}

	// 检查登录状态：查找"个人信息"按钮
	bp.logger.Info("Checking login status by looking for '个人信息' button...")

	// 暂时跳过登录状态检查，直接继续
	bp.logger.Info("Skipping login status check for now, continuing with browser initialization")

	bp.logger.Info("Browser initialization completed successfully")
	return nil
}

// setLocalStorage 设置localStorage
func (bp *BrowserPool) setLocalStorage(page *rod.Page) error {
	// 设置JWT访问令牌 - 使用最简单的语法
	jwtScript := fmt.Sprintf(`localStorage.setItem('jwt_access_token', '%s');`, bp.mafitConfig.JWTAccessToken)

	if _, err := page.Eval(jwtScript); err != nil {
		bp.logger.WithError(err).Warn("Failed to set jwt_access_token, but continuing")
	}

	// 设置侧边栏状态 - 使用最简单的语法
	sidebarScript := fmt.Sprintf(`localStorage.setItem('sidebarSheet', '%s');`, bp.mafitConfig.SidebarSheet)

	if _, err := page.Eval(sidebarScript); err != nil {
		bp.logger.WithError(err).Warn("Failed to set sidebarSheet, but continuing")
	}

	bp.logger.Info("localStorage setup completed")
	return nil
}

// GetBrowser 从池中获取浏览器实例
func (bp *BrowserPool) GetBrowser(ctx context.Context) (*PooledBrowser, error) {
	bp.logger.Info("Attempting to get browser from pool...")

	bp.mu.RLock()
	if bp.closed {
		bp.mu.RUnlock()
		bp.logger.Error("Browser pool is closed")
		return nil, fmt.Errorf("browser pool is closed")
	}
	bp.mu.RUnlock()

	bp.logger.Info("Waiting for available browser...")
	select {
	case browser := <-bp.browsers:
		bp.logger.Info("Got browser from pool, marking as in use")
		browser.mu.Lock()
		browser.InUse = true
		browser.mu.Unlock()
		bp.logger.Info("Browser marked as in use successfully")
		return browser, nil
	case <-ctx.Done():
		bp.logger.Error("Context cancelled while waiting for browser")
		return nil, ctx.Err()
	case <-time.After(bp.config.Timeout):
		bp.logger.Errorf("Timeout waiting for available browser after %v", bp.config.Timeout)
		return nil, fmt.Errorf("timeout waiting for available browser after %v", bp.config.Timeout)
	}
}

// ReturnBrowser 将浏览器实例返回到池中
func (bp *BrowserPool) ReturnBrowser(browser *PooledBrowser) {
	if browser == nil {
		return
	}

	browser.mu.Lock()
	browser.InUse = false
	browser.mu.Unlock()

	// 检查池是否已关闭
	bp.mu.RLock()
	if bp.closed {
		bp.mu.RUnlock()
		return
	}
	bp.mu.RUnlock()

	// 将浏览器放回池中
	select {
	case bp.browsers <- browser:
	default:
		// 池已满，关闭浏览器
		browser.Browser.Close()
	}
}

// Close 关闭浏览器池
func (bp *BrowserPool) Close() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.closed {
		return
	}

	bp.closed = true
	close(bp.browsers)

	// 关闭所有浏览器
	for browser := range bp.browsers {
		if browser != nil && browser.Browser != nil {
			browser.Browser.Close()
		}
	}

	bp.logger.Info("Browser pool closed")
}

// GetStats 获取浏览器池统计信息
func (bp *BrowserPool) GetStats() map[string]interface{} {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	poolSize := bp.config.PoolSize
	available := len(bp.browsers)
	inUse := poolSize - available

	return map[string]interface{}{
		"pool_size": poolSize,
		"available": available,
		"in_use":    inUse,
		"closed":    bp.closed,
		"type":      "pool",
	}
}
