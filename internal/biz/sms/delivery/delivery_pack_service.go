package delivery

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// 投递包服务相关错误
var (
	ErrDeliveryPackNotFound     = fmt.Errorf("delivery pack not found")
	ErrDeliveryPackExists       = fmt.Errorf("delivery pack already exists")
	ErrInvalidDeliveryPackState = fmt.Errorf("invalid delivery pack state")
	ErrDeliveryPackTimeout      = fmt.Errorf("delivery pack timeout")
	ErrInvalidRequest           = fmt.Errorf("invalid request")
	ErrInvalidStatus            = fmt.Errorf("invalid status")
)

// Status 状态类型
type Status string

// 状态常量
const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusCompleted  Status = "completed"
	StatusFailed     Status = "failed"
	StatusCancelled  Status = "cancelled"
	StatusRetrying   Status = "retrying"
)

// SmsBatchStatus 扩展状态
const (
	SmsBatchStatusInitial           SmsBatchStatus = "INITIAL"
	SmsBatchStatusReady             SmsBatchStatus = "READY"
	SmsBatchStatusCompletedSucceeded SmsBatchStatus = "COMPLETED_SUCCEEDED"
	SmsBatchStatusCompletedFailed    SmsBatchStatus = "COMPLETED_FAILED"
)

// TrackRequest 状态追踪请求
type TrackRequest struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Status    Status            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata"`
}

// StatusInfo 状态信息
type StatusInfo struct {
	ID        string            `json:"id"`
	Status    Status            `json:"status"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata"`
}

// StatusTracker 状态追踪器接口
type StatusTracker interface {
	TrackStatus(ctx context.Context, req *TrackRequest) error
	UpdateStatusWithReason(ctx context.Context, id string, status Status, reason string) error
	GetStatus(ctx context.Context, id string) (*StatusInfo, error)
	DeleteStatus(ctx context.Context, id string) error
}

// DeliveryConfig 投递配置
type DeliveryConfig struct {
	Timeout    time.Duration `json:"timeout"`
	MaxRetries int           `json:"max_retries"`
	BatchSize  int           `json:"batch_size"`
}

// DeliveryResult 投递结果
type DeliveryResult struct {
	PackID        string            `json:"pack_id"`
	BatchID       string            `json:"batch_id"`
	Status        SmsBatchStatus    `json:"status"`
	PhoneNumber   string            `json:"phone_number"`
	Content       string            `json:"content"`
	SentTime      *time.Time        `json:"sent_time,omitempty"`
	DeliveredTime *time.Time        `json:"delivered_time,omitempty"`
	Duration      *time.Duration    `json:"duration,omitempty"`
	RetryCount    int32             `json:"retry_count"`
	Error         *DeliveryError    `json:"error,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// DeliveryError 投递错误
type DeliveryError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Retryable bool      `json:"retryable"`
}

// DeliveryProgress 投递进度
type DeliveryProgress struct {
	Stage      string  `json:"stage"`
	Percentage float64 `json:"percentage"`
	Message    string  `json:"message"`
	Total      int     `json:"total"`
	Sent       int     `json:"sent"`
	Delivered  int     `json:"delivered"`
	Failed     int     `json:"failed"`
}

// DeliveryStatus 投递状态
type DeliveryStatus struct {
	PackID        string           `json:"pack_id"`
	CurrentStatus SmsBatchStatus   `json:"current_status"`
	Progress      DeliveryProgress `json:"progress"`
	LastUpdated   time.Time        `json:"last_updated"`
	EstimatedTime *time.Time       `json:"estimated_time,omitempty"`
}

// SMSDeliveryPackService SMS投递包服务接口
type SMSDeliveryPackService interface {
	// CreateDeliveryPack 创建投递包
	CreateDeliveryPack(ctx context.Context, pack *SmsDeliveryPack) (*DeliveryResult, error)

	// SendDeliveryPack 发送投递包
	SendDeliveryPack(ctx context.Context, packID string) (*DeliveryResult, error)

	// ProcessDeliveryPack 处理投递包（异步处理）
	ProcessDeliveryPack(ctx context.Context, packID string) error

	// GetDeliveryPackInfo 获取投递包信息
	GetDeliveryPackInfo(ctx context.Context, packID string) (*DeliveryStatus, error)

	// CancelDeliveryPack 取消投递包
	CancelDeliveryPack(ctx context.Context, packID string) error

	// RetryDeliveryPack 重试投递包
	RetryDeliveryPack(ctx context.Context, packID string) error

	// ListDeliveryPacks 列出投递包
	ListDeliveryPacks(ctx context.Context, batchID string, status SmsBatchStatus, limit int) ([]*DeliveryStatus, error)

	// DeleteDeliveryPack 删除投递包
	DeleteDeliveryPack(ctx context.Context, packID string) error

	// GetDeliveryStats 获取投递统计信息
	GetDeliveryStats(ctx context.Context, packID string) (*DeliveryResult, error)

	// BatchSendDeliveryPacks 批量发送投递包
	BatchSendDeliveryPacks(ctx context.Context, packIDs []string) ([]*DeliveryResult, error)

	// GetDeliveryReports 获取投递报告
	GetDeliveryReports(ctx context.Context, packID string) ([]*DeliveryReport, error)

	// UpdateDeliveryStatus 更新投递状态（用于接收运营商回调）
	UpdateDeliveryStatus(ctx context.Context, packID string, status SmsBatchStatus, report *DeliveryReport) error
}

// SMSDeliveryPackServiceImpl SMS投递包服务实现
type SMSDeliveryPackServiceImpl struct {
	deliveryRepo  SmsDeliveryPackRepo
	reportRepo    DeliveryReportRepo
	statusTracker StatusTracker
	config        *DeliveryConfig
	logger        *log.Helper
}

// NewSMSDeliveryPackService 创建SMS投递包服务
func NewSMSDeliveryPackService(
	deliveryRepo SmsDeliveryPackRepo,
	reportRepo DeliveryReportRepo,
	statusTracker StatusTracker,
	logger log.Logger,
) SMSDeliveryPackService {
	return &SMSDeliveryPackServiceImpl{
		deliveryRepo:  deliveryRepo,
		reportRepo:    reportRepo,
		statusTracker: statusTracker,
		config: &DeliveryConfig{
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			BatchSize:  100,
		},
		logger: log.NewHelper(logger),
	}
}

// CreateDeliveryPack 创建投递包
func (s *SMSDeliveryPackServiceImpl) CreateDeliveryPack(ctx context.Context, pack *SmsDeliveryPack) (*DeliveryResult, error) {
	s.logger.WithContext(ctx).Infof("Creating delivery pack: %s", pack.ID)

	// 验证投递包数据
	if pack.ID == "" {
		return nil, ErrInvalidRequest
	}

	// 设置投递包初始状态
	pack.Status = SmsBatchStatusInitial
	pack.CreateTime = time.Now()
	pack.UpdateTime = time.Now()

	// 保存投递包
	if err := s.deliveryRepo.Save(ctx, pack); err != nil {
		return nil, fmt.Errorf("failed to create delivery pack: %w", err)
	}

	// 创建状态追踪
	trackReq := &TrackRequest{
		ID:        pack.ID,
		Type:      "sms_delivery_pack",
		Status:    StatusPending,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"batch_id":   pack.SmsBatchId,
			"mobile":     pack.Mobile,
			"message_id": pack.MessageId,
		},
	}
	if err := s.statusTracker.TrackStatus(ctx, trackReq); err != nil {
		return nil, fmt.Errorf("failed to track status: %w", err)
	}

	// 返回投递结果
	return &DeliveryResult{
		PackID:      pack.ID,
		BatchID:     pack.SmsBatchId,
		Status:      pack.Status,
		PhoneNumber: pack.Mobile,
		Content:     "", // 内容需要从参数中构建
		RetryCount:  pack.ExecuteTimes,
		Metadata: map[string]string{
			"message_id":    pack.MessageId,
			"provider_type": string(pack.ProviderType),
		},
	}, nil
}

// SendDeliveryPack 发送投递包
func (s *SMSDeliveryPackServiceImpl) SendDeliveryPack(ctx context.Context, packID string) (*DeliveryResult, error) {
	s.logger.WithContext(ctx).Infof("Sending delivery pack: %s", packID)

	// 获取投递包信息
	pack, err := s.deliveryRepo.GetByID(ctx, packID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery pack: %w", err)
	}
	if pack == nil {
		return nil, ErrDeliveryPackNotFound
	}

	// 检查投递包状态
	if pack.Status != SmsBatchStatusInitial {
		return nil, ErrInvalidStatus
	}

	// 更新状态为就绪
	pack.Status = SmsBatchStatusReady
	pack.UpdateTime = time.Now()

	// 保存状态更新
	if err := s.deliveryRepo.Save(ctx, pack); err != nil {
		return nil, fmt.Errorf("failed to update delivery pack: %w", err)
	}

	// 更新状态追踪
	if err := s.statusTracker.UpdateStatusWithReason(ctx, packID, StatusProcessing, "开始发送投递包"); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	// 返回投递结果
	return &DeliveryResult{
		PackID:      pack.ID,
		BatchID:     pack.SmsBatchId,
		Status:      pack.Status,
		PhoneNumber: pack.Mobile,
		Content:     "", // 内容需要从参数中构建
		RetryCount:  pack.ExecuteTimes,
	}, nil
}

// ProcessDeliveryPack 处理投递包（异步处理）
func (s *SMSDeliveryPackServiceImpl) ProcessDeliveryPack(ctx context.Context, packID string) error {
	s.logger.WithContext(ctx).Infof("Processing delivery pack: %s", packID)

	// 获取投递包信息
	pack, err := s.deliveryRepo.GetByID(ctx, packID)
	if err != nil {
		return fmt.Errorf("failed to get delivery pack: %w", err)
	}
	if pack == nil {
		return ErrDeliveryPackNotFound
	}

	// 检查投递包状态
	if pack.Status != SmsBatchStatusInitial && pack.Status != SmsBatchStatusReady {
		return ErrInvalidDeliveryPackState
	}

	// 更新投递包状态为处理中
	pack.Status = SmsBatchStatusReady
	pack.UpdateTime = time.Now()

	if err := s.deliveryRepo.Save(ctx, pack); err != nil {
		return fmt.Errorf("failed to update delivery pack status: %w", err)
	}

	// 更新状态追踪
	return s.statusTracker.UpdateStatusWithReason(ctx, packID, StatusProcessing, "开始处理投递包")
}

// GetDeliveryPackInfo 获取投递包信息
func (s *SMSDeliveryPackServiceImpl) GetDeliveryPackInfo(ctx context.Context, packID string) (*DeliveryStatus, error) {
	// 获取投递包信息
	pack, err := s.deliveryRepo.GetByID(ctx, packID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery pack: %w", err)
	}
	if pack == nil {
		return nil, ErrDeliveryPackNotFound
	}

	// 获取状态追踪信息
	statusInfo, err := s.statusTracker.GetStatus(ctx, packID)
	if err != nil {
		return nil, fmt.Errorf("failed to get status info: %w", err)
	}

	// 计算进度
	progress := s.calculateDeliveryProgress(pack)

	return &DeliveryStatus{
		PackID:        pack.ID,
		CurrentStatus: pack.Status,
		Progress:      progress,
		LastUpdated:   statusInfo.UpdatedAt,
	}, nil
}

// CancelDeliveryPack 取消投递包
func (s *SMSDeliveryPackServiceImpl) CancelDeliveryPack(ctx context.Context, packID string) error {
	s.logger.WithContext(ctx).Infof("Cancelling delivery pack: %s", packID)

	pack, err := s.deliveryRepo.GetByID(ctx, packID)
	if err != nil {
		return fmt.Errorf("failed to get delivery pack: %w", err)
	}
	if pack == nil {
		return ErrDeliveryPackNotFound
	}

	// 只有初始或就绪状态的投递包可以取消
	if pack.Status != SmsBatchStatusInitial && pack.Status != SmsBatchStatusReady {
		return ErrInvalidDeliveryPackState
	}

	// 更新投递包状态
	pack.Status = SmsBatchStatusCompletedFailed
	pack.UpdateTime = time.Now()

	if err := s.deliveryRepo.Save(ctx, pack); err != nil {
		return fmt.Errorf("failed to update delivery pack status: %w", err)
	}

	// 更新状态追踪
	return s.statusTracker.UpdateStatusWithReason(ctx, packID, StatusCancelled, "用户取消投递包")
}

// RetryDeliveryPack 重试投递包
func (s *SMSDeliveryPackServiceImpl) RetryDeliveryPack(ctx context.Context, packID string) error {
	s.logger.WithContext(ctx).Infof("Retrying delivery pack: %s", packID)

	pack, err := s.deliveryRepo.GetByID(ctx, packID)
	if err != nil {
		return fmt.Errorf("failed to get delivery pack: %w", err)
	}
	if pack == nil {
		return ErrDeliveryPackNotFound
	}

	// 只有失败的投递包可以重试
	if pack.Status != SmsBatchStatusCompletedFailed {
		return ErrInvalidDeliveryPackState
	}

	// 检查重试次数
	if pack.ExecuteTimes >= int32(s.config.MaxRetries) {
		return fmt.Errorf("max retry count exceeded: %d", s.config.MaxRetries)
	}

	// 重置投递包状态
	pack.Status = SmsBatchStatusInitial
	pack.UpdateTime = time.Now()
	pack.ExecuteTimes++

	if err := s.deliveryRepo.Save(ctx, pack); err != nil {
		return fmt.Errorf("failed to update delivery pack status: %w", err)
	}

	// 更新状态追踪
	return s.statusTracker.UpdateStatusWithReason(ctx, packID, StatusRetrying, fmt.Sprintf("重试投递包，第%d次", pack.ExecuteTimes))
}

// ListDeliveryPacks 列出投递包
func (s *SMSDeliveryPackServiceImpl) ListDeliveryPacks(ctx context.Context, batchID string, status SmsBatchStatus, limit int) ([]*DeliveryStatus, error) {
	packs, err := s.deliveryRepo.GetByBatchID(ctx, batchID)
	if err != nil {
		return nil, fmt.Errorf("failed to list delivery packs: %w", err)
	}

	var result []*DeliveryStatus
	count := 0
	for _, pack := range packs {
		if count >= limit {
			break
		}

		// 过滤状态
		if status != "" && pack.Status != status {
			continue
		}

		progress := s.calculateDeliveryProgress(pack)
		result = append(result, &DeliveryStatus{
			PackID:        pack.ID,
			CurrentStatus: pack.Status,
			Progress:      progress,
			LastUpdated:   pack.UpdateTime,
		})
		count++
	}

	return result, nil
}

// DeleteDeliveryPack 删除投递包
func (s *SMSDeliveryPackServiceImpl) DeleteDeliveryPack(ctx context.Context, packID string) error {
	s.logger.WithContext(ctx).Infof("Deleting delivery pack: %s", packID)

	pack, err := s.deliveryRepo.GetByID(ctx, packID)
	if err != nil {
		return fmt.Errorf("failed to get delivery pack: %w", err)
	}
	if pack == nil {
		return ErrDeliveryPackNotFound
	}

	// 删除投递包
	if err := s.deliveryRepo.Delete(ctx, pack.ID); err != nil {
		return fmt.Errorf("failed to delete delivery pack: %w", err)
	}

	// 删除状态追踪
	if err := s.statusTracker.DeleteStatus(ctx, packID); err != nil {
		s.logger.WithContext(ctx).Warnf("Failed to delete status tracking: %v", err)
	}

	return nil
}

// GetDeliveryStats 获取投递统计信息
func (s *SMSDeliveryPackServiceImpl) GetDeliveryStats(ctx context.Context, packID string) (*DeliveryResult, error) {
	pack, err := s.deliveryRepo.GetByID(ctx, packID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery pack: %w", err)
	}
	if pack == nil {
		return nil, ErrDeliveryPackNotFound
	}

	return &DeliveryResult{
		PackID:      pack.ID,
		BatchID:     pack.SmsBatchId,
		Status:      pack.Status,
		PhoneNumber: pack.Mobile,
		Content:     "", // 内容需要从参数中构建
		RetryCount:  pack.ExecuteTimes,
		Metadata: map[string]string{
			"message_id":    pack.MessageId,
			"provider_type": string(pack.ProviderType),
		},
	}, nil
}

// BatchSendDeliveryPacks 批量发送投递包
func (s *SMSDeliveryPackServiceImpl) BatchSendDeliveryPacks(ctx context.Context, packIDs []string) ([]*DeliveryResult, error) {
	s.logger.WithContext(ctx).Infof("Batch sending %d delivery packs", len(packIDs))

	var results []*DeliveryResult
	for _, packID := range packIDs {
		result, err := s.SendDeliveryPack(ctx, packID)
		if err != nil {
			results = append(results, &DeliveryResult{
				PackID: packID,
				Status: SmsBatchStatusCompletedFailed,
				Error: &DeliveryError{
					Code:      "SEND_FAILED",
					Message:   err.Error(),
					Timestamp: time.Now(),
					Retryable: true,
				},
			})
		} else {
			results = append(results, result)
		}
	}

	return results, nil
}

// GetDeliveryReports 获取投递报告
func (s *SMSDeliveryPackServiceImpl) GetDeliveryReports(ctx context.Context, packID string) ([]*DeliveryReport, error) {
	reports, err := s.reportRepo.GetByPackID(ctx, packID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery reports: %w", err)
	}
	return reports, nil
}

// UpdateDeliveryStatus 更新投递状态（用于接收运营商回调）
func (s *SMSDeliveryPackServiceImpl) UpdateDeliveryStatus(ctx context.Context, packID string, status SmsBatchStatus, report *DeliveryReport) error {
	s.logger.WithContext(ctx).Infof("Updating delivery status for pack %s to %s", packID, status)

	pack, err := s.deliveryRepo.GetByID(ctx, packID)
	if err != nil {
		return fmt.Errorf("failed to get delivery pack: %w", err)
	}
	if pack == nil {
		return ErrDeliveryPackNotFound
	}

	// 更新投递包状态
	pack.Status = status
	pack.UpdateTime = time.Now()

	if err := s.deliveryRepo.Save(ctx, pack); err != nil {
		return fmt.Errorf("failed to update delivery pack: %w", err)
	}

	// 保存投递报告
	if report != nil {
		if err := s.reportRepo.Save(ctx, report); err != nil {
			s.logger.WithContext(ctx).Warnf("Failed to save delivery report: %v", err)
		}
	}

	// 更新状态追踪
	var trackStatus Status
	switch status {
	case SmsBatchStatusCompletedSucceeded:
		trackStatus = StatusCompleted
	case SmsBatchStatusCompletedFailed:
		trackStatus = StatusFailed
	default:
		trackStatus = StatusProcessing
	}

	return s.statusTracker.UpdateStatusWithReason(ctx, packID, trackStatus, "运营商回调更新状态")
}

// calculateDeliveryProgress 计算投递进度
func (s *SMSDeliveryPackServiceImpl) calculateDeliveryProgress(pack *SmsDeliveryPack) DeliveryProgress {
	switch pack.Status {
	case SmsBatchStatusInitial:
		return DeliveryProgress{
			Stage:      "preparing",
			Percentage: 10.0,
			Message:    "准备发送",
		}
	case SmsBatchStatusReady:
		return DeliveryProgress{
			Stage:      "ready",
			Percentage: 50.0,
			Message:    "准备就绪",
		}
	case SmsBatchStatusCompletedSucceeded:
		return DeliveryProgress{
			Stage:      "delivered",
			Percentage: 100.0,
			Message:    "已送达",
		}
	case SmsBatchStatusCompletedFailed:
		return DeliveryProgress{
			Stage:      "failed",
			Percentage: 0.0,
			Message:    "发送失败",
		}
	default:
		return DeliveryProgress{
			Stage:      "unknown",
			Percentage: 0.0,
			Message:    "未知状态",
		}
	}
}
