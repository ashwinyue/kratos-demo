package step

import (
	"context"
	"fmt"

	delivery "kratos-demo/internal/biz/sms/delivery"
	sync "kratos-demo/internal/biz/sync"
	status "kratos-demo/internal/biz/status"
	types "kratos-demo/internal/biz/sms/types"
)

// 使用types包中的StepType
type StepType = types.StepType

const (
	StepTypePreparation = types.StepTypePreparation
	StepTypeDelivery    = types.StepTypeDelivery
)

// StepStatus 步骤状态
type StepStatus string

const (
	StepStatusPending   StepStatus = "PENDING"
	StepStatusRunning   StepStatus = "RUNNING"
	StepStatusCompleted StepStatus = "COMPLETED"
	StepStatusFailed    StepStatus = "FAILED"
	StepStatusCancelled StepStatus = "CANCELLED"
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

// StepProcessor 步骤处理器接口
type StepProcessor interface {
	Execute(ctx context.Context, batch types.SmsBatchInterface) error
	GetType() StepType
	GetStatus() StepStatus
	GetStatistics() *StepStatistics
}

// PreparationStepProcessor SMS准备步骤处理器
type PreparationStepProcessor struct {
	stepType            StepType
	status              StepStatus
	statistics          *StepStatistics
	smsBatchRepo        types.SmsBatchRepo
	smsDeliveryPackRepo delivery.SmsDeliveryPackRepo
	prepareSyncer       sync.PrepareSyncer
}

// NewPreparationStepProcessor 创建SMS准备步骤处理器
func NewPreparationStepProcessor(
	smsBatchRepo types.SmsBatchRepo,
	smsDeliveryPackRepo delivery.SmsDeliveryPackRepo,
	prepareSyncer sync.PrepareSyncer,
) *PreparationStepProcessor {
	return &PreparationStepProcessor{
		stepType:            StepTypePreparation,
		status:              StepStatusPending,
		statistics:          &StepStatistics{},
		smsBatchRepo:        smsBatchRepo,
		smsDeliveryPackRepo: smsDeliveryPackRepo,
		prepareSyncer:       prepareSyncer,
	}
}

// Execute 执行准备步骤
func (p *PreparationStepProcessor) Execute(ctx context.Context, batch types.SmsBatchInterface) error {
	p.status = StepStatusRunning

	// 初始化同步器
	if err := p.prepareSyncer.InitSyncer(ctx, batch.GetBatchID()); err != nil {
		p.status = StepStatusFailed
		return fmt.Errorf("failed to init prepare syncer: %w", err)
	}

	// 处理SMS批次准备逻辑
	// TODO: 实现具体的准备逻辑

	p.status = StepStatusCompleted
	return nil
}

// GetType 获取步骤类型
func (p *PreparationStepProcessor) GetType() StepType {
	return p.stepType
}

// GetStatus 获取步骤状态
func (p *PreparationStepProcessor) GetStatus() StepStatus {
	return p.status
}

// GetStatistics 获取步骤统计
func (p *PreparationStepProcessor) GetStatistics() *StepStatistics {
	return p.statistics
}

// DeliveryStepProcessor SMS投递步骤处理器
type DeliveryStepProcessor struct {
	stepType            StepType
	status              StepStatus
	statistics          *StepStatistics
	smsBatchRepo        types.SmsBatchRepo
	smsDeliveryPackRepo delivery.SmsDeliveryPackRepo
	deliverySyncer      sync.DeliverySyncer
	messageHandler      status.MessageHandler
}

// NewDeliveryStepProcessor 创建SMS投递步骤处理器
func NewDeliveryStepProcessor(
	smsBatchRepo types.SmsBatchRepo,
	smsDeliveryPackRepo delivery.SmsDeliveryPackRepo,
	deliverySyncer sync.DeliverySyncer,
	messageHandler status.MessageHandler,
) *DeliveryStepProcessor {
	return &DeliveryStepProcessor{
		stepType:            StepTypeDelivery,
		status:              StepStatusPending,
		statistics:          &StepStatistics{},
		smsBatchRepo:        smsBatchRepo,
		smsDeliveryPackRepo: smsDeliveryPackRepo,
		deliverySyncer:      deliverySyncer,
		messageHandler:      messageHandler,
	}
}

// Execute 执行投递步骤
func (d *DeliveryStepProcessor) Execute(ctx context.Context, batch types.SmsBatchInterface) error {
	d.status = StepStatusRunning

	// 初始化同步器
	if err := d.deliverySyncer.InitSyncer(ctx, batch.GetBatchID()); err != nil {
		d.status = StepStatusFailed
		return fmt.Errorf("failed to init delivery syncer: %w", err)
	}

	// 处理SMS批次投递逻辑
	// TODO: 实现具体的投递逻辑

	d.status = StepStatusCompleted
	return nil
}

// GetType 获取步骤类型
func (d *DeliveryStepProcessor) GetType() StepType {
	return d.stepType
}

// GetStatus 获取步骤状态
func (d *DeliveryStepProcessor) GetStatus() StepStatus {
	return d.status
}

// GetStatistics 获取步骤统计
func (d *DeliveryStepProcessor) GetStatistics() *StepStatistics {
	return d.statistics
}

// StepProcessorFactory 步骤处理器工厂接口
type StepProcessorFactory interface {
	CreatePreparationProcessor() StepProcessor
	CreateDeliveryProcessor() StepProcessor
}

// DefaultStepProcessorFactory 默认步骤处理器工厂
type DefaultStepProcessorFactory struct {
	smsBatchRepo        types.SmsBatchRepo
	smsDeliveryPackRepo delivery.SmsDeliveryPackRepo
	prepareSyncer       sync.PrepareSyncer
	deliverySyncer      sync.DeliverySyncer
	messageHandler      status.MessageHandler
}

// NewDefaultStepProcessorFactory 创建默认步骤处理器工厂
func NewDefaultStepProcessorFactory(
	smsBatchRepo types.SmsBatchRepo,
	smsDeliveryPackRepo delivery.SmsDeliveryPackRepo,
	prepareSyncer sync.PrepareSyncer,
	deliverySyncer sync.DeliverySyncer,
	messageHandler status.MessageHandler,
) *DefaultStepProcessorFactory {
	return &DefaultStepProcessorFactory{
		smsBatchRepo:        smsBatchRepo,
		smsDeliveryPackRepo: smsDeliveryPackRepo,
		prepareSyncer:       prepareSyncer,
		deliverySyncer:      deliverySyncer,
		messageHandler:      messageHandler,
	}
}

// CreatePreparationProcessor 创建准备步骤处理器
func (f *DefaultStepProcessorFactory) CreatePreparationProcessor() StepProcessor {
	return NewPreparationStepProcessor(
		f.smsBatchRepo,
		f.smsDeliveryPackRepo,
		f.prepareSyncer,
	)
}

// CreateDeliveryProcessor 创建投递步骤处理器
func (f *DefaultStepProcessorFactory) CreateDeliveryProcessor() StepProcessor {
	return NewDeliveryStepProcessor(
		f.smsBatchRepo,
		f.smsDeliveryPackRepo,
		f.deliverySyncer,
		f.messageHandler,
	)
}
