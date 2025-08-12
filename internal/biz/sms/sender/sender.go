package sender

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ProviderType SMS提供商类型
type ProviderType string

const (
	ProviderAliyun ProviderType = "ALIYUN" // 阿里云SMS
	ProviderWE     ProviderType = "WE"     // WE SMS
	ProviderXSXX   ProviderType = "XSXX"   // XSXX SMS
)

// SendRequest SMS发送请求
type SendRequest struct {
	PhoneNumbers  []string          `json:"phone_numbers"`  // 手机号列表
	TemplateCode  string            `json:"template_code"`  // 模板代码
	TemplateParam map[string]string `json:"template_param"` // 模板参数
	SignName      string            `json:"sign_name"`      // 签名名称
	BatchId       string            `json:"batch_id"`       // 批次ID
}

// SendResponse SMS发送响应
type SendResponse struct {
	Success      bool                   `json:"success"`       // 是否成功
	RequestId    string                 `json:"request_id"`    // 请求ID
	BizId        string                 `json:"biz_id"`        // 业务ID
	Code         string                 `json:"code"`          // 响应码
	Message      string                 `json:"message"`       // 响应消息
	FailedPhones []string               `json:"failed_phones"` // 失败的手机号
	Details      map[string]interface{} `json:"details"`       // 详细信息
	SentTime     time.Time              `json:"sent_time"`     // 发送时间
}

// SmsSender SMS发送器接口
type SmsSender interface {
	// Send 发送SMS
	Send(ctx context.Context, req *SendRequest) (*SendResponse, error)

	// GetProviderType 获取提供商类型
	GetProviderType() ProviderType

	// ValidateConfig 验证配置
	ValidateConfig() error

	// GetMaxBatchSize 获取最大批次大小
	GetMaxBatchSize() int

	// IsHealthy 检查健康状态
	IsHealthy(ctx context.Context) bool
}

// SmsSenderConfig SMS发送器配置
type SmsSenderConfig struct {
	Provider     ProviderType      `yaml:"provider"`       // 提供商类型
	Enabled      bool              `yaml:"enabled"`        // 是否启用
	Timeout      time.Duration     `yaml:"timeout"`        // 超时时间
	RetryTimes   int               `yaml:"retry_times"`    // 重试次数
	MaxBatchSize int               `yaml:"max_batch_size"` // 最大批次大小
	Config       map[string]string `yaml:"config"`         // 具体配置
}

// SmsSenderFactory SMS发送器工厂接口
type SmsSenderFactory interface {
	// CreateSender 创建发送器
	CreateSender(providerType ProviderType, config *SmsSenderConfig) (SmsSender, error)

	// GetAvailableSenders 获取可用的发送器列表
	GetAvailableSenders() []ProviderType

	// GetSender 根据类型获取发送器
	GetSender(providerType ProviderType) (SmsSender, error)
}

// 常见错误定义
var (
	ErrInvalidProvider      = errors.New("invalid SMS provider type")
	ErrInvalidPhoneNumber   = errors.New("invalid phone number format")
	ErrEmptyTemplateCode    = errors.New("template code cannot be empty")
	ErrEmptySignName        = errors.New("sign name cannot be empty")
	ErrExceedMaxBatchSize   = errors.New("exceed maximum batch size")
	ErrProviderUnavailable  = errors.New("SMS provider is unavailable")
	ErrConfigurationMissing = errors.New("SMS provider configuration is missing")
)

// ValidateRequest 验证发送请求的有效性
func ValidateRequest(req *SendRequest) error {
	if req == nil {
		return errors.New("send request cannot be nil")
	}

	if len(req.PhoneNumbers) == 0 {
		return ErrInvalidPhoneNumber
	}

	if req.TemplateCode == "" {
		return ErrEmptyTemplateCode
	}

	if req.SignName == "" {
		return ErrEmptySignName
	}

	// 验证手机号格式（简单验证）
	for _, phone := range req.PhoneNumbers {
		if len(phone) < 11 || len(phone) > 15 {
			return ErrInvalidPhoneNumber
		}
	}

	return nil
}

// ValidateProviderType 验证提供商类型是否有效
func ValidateProviderType(providerType ProviderType) error {
	switch providerType {
	case ProviderAliyun, ProviderWE, ProviderXSXX:
		return nil
	default:
		return fmt.Errorf("invalid provider type: %s", providerType)
	}
}
