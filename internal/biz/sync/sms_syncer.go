package sync

import (
	"context"
	"time"
)

// StepStatistics 步骤统计信息
type StepStatistics struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failure int64 `json:"failure"`
}

// IncreaseTotal 增加总数
func (s *StepStatistics) IncreaseTotal() {
	s.Total++
}

// IncreaseSuccess 增加成功数
func (s *StepStatistics) IncreaseSuccess() {
	s.Success++
}

// IncreaseFailure 增加失败数
func (s *StepStatistics) IncreaseFailure() {
	s.Failure++
}

// DeliverySyncer 投递同步器接口
type DeliverySyncer interface {
	// InitSyncer 初始化同步器
	InitSyncer(ctx context.Context, batchId string) error

	// Sync2Statistics 同步计数器到本地统计
	Sync2Statistics(ctx context.Context, batchId string) (*StepStatistics, error)

	// SyncStatistics2Counter 同步本地统计到计数器
	SyncStatistics2Counter(ctx context.Context, batchId string, stats *StepStatistics) error

	// GetStatistics 获取统计信息
	GetStatistics(ctx context.Context, batchId string) (*StepStatistics, error)

	// SetHeartbeatTime 设置心跳时间
	SetHeartbeatTime(ctx context.Context, batchId string, heartbeatTime time.Time) error

	// GetHeartbeatTime 获取心跳时间
	GetHeartbeatTime(ctx context.Context, batchId string) (time.Time, error)

	// MarkPKTaskComplete 标记PK任务完成
	MarkPKTaskComplete(ctx context.Context, batchId, partitionKey string) error
}

// PrepareSyncer 准备同步器接口
type PrepareSyncer interface {
	// InitSyncer 初始化同步器
	InitSyncer(ctx context.Context, batchId string) error

	// Sync2Statistics 同步计数器到本地统计
	Sync2Statistics(ctx context.Context, batchId string) (*StepStatistics, error)

	// SyncStatistics2Counter 同步本地统计到计数器
	SyncStatistics2Counter(ctx context.Context, batchId string, stats *StepStatistics) error

	// GetStatistics 获取统计信息
	GetStatistics(ctx context.Context, batchId string) (*StepStatistics, error)

	// SetHeartbeatTime 设置心跳时间
	SetHeartbeatTime(ctx context.Context, batchId string, heartbeatTime time.Time) error

	// GetHeartbeatTime 获取心跳时间
	GetHeartbeatTime(ctx context.Context, batchId string) (time.Time, error)

	// MarkPKTaskComplete 标记PK任务完成
	MarkPKTaskComplete(ctx context.Context, batchId, partitionKey string) error
}
