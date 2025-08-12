package sender

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// XsxxSmsSender XSXX SMS发送器
type XsxxSmsSender struct {
	apiUrl    string
	appId     string
	appSecret string
	client    *resty.Client
	config    *SmsSenderConfig
}

// XsxxSmsRequest XSXX SMS请求
type XsxxSmsRequest struct {
	AppId      string `json:"app_id"`
	Timestamp  string `json:"timestamp"`
	Sign       string `json:"sign"`
	Mobiles    string `json:"mobiles"`
	Content    string `json:"content"`
	TemplateId string `json:"template_id,omitempty"`
	Params     string `json:"params,omitempty"`
	BatchNo    string `json:"batch_no,omitempty"`
}

// XsxxSmsResponse XSXX SMS响应
type XsxxSmsResponse struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	MsgId     string                 `json:"msg_id"`
	Timestamp string                 `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
	FailList  []string               `json:"fail_list,omitempty"`
}

// NewXsxxSmsSender 创建XSXX SMS发送器
func NewXsxxSmsSender(config *SmsSenderConfig) (*XsxxSmsSender, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	// 解析XSXX特定配置
	apiUrl := config.Config["api_url"]
	appId := config.Config["app_id"]
	appSecret := config.Config["app_secret"]

	if apiUrl == "" {
		apiUrl = "https://api.xsxx.com"
	}

	if appId == "" || appSecret == "" {
		return nil, fmt.Errorf("app_id and app_secret are required")
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

	return &XsxxSmsSender{
		apiUrl:    apiUrl,
		appId:     appId,
		appSecret: appSecret,
		client:    client,
		config:    config,
	}, nil
}

// Send 发送SMS
func (s *XsxxSmsSender) Send(ctx context.Context, req *SendRequest) (*SendResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}

	if len(req.PhoneNumbers) == 0 {
		return nil, fmt.Errorf("phone numbers are required")
	}

	if len(req.PhoneNumbers) > s.GetMaxBatchSize() {
		return nil, fmt.Errorf("phone numbers exceed max batch size: %d", s.GetMaxBatchSize())
	}

	// 生成时间戳
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	// 构建请求参数
	xsxxReq := &XsxxSmsRequest{
		AppId:      s.appId,
		Timestamp:  timestamp,
		Mobiles:    strings.Join(req.PhoneNumbers, ","),
		TemplateId: req.TemplateCode,
		BatchNo:    req.BatchId,
	}

	// 处理模板参数
	if len(req.TemplateParam) > 0 {
		params := s.buildParamsString(req.TemplateParam)
		xsxxReq.Params = params
		xsxxReq.Content = s.buildContentFromTemplate(req.TemplateCode, req.TemplateParam)
	}

	// 生成签名
	sign := s.generateSign(xsxxReq)
	xsxxReq.Sign = sign

	// 发送请求
	var xsxxResp XsxxSmsResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetBody(xsxxReq).
		SetResult(&xsxxResp).
		Post(s.apiUrl + "/v1/sms/send")

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// 构建响应
	sendResp := &SendResponse{
		Success:      xsxxResp.Code == "0000",
		RequestId:    xsxxResp.MsgId,
		BizId:        fmt.Sprintf("xsxx_%s", xsxxResp.MsgId),
		Code:         xsxxResp.Code,
		Message:      xsxxResp.Message,
		FailedPhones: xsxxResp.FailList,
		Details: map[string]interface{}{
			"provider":      "xsxx",
			"status_code":   resp.StatusCode(),
			"response_time": resp.Time(),
			"timestamp":     xsxxResp.Timestamp,
			"data":          xsxxResp.Data,
		},
		SentTime: time.Now(),
	}

	// 如果没有明确的失败列表，且发送失败，则认为所有号码都失败
	if !sendResp.Success && len(sendResp.FailedPhones) == 0 {
		sendResp.FailedPhones = req.PhoneNumbers
	}

	return sendResp, nil
}

// buildParamsString 构建参数字符串
func (s *XsxxSmsSender) buildParamsString(params map[string]string) string {
	var parts []string
	for key, value := range params {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return strings.Join(parts, "&")
}

// buildContentFromTemplate 从模板构建内容
func (s *XsxxSmsSender) buildContentFromTemplate(templateCode string, params map[string]string) string {
	content := templateCode
	for key, value := range params {
		placeholder := fmt.Sprintf("${%s}", key)
		content = strings.ReplaceAll(content, placeholder, value)
	}
	return content
}

// generateSign 生成签名
func (s *XsxxSmsSender) generateSign(req *XsxxSmsRequest) string {
	// 构建签名字符串
	params := map[string]string{
		"app_id":    req.AppId,
		"timestamp": req.Timestamp,
		"mobiles":   req.Mobiles,
	}

	if req.Content != "" {
		params["content"] = req.Content
	}
	if req.TemplateId != "" {
		params["template_id"] = req.TemplateId
	}
	if req.Params != "" {
		params["params"] = req.Params
	}

	// 按key排序
	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建签名字符串
	var signParts []string
	for _, k := range keys {
		signParts = append(signParts, fmt.Sprintf("%s=%s", k, params[k]))
	}
	signStr := strings.Join(signParts, "&") + "&app_secret=" + s.appSecret

	// MD5签名
	hash := md5.Sum([]byte(signStr))
	return fmt.Sprintf("%x", hash)
}

// GetProviderType 获取提供商类型
func (s *XsxxSmsSender) GetProviderType() ProviderType {
	return ProviderXSXX
}

// ValidateConfig 验证配置
func (s *XsxxSmsSender) ValidateConfig() error {
	if s.appId == "" {
		return fmt.Errorf("app_id is required")
	}
	if s.appSecret == "" {
		return fmt.Errorf("app_secret is required")
	}
	if s.apiUrl == "" {
		return fmt.Errorf("api_url is required")
	}
	return nil
}

// GetMaxBatchSize 获取最大批次大小
func (s *XsxxSmsSender) GetMaxBatchSize() int {
	if s.config != nil && s.config.MaxBatchSize > 0 {
		return s.config.MaxBatchSize
	}
	return 200 // XSXX默认最大批次大小
}

// IsHealthy 检查健康状态
func (s *XsxxSmsSender) IsHealthy(ctx context.Context) bool {
	// 健康检查
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := s.client.R().
		SetContext(ctx).
		Get(s.apiUrl + "/health")

	return err == nil
}
