package status

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// RecoveryPolicy 恢复策略
type RecoveryPolicy struct {
	Enabled          bool          `json:"enabled"`           // 是否启用恢复
	CheckInterval    time.Duration `json:"check_interval"`    // 检查间隔
	MaxRecoveryTime  time.Duration `json:"max_recovery_time"` // 最大恢复时间
	BatchSize        int           `json:"batch_size"`        // 批处理大小
	Concurrency      int           `json:"concurrency"`       // 并发数
	RetryPolicy      *RetryPolicy  `json:"retry_policy"`      // 重试策略
	ConsistencyCheck bool          `json:"consistency_check"` // 一致性检查
	AutoRepair       bool          `json:"auto_repair"`       // 自动修复
}

// RecoveryTask 恢复任务
type RecoveryTask struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Status       Status                 `json:"status"`
	TargetStatus Status                 `json:"target_status"`
	CreatedAt    time.Time              `json:"created_at"`
	LastAttempt  time.Time              `json:"last_attempt"`
	AttemptCount int                    `json:"attempt_count"`
	MaxAttempts  int                    `json:"max_attempts"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	ErrorMsg     string                 `json:"error_msg,omitempty"`
	RecoveryData map[string]interface{} `json:"recovery_data,omitempty"`
}

// ConsistencyIssue 一致性问题
type ConsistencyIssue struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Description string                 `json:"description"`
	Severity    string                 `json:"severity"` // low, medium, high, critical
	DetectedAt  time.Time              `json:"detected_at"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Suggestion  string                 `json:"suggestion,omitempty"`
}

// RecoveryResult 恢复结果
type RecoveryResult struct {
	TaskID       string        `json:"task_id"`
	Success      bool          `json:"success"`
	Message      string        `json:"message"`
	RecoveredAt  time.Time     `json:"recovered_at"`
	Duration     time.Duration `json:"duration"`
	AttemptCount int           `json:"attempt_count"`
}

// RecoveryStats 恢复统计
type RecoveryStats struct {
	TotalTasks        int64         `json:"total_tasks"`
	SuccessfulTasks   int64         `json:"successful_tasks"`
	FailedTasks       int64         `json:"failed_tasks"`
	PendingTasks      int64         `json:"pending_tasks"`
	LastCheckTime     time.Time     `json:"last_check_time"`
	Uptime            time.Duration `json:"uptime"`
	ConsistencyIssues int64         `json:"consistency_issues"`
	AutoRepairs       int64         `json:"auto_repairs"`
}

// StatusRecovery 状态恢复接口
type StatusRecovery interface {
	// Start 启动状态恢复服务
	Start(ctx context.Context) error

	// Stop 停止状态恢复服务
	Stop(ctx context.Context) error

	// RecoverStatus 恢复指定状态
	RecoverStatus(ctx context.Context, id string, targetStatus Status) (*RecoveryResult, error)

	// BatchRecover 批量恢复状态
	BatchRecover(ctx context.Context, ids []string, targetStatus Status) ([]*RecoveryResult, error)

	// CheckConsistency 检查状态一致性
	CheckConsistency(ctx context.Context) ([]*ConsistencyIssue, error)

	// RepairData 修复数据
	RepairData(ctx context.Context, issueID string) error

	// GetRecoveryTasks 获取恢复任务列表
	GetRecoveryTasks(ctx context.Context, status Status) ([]*RecoveryTask, error)

	// GetStats 获取恢复统计信息
	GetStats() *RecoveryStats

	// IsRunning 检查是否运行中
	IsRunning() bool

	// IsHealthy 健康检查
	IsHealthy(ctx context.Context) bool
}

// StatusRecoveryImpl 状态恢复实现
type StatusRecoveryImpl struct {
	policy        *RecoveryPolicy
	statusTracker StatusTracker
	dataRepo      DataRepo
	logger        *log.Helper

	// 内部状态
	mu            sync.RWMutex
	running       bool
	stopChan      chan struct{}
	recoveryTasks map[string]*RecoveryTask
	stats         *RecoveryStats
	startTime     time.Time
}

// NewStatusRecovery 创建状态恢复服务
func NewStatusRecovery(policy *RecoveryPolicy, statusTracker StatusTracker, dataRepo DataRepo, logger log.Logger) StatusRecovery {
	if policy == nil {
		policy = &RecoveryPolicy{
			Enabled:          true,
			CheckInterval:    5 * time.Minute,
			MaxRecoveryTime:  1 * time.Hour,
			BatchSize:        50,
			Concurrency:      5,
			ConsistencyCheck: true,
			AutoRepair:       false,
			RetryPolicy: &RetryPolicy{
				MaxRetries:    3,
				RetryInterval: 2 * time.Minute,
				BackoffFactor: 1.5,
			},
		}
	}

	return &StatusRecoveryImpl{
		policy:        policy,
		statusTracker: statusTracker,
		dataRepo:      dataRepo,
		logger:        log.NewHelper(logger),
		recoveryTasks: make(map[string]*RecoveryTask),
		stats: &RecoveryStats{
			LastCheckTime: time.Now(),
		},
	}
}

// Start 启动状态恢复服务
func (r *StatusRecoveryImpl) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.policy.Enabled {
		r.logger.Info("Status recovery is disabled")
		return nil
	}

	if r.running {
		return nil // 已经在运行
	}

	r.running = true
	r.startTime = time.Now()
	r.stopChan = make(chan struct{})

	// 启动恢复协程
	go r.recoveryLoop(ctx)

	r.logger.Info("Status recovery service started")
	return nil
}

// Stop 停止状态恢复服务
func (r *StatusRecoveryImpl) Stop(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil // 已经停止
	}

	r.running = false
	close(r.stopChan)

	r.logger.Info("Status recovery service stopped")
	return nil
}

// RecoverStatus 恢复指定状态
func (r *StatusRecoveryImpl) RecoverStatus(ctx context.Context, id string, targetStatus Status) (*RecoveryResult, error) {
	if id == "" {
		return nil, ErrInvalidRequest
	}

	startTime := time.Now()
	result := &RecoveryResult{
		TaskID:      id,
		RecoveredAt: startTime,
	}

	// 获取当前状态
	currentStatus, err := r.statusTracker.GetStatus(ctx, id)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Failed to get current status: %v", err)
		result.Duration = time.Since(startTime)
		return result, err
	}

	if currentStatus == nil {
		result.Success = false
		result.Message = "Status not found"
		result.Duration = time.Since(startTime)
		return result, ErrStatusNotFound
	}

	// 检查是否需要恢复
	if currentStatus.Status == targetStatus {
		result.Success = true
		result.Message = "Status already correct, no recovery needed"
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// 创建恢复任务
	metadata := make(map[string]interface{})
	for k, v := range currentStatus.Metadata {
		metadata[k] = v
	}

	recoveryTask := &RecoveryTask{
		ID:           id,
		Type:         currentStatus.Type,
		Status:       currentStatus.Status,
		TargetStatus: targetStatus,
		CreatedAt:    startTime,
		MaxAttempts:  r.policy.RetryPolicy.MaxRetries,
		Metadata:     metadata,
	}

	// 执行恢复
	err = r.executeRecovery(ctx, recoveryTask)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Recovery failed: %v", err)
		result.AttemptCount = recoveryTask.AttemptCount
	} else {
		result.Success = true
		result.Message = "Status recovered successfully"
		result.AttemptCount = recoveryTask.AttemptCount
	}

	result.Duration = time.Since(startTime)
	return result, err
}

// BatchRecover 批量恢复状态
func (r *StatusRecoveryImpl) BatchRecover(ctx context.Context, ids []string, targetStatus Status) ([]*RecoveryResult, error) {
	if len(ids) == 0 {
		return []*RecoveryResult{}, nil
	}

	results := make([]*RecoveryResult, 0, len(ids))
	semaphore := make(chan struct{}, r.policy.Concurrency)
	resultChan := make(chan *RecoveryResult, len(ids))

	// 并发处理恢复任务
	for _, id := range ids {
		go func(taskID string) {
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			result, _ := r.RecoverStatus(ctx, taskID, targetStatus)
			resultChan <- result
		}(id)
	}

	// 收集结果
	for i := 0; i < len(ids); i++ {
		results = append(results, <-resultChan)
	}

	r.logger.Infof("Batch recovery completed for %d tasks", len(ids))
	return results, nil
}

// CheckConsistency 检查状态一致性
func (r *StatusRecoveryImpl) CheckConsistency(ctx context.Context) ([]*ConsistencyIssue, error) {
	if !r.policy.ConsistencyCheck {
		return []*ConsistencyIssue{}, nil
	}

	var issues []*ConsistencyIssue

	// 检查超时任务
	timeoutIssues := r.checkTimeoutConsistency(ctx)
	issues = append(issues, timeoutIssues...)

	// 检查状态转换一致性
	transitionIssues := r.checkTransitionConsistency(ctx)
	issues = append(issues, transitionIssues...)

	// 检查数据完整性
	dataIssues := r.checkDataIntegrity(ctx)
	issues = append(issues, dataIssues...)

	r.mu.Lock()
	r.stats.ConsistencyIssues += int64(len(issues))
	r.mu.Unlock()

	if len(issues) > 0 {
		r.logger.Warnf("Found %d consistency issues", len(issues))
	}

	return issues, nil
}

// RepairData 修复数据
func (r *StatusRecoveryImpl) RepairData(ctx context.Context, issueID string) error {
	if !r.policy.AutoRepair {
		return fmt.Errorf("auto repair is disabled")
	}

	// 这里应该根据具体的问题类型执行相应的修复逻辑
	// 目前只是一个示例实现
	r.logger.Infof("Attempting to repair data for issue: %s", issueID)

	// 模拟修复过程
	time.Sleep(100 * time.Millisecond)

	r.mu.Lock()
	r.stats.AutoRepairs++
	r.mu.Unlock()

	r.logger.Infof("Data repair completed for issue: %s", issueID)
	return nil
}

// GetRecoveryTasks 获取恢复任务列表
func (r *StatusRecoveryImpl) GetRecoveryTasks(ctx context.Context, status Status) ([]*RecoveryTask, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var tasks []*RecoveryTask
	for _, task := range r.recoveryTasks {
		if status == "" || task.Status == status {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// GetStats 获取恢复统计信息
func (r *StatusRecoveryImpl) GetStats() *RecoveryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := *r.stats // 复制统计信息
	if r.running {
		stats.Uptime = time.Since(r.startTime)
	}
	return &stats
}

// IsRunning 检查是否运行中
func (r *StatusRecoveryImpl) IsRunning() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.running
}

// IsHealthy 健康检查
func (r *StatusRecoveryImpl) IsHealthy(ctx context.Context) bool {
	if !r.IsRunning() {
		return false
	}

	if r.statusTracker != nil && !r.statusTracker.IsHealthy(ctx) {
		return false
	}

	if r.dataRepo != nil && !r.dataRepo.IsHealthy(ctx) {
		return false
	}

	return true
}

// recoveryLoop 恢复循环
func (r *StatusRecoveryImpl) recoveryLoop(ctx context.Context) {
	ticker := time.NewTicker(r.policy.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Status recovery stopped due to context cancellation")
			return
		case <-r.stopChan:
			r.logger.Info("Status recovery stopped")
			return
		case <-ticker.C:
			r.performRecoveryCheck(ctx)
		}
	}
}

// performRecoveryCheck 执行恢复检查
func (r *StatusRecoveryImpl) performRecoveryCheck(ctx context.Context) {
	// 检查一致性
	issues, err := r.CheckConsistency(ctx)
	if err != nil {
		r.logger.Errorf("Failed to check consistency: %v", err)
		return
	}

	// 自动修复高优先级问题
	if r.policy.AutoRepair {
		for _, issue := range issues {
			if issue.Severity == "critical" || issue.Severity == "high" {
				if err := r.RepairData(ctx, issue.ID); err != nil {
					r.logger.Errorf("Failed to auto repair issue %s: %v", issue.ID, err)
				}
			}
		}
	}

	r.mu.Lock()
	r.stats.LastCheckTime = time.Now()
	r.mu.Unlock()
}

// executeRecovery 执行恢复
func (r *StatusRecoveryImpl) executeRecovery(ctx context.Context, task *RecoveryTask) error {
	task.AttemptCount++
	task.LastAttempt = time.Now()

	// 存储恢复任务
	r.mu.Lock()
	r.recoveryTasks[task.ID] = task
	r.stats.TotalTasks++
	r.mu.Unlock()

	// 执行状态更新
	err := r.statusTracker.UpdateStatusWithReason(ctx, task.ID, task.TargetStatus, "状态恢复")
	if err != nil {
		task.ErrorMsg = err.Error()
		r.mu.Lock()
		r.stats.FailedTasks++
		r.mu.Unlock()
		return err
	}

	// 恢复成功
	r.mu.Lock()
	delete(r.recoveryTasks, task.ID)
	r.stats.SuccessfulTasks++
	r.mu.Unlock()

	r.logger.Infof("Status recovery successful: %s %s -> %s", task.ID, task.Status, task.TargetStatus)
	return nil
}

// checkTimeoutConsistency 检查超时一致性
func (r *StatusRecoveryImpl) checkTimeoutConsistency(ctx context.Context) []*ConsistencyIssue {
	var issues []*ConsistencyIssue

	// 查找长时间处于处理中状态的任务
	processingTasks, err := r.statusTracker.ListByStatus(ctx, StatusProcessing, 100)
	if err != nil {
		r.logger.Errorf("Failed to list processing tasks: %v", err)
		return issues
	}

	now := time.Now()
	for _, statusInfo := range processingTasks {
		if now.Sub(statusInfo.UpdatedAt) > r.policy.MaxRecoveryTime {
			issue := &ConsistencyIssue{
				ID:          fmt.Sprintf("timeout_%s", statusInfo.ID),
				Type:        "timeout",
				Description: fmt.Sprintf("Task %s has been processing for too long", statusInfo.ID),
				Severity:    "high",
				DetectedAt:  now,
				Data: map[string]interface{}{
					"task_id":    statusInfo.ID,
					"status":     statusInfo.Status,
					"updated_at": statusInfo.UpdatedAt,
					"duration":   now.Sub(statusInfo.UpdatedAt).String(),
				},
				Suggestion: "Consider marking as timeout or failed",
			}
			issues = append(issues, issue)
		}
	}

	return issues
}

// checkTransitionConsistency 检查状态转换一致性
func (r *StatusRecoveryImpl) checkTransitionConsistency(ctx context.Context) []*ConsistencyIssue {
	var issues []*ConsistencyIssue

	// 这里可以添加状态转换规则检查
	// 例如：检查是否有非法的状态转换

	return issues
}

// checkDataIntegrity 检查数据完整性
func (r *StatusRecoveryImpl) checkDataIntegrity(ctx context.Context) []*ConsistencyIssue {
	var issues []*ConsistencyIssue

	// 这里可以添加数据完整性检查
	// 例如：检查关联数据是否存在

	return issues
}
