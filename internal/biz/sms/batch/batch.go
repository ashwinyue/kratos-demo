package batch

import (
	"context"
	"log"
	"time"

	"github.com/looplab/fsm"
	sender "kratos-demo/internal/biz/sms/sender"
	types "kratos-demo/internal/biz/sms/types"
)

// SmsBatchStatus SMS批次状态枚举
type SmsBatchStatus string

const (
	SmsBatchStatusInitial            SmsBatchStatus = "INITIAL"             // 初始状态
	SmsBatchStatusReady              SmsBatchStatus = "READY"               // 准备就绪
	SmsBatchStatusRunning            SmsBatchStatus = "RUNNING"             // 运行中
	SmsBatchStatusPaused             SmsBatchStatus = "PAUSED"              // 已暂停
	SmsBatchStatusPausing            SmsBatchStatus = "PAUSING"             // 暂停中
	SmsBatchStatusCompletedSucceeded SmsBatchStatus = "COMPLETED_SUCCEEDED" // 完成成功
	SmsBatchStatusCompletedFailed    SmsBatchStatus = "COMPLETED_FAILED"    // 完成失败
)

// 步骤类型定义在sms_step.go中

// SmsBatchEvent SMS批次事件枚举
type SmsBatchEvent string

const (
	SmsBatchEventStart    SmsBatchEvent = "START"    // 开始事件
	SmsBatchEventPause    SmsBatchEvent = "PAUSE"    // 暂停事件
	SmsBatchEventResume   SmsBatchEvent = "RESUME"   // 恢复事件
	SmsBatchEventComplete SmsBatchEvent = "COMPLETE" // 完成事件
	SmsBatchEventFail     SmsBatchEvent = "FAIL"     // 失败事件
	SmsBatchEventRetry    SmsBatchEvent = "RETRY"    // 重试事件
)

// MessageType 消息类型枚举
type MessageType string

const (
	MessageTypeSms MessageType = "SMS" // 短信
	MessageTypeMms MessageType = "MMS" // 彩信
)

// Source 来源枚举
type Source string

const (
	SourceCampaign Source = "CAMPAIGN" // 营销活动
	SourceApi      Source = "API"      // API调用
	SourceManual   Source = "MANUAL"   // 手动触发
)

// RegionEnum 区域枚举
type RegionEnum string

const (
	RegionGlobal RegionEnum = "GLOBAL" // 全球
	RegionChina  RegionEnum = "CHINA"  // 中国
	RegionUS     RegionEnum = "US"     // 美国
	RegionEU     RegionEnum = "EU"     // 欧盟
)

// SmsBatch SMS批次实体
type SmsBatch struct {
	// 基础信息
	ID           string    `json:"id" bson:"_id"`                      // 主键ID
	PartitionKey string    `json:"partition_key" bson:"partition_key"` // 分区键
	BatchId      string    `json:"batch_id" bson:"batch_id"`           // 批次号
	TableName    string    `json:"table_name" bson:"table_name"`       // Table Storage名字
	CreateTime   time.Time `json:"create_time" bson:"create_time"`     // 创建时间
	UpdateTime   time.Time `json:"update_time" bson:"update_time"`     // 更新时间

	// 营销活动信息
	CampaignId         string     `json:"campaign_id" bson:"campaign_id"`                         // 运营活动ID
	CampaignName       string     `json:"campaign_name" bson:"campaign_name"`                     // 运营活动名称
	MarketingProgramId string     `json:"marketing_program_id" bson:"marketing_program_id"`       // 市场营销的程序ID
	TaskId             string     `json:"task_id" bson:"task_id"`                                 // 任务号
	TaskCode           string     `json:"task_code" bson:"task_code"`                             // 任务号，用于判断队列消息是否同一批次
	AutoTrigger        bool       `json:"auto_trigger" bson:"auto_trigger"`                       // 自动触发
	ScheduleTime       *time.Time `json:"schedule_time,omitempty" bson:"schedule_time,omitempty"` // Campaign Management的活动调度时间

	// 短信内容信息
	ContentId              string `json:"content_id" bson:"content_id"`                                 // 短信模板的内容ID
	Content                string `json:"content" bson:"content"`                                       // 短信模板的内容
	ContentSignature       string `json:"content_signature" bson:"content_signature"`                   // 短信模板签名
	Url                    string `json:"url,omitempty" bson:"url,omitempty"`                           // 短信模板包含的用户可点击的长链接
	CombineMemberIdWithUrl bool   `json:"combine_member_id_with_url" bson:"combine_member_id_with_url"` // 是否开启短链返回memberId

	// 彩信相关
	MmsId                           string `json:"mms_id,omitempty" bson:"mms_id,omitempty"`                                           // 第三方彩信模板id
	EnableMmsFailedThenSendSms      bool   `json:"enable_mms_failed_then_send_sms" bson:"enable_mms_failed_then_send_sms"`             // 彩信失败是否启用备用短信
	OnceMmsFailedThenSendSmsContent string `json:"once_mms_failed_then_send_sms_content" bson:"once_mms_failed_then_send_sms_content"` // 彩信备用短信

	// 发送配置
	ProviderType sender.ProviderType `json:"provider_type" bson:"provider_type"` // 提供商类型
	ExtCode      int                  `json:"ext_code" bson:"ext_code"`           // 短信扩展码(1个批次对应一个扩展码)
	Region       RegionEnum           `json:"region" bson:"region"`               // 区域
	MessageType  MessageType          `json:"message_type" bson:"message_type"`   // 短信种类：短信/彩信
	Source       Source               `json:"source" bson:"source"`               // 来源

	// 执行状态
	CurrentStep types.StepType `json:"current_step" bson:"current_step"` // 当前步骤
	Status      SmsBatchStatus `json:"status" bson:"status"`             // 当前状态

	// 统计信息
	TotalCount     int64 `json:"total_count" bson:"total_count"`         // 总数量
	SuccessCount   int64 `json:"success_count" bson:"success_count"`     // 成功数量
	FailedCount    int64 `json:"failed_count" bson:"failed_count"`       // 失败数量
	PendingCount   int64 `json:"pending_count" bson:"pending_count"`     // 待处理数量
	ProcessedCount int64 `json:"processed_count" bson:"processed_count"` // 已处理数量

	// 执行信息
	ExecuteTimes          int        `json:"execute_times" bson:"execute_times"`                                         // 执行次数
	LastModifiedTime      time.Time  `json:"last_modified_time" bson:"last_modified_time"`                               // 最后修改时间
	LastExecuteTime       *time.Time `json:"last_execute_time,omitempty" bson:"last_execute_time,omitempty"`             // 最后执行时间
	EstimatedCompleteTime *time.Time `json:"estimated_complete_time,omitempty" bson:"estimated_complete_time,omitempty"` // 预计完成时间

	// 步骤相关字段
	PreparationStep interface{} `json:"preparation_step"` // 准备步骤
	DeliveryStep    interface{} `json:"delivery_step"`    // 投递步骤

	// 扩展字段
	Metadata map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"` // 元数据
	Tags     []string               `json:"tags,omitempty" bson:"tags,omitempty"`         // 标签
}

// 实现 types.SmsBatchInterface 接口
func (b *SmsBatch) GetID() int64 {
	// 由于ID是string类型，这里需要转换或使用其他字段
	// 暂时返回0，实际使用时可能需要调整
	return 0
}

func (b *SmsBatch) GetBatchID() string {
	return b.BatchId
}

func (b *SmsBatch) GetStatus() types.BatchStatus {
	// 将SmsBatchStatus转换为types.BatchStatus
	switch b.Status {
	case SmsBatchStatusInitial:
		return types.BatchStatusPending
	case SmsBatchStatusReady:
		return types.BatchStatusPending
	case SmsBatchStatusRunning:
		return types.BatchStatusRunning
	case SmsBatchStatusPaused:
		return types.BatchStatusPending
	case SmsBatchStatusPausing:
		return types.BatchStatusPending
	case SmsBatchStatusCompletedSucceeded:
		return types.BatchStatusCompleted
	case SmsBatchStatusCompletedFailed:
		return types.BatchStatusFailed
	default:
		return types.BatchStatusPending
	}
}

func (b *SmsBatch) GetCreatedAt() time.Time {
	return b.CreateTime
}

func (b *SmsBatch) GetUpdatedAt() time.Time {
	return b.UpdateTime
}

// SmsBatchStateMachine SMS批次状态机
type SmsBatchStateMachine struct {
	fsm     *fsm.FSM
	batch   *SmsBatch
	logger  interface{} // 日志接口
	metrics interface{} // 指标接口
}

// NewSmsBatchStateMachine 创建SMS批次状态机
func NewSmsBatchStateMachine(batch *SmsBatch) *SmsBatchStateMachine {
	sm := &SmsBatchStateMachine{
		batch: batch,
	}

	// 定义状态机
	sm.fsm = fsm.NewFSM(
		string(batch.Status), // 初始状态
		fsm.Events{
			// 从初始状态开始
			fsm.EventDesc{Name: string(SmsBatchEventStart), Src: []string{string(SmsBatchStatusInitial)}, Dst: string(SmsBatchStatusReady)},

			// 从准备状态到运行状态
			fsm.EventDesc{Name: string(SmsBatchEventStart), Src: []string{string(SmsBatchStatusReady)}, Dst: string(SmsBatchStatusRunning)},

			// 运行状态的转换
			fsm.EventDesc{Name: string(SmsBatchEventPause), Src: []string{string(SmsBatchStatusRunning)}, Dst: string(SmsBatchStatusPausing)},
			fsm.EventDesc{Name: string(SmsBatchEventComplete), Src: []string{string(SmsBatchStatusRunning)}, Dst: string(SmsBatchStatusCompletedSucceeded)},
			fsm.EventDesc{Name: string(SmsBatchEventFail), Src: []string{string(SmsBatchStatusRunning)}, Dst: string(SmsBatchStatusCompletedFailed)},

			// 暂停状态的转换
			fsm.EventDesc{Name: string(SmsBatchEventPause), Src: []string{string(SmsBatchStatusPausing)}, Dst: string(SmsBatchStatusPaused)},
			fsm.EventDesc{Name: string(SmsBatchEventResume), Src: []string{string(SmsBatchStatusPaused)}, Dst: string(SmsBatchStatusRunning)},

			// 失败状态的重试
			fsm.EventDesc{Name: string(SmsBatchEventRetry), Src: []string{string(SmsBatchStatusCompletedFailed)}, Dst: string(SmsBatchStatusReady)},
		},
		fsm.Callbacks{
			"enter_state":  sm.onEnterState,
			"leave_state":  sm.onLeaveState,
			"before_event": sm.onBeforeEvent,
			"after_event":  sm.onAfterEvent,
		},
	)

	return sm
}

// GetCurrentState 获取当前状态
func (sm *SmsBatchStateMachine) GetCurrentState() SmsBatchStatus {
	return SmsBatchStatus(sm.fsm.Current())
}

// CanTransition 检查是否可以进行状态转换
func (sm *SmsBatchStateMachine) CanTransition(event SmsBatchEvent) bool {
	return sm.fsm.Can(string(event))
}

// Transition 执行状态转换
func (sm *SmsBatchStateMachine) Transition(ctx context.Context, event SmsBatchEvent) error {
	err := sm.fsm.Event(ctx, string(event))
	if err != nil {
		return err
	}

	// 更新批次状态
	sm.batch.Status = SmsBatchStatus(sm.fsm.Current())
	sm.batch.LastModifiedTime = time.Now()

	return nil
}

// TriggerEvent 触发状态转换事件
func (sm *SmsBatchStateMachine) TriggerEvent(ctx context.Context, event SmsBatchEvent) error {
	return sm.fsm.Event(ctx, string(event))
}

// Start 开始批次处理
func (sm *SmsBatchStateMachine) Start(ctx context.Context) error {
	return sm.Transition(ctx, SmsBatchEventStart)
}

// Pause 暂停批次处理
func (sm *SmsBatchStateMachine) Pause(ctx context.Context) error {
	return sm.Transition(ctx, SmsBatchEventPause)
}

// Resume 恢复批次处理
func (sm *SmsBatchStateMachine) Resume(ctx context.Context) error {
	return sm.Transition(ctx, SmsBatchEventResume)
}

// Complete 完成批次处理
func (sm *SmsBatchStateMachine) Complete(ctx context.Context) error {
	return sm.Transition(ctx, SmsBatchEventComplete)
}

// Fail 标记批次处理失败
func (sm *SmsBatchStateMachine) Fail(ctx context.Context) error {
	return sm.Transition(ctx, SmsBatchEventFail)
}

// Retry 重试批次处理
func (sm *SmsBatchStateMachine) Retry(ctx context.Context) error {
	return sm.Transition(ctx, SmsBatchEventRetry)
}

// IsCompleted 检查是否已完成
func (sm *SmsBatchStateMachine) IsCompleted() bool {
	currentState := sm.GetCurrentState()
	return currentState == SmsBatchStatusCompletedSucceeded || currentState == SmsBatchStatusCompletedFailed
}

// IsRunning 检查是否正在运行
func (sm *SmsBatchStateMachine) IsRunning() bool {
	return sm.GetCurrentState() == SmsBatchStatusRunning
}

// IsPaused 检查是否已暂停
func (sm *SmsBatchStateMachine) IsPaused() bool {
	currentState := sm.GetCurrentState()
	return currentState == SmsBatchStatusPaused || currentState == SmsBatchStatusPausing
}

// 状态机回调函数
func (sm *SmsBatchStateMachine) onEnterState(ctx context.Context, e *fsm.Event) {
	// 进入状态时的处理逻辑
	log.Printf("SMS Batch %s entering state: %s", sm.batch.BatchId, e.Dst)
	sm.batch.Status = SmsBatchStatus(e.Dst)
	sm.batch.LastModifiedTime = time.Now()
}

func (sm *SmsBatchStateMachine) onLeaveState(ctx context.Context, e *fsm.Event) {
	// 离开状态时的处理逻辑
	log.Printf("SMS Batch %s leaving state: %s", sm.batch.BatchId, e.Src)
}

func (sm *SmsBatchStateMachine) onBeforeEvent(ctx context.Context, e *fsm.Event) {
	// 事件执行前的处理逻辑
	log.Printf("SMS Batch %s before event: %s", sm.batch.BatchId, e.Event)
}

func (sm *SmsBatchStateMachine) onAfterEvent(ctx context.Context, e *fsm.Event) {
	// 事件执行后的处理逻辑
	if e.Dst == string(SmsBatchStatusRunning) {
		now := time.Now()
		sm.batch.LastExecuteTime = &now
	}
}

// SmsBatchRepo SMS批次仓储接口
type SmsBatchRepo interface {
	// CreateSmsBatch 创建SMS批次
	CreateSmsBatch(ctx context.Context, batch *SmsBatch) error

	// GetSmsBatch 根据ID获取SMS批次
	GetSmsBatch(ctx context.Context, id string) (*SmsBatch, error)

	// GetSmsBatchByBatchId 根据批次号获取SMS批次
	GetSmsBatchByBatchId(ctx context.Context, batchId string) (*SmsBatch, error)

	// ListSmsBatch 列出SMS批次
	ListSmsBatch(ctx context.Context, offset, limit int) ([]*SmsBatch, error)

	// Update 更新SMS批次
	Update(ctx context.Context, batch *SmsBatch) error

	// UpdateSmsBatch 更新SMS批次
	UpdateSmsBatch(ctx context.Context, batch *SmsBatch) error

	// UpdateSmsBatchStatus 更新SMS批次状态
	UpdateSmsBatchStatus(ctx context.Context, id string, status SmsBatchStatus) error

	// ListSmsBatchByStatus 根据状态列出SMS批次
	ListSmsBatchByStatus(ctx context.Context, status SmsBatchStatus, limit int) ([]*SmsBatch, error)

	// ListRunningSmsBatch 列出运行中的SMS批次
	ListRunningSmsBatch(ctx context.Context) ([]*SmsBatch, error)

	// GetRunningBatches 获取运行中的批次
	GetRunningBatches(ctx context.Context) ([]*SmsBatch, error)

	// GetByStatus 根据状态获取批次列表
	GetByStatus(ctx context.Context, status SmsBatchStatus) ([]*SmsBatch, error)

	// DeleteSmsBatch 删除SMS批次
	DeleteSmsBatch(ctx context.Context, id string) error

	// GetSmsBatchStats 获取SMS批次统计信息
	GetSmsBatchStats(ctx context.Context, batchId string) (*SmsBatchStats, error)
}

// SmsBatchStats SMS批次统计信息
type SmsBatchStats struct {
	BatchId        string         `json:"batch_id"`
	TotalCount     int64          `json:"total_count"`
	SuccessCount   int64          `json:"success_count"`
	FailedCount    int64          `json:"failed_count"`
	PendingCount   int64          `json:"pending_count"`
	ProcessedCount int64          `json:"processed_count"`
	SuccessRate    float64        `json:"success_rate"`
	StartTime      time.Time      `json:"start_time"`
	EndTime        *time.Time     `json:"end_time,omitempty"`
	Duration       *time.Duration `json:"duration,omitempty"`
	Throughput     float64        `json:"throughput"` // 每秒处理数量
}

// CalculateSuccessRate 计算成功率
func (s *SmsBatchStats) CalculateSuccessRate() {
	if s.ProcessedCount > 0 {
		s.SuccessRate = float64(s.SuccessCount) / float64(s.ProcessedCount) * 100
	}
}

// CalculateThroughput 计算吞吐量
func (s *SmsBatchStats) CalculateThroughput() {
	if s.EndTime != nil && !s.StartTime.IsZero() {
		duration := s.EndTime.Sub(s.StartTime)
		if duration.Seconds() > 0 {
			s.Throughput = float64(s.ProcessedCount) / duration.Seconds()
			s.Duration = &duration
		}
	}
}
