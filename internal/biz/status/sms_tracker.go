package status

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	types "kratos-demo/internal/biz/sms/types"
	sync "kratos-demo/internal/biz/sync"
)

// DeliveryStatusTracker 投递状态跟踪器
type DeliveryStatusTracker struct {
	smsBatchRepo    types.SmsBatchRepo
	deliverySyncer  sync.DeliverySyncer
	stepFactory     types.StepProcessorFactory
	messageHandler  MessageHandler
	timeoutInterval time.Duration // 步骤超时间隔
}

// MessageHandler 消息处理器接口
type MessageHandler interface {
	// SendBatchDeliveryTask 发送批次投递任务
	SendBatchDeliveryTask(ctx context.Context, task string) error

	// SendBatchPreparationTask 发送批次准备任务
	SendBatchPreparationTask(ctx context.Context, task string) error
}

// SmsBatchPartitionTask SMS批次分区任务
type SmsBatchPartitionTask struct {
	SmsBatchId   string            `json:"sms_batch_id"`  // SMS批次ID
	PartitionKey string            `json:"partition_key"` // 分区键
	Status       types.BatchStatus `json:"status"`        // 状态
	TaskCode     string            `json:"task_code"`     // 任务代码
	CreateTime   time.Time         `json:"create_time"`   // 创建时间
}

// NewDeliveryStatusTracker 创建投递状态跟踪器
func NewDeliveryStatusTracker(
	smsBatchRepo types.SmsBatchRepo,
	deliverySyncer sync.DeliverySyncer,
	stepFactory types.StepProcessorFactory,
	messageHandler MessageHandler,
	timeoutInterval time.Duration,
) *DeliveryStatusTracker {
	return &DeliveryStatusTracker{
		smsBatchRepo:    smsBatchRepo,
		deliverySyncer:  deliverySyncer,
		stepFactory:     stepFactory,
		messageHandler:  messageHandler,
		timeoutInterval: timeoutInterval,
	}
}

// TrackRunningBatches 跟踪运行中的批次
func (t *DeliveryStatusTracker) TrackRunningBatches(ctx context.Context) error {
	// 获取所有运行中的批次
	runningBatches, err := t.smsBatchRepo.GetRunningBatches(ctx)
	if err != nil {
		return fmt.Errorf("failed to get running batches: %w", err)
	}

	for _, batch := range runningBatches {
			if err := t.trackBatch(ctx, batch); err != nil {
				log.Printf("Failed to track batch %d: %v", batch.GetID(), err)
				continue
			}
		}

	return nil
}

// trackBatch 跟踪单个批次
func (t *DeliveryStatusTracker) trackBatch(ctx context.Context, batch types.SmsBatchInterface) error {
	// 检查是否所有分区任务都完成
	// 注意：AllPkTasksCompleted方法已从接口中移除，这里需要其他方式实现
	allCompleted := false // 临时实现

	if allCompleted {
		return t.handleComplete(ctx, batch)
	}

	// 检查分区任务超时
	return t.checkPartitionTimeouts(ctx, batch)
}

// checkPartitionTimeouts 检查分区任务超时
func (t *DeliveryStatusTracker) checkPartitionTimeouts(ctx context.Context, batch types.SmsBatchInterface) error {
	// 获取所有分区键（假设有128个分区）
	partitionKeys := t.generatePartitionKeys(128)

	for _, partitionKey := range partitionKeys {
		heartbeatTime, err := t.deliverySyncer.GetHeartbeatTime(ctx, batch.GetBatchID())
		if err != nil {
			// 如果获取心跳时间失败，设置当前时间作为心跳时间
			currentTime := time.Now()
			if err := t.deliverySyncer.SetHeartbeatTime(ctx, batch.GetBatchID(), currentTime); err != nil {
				log.Printf("Failed to set heartbeat time for partition %s: %v", partitionKey, err)
			}
			heartbeatTime = currentTime
		}

		lastUpdatedTime := heartbeatTime.Unix()
		if lastUpdatedTime == 0 {
			// 回退处理：如果调度实例在更新心跳之前终止
			currentTime := time.Now()
			if err := t.deliverySyncer.SetHeartbeatTime(ctx, batch.GetBatchID(), currentTime); err != nil {
				log.Printf("Failed to set fallback heartbeat time for partition %s: %v", partitionKey, err)
			}
			lastUpdatedTime = currentTime.Unix()
		}

		if t.isPartitionTimeout(lastUpdatedTime) {
			log.Printf("Partition %s for SmsBatch %s timeout. Reschedule it.", partitionKey, batch.GetBatchID())

			// 重新调度分区任务
			if err := t.sendPartitionTask2Queue(ctx, batch.GetBatchID(), partitionKey, types.BatchStatusPending, ""); err != nil {
				log.Printf("Failed to reschedule partition task %s: %v", partitionKey, err)
			}
		}
	}

	return nil
}

// handleComplete 处理完成的批次
func (t *DeliveryStatusTracker) handleComplete(ctx context.Context, batch types.SmsBatchInterface) error {
	start := time.Now()

	// 创建投递步骤处理器
	stepProcessor := t.stepFactory.CreateDeliveryProcessor()

	// 执行投递步骤
	if err := stepProcessor.Execute(ctx, batch); err != nil {
		return fmt.Errorf("failed to execute delivery step: %w", err)
	}

	log.Printf("SMS delivery completed for batch %s, duration: %v",
		batch.GetBatchID(), time.Since(start))

	return nil
}

// sendPartitionTask2Queue 发送分区任务到队列
func (t *DeliveryStatusTracker) sendPartitionTask2Queue(ctx context.Context, smsBatchId, partitionKey string, status types.BatchStatus, taskCode string) error {
	task := &SmsBatchPartitionTask{
		SmsBatchId:   smsBatchId,
		PartitionKey: partitionKey,
		Status:       status,
		TaskCode:     taskCode,
		CreateTime:   time.Now(),
	}

	taskJSON, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal partition task: %w", err)
	}

	return t.messageHandler.SendBatchDeliveryTask(ctx, string(taskJSON))
}

// isPartitionTimeout 检查分区是否超时
func (t *DeliveryStatusTracker) isPartitionTimeout(lastUpdatedTime int64) bool {
	return time.Now().Unix()-lastUpdatedTime >= int64(t.timeoutInterval.Seconds())
}

// generatePartitionKeys 生成分区键
func (t *DeliveryStatusTracker) generatePartitionKeys(count int) []string {
	keys := make([]string, count)
	for i := 0; i < count; i++ {
		keys[i] = fmt.Sprintf("pk_%03d", i)
	}
	return keys
}

// PreparationStatusTracker 准备状态跟踪器
type PreparationStatusTracker struct {
	smsBatchRepo   types.SmsBatchRepo
	prepareSyncer  sync.PrepareSyncer
	stepFactory    types.StepProcessorFactory
	messageHandler MessageHandler
}

// NewPreparationStatusTracker 创建准备状态跟踪器
func NewPreparationStatusTracker(
	smsBatchRepo types.SmsBatchRepo,
	prepareSyncer sync.PrepareSyncer,
	stepFactory types.StepProcessorFactory,
	messageHandler MessageHandler,
) *PreparationStatusTracker {
	return &PreparationStatusTracker{
		smsBatchRepo:   smsBatchRepo,
		prepareSyncer:  prepareSyncer,
		stepFactory:    stepFactory,
		messageHandler: messageHandler,
	}
}

// TrackPreparationBatches 跟踪准备中的批次
func (t *PreparationStatusTracker) TrackPreparationBatches(ctx context.Context) error {
	// 获取所有准备中的批次
	preparingBatches, err := t.smsBatchRepo.GetByStatus(ctx, types.BatchStatusPreparing)
	if err != nil {
		return fmt.Errorf("failed to get preparing batches: %w", err)
	}

	// 跟踪每个批次
	for _, batch := range preparingBatches {
		if err := t.trackPreparationBatch(ctx, batch); err != nil {
			log.Printf("Failed to track preparation batch %d: %v", batch.GetID(), err)
			continue
		}
	}

	return nil
}

// trackPreparationBatch 跟踪准备阶段批次
func (t *PreparationStatusTracker) trackPreparationBatch(ctx context.Context, batch types.SmsBatchInterface) error {
	// 创建准备步骤处理器
	stepProcessor := t.stepFactory.CreatePreparationProcessor()

	// 执行准备步骤
	if err := stepProcessor.Execute(ctx, batch); err != nil {
		return fmt.Errorf("failed to execute preparation step: %w", err)
	}

	log.Printf("SMS preparation completed for batch %d", batch.GetID())
	return nil
}

// BatchScheduler 批次调度器
type BatchScheduler struct {
	deliveryTracker    *DeliveryStatusTracker
	preparationTracker *PreparationStatusTracker
	tickerInterval     time.Duration
}

// NewBatchScheduler 创建批次调度器
func NewBatchScheduler(
	deliveryTracker *DeliveryStatusTracker,
	preparationTracker *PreparationStatusTracker,
	tickerInterval time.Duration,
) *BatchScheduler {
	return &BatchScheduler{
		deliveryTracker:    deliveryTracker,
		preparationTracker: preparationTracker,
		tickerInterval:     tickerInterval,
	}
}

// Start 启动调度器
func (s *BatchScheduler) Start(ctx context.Context) error {
	ticker := time.NewTicker(s.tickerInterval)
	defer ticker.Stop()

	log.Printf("Batch scheduler started with interval: %v", s.tickerInterval)

	for {
		select {
		case <-ctx.Done():
			log.Println("Batch scheduler stopped")
			return ctx.Err()
		case <-ticker.C:
			// 跟踪投递状态
			if err := s.deliveryTracker.TrackRunningBatches(ctx); err != nil {
				log.Printf("Failed to track delivery batches: %v", err)
			}

			// 跟踪准备状态
			if err := s.preparationTracker.TrackPreparationBatches(ctx); err != nil {
				log.Printf("Failed to track preparation batches: %v", err)
			}
		}
	}
}

// Stop 停止调度器
func (s *BatchScheduler) Stop() {
	log.Println("Stopping batch scheduler...")
}
