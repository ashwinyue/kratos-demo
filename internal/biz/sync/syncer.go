package sync

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// 同步器相关错误定义
var (
	ErrInvalidSyncRequest   = errors.New("invalid sync request")
	ErrMissingSyncRequestID = errors.New("missing sync request ID")
	ErrInvalidSyncType      = errors.New("invalid sync type")
	ErrMissingSyncData      = errors.New("missing sync data")
	ErrSyncerNotFound       = errors.New("syncer not found")
	ErrSyncTimeout          = errors.New("sync operation timeout")
	ErrSyncFailed           = errors.New("sync operation failed")
)

// SyncType 同步类型枚举
type SyncType string

const (
	SyncTypePrepare  SyncType = "prepare"  // 准备同步
	SyncTypeDelivery SyncType = "delivery" // 投递同步
	SyncTypeSave     SyncType = "save"     // 保存同步
)

// SyncStatus 同步状态
type SyncStatus string

const (
	SyncStatusPending   SyncStatus = "pending"   // 待同步
	SyncStatusRunning   SyncStatus = "running"   // 同步中
	SyncStatusCompleted SyncStatus = "completed" // 已完成
	SyncStatusFailed    SyncStatus = "failed"    // 同步失败
)

// SyncRequest 同步请求
type SyncRequest struct {
	ID         string                 `json:"id"`          // 同步请求ID
	Type       SyncType               `json:"type"`        // 同步类型
	Data       map[string]interface{} `json:"data"`        // 同步数据
	Timestamp  time.Time              `json:"timestamp"`   // 请求时间
	RetryCount int                    `json:"retry_count"` // 重试次数
}

// SyncResponse 同步响应
type SyncResponse struct {
	ID        string                 `json:"id"`        // 同步请求ID
	Status    SyncStatus             `json:"status"`    // 同步状态
	Result    map[string]interface{} `json:"result"`    // 同步结果
	Message   string                 `json:"message"`   // 响应消息
	Timestamp time.Time              `json:"timestamp"` // 响应时间
	Duration  time.Duration          `json:"duration"`  // 处理耗时
}

// SyncConfig 同步器配置
type SyncConfig struct {
	MaxRetries    int           `yaml:"max_retries"`    // 最大重试次数
	RetryInterval time.Duration `yaml:"retry_interval"` // 重试间隔
	Timeout       time.Duration `yaml:"timeout"`        // 超时时间
	BatchSize     int           `yaml:"batch_size"`     // 批处理大小
	Async         bool          `yaml:"async"`          // 是否异步处理
}

// DataRepo 数据仓库接口
type DataRepo interface {
	// UpdateBatch 批量更新数据
	UpdateBatch(ctx context.Context, id string, data map[string]interface{}) error
	// GetBatch 获取批次数据
	GetBatch(ctx context.Context, id string) (map[string]interface{}, error)
	// SaveBatch 保存批次数据
	SaveBatch(ctx context.Context, id string, data map[string]interface{}) error
	// DeleteBatch 删除批次数据
	DeleteBatch(ctx context.Context, id string) error
	// IsHealthy 健康检查
	IsHealthy(ctx context.Context) bool
}

// MessageQueueClient 消息队列客户端接口
type MessageQueueClient interface {
	// SendMessage 发送消息
	SendMessage(ctx context.Context, topic string, data interface{}) error
	// IsHealthy 健康检查
	IsHealthy(ctx context.Context) bool
}

// Syncer 同步器接口
type Syncer interface {
	// Sync 执行同步操作
	Sync(ctx context.Context, req *SyncRequest) (*SyncResponse, error)

	// GetSyncType 获取同步类型
	GetSyncType() SyncType

	// IsAsync 是否异步处理
	IsAsync() bool

	// ValidateRequest 验证同步请求
	ValidateRequest(req *SyncRequest) error

	// GetConfig 获取同步器配置
	GetConfig() *SyncConfig

	// IsHealthy 健康检查
	IsHealthy(ctx context.Context) bool
}

// SyncerFactory 同步器工厂接口
type SyncerFactory interface {
	// CreateSyncer 创建同步器
	CreateSyncer(syncType SyncType) (Syncer, error)

	// GetSyncer 获取同步器
	GetSyncer(syncType SyncType) (Syncer, error)

	// ListAvailableSyncers 列出可用的同步器类型
	ListAvailableSyncers() []SyncType
}

// BaseSyncer 基础同步器实现
type BaseSyncer struct {
	syncType SyncType
	config   *SyncConfig
	logger   *log.Helper
}

// NewBaseSyncer 创建基础同步器
func NewBaseSyncer(syncType SyncType, config *SyncConfig, logger log.Logger) *BaseSyncer {
	return &BaseSyncer{
		syncType: syncType,
		config:   config,
		logger:   log.NewHelper(logger),
	}
}

// GetSyncType 获取同步类型
func (s *BaseSyncer) GetSyncType() SyncType {
	return s.syncType
}

// IsAsync 是否异步处理
func (s *BaseSyncer) IsAsync() bool {
	return s.config.Async
}

// GetConfig 获取同步器配置
func (s *BaseSyncer) GetConfig() *SyncConfig {
	return s.config
}

// ValidateRequest 验证同步请求
func (s *BaseSyncer) ValidateRequest(req *SyncRequest) error {
	if req == nil {
		return ErrInvalidSyncRequest
	}

	if req.ID == "" {
		return ErrMissingSyncRequestID
	}

	if req.Type != s.syncType {
		return ErrInvalidSyncType
	}

	if req.Data == nil {
		return ErrMissingSyncData
	}

	return nil
}

// IsHealthy 健康检查
func (s *BaseSyncer) IsHealthy(ctx context.Context) bool {
	// 基础健康检查，子类可以重写
	return true
}

// createSyncResponse 创建同步响应
func (s *BaseSyncer) createSyncResponse(req *SyncRequest, status SyncStatus, result map[string]interface{}, message string, startTime time.Time) *SyncResponse {
	return &SyncResponse{
		ID:        req.ID,
		Status:    status,
		Result:    result,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  time.Since(startTime),
	}
}
