package step

import (
	"context"
	"fmt"
	"time"

	"kratos-demo/internal/biz/sms/types"
	"kratos-demo/internal/biz/status"
)

// 步骤服务相关错误
var (
	ErrStepNotFound     = fmt.Errorf("step not found")
	ErrStepExists       = fmt.Errorf("step already exists")
	ErrInvalidStepState = fmt.Errorf("invalid step state")
	ErrStepTimeout      = fmt.Errorf("step timeout")
)

// StepConfig 步骤配置
type StepConfig struct {
	Type        StepType `json:"type"`
	Timeout     int      `json:"timeout"`
	RetryCount  int      `json:"retry_count"`
	Concurrency int      `json:"concurrency"`
}

// StepResult 步骤执行结果
type StepResult struct {
	StepID     string          `json:"step_id"`
	Status     StepStatus      `json:"status"`
	StartTime  time.Time       `json:"start_time"`
	EndTime    time.Time       `json:"end_time"`
	Duration   time.Duration   `json:"duration"`
	ErrorMsg   string          `json:"error_msg,omitempty"`
	Statistics *StepStatistics `json:"statistics,omitempty"`
}

// StepError 步骤错误信息
type StepError struct {
	StepID    string    `json:"step_id"`
	ErrorCode string    `json:"error_code"`
	ErrorMsg  string    `json:"error_msg"`
	Timestamp time.Time `json:"timestamp"`
}

// StepProgress 步骤进度信息
type StepProgress struct {
	StepID      string  `json:"step_id"`
	Stage       string  `json:"stage"`
	Percentage  float64 `json:"percentage"`
	Message     string  `json:"message"`
	CurrentItem int     `json:"current_item"`
	TotalItems  int     `json:"total_items"`
}

// SMSStepService SMS步骤服务接口
type SMSStepService interface {
	// CreateStep 创建步骤
	CreateStep(ctx context.Context, batchID string, stepType StepType, config *StepConfig) (*StepResult, error)

	// ExecuteStep 执行步骤
	ExecuteStep(ctx context.Context, stepID string) error

	// ProcessStep 处理步骤
	ProcessStep(ctx context.Context, stepID string, processor StepProcessor) error

	// GetStepInfo 获取步骤信息
	GetStepInfo(ctx context.Context, stepID string) (*StepResult, error)

	// CancelStep 取消步骤
	CancelStep(ctx context.Context, stepID string) error

	// RetryStep 重试步骤
	RetryStep(ctx context.Context, stepID string) error

	// ListSteps 列出步骤
	ListSteps(ctx context.Context, batchID string, stepType StepType, status StepStatus) ([]*StepResult, error)

	// DeleteStep 删除步骤
	DeleteStep(ctx context.Context, stepID string) error

	// GetStepStatistics 获取步骤统计
	GetStepStatistics(ctx context.Context, stepID string) (*StepStatistics, error)

	// UpdateStepStatus 更新步骤状态
	UpdateStepStatus(ctx context.Context, stepID string, status StepStatus) error

	// GetStepProgress 获取步骤进度
	GetStepProgress(ctx context.Context, stepID string) (*StepProgress, error)
}

// SMSStepServiceImpl SMS步骤服务实现
type SMSStepServiceImpl struct {
	stepRepo      SmsStepRepo
	statusTracker status.StatusTracker
	batchRepo     types.SmsBatchRepo
}

// NewSMSStepService 创建SMS步骤服务
func NewSMSStepService(stepRepo SmsStepRepo, statusTracker status.StatusTracker, batchRepo types.SmsBatchRepo) SMSStepService {
	return &SMSStepServiceImpl{
		stepRepo:      stepRepo,
		statusTracker: statusTracker,
		batchRepo:     batchRepo,
	}
}

// CreateStep 创建步骤
func (s *SMSStepServiceImpl) CreateStep(ctx context.Context, batchID string, stepType StepType, config *StepConfig) (*StepResult, error) {
	// 验证批次是否存在
	batch, err := s.batchRepo.GetByBatchID(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}
	if batch == nil {
		return nil, fmt.Errorf("batch not found: %s", batchID)
	}

	// 创建步骤
	step := &SmsStep{
		ID:         generateStepID(),
		BatchId:    batchID,
		Type:       stepType,
		Status:     StepStatusPending,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
		Statistics: &StepStatistics{
			Total:   0,
			Success: 0,
			Failure: 0,
		},
	}

	// 保存步骤
	if err := s.stepRepo.Save(ctx, step); err != nil {
		return nil, fmt.Errorf("failed to save step: %w", err)
	}

	// 更新状态追踪
	if err := s.statusTracker.UpdateStatusWithReason(ctx, step.ID, status.StatusPending, "步骤已创建"); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	return &StepResult{
		StepID:     step.ID,
		Status:     step.Status,
		StartTime:  step.CreateTime,
		Statistics: step.Statistics,
	}, nil
}

// ExecuteStep 执行步骤
func (s *SMSStepServiceImpl) ExecuteStep(ctx context.Context, stepID string) error {
	// 获取步骤信息
	step, err := s.stepRepo.GetByID(ctx, stepID)
	if err != nil {
		return fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return ErrStepNotFound
	}

	// 检查步骤状态
	if step.Status != StepStatusPending {
		return ErrInvalidStepState
	}

	// 更新步骤状态为运行中
	step.Status = StepStatusRunning
	step.UpdateTime = time.Now()

	if err := s.stepRepo.Save(ctx, step); err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}

	// 更新状态追踪
	return s.statusTracker.UpdateStatusWithReason(ctx, stepID, status.StatusProcessing, "步骤开始执行")
}

// ProcessStep 处理步骤
func (s *SMSStepServiceImpl) ProcessStep(ctx context.Context, stepID string, processor StepProcessor) error {
	// 获取步骤信息
	step, err := s.stepRepo.GetByID(ctx, stepID)
	if err != nil {
		return fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return ErrStepNotFound
	}

	// 执行步骤处理
	// 获取批次信息用于处理器
	batch, err := s.batchRepo.GetByBatchID(ctx, step.BatchId)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}

	err = processor.Execute(ctx, batch)
	if err != nil {
		// 更新步骤状态为失败
		step.Status = StepStatusFailed
		step.UpdateTime = time.Now()
		s.stepRepo.Save(ctx, step)
		s.statusTracker.UpdateStatusWithReason(ctx, stepID, status.StatusFailed, fmt.Sprintf("步骤处理失败: %v", err))
		return fmt.Errorf("step processing failed: %w", err)
	}

	// 更新步骤统计信息
	step.Statistics = processor.GetStatistics()
	step.Status = StepStatusCompleted
	step.UpdateTime = time.Now()

	if err := s.stepRepo.Save(ctx, step); err != nil {
		return fmt.Errorf("failed to update step: %w", err)
	}

	// 更新状态追踪
	return s.statusTracker.UpdateStatusWithReason(ctx, stepID, status.StatusCompleted, "步骤处理完成")
}

// GetStepInfo 获取步骤信息
func (s *SMSStepServiceImpl) GetStepInfo(ctx context.Context, stepID string) (*StepResult, error) {
	step, err := s.stepRepo.GetByID(ctx, stepID)
	if err != nil {
		return nil, fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return nil, ErrStepNotFound
	}

	// 获取状态追踪信息（可选）
	_, err = s.statusTracker.GetStatus(ctx, stepID)
	if err != nil {
		// 状态追踪信息获取失败不影响主要功能
	}

	return &StepResult{
		StepID:     step.ID,
		Status:     step.Status,
		StartTime:  step.CreateTime,
		EndTime:    step.UpdateTime,
		Duration:   step.UpdateTime.Sub(step.CreateTime),
		Statistics: step.Statistics,
	}, nil
}

// CancelStep 取消步骤
func (s *SMSStepServiceImpl) CancelStep(ctx context.Context, stepID string) error {
	// 获取步骤信息
	step, err := s.stepRepo.GetByID(ctx, stepID)
	if err != nil {
		return fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return ErrStepNotFound
	}

	// 只有待处理或运行中的步骤可以取消
	if step.Status != StepStatusPending && step.Status != StepStatusRunning {
		return ErrInvalidStepState
	}

	// 更新步骤状态
	step.Status = StepStatusCancelled
	step.UpdateTime = time.Now()

	if err := s.stepRepo.Save(ctx, step); err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}

	// 更新状态追踪
	return s.statusTracker.UpdateStatusWithReason(ctx, stepID, status.StatusCancelled, "用户取消步骤")
}

// RetryStep 重试步骤
func (s *SMSStepServiceImpl) RetryStep(ctx context.Context, stepID string) error {
	// 获取步骤信息
	step, err := s.stepRepo.GetByID(ctx, stepID)
	if err != nil {
		return fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return ErrStepNotFound
	}

	// 只有失败的步骤可以重试
	if step.Status != StepStatusFailed {
		return ErrInvalidStepState
	}

	// 重置步骤状态
	step.Status = StepStatusPending
	step.UpdateTime = time.Now()

	if err := s.stepRepo.Save(ctx, step); err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}

	// 更新状态追踪
	return s.statusTracker.UpdateStatusWithReason(ctx, stepID, status.StatusPending, "重试步骤")
}

// ListSteps 列出步骤
func (s *SMSStepServiceImpl) ListSteps(ctx context.Context, batchID string, stepType StepType, status StepStatus) ([]*StepResult, error) {
	steps, err := s.stepRepo.GetByBatchID(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to list steps: %w", err)
	}

	var results []*StepResult
	for _, step := range steps {
		// 过滤条件
		if stepType != "" && step.Type != stepType {
			continue
		}
		if status != "" && step.Status != status {
			continue
		}

		results = append(results, &StepResult{
			StepID:     step.ID,
			Status:     step.Status,
			StartTime:  step.CreateTime,
			EndTime:    step.UpdateTime,
			Duration:   step.UpdateTime.Sub(step.CreateTime),
			Statistics: step.Statistics,
		})
	}

	return results, nil
}

// DeleteStep 删除步骤
func (s *SMSStepServiceImpl) DeleteStep(ctx context.Context, stepID string) error {
	// 获取步骤信息
	step, err := s.stepRepo.GetByID(ctx, stepID)
	if err != nil {
		return fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return ErrStepNotFound
	}

	// 只有已完成、已取消或失败的步骤可以删除
	if step.Status == StepStatusRunning || step.Status == StepStatusPending {
		return ErrInvalidStepState
	}

	// 删除步骤
	if err := s.stepRepo.Delete(ctx, stepID); err != nil {
		return fmt.Errorf("failed to delete step: %w", err)
	}

	return nil
}

// GetStepStatistics 获取步骤统计
func (s *SMSStepServiceImpl) GetStepStatistics(ctx context.Context, stepID string) (*StepStatistics, error) {
	step, err := s.stepRepo.GetByID(ctx, stepID)
	if err != nil {
		return nil, fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return nil, ErrStepNotFound
	}

	return step.Statistics, nil
}

// UpdateStepStatus 更新步骤状态
func (s *SMSStepServiceImpl) UpdateStepStatus(ctx context.Context, stepID string, stepStatus StepStatus) error {
	// 获取步骤信息
	step, err := s.stepRepo.GetByID(ctx, stepID)
	if err != nil {
		return fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return ErrStepNotFound
	}

	// 更新步骤状态
	step.Status = stepStatus
	step.UpdateTime = time.Now()

	if err := s.stepRepo.Save(ctx, step); err != nil {
		return fmt.Errorf("failed to update step status: %w", err)
	}

	// 映射到状态追踪状态
	var trackStatus status.Status
	switch stepStatus {
	case StepStatusCompleted:
		trackStatus = status.StatusCompleted
	case StepStatusFailed:
		trackStatus = status.StatusFailed
	case StepStatusRunning:
		trackStatus = status.StatusProcessing
	case StepStatusCancelled:
		trackStatus = status.StatusCancelled
	default:
		trackStatus = status.StatusPending
	}

	return s.statusTracker.UpdateStatusWithReason(ctx, stepID, trackStatus, "状态更新")
}

// GetStepProgress 获取步骤进度
func (s *SMSStepServiceImpl) GetStepProgress(ctx context.Context, stepID string) (*StepProgress, error) {
	step, err := s.stepRepo.GetByID(ctx, stepID)
	if err != nil {
		return nil, fmt.Errorf("failed to get step: %w", err)
	}
	if step == nil {
		return nil, ErrStepNotFound
	}

	// 计算进度
	progress := s.calculateStepProgress(step)
	return &progress, nil
}

// calculateStepProgress 计算步骤进度
func (s *SMSStepServiceImpl) calculateStepProgress(step *SmsStep) StepProgress {
	switch step.Status {
	case StepStatusPending:
		return StepProgress{
			StepID:     step.ID,
			Stage:      "pending",
			Percentage: 0,
			Message:    "等待处理",
		}
	case StepStatusRunning:
		// 根据统计信息计算进度
		var percentage float64
		if step.Statistics.Total > 0 {
			processed := step.Statistics.Success + step.Statistics.Failure
			percentage = float64(processed) / float64(step.Statistics.Total) * 100
		}
		return StepProgress{
			StepID:      step.ID,
			Stage:       "running",
			Percentage:  percentage,
			Message:     "正在处理",
			CurrentItem: int(step.Statistics.Success + step.Statistics.Failure),
			TotalItems:  int(step.Statistics.Total),
		}
	case StepStatusCompleted:
		return StepProgress{
			StepID:     step.ID,
			Stage:      "completed",
			Percentage: 100,
			Message:    "已完成",
		}
	case StepStatusFailed:
		return StepProgress{
			StepID:     step.ID,
			Stage:      "failed",
			Percentage: 0,
			Message:    "处理失败",
		}
	case StepStatusCancelled:
		return StepProgress{
			StepID:     step.ID,
			Stage:      "cancelled",
			Percentage: 0,
			Message:    "已取消",
		}
	default:
		return StepProgress{
			StepID:     step.ID,
			Stage:      "unknown",
			Percentage: 0,
			Message:    "未知状态",
		}
	}
}

// generateStepID 生成步骤ID
func generateStepID() string {
	return fmt.Sprintf("step_%d", time.Now().UnixNano())
}
