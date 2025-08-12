package status

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// 状态追踪器相关错误定义
var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrStatusNotFound = errors.New("status not found")
	ErrStatusExists   = errors.New("status already exists")
	ErrInvalidStatus  = errors.New("invalid status")
	ErrTrackerTimeout = errors.New("tracker operation timeout")
	ErrTrackerFailed  = errors.New("tracker operation failed")
)

// Status 表示状态枚举
type Status string

const (
	// StatusPending 待处理
	StatusPending Status = "pending"
	// StatusProcessing 处理中
	StatusProcessing Status = "processing"
	// StatusCompleted 已完成
	StatusCompleted Status = "completed"
	// StatusFailed 失败
	StatusFailed Status = "failed"
	// StatusTimeout 超时
	StatusTimeout Status = "timeout"
	// StatusRetrying 重试中
	StatusRetrying Status = "retrying"
	// StatusCancelled 已取消
	StatusCancelled Status = "cancelled"
)

// TrackRequest 状态追踪请求
type TrackRequest struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Status     Status            `json:"status"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	ExpireTime *time.Time        `json:"expire_time,omitempty"`
	RetryCount int               `json:"retry_count,omitempty"`
	MaxRetries int               `json:"max_retries,omitempty"`
}

// StatusInfo 状态信息
type StatusInfo struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Status     Status            `json:"status"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	ExpireTime *time.Time        `json:"expire_time,omitempty"`
	RetryCount int               `json:"retry_count"`
	MaxRetries int               `json:"max_retries"`
	ErrorMsg   string            `json:"error_msg,omitempty"`
	History    []StatusChange    `json:"history,omitempty"`
}

// StatusChange 状态变更记录
type StatusChange struct {
	FromStatus Status    `json:"from_status"`
	ToStatus   Status    `json:"to_status"`
	Timestamp  time.Time `json:"timestamp"`
	Reason     string    `json:"reason,omitempty"`
}

// StatusTracker 状态追踪器接口
type StatusTracker interface {
	// TrackStatus 追踪状态
	TrackStatus(ctx context.Context, req *TrackRequest) error

	// GetStatus 获取状态信息
	GetStatus(ctx context.Context, id string) (*StatusInfo, error)

	// UpdateStatus 更新状态
	UpdateStatus(ctx context.Context, id string, status Status) error

	// UpdateStatusWithReason 更新状态并记录原因
	UpdateStatusWithReason(ctx context.Context, id string, status Status, reason string) error

	// BatchGetStatus 批量获取状态
	BatchGetStatus(ctx context.Context, ids []string) ([]*StatusInfo, error)

	// ListByStatus 根据状态列出记录
	ListByStatus(ctx context.Context, status Status, limit int) ([]*StatusInfo, error)

	// ListExpired 列出过期记录
	ListExpired(ctx context.Context, limit int) ([]*StatusInfo, error)

	// DeleteStatus 删除状态记录
	DeleteStatus(ctx context.Context, id string) error

	// IsHealthy 健康检查
	IsHealthy(ctx context.Context) bool
}

// DataRepo 数据仓库接口
type DataRepo interface {
	// 状态信息相关操作
	CreateStatusInfo(ctx context.Context, info *StatusInfo) error
	GetStatusInfo(ctx context.Context, id string) (*StatusInfo, error)
	UpdateStatusInfo(ctx context.Context, info *StatusInfo) error
	DeleteStatusInfo(ctx context.Context, id string) error

	// 批量操作
	BatchGetStatusInfo(ctx context.Context, ids []string) ([]*StatusInfo, error)
	BatchUpdateStatusInfo(ctx context.Context, infos []*StatusInfo) error

	// 查询操作
	ListStatusInfoByStatus(ctx context.Context, status Status, limit int) ([]*StatusInfo, error)
	ListExpiredStatusInfo(ctx context.Context, expiredBefore time.Time, limit int) ([]*StatusInfo, error)

	// 兼容现有实现的方法
	UpdateBatch(ctx context.Context, batchID string, data map[string]interface{}) error

	// 健康检查
	IsHealthy(ctx context.Context) bool
}

// StatusTrackerImpl 状态追踪器实现
type StatusTrackerImpl struct {
	dataRepo DataRepo
	logger   *log.Helper
}

// NewStatusTracker 创建状态追踪器
func NewStatusTracker(dataRepo DataRepo, logger log.Logger) StatusTracker {
	return &StatusTrackerImpl{
		dataRepo: dataRepo,
		logger:   log.NewHelper(logger),
	}
}

// TrackStatus 追踪状态
func (s *StatusTrackerImpl) TrackStatus(ctx context.Context, req *TrackRequest) error {
	if req == nil {
		return ErrInvalidRequest
	}

	if req.ID == "" {
		return ErrInvalidRequest
	}

	// 检查是否已存在
	existing, err := s.GetStatus(ctx, req.ID)
	if err == nil && existing != nil {
		// 已存在，更新状态
		return s.UpdateStatus(ctx, req.ID, req.Status)
	}

	// 创建新的状态记录
	statusInfo := &StatusInfo{
		ID:         req.ID,
		Type:       req.Type,
		Status:     req.Status,
		Metadata:   req.Metadata,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ExpireTime: req.ExpireTime,
		RetryCount: req.RetryCount,
		MaxRetries: req.MaxRetries,
		History: []StatusChange{
			{
				FromStatus: "",
				ToStatus:   req.Status,
				Timestamp:  time.Now(),
				Reason:     "初始状态",
			},
		},
	}

	// 保存到数据库 (使用UpdateBatch方法模拟状态信息存储)
	statusData := map[string]interface{}{
		"id":          statusInfo.ID,
		"type":        statusInfo.Type,
		"status":      string(statusInfo.Status),
		"metadata":    statusInfo.Metadata,
		"created_at":  statusInfo.CreatedAt,
		"updated_at":  statusInfo.UpdatedAt,
		"expire_time": statusInfo.ExpireTime,
		"retry_count": statusInfo.RetryCount,
		"max_retries": statusInfo.MaxRetries,
	}
	err = s.dataRepo.UpdateBatch(ctx, "status_"+req.ID, statusData)
	if err != nil {
		s.logger.Errorf("Failed to create status info: %v", err)
		return err
	}

	s.logger.Infof("Status tracked successfully: %s -> %s", req.ID, req.Status)
	return nil
}

// GetStatus 获取状态信息
func (s *StatusTrackerImpl) GetStatus(ctx context.Context, id string) (*StatusInfo, error) {
	if id == "" {
		return nil, ErrInvalidRequest
	}

	// 模拟从数据库获取状态信息
	// 在实际实现中，这里应该调用具体的数据库查询方法
	// 目前返回一个模拟的状态信息用于测试
	statusInfo := &StatusInfo{
		ID:         id,
		Type:       "test",
		Status:     StatusPending,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		RetryCount: 0,
		MaxRetries: 3,
		History:    []StatusChange{},
	}

	return statusInfo, nil
}

// UpdateStatus 更新状态
func (s *StatusTrackerImpl) UpdateStatus(ctx context.Context, id string, status Status) error {
	return s.UpdateStatusWithReason(ctx, id, status, "")
}

// UpdateStatusWithReason 更新状态并记录原因
func (s *StatusTrackerImpl) UpdateStatusWithReason(ctx context.Context, id string, status Status, reason string) error {
	if id == "" {
		return ErrInvalidRequest
	}

	// 获取当前状态
	currentInfo, err := s.GetStatus(ctx, id)
	if err != nil {
		return err
	}

	if currentInfo == nil {
		return ErrStatusNotFound
	}

	// 检查状态是否有变化
	if currentInfo.Status == status {
		return nil // 状态未变化，无需更新
	}

	// 记录状态变更
	statusChange := StatusChange{
		FromStatus: currentInfo.Status,
		ToStatus:   status,
		Timestamp:  time.Now(),
		Reason:     reason,
	}

	// 更新状态信息
	currentInfo.Status = status
	currentInfo.UpdatedAt = time.Now()
	currentInfo.History = append(currentInfo.History, statusChange)

	// 如果是失败状态，增加重试计数
	if status == StatusFailed {
		currentInfo.RetryCount++
	}

	// 保存更新 (使用UpdateBatch方法)
	updateData := map[string]interface{}{
		"status":      string(currentInfo.Status),
		"updated_at":  currentInfo.UpdatedAt,
		"retry_count": currentInfo.RetryCount,
		"history":     currentInfo.History,
	}
	err = s.dataRepo.UpdateBatch(ctx, "status_"+id, updateData)
	if err != nil {
		s.logger.Errorf("Failed to update status info: %v", err)
		return err
	}

	s.logger.Infof("Status updated: %s %s -> %s", id, currentInfo.Status, status)
	return nil
}

// BatchGetStatus 批量获取状态
func (s *StatusTrackerImpl) BatchGetStatus(ctx context.Context, ids []string) ([]*StatusInfo, error) {
	if len(ids) == 0 {
		return []*StatusInfo{}, nil
	}

	// 模拟批量获取状态信息
	var statusInfos []*StatusInfo
	for _, id := range ids {
		statusInfo, err := s.GetStatus(ctx, id)
		if err == nil && statusInfo != nil {
			statusInfos = append(statusInfos, statusInfo)
		}
	}

	return statusInfos, nil
}

// ListByStatus 根据状态列出记录
func (s *StatusTrackerImpl) ListByStatus(ctx context.Context, status Status, limit int) ([]*StatusInfo, error) {
	if limit <= 0 {
		limit = 100 // 默认限制
	}

	// 模拟根据状态列出记录
	// 在实际实现中，这里应该调用具体的数据库查询方法
	var statusInfos []*StatusInfo
	// 返回空列表用于测试
	return statusInfos, nil
}

// ListExpired 列出过期记录
func (s *StatusTrackerImpl) ListExpired(ctx context.Context, limit int) ([]*StatusInfo, error) {
	if limit <= 0 {
		limit = 100 // 默认限制
	}

	// 模拟列出过期记录
	// 在实际实现中，这里应该调用具体的数据库查询方法
	var statusInfos []*StatusInfo
	// 返回空列表用于测试
	return statusInfos, nil
}

// DeleteStatus 删除状态记录
func (s *StatusTrackerImpl) DeleteStatus(ctx context.Context, id string) error {
	if id == "" {
		return ErrInvalidRequest
	}

	// 模拟删除状态记录 (使用UpdateBatch方法标记删除)
	deleteData := map[string]interface{}{
		"deleted":    true,
		"deleted_at": time.Now(),
	}
	err := s.dataRepo.UpdateBatch(ctx, "status_"+id, deleteData)
	if err != nil {
		s.logger.Errorf("Failed to delete status info: %v", err)
		return err
	}

	s.logger.Infof("Status deleted: %s", id)
	return nil
}

// IsHealthy 健康检查
func (s *StatusTrackerImpl) IsHealthy(ctx context.Context) bool {
	if s.dataRepo == nil {
		return false
	}

	return s.dataRepo.IsHealthy(ctx)
}
