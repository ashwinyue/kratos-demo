package sync

import (
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

// SyncerFactoryImpl 同步器工厂实现
type SyncerFactoryImpl struct {
	mu       sync.RWMutex
	syncers  map[SyncType]Syncer      // 同步器实例缓存
	configs  map[SyncType]*SyncConfig // 同步器配置
	logger   log.Logger
	dataRepo DataRepo
	mqClient MessageQueueClient
}

// NewSyncerFactory 创建同步器工厂
func NewSyncerFactory(logger log.Logger, dataRepo DataRepo, mqClient MessageQueueClient) SyncerFactory {
	return &SyncerFactoryImpl{
		syncers:  make(map[SyncType]Syncer),
		configs:  make(map[SyncType]*SyncConfig),
		logger:   logger,
		dataRepo: dataRepo,
		mqClient: mqClient,
	}
}

// CreateSyncer 创建同步器实例（实现SyncerFactory接口）
func (f *SyncerFactoryImpl) CreateSyncer(syncType SyncType) (Syncer, error) {
	return f.GetSyncer(syncType)
}

// ListAvailableSyncers 列出所有可用的同步器类型
func (f *SyncerFactoryImpl) ListAvailableSyncers() []SyncType {
	return f.GetRegisteredTypes()
}

// RegisterSyncer 注册同步器配置
func (f *SyncerFactoryImpl) RegisterSyncer(syncType SyncType, config *SyncConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if config == nil {
		return fmt.Errorf("config cannot be nil for sync type: %s", syncType)
	}

	// 验证配置
	if err := f.validateConfig(config); err != nil {
		return fmt.Errorf("invalid config for sync type %s: %w", syncType, err)
	}

	f.configs[syncType] = config
	log.Infof("Registered syncer config for type: %s", syncType)
	return nil
}

// GetSyncer 获取同步器实例
func (f *SyncerFactoryImpl) GetSyncer(syncType SyncType) (Syncer, error) {
	f.mu.RLock()
	syncer, exists := f.syncers[syncType]
	config, configExists := f.configs[syncType]
	f.mu.RUnlock()

	// 如果同步器已存在，直接返回
	if exists {
		return syncer, nil
	}

	// 如果配置不存在，返回错误
	if !configExists {
		return nil, fmt.Errorf("no config registered for sync type: %s", syncType)
	}

	// 创建新的同步器实例
	f.mu.Lock()
	defer f.mu.Unlock()

	// 双重检查，防止并发创建
	if syncer, exists := f.syncers[syncType]; exists {
		return syncer, nil
	}

	// 根据类型创建对应的同步器
	newSyncer, err := f.createSyncer(syncType, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create syncer for type %s: %w", syncType, err)
	}

	f.syncers[syncType] = newSyncer
	log.Infof("Created syncer instance for type: %s", syncType)
	return newSyncer, nil
}

// GetAllSyncers 获取所有已注册的同步器
func (f *SyncerFactoryImpl) GetAllSyncers() map[SyncType]Syncer {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make(map[SyncType]Syncer)
	for syncType, syncer := range f.syncers {
		result[syncType] = syncer
	}
	return result
}

// GetRegisteredTypes 获取所有已注册的同步器类型
func (f *SyncerFactoryImpl) GetRegisteredTypes() []SyncType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]SyncType, 0, len(f.configs))
	for syncType := range f.configs {
		types = append(types, syncType)
	}
	return types
}

// createSyncer 根据类型创建同步器实例
func (f *SyncerFactoryImpl) createSyncer(syncType SyncType, config *SyncConfig) (Syncer, error) {
	switch syncType {
	case SyncTypePrepare:
		return NewPrepareSyncerImpl(config, f.logger, f.dataRepo, f.mqClient), nil
	case SyncTypeDelivery:
		return NewDeliverySyncerImpl(config, f.logger, f.dataRepo, f.mqClient), nil
	case SyncTypeSave:
		return NewSaveSyncerImpl(config, f.logger, f.dataRepo, f.mqClient), nil
	default:
		return nil, fmt.Errorf("unsupported sync type: %s", syncType)
	}
}

// validateConfig 验证同步器配置
func (f *SyncerFactoryImpl) validateConfig(config *SyncConfig) error {
	if config.MaxRetries < 0 {
		return fmt.Errorf("max_retries cannot be negative")
	}

	if config.RetryInterval <= 0 {
		return fmt.Errorf("retry_interval must be positive")
	}

	if config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	if config.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be positive")
	}

	return nil
}

// UpdateConfig 更新同步器配置
func (f *SyncerFactoryImpl) UpdateConfig(syncType SyncType, config *SyncConfig) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// 验证配置
	if err := f.validateConfig(config); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// 更新配置
	f.configs[syncType] = config

	// 如果同步器实例已存在，需要重新创建
	if _, exists := f.syncers[syncType]; exists {
		newSyncer, err := f.createSyncer(syncType, config)
		if err != nil {
			return fmt.Errorf("failed to recreate syncer: %w", err)
		}
		f.syncers[syncType] = newSyncer
		log.Infof("Recreated syncer instance for type: %s with new config", syncType)
	}

	log.Infof("Updated config for sync type: %s", syncType)
	return nil
}

// RemoveSyncer 移除同步器
func (f *SyncerFactoryImpl) RemoveSyncer(syncType SyncType) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 移除同步器实例
	delete(f.syncers, syncType)
	// 移除配置
	delete(f.configs, syncType)

	log.Infof("Removed syncer for type: %s", syncType)
	return nil
}

// IsHealthy 检查工厂健康状态
func (f *SyncerFactoryImpl) IsHealthy() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// 检查数据仓库健康状态
	if f.dataRepo != nil {
		if !f.dataRepo.IsHealthy(nil) {
			log.Warn("Data repository is not healthy")
			return false
		}
	}

	// 检查消息队列健康状态
	if f.mqClient != nil {
		if !f.mqClient.IsHealthy(nil) {
			log.Warn("Message queue client is not healthy")
			return false
		}
	}

	return true
}

// GetStats 获取工厂统计信息
func (f *SyncerFactoryImpl) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := map[string]interface{}{
		"total_registered_types": len(f.configs),
		"total_active_syncers":   len(f.syncers),
		"registered_types":       f.GetRegisteredTypes(),
	}

	// 添加每个同步器的健康状态
	healthStatus := make(map[string]bool)
	for syncType, syncer := range f.syncers {
		healthStatus[string(syncType)] = syncer.IsHealthy(nil)
	}
	stats["health_status"] = healthStatus

	return stats
}

// Shutdown 关闭工厂，清理资源
func (f *SyncerFactoryImpl) Shutdown() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// 清理所有同步器实例
	f.syncers = make(map[SyncType]Syncer)
	f.configs = make(map[SyncType]*SyncConfig)

	log.Info("Syncer factory shutdown completed")
	return nil
}
