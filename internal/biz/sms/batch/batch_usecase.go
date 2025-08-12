package batch

import (
	"context"
	"fmt"
	"log"
	"time"

	delivery "kratos-demo/internal/biz/sms/delivery"
	sync "kratos-demo/internal/biz/sync"
	types "kratos-demo/internal/biz/sms/types"
	sender "kratos-demo/internal/biz/sms/sender"
	status "kratos-demo/internal/biz/status"
)

// SmsBatchUsecase SMS批次业务用例
type SmsBatchUsecase struct {
	smsBatchRepo        SmsBatchRepo
	smsDeliveryPackRepo delivery.SmsDeliveryPackRepo
	deliverySyncer      sync.DeliverySyncer
	prepareSyncer       sync.PrepareSyncer
	stepFactory         types.StepProcessorFactory
	messageHandler      status.MessageHandler
	stateMachineFactory SmsBatchStateMachineFactory
}

// SmsBatchStateMachineFactory 状态机工厂接口
type SmsBatchStateMachineFactory interface {
	// CreateStateMachine 创建状态机
	CreateStateMachine(batch *SmsBatch) *SmsBatchStateMachine
}

// DefaultSmsBatchStateMachineFactory 默认状态机工厂
type DefaultSmsBatchStateMachineFactory struct{}

// NewDefaultSmsBatchStateMachineFactory 创建默认状态机工厂
func NewDefaultSmsBatchStateMachineFactory() SmsBatchStateMachineFactory {
	return &DefaultSmsBatchStateMachineFactory{}
}

// CreateStateMachine 创建状态机
func (f *DefaultSmsBatchStateMachineFactory) CreateStateMachine(batch *SmsBatch) *SmsBatchStateMachine {
	return NewSmsBatchStateMachine(batch)
}

// NewSmsBatchUsecase 创建SMS批次业务用例
func NewSmsBatchUsecase(
	smsBatchRepo SmsBatchRepo,
	smsDeliveryPackRepo delivery.SmsDeliveryPackRepo,
	deliverySyncer sync.DeliverySyncer,
	prepareSyncer sync.PrepareSyncer,
	stepFactory types.StepProcessorFactory,
	messageHandler status.MessageHandler,
	stateMachineFactory SmsBatchStateMachineFactory,
) *SmsBatchUsecase {
	return &SmsBatchUsecase{
		smsBatchRepo:        smsBatchRepo,
		smsDeliveryPackRepo: smsDeliveryPackRepo,
		deliverySyncer:      deliverySyncer,
		prepareSyncer:       prepareSyncer,
		stepFactory:         stepFactory,
		messageHandler:      messageHandler,
		stateMachineFactory: stateMachineFactory,
	}
}

// CreateSmsBatch 创建SMS批次
func (uc *SmsBatchUsecase) CreateSmsBatch(ctx context.Context, req *CreateSmsBatchRequest) (*SmsBatch, error) {
	// 创建SMS批次实体
	batch := &SmsBatch{
		ID:                     req.ID,
		PartitionKey:           req.PartitionKey,
		BatchId:                req.BatchId,
		Status:                 SmsBatchStatusInitial,
		CurrentStep:            types.StepTypePreparation,
		CampaignId:             req.CampaignId,
		CampaignName:           req.CampaignName,
		MarketingProgramId:     req.MarketingProgramId,
		TaskId:                 req.TaskId,
		ContentId:              req.ContentId,
		Content:                req.Content,
		ContentSignature:       req.ContentSignature,
		Url:                    req.Url,
		CombineMemberIdWithUrl: req.CombineMemberIdWithUrl,
		AutoTrigger:            req.AutoTrigger,
		ScheduleTime:           req.ScheduleTime,
		ExtCode:                req.ExtCode,
		TaskCode:               req.TaskCode,
		ProviderType:           sender.ProviderType(req.ProviderType),
		// ContainsUrl:           req.ContainsUrl, // 字段不存在，暂时注释
		Region:                          RegionEnum(req.Region),
		MmsId:                           req.MmsId,
		EnableMmsFailedThenSendSms:      req.EnableMmsFailedThenSendSms,
		OnceMmsFailedThenSendSmsContent: req.OnceMmsFailedThenSendSmsContent,
		MessageType:                     MessageType(req.MessageType),
		Source:                          Source(req.Source),
		CreateTime:                      time.Now(),
		LastModifiedTime:                time.Now(),
	}

	// 保存到数据库
	if err := uc.smsBatchRepo.CreateSmsBatch(ctx, batch); err != nil {
		return nil, fmt.Errorf("failed to save sms batch: %w", err)
	}

	log.Printf("Created SMS batch: %s", batch.ID)
	return batch, nil
}

// CreateSmsBatchRequest 创建SMS批次请求
type CreateSmsBatchRequest struct {
	ID                              string       `json:"id"`
	PartitionKey                    string       `json:"partition_key"`
	BatchId                         string       `json:"batch_id"`
	CampaignId                      string       `json:"campaign_id"`
	CampaignName                    string       `json:"campaign_name"`
	MarketingProgramId              string       `json:"marketing_program_id"`
	TaskId                          string       `json:"task_id"`
	ContentId                       string       `json:"content_id"`
	Content                         string       `json:"content"`
	ContentSignature                string       `json:"content_signature"`
	Url                             string       `json:"url"`
	CombineMemberIdWithUrl          bool         `json:"combine_member_id_with_url"`
	AutoTrigger                     bool         `json:"auto_trigger"`
	ScheduleTime                    *time.Time   `json:"schedule_time"`
	ExtCode                         int          `json:"ext_code"`
	TaskCode                        string       `json:"task_code"`
	ProviderType                    sender.ProviderType `json:"provider_type"`
	ContainsUrl                     bool         `json:"contains_url"`
	Region                          string       `json:"region"`
	MmsId                           string       `json:"mms_id"`
	EnableMmsFailedThenSendSms      bool         `json:"enable_mms_failed_then_send_sms"`
	OnceMmsFailedThenSendSmsContent string       `json:"once_mms_failed_then_send_sms_content"`
	MessageType                     string       `json:"message_type"`
	Source                          string       `json:"source"`
}

// StartSmsBatch 启动SMS批次
func (uc *SmsBatchUsecase) StartSmsBatch(ctx context.Context, batchId string) error {
	// 获取批次
	batch, err := uc.smsBatchRepo.GetSmsBatchByBatchId(ctx, batchId)
	if err != nil {
		return fmt.Errorf("failed to get sms batch: %w", err)
	}

	// 创建状态机
	stateMachine := uc.stateMachineFactory.CreateStateMachine(batch)

	// 触发开始事件
	if err := stateMachine.Start(ctx); err != nil {
		return fmt.Errorf("failed to start sms batch: %w", err)
	}

	// 更新批次状态
	if err := uc.smsBatchRepo.Update(ctx, batch); err != nil {
		return fmt.Errorf("failed to update sms batch: %w", err)
	}

	log.Printf("Started SMS batch: %s", batch.ID)
	return nil
}

// PauseSmsBatch 暂停SMS批次
func (uc *SmsBatchUsecase) PauseSmsBatch(ctx context.Context, batchId string) error {
	// 获取批次
	batch, err := uc.smsBatchRepo.GetSmsBatchByBatchId(ctx, batchId)
	if err != nil {
		return fmt.Errorf("failed to get sms batch: %w", err)
	}

	// 创建状态机
	stateMachine := uc.stateMachineFactory.CreateStateMachine(batch)

	// 触发暂停事件
	if err := stateMachine.Pause(ctx); err != nil {
		return fmt.Errorf("failed to pause sms batch: %w", err)
	}

	// 更新批次状态
	if err := uc.smsBatchRepo.Update(ctx, batch); err != nil {
		return fmt.Errorf("failed to update sms batch: %w", err)
	}

	log.Printf("Paused SMS batch: %s", batch.ID)
	return nil
}

// ResumeSmsBatch 恢复SMS批次
func (uc *SmsBatchUsecase) ResumeSmsBatch(ctx context.Context, batchId string) error {
	// 获取批次
	batch, err := uc.smsBatchRepo.GetSmsBatchByBatchId(ctx, batchId)
	if err != nil {
		return fmt.Errorf("failed to get sms batch: %w", err)
	}

	// 创建状态机
	stateMachine := uc.stateMachineFactory.CreateStateMachine(batch)

	// 触发恢复事件
	if err := stateMachine.Resume(ctx); err != nil {
		return fmt.Errorf("failed to resume sms batch: %w", err)
	}

	// 更新批次状态
	if err := uc.smsBatchRepo.Update(ctx, batch); err != nil {
		return fmt.Errorf("failed to update sms batch: %w", err)
	}

	log.Printf("Resumed SMS batch: %s", batch.ID)
	return nil
}

// RetrySmsBatch 重试SMS批次
func (uc *SmsBatchUsecase) RetrySmsBatch(ctx context.Context, batchId string) error {
	// 获取批次
	batch, err := uc.smsBatchRepo.GetSmsBatchByBatchId(ctx, batchId)
	if err != nil {
		return fmt.Errorf("failed to get sms batch: %w", err)
	}

	// 创建状态机
	stateMachine := uc.stateMachineFactory.CreateStateMachine(batch)

	// 触发重试事件
	if err := stateMachine.Retry(ctx); err != nil {
		return fmt.Errorf("failed to retry sms batch: %w", err)
	}

	// 更新批次状态
	if err := uc.smsBatchRepo.Update(ctx, batch); err != nil {
		return fmt.Errorf("failed to update sms batch: %w", err)
	}

	log.Printf("Retried SMS batch: %s", batch.ID)
	return nil
}

// GetSmsBatch 获取SMS批次
func (uc *SmsBatchUsecase) GetSmsBatch(ctx context.Context, batchId string) (*SmsBatch, error) {
	batch, err := uc.smsBatchRepo.GetSmsBatchByBatchId(ctx, batchId)
	if err != nil {
		return nil, fmt.Errorf("failed to get sms batch: %w", err)
	}

	return batch, nil
}

// ListSmsBatches 列出SMS批次
func (uc *SmsBatchUsecase) ListSmsBatches(ctx context.Context, offset, limit int) ([]*SmsBatch, error) {
	batches, err := uc.smsBatchRepo.ListSmsBatch(ctx, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list sms batches: %w", err)
	}

	return batches, nil
}

// GetRunningBatches 获取运行中的批次
func (uc *SmsBatchUsecase) GetRunningBatches(ctx context.Context) ([]*SmsBatch, error) {
	batches, err := uc.smsBatchRepo.GetRunningBatches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get running batches: %w", err)
	}

	return batches, nil
}

// ProcessPreparationStep 处理准备步骤
func (uc *SmsBatchUsecase) ProcessPreparationStep(ctx context.Context, batchId string) error {
	// 获取批次
	batch, err := uc.smsBatchRepo.GetSmsBatchByBatchId(ctx, batchId)
	if err != nil {
		return fmt.Errorf("failed to get sms batch: %w", err)
	}

	// 设置当前步骤为准备步骤
	batch.CurrentStep = types.StepTypePreparation

	// 创建准备步骤处理器
	stepProcessor := uc.stepFactory.CreatePreparationProcessor()

	// 执行准备步骤
	if err := stepProcessor.Execute(ctx, batch); err != nil {
		return fmt.Errorf("failed to process preparation step: %w", err)
	}

	log.Printf("Processed preparation step for batch: %s", batch.ID)
	return nil
}

// ProcessDeliveryStep 处理投递步骤
func (uc *SmsBatchUsecase) ProcessDeliveryStep(ctx context.Context, batchId string) error {
	// 获取批次
	batch, err := uc.smsBatchRepo.GetSmsBatchByBatchId(ctx, batchId)
	if err != nil {
		return fmt.Errorf("failed to get sms batch: %w", err)
	}

	// 设置当前步骤为投递步骤
	batch.CurrentStep = types.StepTypeDelivery

	// 创建投递步骤处理器
	stepProcessor := uc.stepFactory.CreateDeliveryProcessor()

	// 执行投递步骤
	if err := stepProcessor.Execute(ctx, batch); err != nil {
		return fmt.Errorf("failed to process delivery step: %w", err)
	}

	log.Printf("Processed delivery step for batch: %s", batch.ID)
	return nil
}

// GetBatchStatistics 获取批次统计信息
func (uc *SmsBatchUsecase) GetBatchStatistics(ctx context.Context, batchId string) (*SmsBatchStats, error) {
	// 获取批次
	batch, err := uc.smsBatchRepo.GetSmsBatchByBatchId(ctx, batchId)
	if err != nil {
		return nil, fmt.Errorf("failed to get sms batch: %w", err)
	}

	// 从同步器获取统计信息
	deliveryStats, err := uc.deliverySyncer.Sync2Statistics(ctx, batch.ID)
	if err != nil {
		log.Printf("Failed to get delivery statistics: %v", err)
		deliveryStats = &sync.StepStatistics{}
	}

	// 构建批次统计信息
	stats := &SmsBatchStats{
		BatchId:      batch.BatchId,
		TotalCount:   batch.TotalCount,
		SuccessCount: deliveryStats.Success,
		PendingCount: batch.TotalCount - deliveryStats.Success - deliveryStats.Failure,
	}

	return stats, nil
}
