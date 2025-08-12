package sender

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// WeSmsSender WE SMS发送器
type WeSmsSender struct {
	apiUrl    string
	apiKey    string
	apiSecret string
	client    *resty.Client
	config    *SmsSenderConfig
}

// WeSmsRequest WE SMS请求
type WeSmsRequest struct {
	ApiKey       string            `json:"api_key"`
	Phones       []string          `json:"phones"`
	Content      string            `json:"content"`
	TemplateId   string            `json:"template_id,omitempty"`
	TemplateVars map[string]string `json:"template_vars,omitempty"`
	Signature    string            `json:"signature,omitempty"`
	BatchId      string            `json:"batch_id,omitempty"`
}

// WeSmsResponse WE SMS响应
type WeSmsResponse struct {
	Code      int                    `json:"code"`
	Message   string                 `json:"message"`
	RequestId string                 `json:"request_id"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Failed    []string               `json:"failed,omitempty"`
}

// NewWeSmsSender 创建WE SMS发送器
func NewWeSmsSender(config *SmsSenderConfig) (*WeSmsSender, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// 解析WE特定配置
	apiUrl := config.Config["api_url"]
	apiKey := config.Config["api_key"]
	apiSecret := config.Config["api_secret"]

	if apiUrl == "" {
		apiUrl = "https://api.we.com"
	}

	if apiKey == "" || apiSecret == "" {
		return nil, fmt.Errorf("api_key and api_secret are required")
	}

	// 创建HTTP客户端
	client := resty.New()
	client.SetTimeout(config.Timeout)
	client.SetRetryCount(config.RetryTimes)
	client.SetRetryWaitTime(1 * time.Second)
	client.SetRetryMaxWaitTime(5 * time.Second)

	// 设置通用请求头
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("User-Agent", "kratos-sms-client/1.0")

	return &WeSmsSender{
		apiUrl:    apiUrl,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client:    client,
		config:    config,
	}, nil
}

// Send 发送SMS
func (s *WeSmsSender) Send(ctx context.Context, req *SendRequest) (*SendResponse, error) {
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
	weReq := &WeSmsRequest{
		ApiKey:       s.apiKey,
		Phones:       req.PhoneNumbers,
		TemplateId:   req.TemplateCode,
		TemplateVars: req.TemplateParam,
		Signature:    req.SignName,
		BatchId:      req.BatchId,
	}

	// 如果没有模板参数，构建纯文本内容
	if req.TemplateCode == "" && len(req.TemplateParam) > 0 {
		content := s.buildContentFromParams(req.TemplateParam)
		weReq.Content = content
	}

	// 发送请求
	var weResp WeSmsResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(weReq).
		SetResult(&weResp).
		Post(s.apiUrl + "/sms/send")

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// 构建响应
	sendResp := &SendResponse{
		Success:      weResp.Code == 200,
		RequestId:    weResp.RequestId,
		BizId:        fmt.Sprintf("we_%s", weResp.RequestId),
		Code:         fmt.Sprintf("%d", weResp.Code),
		Message:      weResp.Message,
		FailedPhones: weResp.Failed,
		Details: map[string]interface{}{
			"provider":      "we",
			"status_code":   resp.StatusCode(),
			"response_time": resp.Time(),
			"data":          weResp.Data,
		},
		SentTime: time.Now(),
	}

	// 如果没有明确的失败列表，且发送失败，则认为所有号码都失败
	if !sendResp.Success && len(sendResp.FailedPhones) == 0 {
		sendResp.FailedPhones = req.PhoneNumbers
	}

	return sendResp, nil
}

// buildContentFromParams 从模板参数构建内容
func (s *WeSmsSender) buildContentFromParams(params map[string]string) string {
	var parts []string
	for key, value := range params {
		parts = append(parts, fmt.Sprintf("%s: %s", key, value))
	}
	return strings.Join(parts, ", ")
}

// GetProviderType 获取提供商类型
func (s *WeSmsSender) GetProviderType() ProviderType {
	return ProviderWE
}

// ValidateConfig 验证配置
func (s *WeSmsSender) ValidateConfig() error {
	if s.apiKey == "" {
		return fmt.Errorf("api_key is required")
	}
	if s.apiSecret == "" {
		return fmt.Errorf("api_secret is required")
	}
	if s.apiUrl == "" {
		return fmt.Errorf("api_url is required")
	}
	return nil
}

// GetMaxBatchSize 获取最大批次大小
func (s *WeSmsSender) GetMaxBatchSize() int {
	if s.config != nil && s.config.MaxBatchSize > 0 {
		return s.config.MaxBatchSize
	}
	return 500 // WE默认最大批次大小
}

// IsHealthy 检查健康状态
func (s *WeSmsSender) IsHealthy(ctx context.Context) bool {
	// 健康检查
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.client.R().
		SetContext(ctx).
		Get(s.apiUrl + "/health")

	return err == nil
}
