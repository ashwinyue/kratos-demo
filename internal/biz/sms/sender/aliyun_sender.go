package sender

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// AliyunSmsSender 阿里云SMS发送器
type AliyunSmsSender struct {
	accessKeyId     string
	accessKeySecret string
	signName        string
	endpoint        string
	client          *resty.Client
	config          *SmsSenderConfig
}

// AliyunSmsConfig 阿里云SMS配置
type AliyunSmsConfig struct {
	AccessKeyId     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	SignName        string `yaml:"sign_name"`
	Endpoint        string `yaml:"endpoint"`
}

// AliyunSmsRequest 阿里云SMS请求
type AliyunSmsRequest struct {
	PhoneNumbers  string `json:"PhoneNumbers"`
	SignName      string `json:"SignName"`
	TemplateCode  string `json:"TemplateCode"`
	TemplateParam string `json:"TemplateParam,omitempty"`
}

// AliyunSmsResponse 阿里云SMS响应
type AliyunSmsResponse struct {
	RequestId string `json:"RequestId"`
	BizId     string `json:"BizId"`
	Code      string `json:"Code"`
	Message   string `json:"Message"`
}

// NewAliyunSmsSender 创建阿里云SMS发送器
func NewAliyunSmsSender(config *SmsSenderConfig) (*AliyunSmsSender, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// 解析阿里云特定配置
	accessKeyId := config.Config["access_key_id"]
	accessKeySecret := config.Config["access_key_secret"]
	signName := config.Config["sign_name"]
	endpoint := config.Config["endpoint"]

	if endpoint == "" {
		endpoint = "https://dysmsapi.aliyuncs.com"
	}

	if accessKeyId == "" || accessKeySecret == "" {
		return nil, fmt.Errorf("access_key_id and access_key_secret are required")
	}

	// 创建HTTP客户端
	client := resty.New()
	client.SetTimeout(config.Timeout)
	client.SetRetryCount(config.RetryTimes)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)

	return &AliyunSmsSender{
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
		signName:        signName,
		endpoint:        endpoint,
		client:          client,
		config:          config,
	}, nil
}

// Send 发送SMS
func (s *AliyunSmsSender) Send(ctx context.Context, req *SendRequest) (*SendResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	if len(req.PhoneNumbers) == 0 {
		return nil, fmt.Errorf("phone numbers are required")
	}

	if len(req.PhoneNumbers) > s.GetMaxBatchSize() {
		return nil, fmt.Errorf("phone numbers exceed max batch size: %d", s.GetMaxBatchSize())
	}

	// 构建请求参数
	templateParam := ""
	if len(req.TemplateParam) > 0 {
		paramBytes, err := json.Marshal(req.TemplateParam)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal template param: %w", err)
		}
		templateParam = string(paramBytes)
	}

	signName := req.SignName
	if signName == "" {
		signName = s.signName
	}

	aliyunReq := &AliyunSmsRequest{
		PhoneNumbers:  strings.Join(req.PhoneNumbers, ","),
		SignName:      signName,
		TemplateCode:  req.TemplateCode,
		TemplateParam: templateParam,
	}

	// 发送请求
	var aliyunResp AliyunSmsResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(aliyunReq).
		SetResult(&aliyunResp).
		Post(s.endpoint + "/SendSms")

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// 构建响应
	sendResp := &SendResponse{
		Success:      aliyunResp.Code == "OK",
		RequestId:    aliyunResp.RequestId,
		BizId:        aliyunResp.BizId,
		Code:         aliyunResp.Code,
		Message:      aliyunResp.Message,
		FailedPhones: []string{},
		Details: map[string]interface{}{
			"provider":     "aliyun",
			"status_code":  resp.StatusCode(),
			"response_time": resp.Time(),
		},
		SentTime: time.Now(),
	}

	// 如果发送失败，记录失败的手机号
	if !sendResp.Success {
		sendResp.FailedPhones = req.PhoneNumbers
	}

	return sendResp, nil
}

// GetProviderType 获取提供商类型
func (s *AliyunSmsSender) GetProviderType() ProviderType {
	return ProviderAliyun
}

// ValidateConfig 验证配置
func (s *AliyunSmsSender) ValidateConfig() error {
	if s.accessKeyId == "" {
		return fmt.Errorf("access_key_id is required")
	}
	if s.accessKeySecret == "" {
		return fmt.Errorf("access_key_secret is required")
	}
	if s.endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}
	return nil
}

// GetMaxBatchSize 获取最大批次大小
func (s *AliyunSmsSender) GetMaxBatchSize() int {
	if s.config != nil && s.config.MaxBatchSize > 0 {
		return s.config.MaxBatchSize
	}
	return 1000 // 阿里云默认最大批次大小
}

// IsHealthy 检查健康状态
func (s *AliyunSmsSender) IsHealthy(ctx context.Context) bool {
	// 简单的健康检查，可以发送一个测试请求
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.client.R().
		SetContext(ctx).
		Get(s.endpoint)

	return err == nil
}