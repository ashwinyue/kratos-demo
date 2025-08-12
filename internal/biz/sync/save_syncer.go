package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// SaveSyncData 保存同步数据结构
type SaveSyncData struct {
	BatchID       string            `json:"batch_id"`       // 批次ID
	PackID        string            `json:"pack_id"`        // 投递包ID
	FinalStatus   string            `json:"final_status"`   // 最终状态
	TotalCount    int               `json:"total_count"`    // 总数量
	SuccessCount  int               `json:"success_count"`  // 成功数量
	FailedCount   int               `json:"failed_count"`   // 失败数量
	ProcessResult map[string]string `json:"process_result"` // 处理结果
	Metadata      map[string]string `json:"metadata"`       // 元数据
	CompletedAt   time.Time         `json:"completed_at"`   // 完成时间
}

// SaveSyncerImpl 保存同步器实现
type SaveSyncerImpl struct {
	*BaseSyncer
	dataRepo DataRepo           // 数据仓库接口
	mqClient MessageQueueClient // 消息队列客户端
}

// NewSaveSyncerImpl 创建保存同步器实现
func NewSaveSyncerImpl(config *SyncConfig, logger log.Logger, dataRepo DataRepo, mqClient MessageQueueClient) *SaveSyncerImpl {
	return &SaveSyncerImpl{
		BaseSyncer: NewBaseSyncer(SyncTypeSave, config, logger),
		dataRepo:   dataRepo,
		mqClient:   mqClient,
	}
}

// Sync 执行保存同步操作
func (s *SaveSyncerImpl) Sync(ctx context.Context, req *SyncRequest) (*SyncResponse, error) {
	startTime := time.Now()
	s.logger.WithContext(ctx).Infof("Starting save sync for request: %s", req.ID)

	// 验证请求
	if err := s.ValidateRequest(req); err != nil {
		s.logger.WithContext(ctx).Errorf("Invalid save sync request: %v", err)
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
		s.logger.WithContext(ctx).Errorf("Save sync failed: %v", err)
		return s.createSyncResponse(req, SyncStatusFailed, nil, err.Error(), startTime), err
	}

	s.logger.WithContext(ctx).Infof("Save sync completed successfully for request: %s", req.ID)
	return s.createSyncResponse(req, SyncStatusCompleted, result, "Save sync completed", startTime), nil
}

// parseSyncData 解析同步数据
func (s *SaveSyncerImpl) parseSyncData(data map[string]interface{}) (*SaveSyncData, error) {
	syncData := &SaveSyncData{}

	// 解析批次ID
	if batchID, ok := data["batch_id"].(string); ok {
		syncData.BatchID = batchID
	} else {
		return nil, fmt.Errorf("missing or invalid batch_id")
	}

	// 解析投递包ID（可选）
	if packID, ok := data["pack_id"].(string); ok {
		syncData.PackID = packID
	}

	// 解析最终状态
	if status, ok := data["final_status"].(string); ok {
		syncData.FinalStatus = status
	} else {
		return nil, fmt.Errorf("missing or invalid final_status")
	}

	// 解析总数量
	if count, ok := data["total_count"].(float64); ok {
		syncData.TotalCount = int(count)
	} else if count, ok := data["total_count"].(int); ok {
		syncData.TotalCount = count
	} else {
		syncData.TotalCount = 0
	}

	// 解析成功数量
	if count, ok := data["success_count"].(float64); ok {
		syncData.SuccessCount = int(count)
	} else if count, ok := data["success_count"].(int); ok {
		syncData.SuccessCount = count
	} else {
		syncData.SuccessCount = 0
	}

	// 解析失败数量
	if count, ok := data["failed_count"].(float64); ok {
		syncData.FailedCount = int(count)
	} else if count, ok := data["failed_count"].(int); ok {
		syncData.FailedCount = count
	} else {
		syncData.FailedCount = 0
	}

	// 解析处理结果
	if result, ok := data["process_result"].(map[string]interface{}); ok {
		syncData.ProcessResult = make(map[string]string)
		for k, v := range result {
			if str, ok := v.(string); ok {
				syncData.ProcessResult[k] = str
			}
		}
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

	// 设置完成时间
	if completedAt, ok := data["completed_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, completedAt); err == nil {
			syncData.CompletedAt = t
		} else {
			syncData.CompletedAt = time.Now()
		}
	} else {
		syncData.CompletedAt = time.Now()
	}

	return syncData, nil
}

// performSync 执行同步操作
func (s *SaveSyncerImpl) performSync(ctx context.Context, data *SaveSyncData) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// 1. 保存最终处理结果
	if err := s.saveFinalResult(ctx, data); err != nil {
		return nil, fmt.Errorf("failed to save final result: %w", err)
	}

	// 2. 更新最终状态
	if err := s.updateFinalStatus(ctx, data); err != nil {
		return nil, fmt.Errorf("failed to update final status: %w", err)
	}

	// 3. 清理临时数据
	if err := s.cleanupTempData(ctx, data); err != nil {
		s.logger.WithContext(ctx).Warnf("Failed to cleanup temp data: %v", err)
		// 清理失败不影响同步结果，只记录警告
	}

	// 4. 发送完成通知消息
	if err := s.sendCompletionNotification(ctx, data); err != nil {
		s.logger.WithContext(ctx).Warnf("Failed to send completion notification: %v", err)
		// 通知发送失败不影响同步结果，只记录警告
	}

	// 5. 记录同步结果
	result["batch_id"] = data.BatchID
	result["pack_id"] = data.PackID
	result["final_status"] = data.FinalStatus
	result["total_count"] = data.TotalCount
	result["success_count"] = data.SuccessCount
	result["failed_count"] = data.FailedCount
	result["synced_at"] = time.Now().Format(time.RFC3339)

	return result, nil
}

// saveFinalResult 保存最终处理结果
func (s *SaveSyncerImpl) saveFinalResult(ctx context.Context, data *SaveSyncData) error {
	// 构建结果数据
	resultData := map[string]interface{}{
		"batch_id":       data.BatchID,
		"pack_id":        data.PackID,
		"final_status":   data.FinalStatus,
		"total_count":    data.TotalCount,
		"success_count":  data.SuccessCount,
		"failed_count":   data.FailedCount,
		"process_result": data.ProcessResult,
		"completed_at":   data.CompletedAt,
		"created_at":     time.Now(),
	}

	// 添加元数据
	if data.Metadata != nil {
		resultData["metadata"] = data.Metadata
	}

	// 保存到数据库（这里假设有专门的结果表）
	if err := s.dataRepo.UpdateBatch(ctx, fmt.Sprintf("result_%s", data.BatchID), resultData); err != nil {
		return fmt.Errorf("failed to save final result for batch %s: %w", data.BatchID, err)
	}

	s.logger.WithContext(ctx).Infof("Saved final result for batch: %s (success: %d, failed: %d)",
		data.BatchID, data.SuccessCount, data.FailedCount)
	return nil
}

// updateFinalStatus 更新最终状态
func (s *SaveSyncerImpl) updateFinalStatus(ctx context.Context, data *SaveSyncData) error {
	// 构建状态更新数据
	statusUpdate := map[string]interface{}{
		"final_status":  data.FinalStatus,
		"total_count":   data.TotalCount,
		"success_count": data.SuccessCount,
		"failed_count":  data.FailedCount,
		"completed_at":  data.CompletedAt,
		"updated_at":    time.Now(),
	}

	// 更新批次最终状态
	if err := s.dataRepo.UpdateBatch(ctx, data.BatchID, statusUpdate); err != nil {
		return fmt.Errorf("failed to update final status for batch %s: %w", data.BatchID, err)
	}

	// 如果有投递包ID，也更新投递包状态
	if data.PackID != "" {
		packUpdate := map[string]interface{}{
			"final_status": data.FinalStatus,
			"completed_at": data.CompletedAt,
			"updated_at":   time.Now(),
		}
		if err := s.dataRepo.UpdateBatch(ctx, data.PackID, packUpdate); err != nil {
			s.logger.WithContext(ctx).Warnf("Failed to update pack status: %v", err)
		}
	}

	s.logger.WithContext(ctx).Infof("Updated final status: %s -> %s", data.BatchID, data.FinalStatus)
	return nil
}

// cleanupTempData 清理临时数据
func (s *SaveSyncerImpl) cleanupTempData(ctx context.Context, data *SaveSyncData) error {
	// 清理临时处理数据
	tempKeys := []string{
		fmt.Sprintf("temp_batch_%s", data.BatchID),
		fmt.Sprintf("temp_pack_%s", data.PackID),
		fmt.Sprintf("processing_%s", data.BatchID),
	}

	for _, key := range tempKeys {
		if key != "temp_pack_" { // 避免空的pack_id
			if err := s.dataRepo.UpdateBatch(ctx, key, map[string]interface{}{"deleted": true}); err != nil {
				s.logger.WithContext(ctx).Warnf("Failed to cleanup temp data %s: %v", key, err)
			}
		}
	}

	s.logger.WithContext(ctx).Infof("Cleaned up temp data for batch: %s", data.BatchID)
	return nil
}

// sendCompletionNotification 发送完成通知消息
func (s *SaveSyncerImpl) sendCompletionNotification(ctx context.Context, data *SaveSyncData) error {
	message := map[string]interface{}{
		"event_type":    "batch_completed",
		"batch_id":      data.BatchID,
		"pack_id":       data.PackID,
		"final_status":  data.FinalStatus,
		"total_count":   data.TotalCount,
		"success_count": data.SuccessCount,
		"failed_count":  data.FailedCount,
		"completed_at":  data.CompletedAt.Format(time.RFC3339),
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	// 添加处理结果摘要
	if data.ProcessResult != nil {
		message["process_summary"] = data.ProcessResult
	}

	if err := s.mqClient.SendMessage(ctx, "completion_events", message); err != nil {
		return fmt.Errorf("failed to send completion notification: %w", err)
	}

	s.logger.WithContext(ctx).Infof("Sent completion notification for batch: %s", data.BatchID)
	return nil
}

// IsHealthy 健康检查
func (s *SaveSyncerImpl) IsHealthy(ctx context.Context) bool {
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

// ValidateRequest 验证保存同步请求
func (s *SaveSyncerImpl) ValidateRequest(req *SyncRequest) error {
	// 调用基础验证
	if err := s.BaseSyncer.ValidateRequest(req); err != nil {
		return err
	}

	// 验证批次ID
	if _, ok := req.Data["batch_id"]; !ok {
		return fmt.Errorf("missing batch_id in sync data")
	}

	// 验证最终状态
	if _, ok := req.Data["final_status"]; !ok {
		return fmt.Errorf("missing final_status in sync data")
	}

	return nil
}
