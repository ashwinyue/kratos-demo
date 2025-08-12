package delivery

import (
	"context"
	"time"
)

// SmsBatchStatus SMS批次状态
type SmsBatchStatus string

const (
	SmsBatchStatusPending   SmsBatchStatus = "PENDING"
	SmsBatchStatusRunning   SmsBatchStatus = "RUNNING"
	SmsBatchStatusCompleted SmsBatchStatus = "COMPLETED"
	SmsBatchStatusFailed    SmsBatchStatus = "FAILED"
	SmsBatchStatusCancelled SmsBatchStatus = "CANCELLED"
)

// ProviderType 提供商类型
type ProviderType string

const (
	ProviderTypeAliyun ProviderType = "ALIYUN"
	ProviderTypeWe     ProviderType = "WE"
	ProviderTypeXsxx   ProviderType = "XSXX"
)

// SmsDeliveryPack SMS投递包
type SmsDeliveryPack struct {
	ID           string            `json:"id"`            // 投递包ID
	PartitionKey string            `json:"partition_key"` // 分区键
	Status       SmsBatchStatus    `json:"status"`        // 状态
	MemberId     string            `json:"member_id"`     // 会员ID
	Mobile       string            `json:"mobile"`        // 手机号
	ShortCode    string            `json:"short_code"`    // 短链接代码
	MessageId    string            `json:"message_id"`    // 消息ID
	ExtCode      int               `json:"ext_code"`      // 扩展代码
	Parameters   map[string]string `json:"parameters"`    // 参数
	SmsBatchId   string            `json:"sms_batch_id"`  // SMS批次ID
	ExecuteTimes int32             `json:"execute_times"` // 执行次数
	ProviderType ProviderType      `json:"provider_type"` // 提供商类型
	Reports      []*DeliveryReport `json:"reports"`       // 投递报告
	AliMsgId     string            `json:"ali_msg_id"`    // 阿里云消息ID
	CTRReports   []*CTRReport      `json:"ctr_reports"`   // CTR报告
	CreateTime   time.Time         `json:"create_time"`   // 创建时间
	UpdateTime   time.Time         `json:"update_time"`   // 更新时间
}

// DeliveryReport 投递报告
type DeliveryReport struct {
	ID           string       `json:"id"`            // 报告ID
	PackId       string       `json:"pack_id"`       // 投递包ID
	Status       string       `json:"status"`        // 状态
	ErrorCode    string       `json:"error_code"`    // 错误代码
	ErrorMessage string       `json:"error_message"` // 错误消息
	ProviderType ProviderType `json:"provider_type"` // 提供商类型
	ReportTime   time.Time    `json:"report_time"`   // 报告时间
	CreateTime   time.Time    `json:"create_time"`   // 创建时间
}

// CTRReport CTR报告
type CTRReport struct {
	ID         string    `json:"id"`          // 报告ID
	PackId     string    `json:"pack_id"`     // 投递包ID
	ClickTime  time.Time `json:"click_time"`  // 点击时间
	UserAgent  string    `json:"user_agent"`  // 用户代理
	IPAddress  string    `json:"ip_address"`  // IP地址
	CreateTime time.Time `json:"create_time"` // 创建时间
}

// AddExecuteTimes 增加执行次数
func (p *SmsDeliveryPack) AddExecuteTimes(num int32) int32 {
	p.ExecuteTimes += num
	p.UpdateTime = time.Now()
	return p.ExecuteTimes
}

// SmsDeliveryPackRepo SMS投递包仓储接口
type SmsDeliveryPackRepo interface {
	// Save 保存投递包
	Save(ctx context.Context, pack *SmsDeliveryPack) error

	// Update 更新投递包
	Update(ctx context.Context, pack *SmsDeliveryPack) error

	// GetByID 根据ID获取投递包
	GetByID(ctx context.Context, id string) (*SmsDeliveryPack, error)

	// GetByBatchID 根据批次ID获取投递包列表
	GetByBatchID(ctx context.Context, batchId string) ([]*SmsDeliveryPack, error)

	// GetByBatchIDAndStatus 根据批次ID和状态获取投递包列表
	GetByBatchIDAndStatus(ctx context.Context, batchId string, status SmsBatchStatus) ([]*SmsDeliveryPack, error)

	// GetByPartitionKey 根据分区键获取投递包列表
	GetByPartitionKey(ctx context.Context, partitionKey string) ([]*SmsDeliveryPack, error)

	// BatchSave 批量保存投递包
	BatchSave(ctx context.Context, packs []*SmsDeliveryPack) error

	// BatchUpdate 批量更新投递包
	BatchUpdate(ctx context.Context, packs []*SmsDeliveryPack) error

	// Delete 删除投递包
	Delete(ctx context.Context, id string) error

	// Count 统计投递包数量
	Count(ctx context.Context, batchId string) (int64, error)

	// CountByStatus 根据状态统计投递包数量
	CountByStatus(ctx context.Context, batchId string, status SmsBatchStatus) (int64, error)
}

// DeliveryReportRepo 投递报告仓储接口
type DeliveryReportRepo interface {
	// Save 保存投递报告
	Save(ctx context.Context, report *DeliveryReport) error

	// GetByPackID 根据投递包ID获取报告列表
	GetByPackID(ctx context.Context, packId string) ([]*DeliveryReport, error)

	// BatchSave 批量保存投递报告
	BatchSave(ctx context.Context, reports []*DeliveryReport) error
}

// CTRReportRepo CTR报告仓储接口
type CTRReportRepo interface {
	// Save 保存CTR报告
	Save(ctx context.Context, report *CTRReport) error

	// GetByPackID 根据投递包ID获取CTR报告列表
	GetByPackID(ctx context.Context, packId string) ([]*CTRReport, error)

	// BatchSave 批量保存CTR报告
	BatchSave(ctx context.Context, reports []*CTRReport) error

	// GetStatsByBatchID 根据批次ID获取CTR统计
	GetStatsByBatchID(ctx context.Context, batchId string) (map[string]int64, error)
}
