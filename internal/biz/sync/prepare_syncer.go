package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// DataRepo 接口定义在 syncer.go 中
// MessageQueueClient 接口定义在 syncer.go 中

// PrepareSyncData 准备同步数据结构
type PrepareSyncData struct {
	BatchID      string            `json:"batch_id"`      // 批次ID
	Status       string            `json:"status"`        // 批次状态
	MessageCount int               `json:"message_count"` // 消息数量
	Metadata     map[string]string `json:"metadata"`      // 元数据
	PreparedAt   time.Time         `json:"prepared_at"`   // 准备时间
}

// PrepareSyncerImpl 准备同步器实现
type PrepareSyncerImpl struct {
	*BaseSyncer
	dataRepo DataRepo           // 数据仓库接口
	mqClient MessageQueueClient // 消息队列客户端
}

// NewPrepareSyncerImpl 创建准备同步器实现
func NewPrepareSyncerImpl(config *SyncConfig, logger log.Logger, dataRepo DataRepo, mqClient MessageQueueClient) *PrepareSyncerImpl {
	return &PrepareSyncerImpl{
		BaseSyncer: NewBaseSyncer(SyncTypePrepare, config, logger),
		dataRepo:   dataRepo,
		mqClient:   mqClient,
	}
}

// Sync 执行准备同步操作
func (s *PrepareSyncerImpl) Sync(ctx context.Context, req *SyncRequest) (*SyncResponse, error) {
	startTime := time.Now()
	s.logger.WithContext(ctx).Infof("Starting prepare sync for request: %s", req.ID)

	// 验证请求
	if err := s.ValidateRequest(req); err != nil {
		s.logger.WithContext(ctx).Errorf("Invalid prepare sync request: %v", err)
		return s.createSyncResponse(req, SyncStatusFailed, nil, err.Error(), startTime), err
	}

	// 解析同步数据
	syncData, err := s.parseSyncData(req.Data)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("Failed to parse sync data: %v", err)
		return s.createSyncResponse(req, SyncStatusFailed, nil, err.Error(), startTime), err
	}

	// 执行同步操作
	result, err := s.performSync(ctx, syncData)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("Prepare sync failed: %v", err)
		return s.createSyncResponse(req, SyncStatusFailed, nil, err.Error(), startTime), err
	}

	s.logger.WithContext(ctx).Infof("Prepare sync completed successfully for request: %s", req.ID)
	return s.createSyncResponse(req, SyncStatusCompleted, result, "Prepare sync completed", startTime), nil
}

// parseSyncData 解析同步数据
func (s *PrepareSyncerImpl) parseSyncData(data map[string]interface{}) (*PrepareSyncData, error) {
	syncData := &PrepareSyncData{}

	// 解析批次ID
	if batchID, ok := data["batch_id"].(string); ok {
		syncData.BatchID = batchID
	} else {
		return nil, fmt.Errorf("missing or invalid batch_id")
	}

	// 解析状态
	if status, ok := data["status"].(string); ok {
		syncData.Status = status
	} else {
		return nil, fmt.Errorf("missing or invalid status")
	}

	// 解析消息数量
	if count, ok := data["message_count"].(float64); ok {
		syncData.MessageCount = int(count)
	} else if count, ok := data["message_count"].(int); ok {
		syncData.MessageCount = count
	} else {
		syncData.MessageCount = 0
	}

	// 解析元数据
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		syncData.Metadata = make(map[string]string)
		for k, v := range metadata {
			if str, ok := v.(string); ok {
				syncData.Metadata[k] = str
			}
		}
	}

	// 设置准备时间
	if preparedAt, ok := data["prepared_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, preparedAt); err == nil {
			syncData.PreparedAt = t
		} else {
			syncData.PreparedAt = time.Now()
		}
	} else {
		syncData.PreparedAt = time.Now()
	}

	return syncData, nil
}

// performSync 执行同步操作
func (s *PrepareSyncerImpl) performSync(ctx context.Context, data *PrepareSyncData) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 1. 更新数据库中的批次状态
	if err := s.updateBatchStatus(ctx, data); err != nil {
		return nil, fmt.Errorf("failed to update batch status: %w", err)
	}

	// 2. 发送准备完成消息到消息队列
	if err := s.sendPrepareCompleteMessage(ctx, data); err != nil {
		s.logger.WithContext(ctx).Warnf("Failed to send prepare complete message: %v", err)
		// 消息发送失败不影响同步结果，只记录警告
	}

	// 3. 记录同步结果
	result["batch_id"] = data.BatchID
	result["status"] = data.Status
	result["message_count"] = data.MessageCount
	result["synced_at"] = time.Now().Format(time.RFC3339)

	return result, nil
}

// updateBatchStatus 更新批次状态
func (s *PrepareSyncerImpl) updateBatchStatus(ctx context.Context, data *PrepareSyncData) error {
	// 构建更新数据
	updateData := map[string]interface{}{
		"status":        data.Status,
		"message_count": data.MessageCount,
		"prepared_at":   data.PreparedAt,
		"updated_at":    time.Now(),
	}

	// 添加元数据
	if data.Metadata != nil {
		updateData["metadata"] = data.Metadata
	}

	// 执行数据库更新
	if err := s.dataRepo.UpdateBatch(ctx, data.BatchID, updateData); err != nil {
		return fmt.Errorf("failed to update batch %s: %w", data.BatchID, err)
	}

	s.logger.WithContext(ctx).Infof("Updated batch status: %s -> %s", data.BatchID, data.Status)
	return nil
}

// sendPrepareCompleteMessage 发送准备完成消息
func (s *PrepareSyncerImpl) sendPrepareCompleteMessage(ctx context.Context, data *PrepareSyncData) error {
	message := map[string]interface{}{
		"event_type":    "batch_prepare_completed",
		"batch_id":      data.BatchID,
		"status":        data.Status,
		"message_count": data.MessageCount,
		"prepared_at":   data.PreparedAt.Format(time.RFC3339),
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	if err := s.mqClient.SendMessage(ctx, "batch_events", message); err != nil {
		return fmt.Errorf("failed to send prepare complete message: %w", err)
	}

	s.logger.WithContext(ctx).Infof("Sent prepare complete message for batch: %s", data.BatchID)
	return nil
}

// IsHealthy 健康检查
func (s *PrepareSyncerImpl) IsHealthy(ctx context.Context) bool {
	// 检查数据仓库连接
	if !s.dataRepo.IsHealthy(ctx) {
		s.logger.WithContext(ctx).Warn("Data repository is not healthy")
		return false
	}

	// 检查消息队列连接
	if !s.mqClient.IsHealthy(ctx) {
		s.logger.WithContext(ctx).Warn("Message queue client is not healthy")
		return false
	}

	return true
}

// ValidateRequest 验证准备同步请求
func (s *PrepareSyncerImpl) ValidateRequest(req *SyncRequest) error {
	// 调用基础验证
	if err := s.BaseSyncer.ValidateRequest(req); err != nil {
		return err
	}

	// 验证批次ID
	if _, ok := req.Data["batch_id"]; !ok {
		return fmt.Errorf("missing batch_id in sync data")
	}

	// 验证状态
	if _, ok := req.Data["status"]; !ok {
		return fmt.Errorf("missing status in sync data")
	}

	return nil
}
