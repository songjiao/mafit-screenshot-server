package browser

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"makeprofit/internal/config"
	"makeprofit/pkg/utils"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/sirupsen/logrus"
)

// BrowserPool 浏览器实例池
type BrowserPool struct {
	browsers    []*PooledBrowser  // 浏览器实例池
	config      *config.BrowserConfig
	mafitConfig *config.MafitConfig
	logger      *logrus.Logger
	mu          sync.RWMutex
	closed      bool
}

// PooledBrowser 池化的浏览器实例
type PooledBrowser struct {
	Browser     *rod.Browser
	config      *config.BrowserConfig
	mafitConfig *config.MafitConfig
	logger      *logrus.Logger
	mu          sync.RWMutex
	closed      bool
	
	// 页面计数
	activePages int32  // 当前活跃页面数
	maxPages    int32  // 最大页面数
}



// NewBrowserPool 创建新的浏览器实例池
func NewBrowserPool(browserConfig *config.BrowserConfig, mafitConfig *config.MafitConfig) (*BrowserPool, error) {
	logger := utils.GetLogger()

	// 设置默认值
	if browserConfig.PoolSize <= 0 {
		browserConfig.PoolSize = 1 // 默认1个浏览器实例
	}
	if browserConfig.MaxConcurrentPages <= 0 {
		browserConfig.MaxConcurrentPages = 20 // 默认每个浏览器最大20个页面
	}
	if browserConfig.RendererProcesses <= 0 {
		browserConfig.RendererProcesses = 16 // 默认16个渲染进程
	}
	if browserConfig.WebGLContexts <= 0 {
		browserConfig.WebGLContexts = 8 // 默认8个WebGL上下文
	}
	if browserConfig.MemoryLimit <= 0 {
		browserConfig.MemoryLimit = 2048 // 默认2048MB内存限制
	}

	pool := &BrowserPool{
		browsers:    make([]*PooledBrowser, 0, browserConfig.PoolSize),
		config:      browserConfig,
		mafitConfig: mafitConfig,
		logger:      logger,
	}

	logger.WithFields(logrus.Fields{
		"pool_size":        browserConfig.PoolSize,
		"max_concurrent_pages": browserConfig.MaxConcurrentPages,
		"renderer_processes": browserConfig.RendererProcesses,
		"webgl_contexts":   browserConfig.WebGLContexts,
		"memory_limit":     browserConfig.MemoryLimit,
	}).Info("Initializing browser instance pool")

	// 初始化浏览器实例池
	for i := 0; i < browserConfig.PoolSize; i++ {
		logger.Infof("Creating browser %d/%d", i+1, browserConfig.PoolSize)
		browser, err := pool.createBrowser()
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create browser %d: %w", i+1, err)
		}
		logger.Infof("Successfully created browser %d, adding to pool", i+1)
		pool.browsers = append(pool.browsers, browser)
		logger.Infof("Browser %d added to pool successfully", i+1)
	}
	
	logger.Infof("Browser pool initialization completed. Pool size: %d, Browsers count: %d", 
		browserConfig.PoolSize, len(pool.browsers))

	logger.WithFields(logrus.Fields{
		"pool_size": browserConfig.PoolSize,
	}).Info("Browser instance pool initialized successfully")

	return pool, nil
}

// createBrowser 创建新的浏览器实例（针对高配服务器优化）
func (bp *BrowserPool) createBrowser() (*PooledBrowser, error) {
	bp.logger.Info("Creating new browser instance with high-performance settings...")

	// 配置启动器，针对高配服务器优化
	launcher := launcher.New().
		Headless(bp.config.Headless).
		Leakless(false)
	
	// 尝试使用系统已安装的Chrome浏览器
	if chromePath := os.Getenv("CHROME_PATH"); chromePath != "" {
		bp.logger.Infof("Using system Chrome browser: %s", chromePath)
		launcher = launcher.Bin(chromePath)
	} else if chromeBin := os.Getenv("CHROME_BIN"); chromeBin != "" {
		bp.logger.Infof("Using system Chrome browser: %s", chromeBin)
		launcher = launcher.Bin(chromeBin)
	} else {
		bp.logger.Info("No system Chrome found, will download browser")
	}
	
	launcherURL, err := launcher.
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
		// 高配服务器内存优化
		Set("max_old_space_size", fmt.Sprintf("%d", bp.config.MemoryLimit)).
		Set("js-flags", fmt.Sprintf("--max-old-space-size=%d", bp.config.MemoryLimit)).
		// 高配服务器进程优化
		Set("renderer-process-limit", fmt.Sprintf("%d", bp.config.RendererProcesses)).
		Set("max-active-webgl-contexts", fmt.Sprintf("%d", bp.config.WebGLContexts)).
		// 网络优化
		Set("disable-background-timer-throttling", "").
		Set("disable-renderer-backgrounding", "").
		// 高配服务器特定优化
		Set("enable-features", "VaapiVideoDecoder,VaapiVideoEncoder").
		Set("ignore-gpu-blocklist", "").
		Set("enable-gpu-rasterization", "").
		Set("enable-zero-copy", "").
		Set("enable-oop-rasterization", "").
		Set("enable-raw-draw", "").
		// 内存管理优化
		Set("memory-pressure-off", "").
		Set("aggressive-cache-discard", "").
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

	// 创建池化浏览器实例
	pooledBrowser := &PooledBrowser{
		Browser:     browser,
		config:      bp.config,
		mafitConfig: bp.mafitConfig,
		logger:      bp.logger,
		activePages: 0,
		maxPages:    int32(bp.config.MaxConcurrentPages), // 使用最大并发页面数作为限制
	}

	// 初始化浏览器：设置一次localStorage
	if err := pooledBrowser.initializeBrowser(); err != nil {
		browser.Close()
		return nil, fmt.Errorf("failed to initialize browser: %w", err)
	}

	bp.logger.Info("Successfully created browser instance")
	return pooledBrowser, nil
}

// initializeBrowser 初始化浏览器：设置一次localStorage
func (pb *PooledBrowser) initializeBrowser() error {
	// 创建一个临时页面来设置localStorage
	tempPage := pb.Browser.MustPage()
	defer tempPage.Close()

	// 先导航到目标网站，这样才能设置localStorage
	baseURL := pb.mafitConfig.BaseURL
	pb.logger.Infof("Navigating to base URL for localStorage setup: %s", baseURL)
	
	if err := tempPage.Navigate(baseURL); err != nil {
		return fmt.Errorf("failed to navigate to base URL: %w", err)
	}

	// 等待页面加载
	if err := tempPage.WaitLoad(); err != nil {
		pb.logger.WithError(err).Warn("Page load timeout during initialization, but continuing")
	}

	// 设置localStorage（只需要设置一次）
	if err := pb.setLocalStorage(tempPage); err != nil {
		return fmt.Errorf("failed to set localStorage: %w", err)
	}

	pb.logger.Info("Browser initialized with localStorage")
	return nil
}

// GetActivePages 获取活跃页面数
func (pb *PooledBrowser) GetActivePages() int32 {
	pb.mu.RLock()
	defer pb.mu.RUnlock()
	return pb.activePages
}

// CreatePage 创建新页面
func (pb *PooledBrowser) CreatePage(ctx context.Context) (*rod.Page, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()
	
	if pb.closed {
		return nil, fmt.Errorf("browser is closed")
	}
	
	// 检查页面数量限制
	if pb.activePages >= pb.maxPages {
		return nil, fmt.Errorf("browser reached max pages limit: %d", pb.maxPages)
	}
	
	// 创建新页面
	page := pb.Browser.MustPage()
	page.MustSetViewport(1920, 1080, 1, false)
	
	// 增加活跃页面计数
	pb.activePages++
	
	pb.logger.Infof("Created new page, active pages: %d/%d", pb.activePages, pb.maxPages)
	return page, nil
}

// ClosePage 关闭页面
func (pb *PooledBrowser) ClosePage(page *rod.Page) {
	if page == nil {
		return
	}
	
	pb.mu.Lock()
	defer pb.mu.Unlock()
	
	// 关闭页面
	page.Close()
	
	// 减少活跃页面计数
	pb.activePages--
	
	pb.logger.Infof("Closed page, active pages: %d/%d", pb.activePages, pb.maxPages)
}

// setLocalStorage 设置localStorage
func (pb *PooledBrowser) setLocalStorage(page *rod.Page) error {
	// 设置JWT访问令牌 - 使用最简单的语法
	jwtScript := fmt.Sprintf(`localStorage.setItem('jwt_access_token', '%s');`, pb.mafitConfig.JWTAccessToken)

	if _, err := page.Eval(jwtScript); err != nil {
		pb.logger.WithError(err).Warn("Failed to set jwt_access_token, but continuing")
	}

	// 设置侧边栏状态 - 使用最简单的语法
	sidebarScript := fmt.Sprintf(`localStorage.setItem('sidebarSheet', '%s');`, pb.mafitConfig.SidebarSheet)

	if _, err := page.Eval(sidebarScript); err != nil {
		pb.logger.WithError(err).Warn("Failed to set sidebarSheet, but continuing")
	}

	pb.logger.Info("localStorage setup completed")
	return nil
}

// GetBrowser 从池中获取浏览器实例（简单遍历查找）
func (bp *BrowserPool) GetBrowser(ctx context.Context) (*PooledBrowser, error) {
	bp.logger.Info("Attempting to get browser from pool...")

	// 非阻塞轮询获取可用的浏览器
	for {
		select {
		case <-ctx.Done():
			bp.logger.Error("Context cancelled while getting browser")
			return nil, ctx.Err()
		default:
			// 遍历所有浏览器实例，查找可用的
			bp.mu.RLock()
			if bp.closed {
				bp.mu.RUnlock()
				bp.logger.Error("Browser pool is closed")
				return nil, fmt.Errorf("browser pool is closed")
			}

			for _, browser := range bp.browsers {
				if browser == nil {
					continue
				}
				
				activePages := browser.GetActivePages()
				bp.logger.Debugf("Checking browser availability. Active pages: %d/%d, closed: %v", 
					activePages, browser.maxPages, browser.closed)
				
				// 检查浏览器是否可用且未达到页面限制
				if !browser.closed && activePages < browser.maxPages {
					bp.mu.RUnlock()
					bp.logger.Infof("Found available browser. Active pages: %d/%d", 
						activePages, browser.maxPages)
					return browser, nil
				}
			}
			bp.mu.RUnlock()

			// 没有找到可用的浏览器，短暂等待后继续
			select {
			case <-time.After(100 * time.Millisecond):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
}

// ReturnBrowser 将浏览器实例返回到池中（简化版本）
func (bp *BrowserPool) ReturnBrowser(browser *PooledBrowser) {
	if browser == nil {
		return
	}

	// 浏览器已经在池中，不需要额外操作
	// 只需要确保页面计数正确即可
	bp.logger.Debugf("Browser returned to pool (already in pool). Active pages: %d/%d", 
		browser.GetActivePages(), browser.maxPages)
}

// Close 关闭浏览器池
func (bp *BrowserPool) Close() {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.closed {
		return
	}

	bp.closed = true

	// 关闭所有浏览器
	for _, browser := range bp.browsers {
		if browser != nil {
			browser.Close()
		}
	}

	bp.logger.Info("Browser pool closed")
}

// Close 关闭池化浏览器实例
func (pb *PooledBrowser) Close() {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if pb.closed {
		return
	}

	pb.closed = true

	// 关闭浏览器
	if pb.Browser != nil {
		pb.Browser.Close()
	}

	pb.logger.Info("Pooled browser closed")
}

// GetStats 获取浏览器池统计信息
func (bp *BrowserPool) GetStats() map[string]interface{} {
	bp.mu.RLock()
	defer bp.mu.RUnlock()

	poolSize := bp.config.PoolSize
	totalActivePages := 0
	availableBrowsers := 0
	browserStats := make([]map[string]interface{}, 0)
	
	// 遍历所有浏览器实例
	for _, browser := range bp.browsers {
		if browser != nil {
			activePages := browser.GetActivePages()
			totalActivePages += int(activePages)
			
			// 检查浏览器是否可用
			isAvailable := !browser.closed && activePages < browser.maxPages
			if isAvailable {
				availableBrowsers++
			}
			
			// 收集每个浏览器的统计信息
			stats := map[string]interface{}{
				"active_pages":   activePages,
				"max_pages":      browser.maxPages,
				"available":      isAvailable,
				"closed":         browser.closed,
			}
			browserStats = append(browserStats, stats)
		}
	}

	return map[string]interface{}{
		"pool_size":        poolSize,
		"available":        availableBrowsers,
		"in_use":           poolSize - availableBrowsers,
		"closed":           bp.closed,
		"type":             "pool",
		"total_active_pages": totalActivePages,
		"max_concurrent":   bp.config.MaxConcurrentPages,
		"renderer_process": bp.config.RendererProcesses,
		"webgl_contexts":   bp.config.WebGLContexts,
		"memory_limit":     bp.config.MemoryLimit,
		"browser_stats":    browserStats,
	}
}
