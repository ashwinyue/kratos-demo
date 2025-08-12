package status

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
)

// StatsPeriod 统计周期
type StatsPeriod string

const (
	PeriodMinute StatsPeriod = "minute"
	PeriodHour   StatsPeriod = "hour"
	PeriodDay    StatsPeriod = "day"
	PeriodWeek   StatsPeriod = "week"
	PeriodMonth  StatsPeriod = "month"
)

// StatusCount 状态计数
type StatusCount struct {
	Status Status `json:"status"`
	Count  int64  `json:"count"`
}

// TypeCount 类型计数
type TypeCount struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
}

// TimeSeriesPoint 时间序列数据点
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     int64     `json:"value"`
}

// StatusDistribution 状态分布
type StatusDistribution struct {
	Period     StatsPeriod   `json:"period"`
	StartTime  time.Time     `json:"start_time"`
	EndTime    time.Time     `json:"end_time"`
	TotalCount int64         `json:"total_count"`
	Counts     []StatusCount `json:"counts"`
	TypeCounts []TypeCount   `json:"type_counts,omitempty"`
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	Period          StatsPeriod   `json:"period"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         time.Time     `json:"end_time"`
	TotalRequests   int64         `json:"total_requests"`
	SuccessfulCount int64         `json:"successful_count"`
	FailedCount     int64         `json:"failed_count"`
	TimeoutCount    int64         `json:"timeout_count"`
	RetryCount      int64         `json:"retry_count"`
	SuccessRate     float64       `json:"success_rate"`
	FailureRate     float64       `json:"failure_rate"`
	TimeoutRate     float64       `json:"timeout_rate"`
	AvgProcessTime  time.Duration `json:"avg_process_time"`
	MaxProcessTime  time.Duration `json:"max_process_time"`
	MinProcessTime  time.Duration `json:"min_process_time"`
	P95ProcessTime  time.Duration `json:"p95_process_time"`
	P99ProcessTime  time.Duration `json:"p99_process_time"`
}

// TrendAnalysis 趋势分析
type TrendAnalysis struct {
	Period      StatsPeriod       `json:"period"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     time.Time         `json:"end_time"`
	TimeSeries  []TimeSeriesPoint `json:"time_series"`
	Trend       string            `json:"trend"` // increasing, decreasing, stable
	GrowthRate  float64           `json:"growth_rate"`
	PeakTime    time.Time         `json:"peak_time,omitempty"`
	PeakValue   int64             `json:"peak_value,omitempty"`
	ValleyTime  time.Time         `json:"valley_time,omitempty"`
	ValleyValue int64             `json:"valley_value,omitempty"`
}

// AlertRule 告警规则
type AlertRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Enabled     bool          `json:"enabled"`
	Metric      string        `json:"metric"`   // success_rate, failure_rate, timeout_rate, etc.
	Operator    string        `json:"operator"` // >, <, >=, <=, ==
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"` // 持续时间
	Severity    string        `json:"severity"` // low, medium, high, critical
	Actions     []string      `json:"actions"`  // email, sms, webhook
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// Alert 告警
type Alert struct {
	ID        string                 `json:"id"`
	RuleID    string                 `json:"rule_id"`
	RuleName  string                 `json:"rule_name"`
	Message   string                 `json:"message"`
	Severity  string                 `json:"severity"`
	Status    string                 `json:"status"` // firing, resolved
	StartTime time.Time              `json:"start_time"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Notified  bool                   `json:"notified"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// StatisticsQuery 统计查询
type StatisticsQuery struct {
	StartTime time.Time   `json:"start_time"`
	EndTime   time.Time   `json:"end_time"`
	Period    StatsPeriod `json:"period"`
	Statuses  []Status    `json:"statuses,omitempty"`
	Types     []string    `json:"types,omitempty"`
	GroupBy   []string    `json:"group_by,omitempty"` // status, type, hour, day
	Limit     int         `json:"limit,omitempty"`
	Offset    int         `json:"offset,omitempty"`
}

// StatusStatistics 状态统计接口
type StatusStatistics interface {
	// GetDistribution 获取状态分布
	GetDistribution(ctx context.Context, query *StatisticsQuery) (*StatusDistribution, error)

	// GetPerformanceMetrics 获取性能指标
	GetPerformanceMetrics(ctx context.Context, query *StatisticsQuery) (*PerformanceMetrics, error)

	// GetTrendAnalysis 获取趋势分析
	GetTrendAnalysis(ctx context.Context, query *StatisticsQuery) (*TrendAnalysis, error)

	// GetTopFailureReasons 获取主要失败原因
	GetTopFailureReasons(ctx context.Context, query *StatisticsQuery) ([]map[string]interface{}, error)

	// GetHourlyStats 获取小时统计
	GetHourlyStats(ctx context.Context, date time.Time) ([]*StatusDistribution, error)

	// GetDailyStats 获取日统计
	GetDailyStats(ctx context.Context, startDate, endDate time.Time) ([]*StatusDistribution, error)

	// RecordStatusChange 记录状态变更（用于实时统计）
	RecordStatusChange(ctx context.Context, change *StatusChange) error

	// CreateAlertRule 创建告警规则
	CreateAlertRule(ctx context.Context, rule *AlertRule) error

	// UpdateAlertRule 更新告警规则
	UpdateAlertRule(ctx context.Context, rule *AlertRule) error

	// DeleteAlertRule 删除告警规则
	DeleteAlertRule(ctx context.Context, ruleID string) error

	// GetAlertRules 获取告警规则列表
	GetAlertRules(ctx context.Context) ([]*AlertRule, error)

	// GetActiveAlerts 获取活跃告警
	GetActiveAlerts(ctx context.Context) ([]*Alert, error)

	// ResolveAlert 解决告警
	ResolveAlert(ctx context.Context, alertID string) error

	// IsHealthy 健康检查
	IsHealthy(ctx context.Context) bool
}

// StatusStatisticsImpl 状态统计实现
type StatusStatisticsImpl struct {
	statusTracker StatusTracker
	dataRepo      DataRepo
	logger        *log.Helper

	// 缓存和实时统计
	mu           sync.RWMutex
	cache        map[string]interface{}
	cacheExpiry  map[string]time.Time
	alertRules   map[string]*AlertRule
	activeAlerts map[string]*Alert

	// 实时计数器
	counters  map[string]int64
	lastReset time.Time
}

// NewStatusStatistics 创建状态统计服务
func NewStatusStatistics(statusTracker StatusTracker, dataRepo DataRepo, logger log.Logger) StatusStatistics {
	return &StatusStatisticsImpl{
		statusTracker: statusTracker,
		dataRepo:      dataRepo,
		logger:        log.NewHelper(logger),
		cache:         make(map[string]interface{}),
		cacheExpiry:   make(map[string]time.Time),
		alertRules:    make(map[string]*AlertRule),
		activeAlerts:  make(map[string]*Alert),
		counters:      make(map[string]int64),
		lastReset:     time.Now(),
	}
}

// GetDistribution 获取状态分布
func (s *StatusStatisticsImpl) GetDistribution(ctx context.Context, query *StatisticsQuery) (*StatusDistribution, error) {
	if query == nil {
		return nil, ErrInvalidRequest
	}

	// 检查缓存
	cacheKey := fmt.Sprintf("distribution_%d_%d_%s", query.StartTime.Unix(), query.EndTime.Unix(), query.Period)
	if cached := s.getFromCache(cacheKey); cached != nil {
		if dist, ok := cached.(*StatusDistribution); ok {
			return dist, nil
		}
	}

	// 模拟数据查询（实际应该从数据库查询）
	distribution := &StatusDistribution{
		Period:     query.Period,
		StartTime:  query.StartTime,
		EndTime:    query.EndTime,
		TotalCount: 1000,
		Counts: []StatusCount{
			{Status: StatusCompleted, Count: 800},
			{Status: StatusFailed, Count: 150},
			{Status: StatusTimeout, Count: 30},
			{Status: StatusProcessing, Count: 20},
		},
		TypeCounts: []TypeCount{
			{Type: "sms", Count: 600},
			{Type: "email", Count: 300},
			{Type: "push", Count: 100},
		},
	}

	// 缓存结果
	s.setCache(cacheKey, distribution, 5*time.Minute)

	return distribution, nil
}

// GetPerformanceMetrics 获取性能指标
func (s *StatusStatisticsImpl) GetPerformanceMetrics(ctx context.Context, query *StatisticsQuery) (*PerformanceMetrics, error) {
	if query == nil {
		return nil, ErrInvalidRequest
	}

	// 检查缓存
	cacheKey := fmt.Sprintf("performance_%d_%d_%s", query.StartTime.Unix(), query.EndTime.Unix(), query.Period)
	if cached := s.getFromCache(cacheKey); cached != nil {
		if metrics, ok := cached.(*PerformanceMetrics); ok {
			return metrics, nil
		}
	}

	// 模拟性能指标计算
	totalRequests := int64(1000)
	successfulCount := int64(800)
	failedCount := int64(150)
	timeoutCount := int64(30)
	retryCount := int64(50)

	metrics := &PerformanceMetrics{
		Period:          query.Period,
		StartTime:       query.StartTime,
		EndTime:         query.EndTime,
		TotalRequests:   totalRequests,
		SuccessfulCount: successfulCount,
		FailedCount:     failedCount,
		TimeoutCount:    timeoutCount,
		RetryCount:      retryCount,
		SuccessRate:     float64(successfulCount) / float64(totalRequests) * 100,
		FailureRate:     float64(failedCount) / float64(totalRequests) * 100,
		TimeoutRate:     float64(timeoutCount) / float64(totalRequests) * 100,
		AvgProcessTime:  2 * time.Second,
		MaxProcessTime:  10 * time.Second,
		MinProcessTime:  100 * time.Millisecond,
		P95ProcessTime:  5 * time.Second,
		P99ProcessTime:  8 * time.Second,
	}

	// 缓存结果
	s.setCache(cacheKey, metrics, 5*time.Minute)

	return metrics, nil
}

// GetTrendAnalysis 获取趋势分析
func (s *StatusStatisticsImpl) GetTrendAnalysis(ctx context.Context, query *StatisticsQuery) (*TrendAnalysis, error) {
	if query == nil {
		return nil, ErrInvalidRequest
	}

	// 检查缓存
	cacheKey := fmt.Sprintf("trend_%d_%d_%s", query.StartTime.Unix(), query.EndTime.Unix(), query.Period)
	if cached := s.getFromCache(cacheKey); cached != nil {
		if trend, ok := cached.(*TrendAnalysis); ok {
			return trend, nil
		}
	}

	// 模拟趋势数据生成
	timeSeries := s.generateTimeSeriesData(query.StartTime, query.EndTime, query.Period)

	analysis := &TrendAnalysis{
		Period:     query.Period,
		StartTime:  query.StartTime,
		EndTime:    query.EndTime,
		TimeSeries: timeSeries,
		Trend:      "increasing",
		GrowthRate: 5.2,
	}

	// 计算峰值和谷值
	if len(timeSeries) > 0 {
		peakValue := timeSeries[0].Value
		valleyValue := timeSeries[0].Value
		peakTime := timeSeries[0].Timestamp
		valleyTime := timeSeries[0].Timestamp

		for _, point := range timeSeries {
			if point.Value > peakValue {
				peakValue = point.Value
				peakTime = point.Timestamp
			}
			if point.Value < valleyValue {
				valleyValue = point.Value
				valleyTime = point.Timestamp
			}
		}

		analysis.PeakValue = peakValue
		analysis.PeakTime = peakTime
		analysis.ValleyValue = valleyValue
		analysis.ValleyTime = valleyTime
	}

	// 缓存结果
	s.setCache(cacheKey, analysis, 10*time.Minute)

	return analysis, nil
}

// GetTopFailureReasons 获取主要失败原因
func (s *StatusStatisticsImpl) GetTopFailureReasons(ctx context.Context, query *StatisticsQuery) ([]map[string]interface{}, error) {
	if query == nil {
		return nil, ErrInvalidRequest
	}

	// 模拟失败原因统计
	reasons := []map[string]interface{}{
		{
			"reason": "网络超时",
			"count":  45,
			"rate":   30.0,
		},
		{
			"reason": "服务不可用",
			"count":  38,
			"rate":   25.3,
		},
		{
			"reason": "参数错误",
			"count":  32,
			"rate":   21.3,
		},
		{
			"reason": "限流",
			"count":  20,
			"rate":   13.3,
		},
		{
			"reason": "其他",
			"count":  15,
			"rate":   10.0,
		},
	}

	return reasons, nil
}

// GetHourlyStats 获取小时统计
func (s *StatusStatisticsImpl) GetHourlyStats(ctx context.Context, date time.Time) ([]*StatusDistribution, error) {
	var stats []*StatusDistribution

	// 生成24小时的统计数据
	for hour := 0; hour < 24; hour++ {
		startTime := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, date.Location())
		endTime := startTime.Add(time.Hour)

		stat := &StatusDistribution{
			Period:     PeriodHour,
			StartTime:  startTime,
			EndTime:    endTime,
			TotalCount: int64(50 + hour*2), // 模拟数据
			Counts: []StatusCount{
				{Status: StatusCompleted, Count: int64(40 + hour*2)},
				{Status: StatusFailed, Count: int64(8 + hour/3)},
				{Status: StatusTimeout, Count: int64(2)},
			},
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// GetDailyStats 获取日统计
func (s *StatusStatisticsImpl) GetDailyStats(ctx context.Context, startDate, endDate time.Time) ([]*StatusDistribution, error) {
	var stats []*StatusDistribution

	// 生成每日统计数据
	for d := startDate; d.Before(endDate) || d.Equal(endDate); d = d.AddDate(0, 0, 1) {
		stat := &StatusDistribution{
			Period:     PeriodDay,
			StartTime:  d,
			EndTime:    d.AddDate(0, 0, 1),
			TotalCount: int64(1000 + d.Day()*10), // 模拟数据
			Counts: []StatusCount{
				{Status: StatusCompleted, Count: int64(800 + d.Day()*8)},
				{Status: StatusFailed, Count: int64(150 + d.Day())},
				{Status: StatusTimeout, Count: int64(30)},
				{Status: StatusProcessing, Count: int64(20)},
			},
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

// RecordStatusChange 记录状态变更
func (s *StatusStatisticsImpl) RecordStatusChange(ctx context.Context, change *StatusChange) error {
	if change == nil {
		return ErrInvalidRequest
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 更新实时计数器
	key := fmt.Sprintf("%s", change.ToStatus)
	s.counters[key]++

	// 检查告警规则
	go s.checkAlertRules(ctx, change)

	s.logger.Debugf("Recorded status change: %s -> %s", change.FromStatus, change.ToStatus)
	return nil
}

// CreateAlertRule 创建告警规则
func (s *StatusStatisticsImpl) CreateAlertRule(ctx context.Context, rule *AlertRule) error {
	if rule == nil || rule.ID == "" {
		return ErrInvalidRequest
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	s.alertRules[rule.ID] = rule

	s.logger.Infof("Created alert rule: %s", rule.Name)
	return nil
}

// UpdateAlertRule 更新告警规则
func (s *StatusStatisticsImpl) UpdateAlertRule(ctx context.Context, rule *AlertRule) error {
	if rule == nil || rule.ID == "" {
		return ErrInvalidRequest
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.alertRules[rule.ID]; !exists {
		return ErrStatusNotFound
	}

	rule.UpdatedAt = time.Now()
	s.alertRules[rule.ID] = rule

	s.logger.Infof("Updated alert rule: %s", rule.Name)
	return nil
}

// DeleteAlertRule 删除告警规则
func (s *StatusStatisticsImpl) DeleteAlertRule(ctx context.Context, ruleID string) error {
	if ruleID == "" {
		return ErrInvalidRequest
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.alertRules[ruleID]; !exists {
		return ErrStatusNotFound
	}

	delete(s.alertRules, ruleID)

	s.logger.Infof("Deleted alert rule: %s", ruleID)
	return nil
}

// GetAlertRules 获取告警规则列表
func (s *StatusStatisticsImpl) GetAlertRules(ctx context.Context) ([]*AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rules []*AlertRule
	for _, rule := range s.alertRules {
		rules = append(rules, rule)
	}

	return rules, nil
}

// GetActiveAlerts 获取活跃告警
func (s *StatusStatisticsImpl) GetActiveAlerts(ctx context.Context) ([]*Alert, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var alerts []*Alert
	for _, alert := range s.activeAlerts {
		if alert.Status == "firing" {
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

// ResolveAlert 解决告警
func (s *StatusStatisticsImpl) ResolveAlert(ctx context.Context, alertID string) error {
	if alertID == "" {
		return ErrInvalidRequest
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	alert, exists := s.activeAlerts[alertID]
	if !exists {
		return ErrStatusNotFound
	}

	now := time.Now()
	alert.Status = "resolved"
	alert.EndTime = &now
	alert.Duration = now.Sub(alert.StartTime)
	alert.UpdatedAt = now

	s.logger.Infof("Resolved alert: %s", alertID)
	return nil
}

// IsHealthy 健康检查
func (s *StatusStatisticsImpl) IsHealthy(ctx context.Context) bool {
	if s.statusTracker != nil && !s.statusTracker.IsHealthy(ctx) {
		return false
	}

	if s.dataRepo != nil && !s.dataRepo.IsHealthy(ctx) {
		return false
	}

	return true
}

// getFromCache 从缓存获取数据
func (s *StatusStatisticsImpl) getFromCache(key string) interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if expiry, exists := s.cacheExpiry[key]; exists {
		if time.Now().After(expiry) {
			// 缓存过期
			delete(s.cache, key)
			delete(s.cacheExpiry, key)
			return nil
		}
		return s.cache[key]
	}
	return nil
}

// setCache 设置缓存
func (s *StatusStatisticsImpl) setCache(key string, value interface{}, ttl time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache[key] = value
	s.cacheExpiry[key] = time.Now().Add(ttl)
}

// generateTimeSeriesData 生成时间序列数据
func (s *StatusStatisticsImpl) generateTimeSeriesData(startTime, endTime time.Time, period StatsPeriod) []TimeSeriesPoint {
	var points []TimeSeriesPoint
	var interval time.Duration

	switch period {
	case PeriodMinute:
		interval = time.Minute
	case PeriodHour:
		interval = time.Hour
	case PeriodDay:
		interval = 24 * time.Hour
	default:
		interval = time.Hour
	}

	for t := startTime; t.Before(endTime); t = t.Add(interval) {
		// 模拟数据生成
		value := int64(100 + t.Hour()*5 + t.Minute()/10)
		points = append(points, TimeSeriesPoint{
			Timestamp: t,
			Value:     value,
		})
	}

	return points
}

// checkAlertRules 检查告警规则
func (s *StatusStatisticsImpl) checkAlertRules(ctx context.Context, change *StatusChange) {
	s.mu.RLock()
	rules := make([]*AlertRule, 0, len(s.alertRules))
	for _, rule := range s.alertRules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}
	s.mu.RUnlock()

	// 检查每个规则
	for _, rule := range rules {
		if s.shouldTriggerAlert(rule, change) {
			s.triggerAlert(ctx, rule, change)
		}
	}
}

// shouldTriggerAlert 判断是否应该触发告警
func (s *StatusStatisticsImpl) shouldTriggerAlert(rule *AlertRule, change *StatusChange) bool {
	// 这里应该根据具体的规则逻辑判断
	// 目前只是一个简单的示例
	if rule.Metric == "failure_rate" && change.ToStatus == StatusFailed {
		return true
	}
	return false
}

// triggerAlert 触发告警
func (s *StatusStatisticsImpl) triggerAlert(ctx context.Context, rule *AlertRule, change *StatusChange) {
	alertID := fmt.Sprintf("%s_%d", rule.ID, time.Now().Unix())
	now := time.Now()

	alert := &Alert{
		ID:        alertID,
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Message:   fmt.Sprintf("Alert triggered: %s", rule.Description),
		Severity:  rule.Severity,
		Status:    "firing",
		StartTime: now,
		Duration:  0,
		Metadata: map[string]interface{}{
			"from_status": change.FromStatus,
			"to_status":   change.ToStatus,
			"timestamp":   change.Timestamp,
		},
		Notified:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	s.activeAlerts[alertID] = alert
	s.mu.Unlock()

	s.logger.Warnf("Alert triggered: %s - %s", rule.Name, alert.Message)
}
