package data

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-kratos/kratos/v2/log"

	"kratos-demo/internal/biz/sync"
)

// RedisClient Redis客户端接口
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
}

// redisDeliverySyncer Redis投递同步器实现
type redisDeliverySyncer struct {
	redisClient RedisClient
	logger      *log.Helper
}

// NewRedisDeliverySyncer 创建Redis投递同步器
func NewRedisDeliverySyncer(redisClient RedisClient, logger log.Logger) sync.DeliverySyncer {
	return &redisDeliverySyncer{
		redisClient: redisClient,
		logger:      log.NewHelper(logger),
	}
}

// InitSyncer 初始化同步器
func (r *redisDeliverySyncer) InitSyncer(ctx context.Context, batchId string) error {
	// 初始化Redis中的计数器
	keys := []string{
		fmt.Sprintf("sms:delivery:total:%s", batchId),
		fmt.Sprintf("sms:delivery:success:%s", batchId),
		fmt.Sprintf("sms:delivery:failure:%s", batchId),
	}

	for _, key := range keys {
		if err := r.redisClient.Set(ctx, key, 0, 24*time.Hour).Err(); err != nil {
			return fmt.Errorf("failed to init redis key %s: %w", key, err)
		}
	}

	return nil
}

// Sync2Statistics 同步计数器到本地统计
func (r *redisDeliverySyncer) Sync2Statistics(ctx context.Context, batchId string) (*sync.StepStatistics, error) {
	totalKey := fmt.Sprintf("sms:delivery:total:%s", batchId)
	successKey := fmt.Sprintf("sms:delivery:success:%s", batchId)
	failureKey := fmt.Sprintf("sms:delivery:failure:%s", batchId)

	total, err := r.redisClient.Get(ctx, totalKey).Int64()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	success, err := r.redisClient.Get(ctx, successKey).Int64()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get success count: %w", err)
	}

	failure, err := r.redisClient.Get(ctx, failureKey).Int64()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get failure count: %w", err)
	}

	return &sync.StepStatistics{
		Total:   total,
		Success: success,
		Failure: failure,
	}, nil
}

// SyncStatistics2Counter 同步本地统计到计数器
func (r *redisDeliverySyncer) SyncStatistics2Counter(ctx context.Context, batchId string, stats *sync.StepStatistics) error {
	totalKey := fmt.Sprintf("sms:delivery:total:%s", batchId)
	successKey := fmt.Sprintf("sms:delivery:success:%s", batchId)
	failureKey := fmt.Sprintf("sms:delivery:failure:%s", batchId)

	if err := r.redisClient.Set(ctx, totalKey, stats.Total, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to set total count: %w", err)
	}

	if err := r.redisClient.Set(ctx, successKey, stats.Success, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to set success count: %w", err)
	}

	if err := r.redisClient.Set(ctx, failureKey, stats.Failure, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to set failure count: %w", err)
	}

	return nil
}

// GetStatistics 获取统计信息
func (r *redisDeliverySyncer) GetStatistics(ctx context.Context, batchId string) (*sync.StepStatistics, error) {
	return r.Sync2Statistics(ctx, batchId)
}

// SetHeartbeatTime 设置心跳时间
func (r *redisDeliverySyncer) SetHeartbeatTime(ctx context.Context, batchId string, heartbeatTime time.Time) error {
	key := fmt.Sprintf("sms:delivery:heartbeat:%s", batchId)
	timestamp := heartbeatTime.Unix()
	return r.redisClient.Set(ctx, key, timestamp, 24*time.Hour).Err()
}

// GetHeartbeatTime 获取心跳时间
func (r *redisDeliverySyncer) GetHeartbeatTime(ctx context.Context, batchId string) (time.Time, error) {
	key := fmt.Sprintf("sms:delivery:heartbeat:%s", batchId)
	timestampStr, err := r.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to get heartbeat time: %w", err)
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return time.Unix(timestamp, 0), nil
}

// MarkPKTaskComplete 标记分区任务完成
func (r *redisDeliverySyncer) MarkPKTaskComplete(ctx context.Context, batchId, partitionKey string) error {
	key := fmt.Sprintf("sms:delivery:pk_complete:%s:%s", batchId, partitionKey)
	return r.redisClient.Set(ctx, key, 1, 24*time.Hour).Err()
}

// AllPkTasksCompleted 检查所有PK任务是否完成
func (r *redisDeliverySyncer) AllPkTasksCompleted(ctx context.Context, batchId string) (bool, error) {
	// 简化实现：检查是否有完成标记
	// 在实际项目中，可以通过扫描所有分区键来实现
	key := fmt.Sprintf("sms:delivery:all_pk_complete:%s", batchId)
	exists, err := r.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check all pk tasks completion: %w", err)
	}
	return exists > 0, nil
}

// redisPrepareSyncer Redis准备同步器实现
type redisPrepareSyncer struct {
	redisClient RedisClient
	logger      *log.Helper
}

// NewRedisPrepareSyncer 创建Redis准备同步器
func NewRedisPrepareSyncer(redisClient RedisClient, logger log.Logger) sync.PrepareSyncer {
	return &redisPrepareSyncer{
		redisClient: redisClient,
		logger:      log.NewHelper(logger),
	}
}

// InitSyncer 初始化同步器
func (r *redisPrepareSyncer) InitSyncer(ctx context.Context, batchId string) error {
	// 初始化Redis中的计数器
	keys := []string{
		fmt.Sprintf("sms:prepare:total:%s", batchId),
		fmt.Sprintf("sms:prepare:success:%s", batchId),
		fmt.Sprintf("sms:prepare:failure:%s", batchId),
	}

	for _, key := range keys {
		if err := r.redisClient.Set(ctx, key, 0, 24*time.Hour).Err(); err != nil {
			return fmt.Errorf("failed to init redis key %s: %w", key, err)
		}
	}

	return nil
}

// Sync2Statistics 同步计数器到本地统计
func (r *redisPrepareSyncer) Sync2Statistics(ctx context.Context, batchId string) (*sync.StepStatistics, error) {
	totalKey := fmt.Sprintf("sms:prepare:total:%s", batchId)
	successKey := fmt.Sprintf("sms:prepare:success:%s", batchId)
	failureKey := fmt.Sprintf("sms:prepare:failure:%s", batchId)

	total, err := r.redisClient.Get(ctx, totalKey).Int64()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	success, err := r.redisClient.Get(ctx, successKey).Int64()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get success count: %w", err)
	}

	failure, err := r.redisClient.Get(ctx, failureKey).Int64()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get failure count: %w", err)
	}

	return &sync.StepStatistics{
		Total:   total,
		Success: success,
		Failure: failure,
	}, nil
}

// SyncStatistics2Counter 同步本地统计到计数器
func (r *redisPrepareSyncer) SyncStatistics2Counter(ctx context.Context, batchId string, stats *sync.StepStatistics) error {
	totalKey := fmt.Sprintf("sms:prepare:total:%s", batchId)
	successKey := fmt.Sprintf("sms:prepare:success:%s", batchId)
	failureKey := fmt.Sprintf("sms:prepare:failure:%s", batchId)

	if err := r.redisClient.Set(ctx, totalKey, stats.Total, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to set total count: %w", err)
	}

	if err := r.redisClient.Set(ctx, successKey, stats.Success, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to set success count: %w", err)
	}

	if err := r.redisClient.Set(ctx, failureKey, stats.Failure, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("failed to set failure count: %w", err)
	}

	return nil
}

// GetStatistics 获取统计信息
func (r *redisPrepareSyncer) GetStatistics(ctx context.Context, batchId string) (*sync.StepStatistics, error) {
	return r.Sync2Statistics(ctx, batchId)
}

// SetHeartbeatTime 设置心跳时间
func (r *redisPrepareSyncer) SetHeartbeatTime(ctx context.Context, batchId string, heartbeatTime time.Time) error {
	key := fmt.Sprintf("sms:prepare:heartbeat:%s", batchId)
	timestamp := heartbeatTime.Unix()
	return r.redisClient.Set(ctx, key, timestamp, 24*time.Hour).Err()
}

// GetHeartbeatTime 获取心跳时间
func (r *redisPrepareSyncer) GetHeartbeatTime(ctx context.Context, batchId string) (time.Time, error) {
	key := fmt.Sprintf("sms:prepare:heartbeat:%s", batchId)
	timestampStr, err := r.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to get heartbeat time: %w", err)
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse timestamp: %w", err)
	}

	return time.Unix(timestamp, 0), nil
}

// MarkPKTaskComplete 标记分区任务完成
func (r *redisPrepareSyncer) MarkPKTaskComplete(ctx context.Context, batchId, partitionKey string) error {
	key := fmt.Sprintf("sms:prepare:pk_complete:%s:%s", batchId, partitionKey)
	return r.redisClient.Set(ctx, key, 1, 24*time.Hour).Err()
}

// IncreaseTotal 增加总数
func (r *redisPrepareSyncer) IncreaseTotal(ctx context.Context, batchId string, num int64) error {
	key := fmt.Sprintf("sms:prepare:total:%s", batchId)
	return r.redisClient.IncrBy(ctx, key, num).Err()
}

// IncreaseSuccess 增加成功数
func (r *redisPrepareSyncer) IncreaseSuccess(ctx context.Context, batchId string, num int64) error {
	key := fmt.Sprintf("sms:prepare:success:%s", batchId)
	return r.redisClient.IncrBy(ctx, key, num).Err()
}

// IncreaseFailure 增加失败数
func (r *redisPrepareSyncer) IncreaseFailure(ctx context.Context, batchId string, num int64) error {
	key := fmt.Sprintf("sms:prepare:failure:%s", batchId)
	return r.redisClient.IncrBy(ctx, key, num).Err()
}