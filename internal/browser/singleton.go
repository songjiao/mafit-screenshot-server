package browser

import (
	"sync"

	"makeprofit/internal/config"
	"makeprofit/pkg/utils"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/sirupsen/logrus"
)

// SingletonBrowser 单例浏览器服务
type SingletonBrowser struct {
	browser *rod.Browser
	mu      sync.RWMutex
	config  *config.BrowserConfig
	logger  *logrus.Logger
}

var (
	singletonInstance *SingletonBrowser
	singletonOnce     sync.Once
)

// GetSingletonBrowser 获取单例浏览器实例
func GetSingletonBrowser(cfg *config.BrowserConfig) *SingletonBrowser {
	singletonOnce.Do(func() {
		singletonInstance = &SingletonBrowser{
			config: cfg,
			logger: utils.GetLogger(),
		}
	})
	return singletonInstance
}

// GetBrowser 获取浏览器实例，如果不存在则创建
func (sb *SingletonBrowser) GetBrowser() (*rod.Browser, error) {
	sb.mu.RLock()
	if sb.browser != nil {
		// 检查浏览器是否还可用
		if err := sb.checkBrowser(sb.browser); err == nil {
			sb.mu.RUnlock()
			return sb.browser, nil
		}
		// 浏览器不可用，需要重新创建
		sb.logger.Warn("Browser instance is not available, recreating...")
		sb.browser.Close()
		sb.browser = nil
	}
	sb.mu.RUnlock()

	// 需要创建新的浏览器实例
	sb.mu.Lock()
	defer sb.mu.Unlock()

	// 双重检查，防止并发创建
	if sb.browser != nil {
		return sb.browser, nil
	}

	browser, err := sb.createBrowser()
	if err != nil {
		return nil, err
	}

	sb.browser = browser
	sb.logger.Info("Created singleton browser instance")
	return browser, nil
}

// createBrowser 创建新的浏览器实例
func (sb *SingletonBrowser) createBrowser() (*rod.Browser, error) {
	sb.logger.Info("Creating new browser instance...")

	// 配置启动器，针对2GB内存优化（适当放宽限制）
	launcherURL, err := launcher.New().
		Headless(sb.config.Headless).
		Leakless(false). // 禁用leakless模式以节省内存
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
		// 内存优化（2G内存可以适当放宽）
		Set("max_old_space_size", "512"). // 从256MB提升到512MB
		Set("js-flags", "--max-old-space-size=512").
		// 进程优化（2G内存可以支持更多进程）
		Set("renderer-process-limit", "2").    // 从1个进程提升到2个进程
		Set("max-active-webgl-contexts", "2"). // 从1个提升到2个
		// 关闭不必要的服务
		Set("disable-audio-service", "").
		Set("disable-crash-reporter", "").
		Set("disable-breakpad", "").
		Set("disable-features", "TranslateUI,AudioServiceOutOfProcess"). // 移除site-per-process限制
		// 移除一些过于激进的内存限制
		// Set("memory-pressure-off", ""). // 移除，让Chrome自己管理内存压力
		// Set("aggressive-cache-discard", ""). // 移除，减少缓存丢弃
		// Set("enable-low-end-device-mode", ""). // 移除，2G内存不算低端设备
		Launch()

	if err != nil {
		return nil, err
	}
	// 连接到浏览器
	browser := rod.New().ControlURL(launcherURL)

	if err := browser.Connect(); err != nil {
		sb.logger.WithError(err).Error("Failed to connect to browser")
		return nil, err
	}

	sb.logger.Info("Successfully created browser instance with optimized settings")
	return browser, nil
}

// checkBrowser 检查浏览器是否可用
func (sb *SingletonBrowser) checkBrowser(browser *rod.Browser) error {
	// 简单检查：尝试创建一个页面
	page := browser.MustPage("about:blank")
	defer page.Close()

	// 如果页面创建成功，说明浏览器可用
	return nil
}

// Close 关闭浏览器实例
func (sb *SingletonBrowser) Close() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if sb.browser != nil {
		sb.logger.Info("Closing singleton browser instance")
		sb.browser.Close()
		sb.browser = nil
	}
}

// GetStats 获取浏览器统计信息
func (sb *SingletonBrowser) GetStats() map[string]interface{} {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	stats := map[string]interface{}{
		"type": "singleton",
	}

	if sb.browser != nil {
		stats["status"] = "active"
		stats["available"] = true
	} else {
		stats["status"] = "inactive"
		stats["available"] = false
	}

	return stats
}

// IsAvailable 检查浏览器是否可用
func (sb *SingletonBrowser) IsAvailable() bool {
	sb.mu.RLock()
	defer sb.mu.RUnlock()

	if sb.browser == nil {
		return false
	}

	return sb.checkBrowser(sb.browser) == nil
}
