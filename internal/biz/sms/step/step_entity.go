package step

import (
	"context"
	"time"
)

// SmsStep SMS步骤实体
type SmsStep struct {
	ID         string                 `json:"id" bson:"_id"`
	BatchId    string                 `json:"batch_id" bson:"batch_id"`
	Type       StepType               `json:"type" bson:"type"`
	Status     StepStatus             `json:"status" bson:"status"`
	CreateTime time.Time              `json:"create_time" bson:"create_time"`
	UpdateTime time.Time              `json:"update_time" bson:"update_time"`
	Statistics *StepStatistics        `json:"statistics" bson:"statistics"`
	ErrorMsg   string                 `json:"error_msg,omitempty" bson:"error_msg,omitempty"`
	RetryCount int                    `json:"retry_count" bson:"retry_count"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

// SmsStepRepo SMS步骤仓储接口
type SmsStepRepo interface {
	// Save 保存步骤
	Save(ctx context.Context, step *SmsStep) error

	// GetByID 根据ID获取步骤
	GetByID(ctx context.Context, id string) (*SmsStep, error)

	// GetByBatchID 根据批次ID获取步骤列表
	GetByBatchID(ctx context.Context, batchID string) ([]*SmsStep, error)

	// GetByBatchIDAndType 根据批次ID和类型获取步骤
	GetByBatchIDAndType(ctx context.Context, batchID string, stepType StepType) (*SmsStep, error)

	// GetByBatchIDAndStatus 根据批次ID和状态获取步骤列表
	GetByBatchIDAndStatus(ctx context.Context, batchID string, status StepStatus) ([]*SmsStep, error)

	// Update 更新步骤
	Update(ctx context.Context, step *SmsStep) error

	// Delete 删除步骤
	Delete(ctx context.Context, id string) error

	// ListByStatus 根据状态列出步骤
	ListByStatus(ctx context.Context, status StepStatus, limit int) ([]*SmsStep, error)

	// Count 统计步骤数量
	Count(ctx context.Context, batchID string) (int64, error)

	// CountByStatus 根据状态统计步骤数量
	CountByStatus(ctx context.Context, batchID string, status StepStatus) (int64, error)
}
