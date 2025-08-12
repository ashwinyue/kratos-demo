package types

import (
	"context"
	"time"
)

// BatchStatus 批次状态
type BatchStatus string

const (
	BatchStatusPending   BatchStatus = "PENDING"
	BatchStatusPreparing BatchStatus = "PREPARING"
	BatchStatusRunning   BatchStatus = "RUNNING"
	BatchStatusCompleted BatchStatus = "COMPLETED"
	BatchStatusFailed    BatchStatus = "FAILED"
	BatchStatusCancelled BatchStatus = "CANCELLED"
)

// StepType 步骤类型
type StepType string

const (
	StepTypePreparation StepType = "PREPARATION"
	StepTypeDelivery    StepType = "DELIVERY"
)

// SmsBatchInterface SMS批次接口
type SmsBatchInterface interface {
	GetID() int64
	GetBatchID() string
	GetStatus() BatchStatus
	GetCreatedAt() time.Time
	GetUpdatedAt() time.Time
}

// SmsBatchRepo SMS批次仓储接口
type SmsBatchRepo interface {
	Save(ctx context.Context, batch SmsBatchInterface) error
	Get(ctx context.Context, id int64) (SmsBatchInterface, error)
	GetByBatchID(ctx context.Context, batchID string) (SmsBatchInterface, error)
	Update(ctx context.Context, batch SmsBatchInterface) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, offset, limit int) ([]SmsBatchInterface, error)
	GetRunningBatches(ctx context.Context) ([]SmsBatchInterface, error)
	GetByStatus(ctx context.Context, status BatchStatus) ([]SmsBatchInterface, error)
}

// StepProcessor 步骤处理器接口 - 这里只是为了向后兼容，实际使用step包中的接口
type StepProcessor interface {
	Execute(ctx context.Context, batch SmsBatchInterface) error
	GetStatus() string
}

// StepProcessorFactory 步骤处理器工厂接口
type StepProcessorFactory interface {
	CreatePreparationProcessor() StepProcessor
	CreateDeliveryProcessor() StepProcessor
}