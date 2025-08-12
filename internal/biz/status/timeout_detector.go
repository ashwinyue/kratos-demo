package status

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// TimeoutPolicy 超时策略
type TimeoutPolicy struct {
	DefaultTimeout time.Duration `json:"default_timeout"` // 默认超时时间
	MaxTimeout     time.Duration `json:"max_timeout"`     // 最大超时时间
	CheckInterval  time.Duration `json:"check_interval"`  // 检查间隔
	RetryPolicy    *RetryPolicy  `json:"retry_policy"`    // 重试策略
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`    // 最大重试次数
	RetryInterval time.Duration `json:"retry_interval"` // 重试间隔
	BackoffFactor float64       `json:"backoff_factor"` // 退避因子
}

// TimeoutTask 超时任务
type TimeoutTask struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	StartTime  time.Time              `json:"start_time"`
	Timeout    time.Duration          `json:"timeout"`
	ExpireTime time.Time              `json:"expire_time"`
	RetryCount int                    `json:"retry_count"`
	MaxRetries int                    `json:"max_retries"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Callback   TimeoutCallback        `json:"-"` // 超时回调函数
}

// TimeoutCallback 超时回调函数类型
type TimeoutCallback func(ctx context.Context, task *TimeoutTask) error

// TimeoutDetector 超时检测器接口
type TimeoutDetector interface {
	// Start 启动超时检测器
	Start(ctx context.Context) error

	// Stop 停止超时检测器
	Stop(ctx context.Context) error

	// AddTask 添加超时任务
	AddTask(ctx context.Context, task *TimeoutTask) error

	// RemoveTask 移除超时任务
	RemoveTask(ctx context.Context, taskID string) error

	// GetTask 获取超时任务
	GetTask(ctx context.Context, taskID string) (*TimeoutTask, error)

	// ListExpiredTasks 列出已过期的任务
	ListExpiredTasks(ctx context.Context) ([]*TimeoutTask, error)

	// UpdateTaskTimeout 更新任务超时时间
	UpdateTaskTimeout(ctx context.Context, taskID string, timeout time.Duration) error

	// IsRunning 检查检测器是否运行中
	IsRunning() bool

	// GetStats 获取统计信息
	GetStats() *TimeoutStats

	// IsHealthy 健康检查
	IsHealthy(ctx context.Context) bool
}

// TimeoutStats 超时统计信息
type TimeoutStats struct {
	TotalTasks     int64         `json:"total_tasks"`     // 总任务数
	ActiveTasks    int64         `json:"active_tasks"`    // 活跃任务数
	ExpiredTasks   int64         `json:"expired_tasks"`   // 过期任务数
	ProcessedTasks int64         `json:"processed_tasks"` // 已处理任务数
	LastCheckTime  time.Time     `json:"last_check_time"` // 最后检查时间
	Uptime         time.Duration `json:"uptime"`          // 运行时间
}

// TimeoutDetectorImpl 超时检测器实现
type TimeoutDetectorImpl struct {
	policy        *TimeoutPolicy
	statusTracker StatusTracker
	logger        *log.Helper

	// 内部状态
	mu        sync.RWMutex
	tasks     map[string]*TimeoutTask
	running   bool
	stopChan  chan struct{}
	stats     *TimeoutStats
	startTime time.Time
}

// NewTimeoutDetector 创建超时检测器
func NewTimeoutDetector(policy *TimeoutPolicy, statusTracker StatusTracker, logger log.Logger) TimeoutDetector {
	if policy == nil {
		policy = &TimeoutPolicy{
			DefaultTimeout: 30 * time.Minute,
			MaxTimeout:     2 * time.Hour,
			CheckInterval:  1 * time.Minute,
			RetryPolicy: &RetryPolicy{
				MaxRetries:    3,
				RetryInterval: 5 * time.Minute,
				BackoffFactor: 2.0,
			},
		}
	}

	return &TimeoutDetectorImpl{
		policy:        policy,
		statusTracker: statusTracker,
		logger:        log.NewHelper(logger),
		tasks:         make(map[string]*TimeoutTask),
		running:       false,
		stopChan:      make(chan struct{}),
		stats: &TimeoutStats{
			LastCheckTime: time.Now(),
		},
	}
}

// Start 启动超时检测器
func (d *TimeoutDetectorImpl) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return nil // 已经在运行
	}

	d.running = true
	d.startTime = time.Now()
	d.stopChan = make(chan struct{})

	// 启动检测协程
	go d.detectLoop(ctx)

	d.logger.Info("Timeout detector started")
	return nil
}

// Stop 停止超时检测器
func (d *TimeoutDetectorImpl) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return nil // 已经停止
	}

	d.running = false
	close(d.stopChan)

	d.logger.Info("Timeout detector stopped")
	return nil
}

// AddTask 添加超时任务
func (d *TimeoutDetectorImpl) AddTask(ctx context.Context, task *TimeoutTask) error {
	if task == nil || task.ID == "" {
		return ErrInvalidRequest
	}

	// 设置默认值
	if task.Timeout <= 0 {
		task.Timeout = d.policy.DefaultTimeout
	}
	if task.Timeout > d.policy.MaxTimeout {
		task.Timeout = d.policy.MaxTimeout
	}
	if task.StartTime.IsZero() {
		task.StartTime = time.Now()
	}
	task.ExpireTime = task.StartTime.Add(task.Timeout)

	d.mu.Lock()
	d.tasks[task.ID] = task
	d.stats.TotalTasks++
	d.stats.ActiveTasks++
	d.mu.Unlock()

	d.logger.Infof("Added timeout task: %s, expire at: %v", task.ID, task.ExpireTime)
	return nil
}

// RemoveTask 移除超时任务
func (d *TimeoutDetectorImpl) RemoveTask(ctx context.Context, taskID string) error {
	if taskID == "" {
		return ErrInvalidRequest
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.tasks[taskID]; exists {
		delete(d.tasks, taskID)
		d.stats.ActiveTasks--
		d.logger.Infof("Removed timeout task: %s", taskID)
	}

	return nil
}

// GetTask 获取超时任务
func (d *TimeoutDetectorImpl) GetTask(ctx context.Context, taskID string) (*TimeoutTask, error) {
	if taskID == "" {
		return nil, ErrInvalidRequest
	}

	d.mu.RLock()
	defer d.mu.RUnlock()

	task, exists := d.tasks[taskID]
	if !exists {
		return nil, ErrStatusNotFound
	}

	return task, nil
}

// ListExpiredTasks 列出已过期的任务
func (d *TimeoutDetectorImpl) ListExpiredTasks(ctx context.Context) ([]*TimeoutTask, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now()
	var expiredTasks []*TimeoutTask

	for _, task := range d.tasks {
		if now.After(task.ExpireTime) {
			expiredTasks = append(expiredTasks, task)
		}
	}

	return expiredTasks, nil
}

// UpdateTaskTimeout 更新任务超时时间
func (d *TimeoutDetectorImpl) UpdateTaskTimeout(ctx context.Context, taskID string, timeout time.Duration) error {
	if taskID == "" || timeout <= 0 {
		return ErrInvalidRequest
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	task, exists := d.tasks[taskID]
	if !exists {
		return ErrStatusNotFound
	}

	// 限制最大超时时间
	if timeout > d.policy.MaxTimeout {
		timeout = d.policy.MaxTimeout
	}

	task.Timeout = timeout
	task.ExpireTime = task.StartTime.Add(timeout)

	d.logger.Infof("Updated timeout for task %s: %v, new expire time: %v", taskID, timeout, task.ExpireTime)
	return nil
}

// IsRunning 检查检测器是否运行中
func (d *TimeoutDetectorImpl) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

// GetStats 获取统计信息
func (d *TimeoutDetectorImpl) GetStats() *TimeoutStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := *d.stats // 复制统计信息
	if d.running {
		stats.Uptime = time.Since(d.startTime)
	}
	return &stats
}

// IsHealthy 健康检查
func (d *TimeoutDetectorImpl) IsHealthy(ctx context.Context) bool {
	if !d.IsRunning() {
		return false
	}

	if d.statusTracker != nil {
		return d.statusTracker.IsHealthy(ctx)
	}

	return true
}

// detectLoop 检测循环
func (d *TimeoutDetectorImpl) detectLoop(ctx context.Context) {
	ticker := time.NewTicker(d.policy.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("Timeout detector stopped due to context cancellation")
			return
		case <-d.stopChan:
			d.logger.Info("Timeout detector stopped")
			return
		case <-ticker.C:
			d.checkTimeouts(ctx)
		}
	}
}

// checkTimeouts 检查超时任务
func (d *TimeoutDetectorImpl) checkTimeouts(ctx context.Context) {
	expiredTasks, err := d.ListExpiredTasks(ctx)
	if err != nil {
		d.logger.Errorf("Failed to list expired tasks: %v", err)
		return
	}

	if len(expiredTasks) == 0 {
		d.mu.Lock()
		d.stats.LastCheckTime = time.Now()
		d.mu.Unlock()
		return
	}

	d.logger.Infof("Found %d expired tasks", len(expiredTasks))

	for _, task := range expiredTasks {
		d.handleTimeoutTask(ctx, task)
	}

	d.mu.Lock()
	d.stats.LastCheckTime = time.Now()
	d.stats.ExpiredTasks += int64(len(expiredTasks))
	d.mu.Unlock()
}

// handleTimeoutTask 处理超时任务
func (d *TimeoutDetectorImpl) handleTimeoutTask(ctx context.Context, task *TimeoutTask) {
	d.logger.Warnf("Task %s has timed out, started at: %v, expired at: %v",
		task.ID, task.StartTime, task.ExpireTime)

	// 更新状态追踪器中的状态
	if d.statusTracker != nil {
		err := d.statusTracker.UpdateStatusWithReason(ctx, task.ID, StatusTimeout, "任务超时")
		if err != nil {
			d.logger.Errorf("Failed to update status to timeout for task %s: %v", task.ID, err)
		}
	}

	// 执行回调函数
	if task.Callback != nil {
		if err := task.Callback(ctx, task); err != nil {
			d.logger.Errorf("Timeout callback failed for task %s: %v", task.ID, err)
		}
	}

	// 检查是否需要重试
	if d.shouldRetry(task) {
		d.scheduleRetry(ctx, task)
	} else {
		// 移除任务
		d.RemoveTask(ctx, task.ID)
		d.mu.Lock()
		d.stats.ProcessedTasks++
		d.mu.Unlock()
	}
}

// shouldRetry 判断是否应该重试
func (d *TimeoutDetectorImpl) shouldRetry(task *TimeoutTask) bool {
	if d.policy.RetryPolicy == nil {
		return false
	}

	return task.RetryCount < task.MaxRetries && task.RetryCount < d.policy.RetryPolicy.MaxRetries
}

// scheduleRetry 安排重试
func (d *TimeoutDetectorImpl) scheduleRetry(ctx context.Context, task *TimeoutTask) {
	task.RetryCount++

	// 计算重试间隔（带退避）
	retryInterval := d.policy.RetryPolicy.RetryInterval
	if d.policy.RetryPolicy.BackoffFactor > 1.0 {
		for i := 0; i < task.RetryCount-1; i++ {
			retryInterval = time.Duration(float64(retryInterval) * d.policy.RetryPolicy.BackoffFactor)
		}
	}

	// 更新任务时间
	task.StartTime = time.Now().Add(retryInterval)
	task.ExpireTime = task.StartTime.Add(task.Timeout)

	d.logger.Infof("Scheduled retry for task %s (attempt %d/%d), next start: %v",
		task.ID, task.RetryCount, task.MaxRetries, task.StartTime)

	// 更新状态追踪器
	if d.statusTracker != nil {
		err := d.statusTracker.UpdateStatusWithReason(ctx, task.ID, StatusRetrying,
			fmt.Sprintf("重试第%d次", task.RetryCount))
		if err != nil {
			d.logger.Errorf("Failed to update status to retrying for task %s: %v", task.ID, err)
		}
	}
}
