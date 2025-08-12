package sender

import (
	"fmt"
	"sync"
)

// DefaultSmsSenderFactory 默认SMS发送器工厂
type DefaultSmsSenderFactory struct {
	senders map[ProviderType]SmsSender
	configs map[ProviderType]*SmsSenderConfig
	mu      sync.RWMutex
}

// NewSmsSenderFactory 创建SMS发送器工厂
func NewSmsSenderFactory() SmsSenderFactory {
	return &DefaultSmsSenderFactory{
		senders: make(map[ProviderType]SmsSender),
		configs: make(map[ProviderType]*SmsSenderConfig),
	}
}

// CreateSender 创建发送器
func (f *DefaultSmsSenderFactory) CreateSender(providerType ProviderType, config *SmsSenderConfig) (SmsSender, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if !config.Enabled {
		return nil, fmt.Errorf("provider %s is disabled", providerType)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// 如果已存在，直接返回
	if sender, exists := f.senders[providerType]; exists {
		return sender, nil
	}

	var sender SmsSender
	var err error

	// 根据提供商类型创建对应的发送器
	switch providerType {
	case ProviderAliyun:
		sender, err = NewAliyunSmsSender(config)
	case ProviderWE:
		sender, err = NewWeSmsSender(config)
	case ProviderXSXX:
		sender, err = NewXsxxSmsSender(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create sender for %s: %w", providerType, err)
	}

	// 验证配置
	if err := sender.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid config for %s: %w", providerType, err)
	}

	// 缓存发送器和配置
	f.senders[providerType] = sender
	f.configs[providerType] = config

	return sender, nil
}

// GetAvailableSenders 获取可用的发送器列表
func (f *DefaultSmsSenderFactory) GetAvailableSenders() []ProviderType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var providers []ProviderType
	for providerType, config := range f.configs {
		if config.Enabled {
			providers = append(providers, providerType)
		}
	}

	return providers
}

// GetSender 根据类型获取发送器
func (f *DefaultSmsSenderFactory) GetSender(providerType ProviderType) (SmsSender, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	sender, exists := f.senders[providerType]
	if !exists {
		return nil, fmt.Errorf("sender not found for provider: %s", providerType)
	}

	config, configExists := f.configs[providerType]
	if !configExists || !config.Enabled {
		return nil, fmt.Errorf("provider %s is disabled", providerType)
	}

	return sender, nil
}

// RegisterSender 注册发送器
func (f *DefaultSmsSenderFactory) RegisterSender(providerType ProviderType, sender SmsSender, config *SmsSenderConfig) error {
	if sender == nil {
		return fmt.Errorf("sender is required")
	}
	if config == nil {
		return fmt.Errorf("config is required")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.senders[providerType] = sender
	f.configs[providerType] = config

	return nil
}

// RemoveSender 移除发送器
func (f *DefaultSmsSenderFactory) RemoveSender(providerType ProviderType) {
	f.mu.Lock()
	defer f.mu.Unlock()

	delete(f.senders, providerType)
	delete(f.configs, providerType)
}

// UpdateConfig 更新配置
func (f *DefaultSmsSenderFactory) UpdateConfig(providerType ProviderType, config *SmsSenderConfig) error {
	if config == nil {
		return fmt.Errorf("config is required")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// 更新配置
	f.configs[providerType] = config

	// 如果发送器已存在，需要重新创建
	if _, exists := f.senders[providerType]; exists {
		delete(f.senders, providerType)

		// 重新创建发送器
		sender, err := f.createSenderInternal(providerType, config)
		if err != nil {
			return fmt.Errorf("failed to recreate sender: %w", err)
		}

		f.senders[providerType] = sender
	}

	return nil
}

// createSenderInternal 内部创建发送器方法（不加锁）
func (f *DefaultSmsSenderFactory) createSenderInternal(providerType ProviderType, config *SmsSenderConfig) (SmsSender, error) {
	switch providerType {
	case ProviderAliyun:
		return NewAliyunSmsSender(config)
	case ProviderWE:
		return NewWeSmsSender(config)
	case ProviderXSXX:
		return NewXsxxSmsSender(config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// GetSenderStats 获取发送器统计信息
func (f *DefaultSmsSenderFactory) GetSenderStats() map[ProviderType]map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := make(map[ProviderType]map[string]interface{})

	for providerType, config := range f.configs {
		sender, senderExists := f.senders[providerType]

		stats[providerType] = map[string]interface{}{
			"enabled":        config.Enabled,
			"timeout":        config.Timeout,
			"retry_times":    config.RetryTimes,
			"max_batch_size": config.MaxBatchSize,
			"sender_exists":  senderExists,
		}

		if senderExists {
			stats[providerType]["max_batch_size_actual"] = sender.GetMaxBatchSize()
		}
	}

	return stats
}

// Shutdown 关闭工厂，清理资源
func (f *DefaultSmsSenderFactory) Shutdown() {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 清理所有发送器和配置
	f.senders = make(map[ProviderType]SmsSender)
	f.configs = make(map[ProviderType]*SmsSenderConfig)
}
