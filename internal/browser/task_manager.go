package browser

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"makeprofit/internal/config"
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
)

// TaskManager 任务管理器
type TaskManager struct {
	runningTasks map[string]*TaskInfo // 正在运行的任务
	mu           sync.RWMutex
	config       *config.CDNConfig
}

// TaskInfo 任务信息
type TaskInfo struct {
	Done        chan struct{}
	StartTime   time.Time
	Timeout     time.Duration
	Status      TaskStatus
	CompletedAt time.Time
}

// NewTaskManager 创建新的任务管理器
func NewTaskManager(cfg *config.CDNConfig) *TaskManager {
	tm := &TaskManager{
		runningTasks: make(map[string]*TaskInfo),
		config:       cfg,
	}

	// 启动清理协程
	go tm.cleanupExpiredTasks()

	return tm
}

// generateTaskKey 生成任务键（包含时间信息）
func (tm *TaskManager) generateTaskKey(symbol, market, timeframe string) string {
	now := time.Now()

	switch timeframe {
	case "1d":
		// 日线：同一天内一支股票只有一张
		return fmt.Sprintf("%s_%s_1d_%s", symbol, market, now.Format("20060102"))
	case "1h":
		// 小时线：包含小时信息
		return fmt.Sprintf("%s_%s_1h_%s_%02d", symbol, market, now.Format("20060102"), now.Hour())
	case "1wk":
		// 周线：同一周内一支股票只有一张
		year, week := now.ISOWeek()
		return fmt.Sprintf("%s_%s_1wk_%d_%02d", symbol, market, year, week)
	default:
		// 其他时间框架使用时间戳
		return fmt.Sprintf("%s_%s_%s_%s", symbol, market, timeframe, now.Format("20060102_150405"))
	}
}

// generateCDNURL 生成CDN URL
func (tm *TaskManager) generateCDNURL(symbol, market, timeframe string) string {
	now := time.Now()
	var fileName string

	switch timeframe {
	case "1d":
		// 日线：同一天内一支股票只有一张，格式：{symbol}_{market}_1d_{date}.png
		fileName = fmt.Sprintf("%s_%s_1d_%s.png", symbol, market, now.Format("20060102"))
	case "1h":
		// 小时线：根据市场开市时间生成，格式：{symbol}_{market}_1h_{date}_{hour}.png
		fileName = fmt.Sprintf("%s_%s_1h_%s_%02d.png", symbol, market, now.Format("20060102"), now.Hour())
	case "1wk":
		// 周线：同一周内一支股票只有一张，格式：{symbol}_{market}_1wk_{year}_{week}.png
		year, week := now.ISOWeek()
		fileName = fmt.Sprintf("%s_%s_1wk_%d_%02d.png", symbol, market, year, week)
	default:
		// 其他时间框架使用时间戳（保持向后兼容）
		timestamp := now.Format("20060102_150405")
		fileName = fmt.Sprintf("%s_%s_%s_%s.png", symbol, market, timeframe, timestamp)
	}

	return fmt.Sprintf("%s/%s/%s", tm.config.BaseURL, tm.config.ResultPath, fileName)
}

// checkCDNExists 检查CDN上是否已存在图片
func (tm *TaskManager) checkCDNExists(cdnURL string) bool {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Head(cdnURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// isTaskRunning 检查任务是否正在运行
func (tm *TaskManager) isTaskRunning(taskKey string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	taskInfo, exists := tm.runningTasks[taskKey]
	if !exists {
		return false
	}

	// 检查任务状态
	if taskInfo.Status == TaskStatusCompleted || taskInfo.Status == TaskStatusFailed {
		return false
	}

	// 检查任务是否超时
	elapsed := time.Since(taskInfo.StartTime)
	if elapsed > taskInfo.Timeout || elapsed < 0 {
		// 任务超时或时间异常（未来时间），标记为失败
		taskInfo.Status = TaskStatusFailed
		return false
	}

	return true
}

// isTaskCompleted 检查任务是否已完成
func (tm *TaskManager) isTaskCompleted(taskKey string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	taskInfo, exists := tm.runningTasks[taskKey]
	if !exists {
		return false
	}

	return taskInfo.Status == TaskStatusCompleted
}

// startTask 开始任务
func (tm *TaskManager) startTask(taskKey string) (chan struct{}, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 检查任务是否已经在运行
	if taskInfo, exists := tm.runningTasks[taskKey]; exists {
		// 如果任务已完成，返回错误
		if taskInfo.Status == TaskStatusCompleted {
			return nil, fmt.Errorf("task %s is already completed", taskKey)
		}

		// 如果任务失败，可以重新开始
		if taskInfo.Status == TaskStatusFailed {
			// 清理旧任务
			close(taskInfo.Done)
			delete(tm.runningTasks, taskKey)
		} else {
			// 检查任务是否超时
			elapsed := time.Since(taskInfo.StartTime)
			if elapsed <= taskInfo.Timeout && elapsed >= 0 {
				return nil, fmt.Errorf("task %s is already running", taskKey)
			}
			// 任务超时，清理旧任务
			close(taskInfo.Done)
			delete(tm.runningTasks, taskKey)
		}
	}

	// 创建任务完成通道
	done := make(chan struct{})
	taskInfo := &TaskInfo{
		Done:      done,
		StartTime: time.Now(),
		Timeout:   5 * time.Minute, // 5分钟超时
		Status:    TaskStatusRunning,
	}
	tm.runningTasks[taskKey] = taskInfo

	return done, nil
}

// completeTask 完成任务
func (tm *TaskManager) completeTask(taskKey string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if taskInfo, exists := tm.runningTasks[taskKey]; exists {
		// 标记任务为已完成
		taskInfo.Status = TaskStatusCompleted
		taskInfo.CompletedAt = time.Now()

		// 安全关闭channel
		select {
		case <-taskInfo.Done:
			// channel已经关闭，不需要再次关闭
		default:
			close(taskInfo.Done)
		}
		// 注意：不删除任务，保留一段时间供后续请求检查
	}
}

// failTask 标记任务失败并清理
func (tm *TaskManager) failTask(taskKey string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if taskInfo, exists := tm.runningTasks[taskKey]; exists {
		// 标记任务为失败
		taskInfo.Status = TaskStatusFailed
		taskInfo.CompletedAt = time.Now()

		// 安全关闭channel
		select {
		case <-taskInfo.Done:
			// channel已经关闭，不需要再次关闭
		default:
			close(taskInfo.Done)
		}
		// 注意：不删除任务，保留一段时间供后续请求检查
	}
}

// waitForTask 等待任务完成
func (tm *TaskManager) waitForTask(taskKey string, timeout time.Duration) error {
	tm.mu.RLock()
	taskInfo, exists := tm.runningTasks[taskKey]
	tm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("task %s not found", taskKey)
	}

	// 记录等待开始时间
	startTime := time.Now()

	select {
	case <-taskInfo.Done:
		waitTime := time.Since(startTime)
		// 记录等待时间到日志
		_ = waitTime // 避免未使用变量警告
		return nil
	case <-time.After(timeout):
		waitTime := time.Since(startTime)
		return fmt.Errorf("wait for task %s timeout after %v (waited %v)", taskKey, timeout, waitTime)
	}
}

// cleanupExpiredTasks 清理过期任务
func (tm *TaskManager) cleanupExpiredTasks() {
	ticker := time.NewTicker(1 * time.Minute) // 每分钟检查一次
	defer ticker.Stop()

	for range ticker.C {
		tm.mu.Lock()
		now := time.Now()
		for taskKey, taskInfo := range tm.runningTasks {
			// 清理超时的运行中任务
			if taskInfo.Status == TaskStatusRunning {
				elapsed := now.Sub(taskInfo.StartTime)
				if elapsed > taskInfo.Timeout || elapsed < 0 {
					// 任务超时或时间异常（未来时间），清理
					close(taskInfo.Done)
					delete(tm.runningTasks, taskKey)
				}
			}

			// 清理已完成的任务（保留30分钟）
			if taskInfo.Status == TaskStatusCompleted || taskInfo.Status == TaskStatusFailed {
				if taskInfo.CompletedAt.Add(30 * time.Minute).Before(now) {
					close(taskInfo.Done)
					delete(tm.runningTasks, taskKey)
				}
			}
		}
		tm.mu.Unlock()
	}
}

// GetRunningTasksCount 获取正在运行的任务数量
func (tm *TaskManager) GetRunningTasksCount() int {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return len(tm.runningTasks)
}

// GetRunningTasks 获取正在运行的任务列表
func (tm *TaskManager) GetRunningTasks() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tasks := make([]string, 0, len(tm.runningTasks))
	for taskKey := range tm.runningTasks {
		tasks = append(tasks, taskKey)
	}
	return tasks
}
