package batch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	delivery "kratos-demo/internal/biz/sms/delivery"
	status "kratos-demo/internal/biz/status"
)

// SMS批次处理相关错误定义
var (
	ErrBatchNotFound     = errors.New("batch not found")
	ErrBatchExists       = errors.New("batch already exists")
	ErrInvalidBatchState = errors.New("invalid batch state")
	ErrBatchTimeout      = errors.New("batch operation timeout")
	ErrBatchFailed       = errors.New("batch operation failed")
)

// BatchConfig 批次配置
type BatchConfig struct {
	MaxRetries    int           `json:"max_retries"`
	RetryInterval time.Duration `json:"retry_interval"`
	Timeout       time.Duration `json:"timeout"`
	BatchSize     int           `json:"batch_size"`
	Concurrency   int           `json:"concurrency"`
	EnableMetrics bool          `json:"enable_metrics"`
	EnableTracing bool          `json:"enable_tracing"`
}

// BatchResult 批次处理结果
type BatchResult struct {
	BatchID      string            `json:"batch_id"`
	Status       SmsBatchStatus    `json:"status"`
	TotalCount   int64             `json:"total_count"`
	SuccessCount int64             `json:"success_count"`
	FailedCount  int64             `json:"failed_count"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      *time.Time        `json:"end_time,omitempty"`
	Duration     *time.Duration    `json:"duration,omitempty"`
	Errors       []BatchError      `json:"errors,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// BatchError 批次错误信息
type BatchError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"`
}

// BatchStatus 批次状态
type BatchStatus struct {
	BatchID       string         `json:"batch_id"`
	CurrentStatus SmsBatchStatus `json:"current_status"`
	Progress      BatchProgress  `json:"progress"`
	LastUpdated   time.Time      `json:"last_updated"`
	EstimatedTime *time.Time     `json:"estimated_time,omitempty"`
}

// BatchProgress 批次进度
type BatchProgress struct {
	Total      int64   `json:"total"`
	Processed  int64   `json:"processed"`
	Successful int64   `json:"successful"`
	Failed     int64   `json:"failed"`
	Percentage float64 `json:"percentage"`
}

// SMSBatchService SMS批次处理业务服务接口
type SMSBatchService interface {
	// CreateBatch 创建新的批次任务
	CreateBatch(ctx context.Context, batch *SmsBatch) (*BatchResult, error)

	// StartBatch 启动批次处理
	StartBatch(ctx context.Context, batchID string) error

	// ProcessBatch 处理批次任务
	ProcessBatch(ctx context.Context, batchID string) error

	// GetBatchInfo 获取批次信息
	GetBatchInfo(ctx context.Context, batchID string) (*BatchStatus, error)

	// CancelBatch 取消批次处理
	CancelBatch(ctx context.Context, batchID string) error

	// RetryBatch 重试失败的批次
	RetryBatch(ctx context.Context, batchID string) error

	// ListBatches 列出批次
	ListBatches(ctx context.Context, status SmsBatchStatus, limit int) ([]*BatchStatus, error)

	// DeleteBatch 删除批次
	DeleteBatch(ctx context.Context, batchID string) error

	// GetBatchStats 获取批次统计信息
	GetBatchStats(ctx context.Context, batchID string) (*BatchResult, error)
}

// SMSBatchServiceImpl SMS批次处理业务服务实现
type SMSBatchServiceImpl struct {
	batchRepo     SmsBatchRepo
	deliveryRepo  delivery.SmsDeliveryPackRepo
	statusTracker status.StatusTracker
	logger        *log.Helper
	config        BatchConfig
}

// NewSMSBatchService 创建SMS批次处理服务
func NewSMSBatchService(
	batchRepo SmsBatchRepo,
	deliveryRepo delivery.SmsDeliveryPackRepo,
	statusTracker status.StatusTracker,
	logger log.Logger,
	config BatchConfig,
) SMSBatchService {
	return &SMSBatchServiceImpl{
		batchRepo:     batchRepo,
		deliveryRepo:  deliveryRepo,
		statusTracker: statusTracker,
		logger:        log.NewHelper(logger),
		config:        config,
	}
}

// CreateBatch 创建新的批次任务
func (s *SMSBatchServiceImpl) CreateBatch(ctx context.Context, batch *SmsBatch) (*BatchResult, error) {
	s.logger.WithContext(ctx).Infof("Creating batch: %s", batch.BatchId)

	// 验证批次数据
	if batch.BatchId == "" {
		return nil, status.ErrInvalidRequest
	}

	// 检查批次是否已存在
	existingBatch, err := s.batchRepo.GetSmsBatchByBatchId(ctx, batch.BatchId)
	if err == nil && existingBatch != nil {
		return nil, ErrBatchExists
	}

	// 设置批次初始状态
	batch.Status = SmsBatchStatusInitial
	batch.CreateTime = time.Now()
	batch.UpdateTime = time.Now()

	// 保存批次
	if err := s.batchRepo.CreateSmsBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to create batch: %w", err)
	}

	// 创建状态追踪
	trackReq := &status.TrackRequest{
		ID:        batch.BatchId,
		Type:      "sms_batch",
		Status:    status.StatusPending,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"batch_type":  "sms",
			"total_count": fmt.Sprintf("%d", batch.TotalCount),
		},
	}

	if err := s.statusTracker.TrackStatus(ctx, trackReq); err != nil {
		return nil, fmt.Errorf("failed to track batch status: %w", err)
	}

	// 返回批次结果
	return &BatchResult{
		BatchID:      batch.BatchId,
		Status:       batch.Status,
		TotalCount:   batch.TotalCount,
		SuccessCount: batch.SuccessCount,
		FailedCount:  batch.FailedCount,
		StartTime:    batch.CreateTime,
		Metadata: map[string]string{
			"campaign_id": batch.CampaignId,
			"task_id":     batch.TaskId,
		},
	}, nil
}

// StartBatch 启动批次处理
func (s *SMSBatchServiceImpl) StartBatch(ctx context.Context, batchID string) error {
	s.logger.WithContext(ctx).Infof("Starting batch: %s", batchID)

	// 获取批次信息
	batch, err := s.batchRepo.GetSmsBatchByBatchId(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}

	if batch == nil {
		return ErrBatchNotFound
	}

	// 检查批次状态
	if batch.Status != SmsBatchStatusInitial && batch.Status != SmsBatchStatusReady {
		return ErrInvalidBatchState
	}

	// 更新批次状态为运行中
	batch.Status = SmsBatchStatusRunning
	batch.UpdateTime = time.Now()
	batch.LastExecuteTime = &batch.UpdateTime

	if err := s.batchRepo.UpdateSmsBatch(ctx, batch); err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	// 更新状态追踪
	if err := s.statusTracker.UpdateStatus(ctx, batchID, status.StatusProcessing); err != nil {
		s.logger.WithContext(ctx).Warnf("Failed to update status tracker: %v", err)
	}

	return nil
}

// ProcessBatch 处理批次任务
func (s *SMSBatchServiceImpl) ProcessBatch(ctx context.Context, batchID string) error {
	s.logger.WithContext(ctx).Infof("Processing batch: %s", batchID)

	// 获取批次信息
	batch, err := s.batchRepo.GetSmsBatchByBatchId(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}

	if batch == nil {
		return ErrBatchNotFound
	}

	// 检查批次状态
	if batch.Status != SmsBatchStatusRunning {
		return ErrInvalidBatchState
	}

	// 这里可以添加具体的批次处理逻辑
	// 例如：处理投递包、发送短信等

	// 模拟处理完成
	batch.Status = SmsBatchStatusCompletedSucceeded
	batch.UpdateTime = time.Now()
	batch.ProcessedCount = batch.TotalCount
	batch.SuccessCount = batch.TotalCount

	if err := s.batchRepo.UpdateSmsBatch(ctx, batch); err != nil {
		return fmt.Errorf("failed to update batch: %w", err)
	}

	// 更新状态追踪
	if err := s.statusTracker.UpdateStatus(ctx, batchID, status.StatusCompleted); err != nil {
		s.logger.WithContext(ctx).Warnf("Failed to update status tracker: %v", err)
	}

	return nil
}

// GetBatchInfo 获取批次信息
func (s *SMSBatchServiceImpl) GetBatchInfo(ctx context.Context, batchID string) (*BatchStatus, error) {
	batch, err := s.batchRepo.GetSmsBatchByBatchId(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	if batch == nil {
		return nil, ErrBatchNotFound
	}

	progress := s.calculateProgress(batch)

	return &BatchStatus{
		BatchID:       batch.BatchId,
		CurrentStatus: batch.Status,
		Progress:      progress,
		LastUpdated:   batch.UpdateTime,
	}, nil
}

// CancelBatch 取消批次处理
func (s *SMSBatchServiceImpl) CancelBatch(ctx context.Context, batchID string) error {
	s.logger.WithContext(ctx).Infof("Cancelling batch: %s", batchID)

	batch, err := s.batchRepo.GetSmsBatchByBatchId(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}

	if batch == nil {
		return ErrBatchNotFound
	}

	// 只有运行中或暂停的批次可以取消
	if batch.Status != SmsBatchStatusRunning && batch.Status != SmsBatchStatusPaused {
		return ErrInvalidBatchState
	}

	// 更新批次状态
	batch.Status = SmsBatchStatusCompletedFailed
	batch.UpdateTime = time.Now()

	if err := s.batchRepo.UpdateSmsBatch(ctx, batch); err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	// 更新状态追踪
	if err := s.statusTracker.UpdateStatus(ctx, batchID, status.StatusFailed); err != nil {
		s.logger.WithContext(ctx).Warnf("Failed to update status tracker: %v", err)
	}

	return nil
}

// RetryBatch 重试失败的批次
func (s *SMSBatchServiceImpl) RetryBatch(ctx context.Context, batchID string) error {
	s.logger.WithContext(ctx).Infof("Retrying batch: %s", batchID)

	batch, err := s.batchRepo.GetSmsBatchByBatchId(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}

	if batch == nil {
		return ErrBatchNotFound
	}

	// 只有失败的批次可以重试
	if batch.Status != SmsBatchStatusCompletedFailed {
		return ErrInvalidBatchState
	}

	// 重置批次状态
	batch.Status = SmsBatchStatusReady
	batch.UpdateTime = time.Now()
	batch.ExecuteTimes++

	if err := s.batchRepo.UpdateSmsBatch(ctx, batch); err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	// 更新状态追踪
	if err := s.statusTracker.UpdateStatus(ctx, batchID, status.StatusRetrying); err != nil {
		s.logger.WithContext(ctx).Warnf("Failed to update status tracker: %v", err)
	}

	return nil
}

// ListBatches 列出批次
func (s *SMSBatchServiceImpl) ListBatches(ctx context.Context, status SmsBatchStatus, limit int) ([]*BatchStatus, error) {
	batches, err := s.batchRepo.ListSmsBatchByStatus(ctx, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list batches: %w", err)
	}

	result := make([]*BatchStatus, 0, len(batches))
	for _, batch := range batches {
		progress := s.calculateProgress(batch)
		result = append(result, &BatchStatus{
			BatchID:       batch.BatchId,
			CurrentStatus: batch.Status,
			Progress:      progress,
			LastUpdated:   batch.UpdateTime,
		})
	}

	return result, nil
}

// DeleteBatch 删除批次
func (s *SMSBatchServiceImpl) DeleteBatch(ctx context.Context, batchID string) error {
	s.logger.WithContext(ctx).Infof("Deleting batch: %s", batchID)

	batch, err := s.batchRepo.GetSmsBatchByBatchId(ctx, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch: %w", err)
	}

	if batch == nil {
		return ErrBatchNotFound
	}

	// 删除批次
	if err := s.batchRepo.DeleteSmsBatch(ctx, batch.ID); err != nil {
		return fmt.Errorf("failed to delete batch: %w", err)
	}

	// 删除状态追踪
	if err := s.statusTracker.DeleteStatus(ctx, batchID); err != nil {
		s.logger.WithContext(ctx).Warnf("Failed to delete status tracker: %v", err)
	}

	return nil
}

// GetBatchStats 获取批次统计信息
func (s *SMSBatchServiceImpl) GetBatchStats(ctx context.Context, batchID string) (*BatchResult, error) {
	batch, err := s.batchRepo.GetSmsBatchByBatchId(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	if batch == nil {
		return nil, ErrBatchNotFound
	}

	// 计算持续时间
	var duration *time.Duration
	var endTime *time.Time
	if batch.LastExecuteTime != nil {
		d := batch.LastExecuteTime.Sub(batch.CreateTime)
		duration = &d
		endTime = batch.LastExecuteTime
	}

	return &BatchResult{
		BatchID:      batch.BatchId,
		Status:       batch.Status,
		TotalCount:   batch.TotalCount,
		SuccessCount: batch.SuccessCount,
		FailedCount:  batch.FailedCount,
		StartTime:    batch.CreateTime,
		EndTime:      endTime,
		Duration:     duration,
		Metadata: map[string]string{
			"campaign_id":   batch.CampaignId,
			"task_id":       batch.TaskId,
			"execute_times": fmt.Sprintf("%d", batch.ExecuteTimes),
		},
	}, nil
}

// calculateProgress 计算批次进度
func (s *SMSBatchServiceImpl) calculateProgress(batch *SmsBatch) BatchProgress {
	progress := BatchProgress{
		Total:      batch.TotalCount,
		Processed:  batch.ProcessedCount,
		Successful: batch.SuccessCount,
		Failed:     batch.FailedCount,
	}

	if batch.TotalCount > 0 {
		progress.Percentage = float64(batch.ProcessedCount) / float64(batch.TotalCount) * 100
	}

	return progress
}
