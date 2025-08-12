package sender

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// SmsSenderConfigManager SMS发送器配置管理器接口
type SmsSenderConfigManager interface {
	// GetConfig 获取指定提供商的配置
	GetConfig(providerType ProviderType) (*SmsSenderConfig, error)

	// GetAllConfigs 获取所有提供商的配置
	GetAllConfigs() map[ProviderType]*SmsSenderConfig

	// ValidateConfig 验证指定提供商的配置
	ValidateConfig(providerType ProviderType) error

	// ValidateAllConfigs 验证所有配置
	ValidateAllConfigs() map[ProviderType]error

	// GetEnabledProviders 获取所有启用的提供商
	GetEnabledProviders() []ProviderType

	// IsProviderEnabled 检查指定提供商是否启用
	IsProviderEnabled(providerType ProviderType) bool

	// UpdateConfig 更新指定提供商的配置
	UpdateConfig(providerType ProviderType, config *SmsSenderConfig) error
}

// DefaultSmsSenderConfigManager 默认SMS发送器配置管理器实现
type DefaultSmsSenderConfigManager struct {
	configs map[ProviderType]*SmsSenderConfig
	logger  *log.Helper
}

// NewDefaultSmsSenderConfigManager 创建默认SMS发送器配置管理器
func NewDefaultSmsSenderConfigManager(configs map[ProviderType]*SmsSenderConfig, logger log.Logger) SmsSenderConfigManager {
	if configs == nil {
		configs = make(map[ProviderType]*SmsSenderConfig)
	}

	return &DefaultSmsSenderConfigManager{
		configs: configs,
		logger:  log.NewHelper(logger),
	}
}

// GetConfig 获取指定提供商的配置
func (m *DefaultSmsSenderConfigManager) GetConfig(providerType ProviderType) (*SmsSenderConfig, error) {
	config, exists := m.configs[providerType]
	if !exists {
		return nil, fmt.Errorf("config not found for provider: %s", providerType)
	}

	// 返回配置的副本，避免外部修改
	return m.copyConfig(config), nil
}

// GetAllConfigs 获取所有提供商的配置
func (m *DefaultSmsSenderConfigManager) GetAllConfigs() map[ProviderType]*SmsSenderConfig {
	allConfigs := make(map[ProviderType]*SmsSenderConfig)

	for providerType, config := range m.configs {
		allConfigs[providerType] = m.copyConfig(config)
	}

	return allConfigs
}

// ValidateConfig 验证指定提供商的配置
func (m *DefaultSmsSenderConfigManager) ValidateConfig(providerType ProviderType) error {
	config, exists := m.configs[providerType]
	if !exists {
		return fmt.Errorf("config not found for provider: %s", providerType)
	}

	return m.validateSingleConfig(providerType, config)
}

// ValidateAllConfigs 验证所有配置
func (m *DefaultSmsSenderConfigManager) ValidateAllConfigs() map[ProviderType]error {
	validationErrors := make(map[ProviderType]error)

	for providerType, config := range m.configs {
		if err := m.validateSingleConfig(providerType, config); err != nil {
			validationErrors[providerType] = err
		}
	}

	return validationErrors
}

// GetEnabledProviders 获取所有启用的提供商
func (m *DefaultSmsSenderConfigManager) GetEnabledProviders() []ProviderType {
	enabledProviders := make([]ProviderType, 0)

	for providerType, config := range m.configs {
		if config.Enabled {
			enabledProviders = append(enabledProviders, providerType)
		}
	}

	return enabledProviders
}

// IsProviderEnabled 检查指定提供商是否启用
func (m *DefaultSmsSenderConfigManager) IsProviderEnabled(providerType ProviderType) bool {
	config, exists := m.configs[providerType]
	if !exists {
		return false
	}

	return config.Enabled
}

// UpdateConfig 更新指定提供商的配置
func (m *DefaultSmsSenderConfigManager) UpdateConfig(providerType ProviderType, config *SmsSenderConfig) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}

	// 验证新配置
	if err := m.validateSingleConfig(providerType, config); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	// 更新配置
	m.configs[providerType] = m.copyConfig(config)
	m.logger.Infof("config updated for provider: %s", providerType)

	return nil
}

// validateSingleConfig 验证单个配置
func (m *DefaultSmsSenderConfigManager) validateSingleConfig(providerType ProviderType, config *SmsSenderConfig) error {
	if config == nil {
		return errors.New("config cannot be nil")
	}

	// 验证提供商类型
	if config.Provider != providerType {
		return fmt.Errorf("provider type mismatch: expected %s, got %s", providerType, config.Provider)
	}

	// 验证基础配置
	if config.MaxBatchSize <= 0 {
		return errors.New("max_batch_size must be greater than 0")
	}

	if config.RetryTimes < 0 {
		return errors.New("retry_times cannot be negative")
	}

	if config.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}

	// 根据提供商类型验证特定配置
	switch providerType {
	case ProviderAliyun:
		return m.validateAliyunConfig(config)
	case ProviderWE:
		return m.validateWeConfig(config)
	case ProviderXSXX:
		return m.validateXsxxConfig(config)
	default:
		return fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// validateAliyunConfig 验证阿里云配置
func (m *DefaultSmsSenderConfigManager) validateAliyunConfig(config *SmsSenderConfig) error {
	requiredFields := []string{"access_key_id", "access_key_secret", "sign_name"}

	for _, field := range requiredFields {
		if value, exists := config.Config[field]; !exists || value == "" {
			return fmt.Errorf("aliyun config missing required field: %s", field)
		}
	}

	return nil
}

// validateWeConfig 验证WE配置
func (m *DefaultSmsSenderConfigManager) validateWeConfig(config *SmsSenderConfig) error {
	requiredFields := []string{"api_url", "api_key", "api_secret", "sign_name"}

	for _, field := range requiredFields {
		if value, exists := config.Config[field]; !exists || value == "" {
			return fmt.Errorf("WE config missing required field: %s", field)
		}
	}

	return nil
}

// validateXsxxConfig 验证XSXX配置
func (m *DefaultSmsSenderConfigManager) validateXsxxConfig(config *SmsSenderConfig) error {
	requiredFields := []string{"api_url", "username", "password", "sign_name"}

	for _, field := range requiredFields {
		if value, exists := config.Config[field]; !exists || value == "" {
			return fmt.Errorf("XSXX config missing required field: %s", field)
		}
	}

	return nil
}

// copyConfig 复制配置对象
func (m *DefaultSmsSenderConfigManager) copyConfig(config *SmsSenderConfig) *SmsSenderConfig {
	if config == nil {
		return nil
	}

	// 复制配置映射
	configCopy := make(map[string]string)
	for key, value := range config.Config {
		configCopy[key] = value
	}

	return &SmsSenderConfig{
		Provider:     config.Provider,
		Enabled:      config.Enabled,
		Timeout:      config.Timeout,
		RetryTimes:   config.RetryTimes,
		MaxBatchSize: config.MaxBatchSize,
		Config:       configCopy,
	}
}

// CreateDefaultConfigs 创建默认配置
func CreateDefaultConfigs() map[ProviderType]*SmsSenderConfig {
	configs := make(map[ProviderType]*SmsSenderConfig)

	// 阿里云默认配置
	configs[ProviderAliyun] = &SmsSenderConfig{
		Provider:     ProviderAliyun,
		Enabled:      false, // 默认禁用，需要手动配置后启用
		Timeout:      30 * time.Second,
		RetryTimes:   3,
		MaxBatchSize: 1000,
		Config: map[string]string{
			"access_key_id":     "",
			"access_key_secret": "",
			"sign_name":         "",
			"template_code":     "",
			"region":            "cn-hangzhou",
			"endpoint":          "dysmsapi.aliyuncs.com",
		},
	}

	// WE默认配置
	configs[ProviderWE] = &SmsSenderConfig{
		Provider:     ProviderWE,
		Enabled:      false,
		Timeout:      30 * time.Second,
		RetryTimes:   3,
		MaxBatchSize: 500,
		Config: map[string]string{
			"api_url":    "https://api.we.com/sms/send",
			"api_key":    "",
			"api_secret": "",
			"sign_name":  "",
		},
	}

	// XSXX默认配置
	configs[ProviderXSXX] = &SmsSenderConfig{
		Provider:     ProviderXSXX,
		Enabled:      false,
		Timeout:      30 * time.Second,
		RetryTimes:   3,
		MaxBatchSize: 200,
		Config: map[string]string{
			"api_url":   "https://api.xsxx.com/sms/send",
			"username":  "",
			"password":  "",
			"sign_name": "",
		},
	}

	return configs
}

// LoadConfigsFromYaml 从YAML配置加载SMS发送器配置
// 这个函数需要在实际项目中根据具体的配置结构实现
func LoadConfigsFromYaml(yamlConfig interface{}) (map[ProviderType]*SmsSenderConfig, error) {
	// 这里是示例实现，实际项目中需要根据具体的YAML结构解析
	configs := CreateDefaultConfigs()

	// TODO: 实现从YAML配置解析的逻辑
	// 例如：
	// if smsConfig, ok := yamlConfig.(*SmsConfig); ok {
	//     // 解析并更新configs
	// }

	return configs, nil
}
