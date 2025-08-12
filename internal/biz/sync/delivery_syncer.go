package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// DeliverySyncData 投递同步数据结构
type DeliverySyncData struct {
	BatchID        string            `json:"batch_id"`        // 批次ID
	PackID         string            `json:"pack_id"`         // 投递包ID
	Status         string            `json:"status"`          // 投递状态
	DeliveredCount int               `json:"delivered_count"` // 已投递数量
	FailedCount    int               `json:"failed_count"`    // 失败数量
	Metadata       map[string]string `json:"metadata"`        // 元数据
	DeliveredAt    time.Time         `json:"delivered_at"`    // 投递时间
}

// DeliverySyncerImpl 投递同步器实现
type DeliverySyncerImpl struct {
	*BaseSyncer
	dataRepo DataRepo           // 数据仓库接口
	mqClient MessageQueueClient // 消息队列客户端
}

// NewDeliverySyncerImpl 创建投递同步器实现
func NewDeliverySyncerImpl(config *SyncConfig, logger log.Logger, dataRepo DataRepo, mqClient MessageQueueClient) *DeliverySyncerImpl {
	return &DeliverySyncerImpl{
		BaseSyncer: NewBaseSyncer(SyncTypeDelivery, config, logger),
		dataRepo:   dataRepo,
		mqClient:   mqClient,
	}
}

// Sync 执行投递同步操作
func (s *DeliverySyncerImpl) Sync(ctx context.Context, req *SyncRequest) (*SyncResponse, error) {
	startTime := time.Now()
	s.logger.WithContext(ctx).Infof("Starting delivery sync for request: %s", req.ID)

	// 验证请求
	if err := s.ValidateRequest(req); err != nil {
		s.logger.WithContext(ctx).Errorf("Invalid delivery sync request: %v", err)
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
		s.logger.WithContext(ctx).Errorf("Delivery sync failed: %v", err)
		return s.createSyncResponse(req, SyncStatusFailed, nil, err.Error(), startTime), err
	}

	s.logger.WithContext(ctx).Infof("Delivery sync completed successfully for request: %s", req.ID)
	return s.createSyncResponse(req, SyncStatusCompleted, result, "Delivery sync completed", startTime), nil
}

// parseSyncData 解析同步数据
func (s *DeliverySyncerImpl) parseSyncData(data map[string]interface{}) (*DeliverySyncData, error) {
	syncData := &DeliverySyncData{}

	// 解析批次ID
	if batchID, ok := data["batch_id"].(string); ok {
		syncData.BatchID = batchID
	} else {
		return nil, fmt.Errorf("missing or invalid batch_id")
	}

	// 解析投递包ID
	if packID, ok := data["pack_id"].(string); ok {
		syncData.PackID = packID
	} else {
		return nil, fmt.Errorf("missing or invalid pack_id")
	}

	// 解析状态
	if status, ok := data["status"].(string); ok {
		syncData.Status = status
	} else {
		return nil, fmt.Errorf("missing or invalid status")
	}

	// 解析已投递数量
	if count, ok := data["delivered_count"].(float64); ok {
		syncData.DeliveredCount = int(count)
	} else if count, ok := data["delivered_count"].(int); ok {
		syncData.DeliveredCount = count
	} else {
		syncData.DeliveredCount = 0
	}

	// 解析失败数量
	if count, ok := data["failed_count"].(float64); ok {
		syncData.FailedCount = int(count)
	} else if count, ok := data["failed_count"].(int); ok {
		syncData.FailedCount = count
	} else {
		syncData.FailedCount = 0
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

	// 设置投递时间
	if deliveredAt, ok := data["delivered_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, deliveredAt); err == nil {
			syncData.DeliveredAt = t
		} else {
			syncData.DeliveredAt = time.Now()
		}
	} else {
		syncData.DeliveredAt = time.Now()
	}

	return syncData, nil
}

// performSync 执行同步操作
func (s *DeliverySyncerImpl) performSync(ctx context.Context, data *DeliverySyncData) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 1. 更新投递包状态
	if err := s.updateDeliveryPackStatus(ctx, data); err != nil {
		return nil, fmt.Errorf("failed to update delivery pack status: %w", err)
	}

	// 2. 更新批次投递统计
	if err := s.updateBatchDeliveryStats(ctx, data); err != nil {
		return nil, fmt.Errorf("failed to update batch delivery stats: %w", err)
	}

	// 3. 发送投递完成消息到消息队列
	if err := s.sendDeliveryCompleteMessage(ctx, data); err != nil {
		s.logger.WithContext(ctx).Warnf("Failed to send delivery complete message: %v", err)
		// 消息发送失败不影响同步结果，只记录警告
	}

	// 4. 记录同步结果
	result["batch_id"] = data.BatchID
	result["pack_id"] = data.PackID
	result["status"] = data.Status
	result["delivered_count"] = data.DeliveredCount
	result["failed_count"] = data.FailedCount
	result["synced_at"] = time.Now().Format(time.RFC3339)

	return result, nil
}

// updateDeliveryPackStatus 更新投递包状态
func (s *DeliverySyncerImpl) updateDeliveryPackStatus(ctx context.Context, data *DeliverySyncData) error {
	// 构建更新数据
	updateData := map[string]interface{}{
		"status":          data.Status,
		"delivered_count": data.DeliveredCount,
		"failed_count":    data.FailedCount,
		"delivered_at":    data.DeliveredAt,
		"updated_at":      time.Now(),
	}

	// 添加元数据
	if data.Metadata != nil {
		updateData["metadata"] = data.Metadata
	}

	// 执行数据库更新
	if err := s.dataRepo.UpdateBatch(ctx, data.PackID, updateData); err != nil {
		return fmt.Errorf("failed to update delivery pack %s: %w", data.PackID, err)
	}

	s.logger.WithContext(ctx).Infof("Updated delivery pack status: %s -> %s", data.PackID, data.Status)
	return nil
}

// updateBatchDeliveryStats 更新批次投递统计
func (s *DeliverySyncerImpl) updateBatchDeliveryStats(ctx context.Context, data *DeliverySyncData) error {
	// 构建统计更新数据
	statsUpdate := map[string]interface{}{
		"delivered_count":  data.DeliveredCount,
		"failed_count":     data.FailedCount,
		"last_delivery_at": data.DeliveredAt,
		"updated_at":       time.Now(),
	}

	// 执行批次统计更新
	if err := s.dataRepo.UpdateBatch(ctx, data.BatchID, statsUpdate); err != nil {
		return fmt.Errorf("failed to update batch delivery stats %s: %w", data.BatchID, err)
	}

	s.logger.WithContext(ctx).Infof("Updated batch delivery stats: %s (delivered: %d, failed: %d)",
		data.BatchID, data.DeliveredCount, data.FailedCount)
	return nil
}

// sendDeliveryCompleteMessage 发送投递完成消息
func (s *DeliverySyncerImpl) sendDeliveryCompleteMessage(ctx context.Context, data *DeliverySyncData) error {
	message := map[string]interface{}{
		"event_type":      "delivery_completed",
		"batch_id":        data.BatchID,
		"pack_id":         data.PackID,
		"status":          data.Status,
		"delivered_count": data.DeliveredCount,
		"failed_count":    data.FailedCount,
		"delivered_at":    data.DeliveredAt.Format(time.RFC3339),
		"timestamp":       time.Now().Format(time.RFC3339),
	}

	if err := s.mqClient.SendMessage(ctx, "delivery_events", message); err != nil {
		return fmt.Errorf("failed to send delivery complete message: %w", err)
	}

	s.logger.WithContext(ctx).Infof("Sent delivery complete message for pack: %s", data.PackID)
	return nil
}

// IsHealthy 健康检查
func (s *DeliverySyncerImpl) IsHealthy(ctx context.Context) bool {
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

// ValidateRequest 验证投递同步请求
func (s *DeliverySyncerImpl) ValidateRequest(req *SyncRequest) error {
	// 调用基础验证
	if err := s.BaseSyncer.ValidateRequest(req); err != nil {
		return err
	}

	// 验证批次ID
	if _, ok := req.Data["batch_id"]; !ok {
		return fmt.Errorf("missing batch_id in sync data")
	}

	// 验证投递包ID
	if _, ok := req.Data["pack_id"]; !ok {
		return fmt.Errorf("missing pack_id in sync data")
	}

	// 验证状态
	if _, ok := req.Data["status"]; !ok {
		return fmt.Errorf("missing status in sync data")
	}

	return nil
}
