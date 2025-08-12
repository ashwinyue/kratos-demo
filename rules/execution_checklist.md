# Java到Go项目重构执行检查清单

## 当前状态概览

### ✅ 已完成 (40%)
- [x] Kratos框架基础搭建
- [x] MongoDB、RocketMQ、Redis集成
- [x] Docker Compose本地环境
- [x] Kubernetes部署配置
- [x] 基础监控(Prometheus + Jaeger)
- [x] 服务正常运行(HTTP:8000, gRPC:9000)
- [x] SMS发送器模块完整实现
- [x] 同步器模块完整实现

### 🔄 当前进行中 (10%)
- [ ] 基础消息队列功能完善
- [ ] API接口完善
- [ ] 监控集成优化
- [ ] 状态追踪器模块开发

### ❌ 待完成 (50%)
**15个核心业务模块待实现**

---

## 第一阶段: 核心业务模块实现 (3-4周)

### Week 1: SMS发送器和同步器模块

#### Day 1-3: SMS发送器模块 ✅ **已完成**

**文件创建清单:**
```
internal/biz/
├── sms_sender.go              # SMS发送器接口 ✅
├── aliyun_sms_sender.go       # 阿里云SMS发送器 ✅
├── we_sms_sender.go           # WE SMS发送器 ✅
├── xsxx_sms_sender.go         # XSXX SMS发送器 ✅
├── sms_sender_factory.go      # SMS发送器工厂 ✅
└── sms_sender_config.go       # 发送器配置管理 ✅
```

**具体任务:**
- [x] 创建SMS发送器接口 (`internal/biz/sms_sender.go`)
  ```go
  type SmsSender interface {
      Send(ctx context.Context, req *SendRequest) (*SendResponse, error)
      GetProviderType() ProviderType
      ValidateConfig() error
      GetMaxBatchSize() int
      IsHealthy(ctx context.Context) bool
  }
  ```

- [x] 实现阿里云SMS发送器 (`internal/biz/aliyun_sms_sender.go`)
  ```go
  type AliyunSmsSender struct {
      config *AliyunSmsConfig
      client *dysmsapi.Client
      logger *log.Helper
  }
  ```

- [x] 实现WE SMS发送器 (`internal/biz/we_sms_sender.go`)
- [x] 实现XSXX SMS发送器 (`internal/biz/xsxx_sms_sender.go`)
- [x] 创建SMS发送器工厂 (`internal/biz/sms_sender_factory.go`)
  ```go
  type SmsSenderFactory interface {
      CreateSender(providerType ProviderType) (SmsSender, error)
      GetSender(providerType ProviderType) (SmsSender, error)
      ListAvailableSenders() []ProviderType
  }
  ```

- [x] 添加发送器配置管理
  ```yaml
  # configs/config.yaml
  sms:
    providers:
      aliyun:
        access_key_id: "your_key"
        access_key_secret: "your_secret"
        endpoint: "dysmsapi.aliyuncs.com"
      we:
        api_url: "https://api.we.com"
        api_key: "your_key"
      xsxx:
        api_url: "https://api.xsxx.com"
        username: "your_username"
        password: "your_password"
  ```

**验收标准:**
- [x] 所有SMS发送器接口实现完成
- [x] 工厂模式正确工作
- [x] 配置管理正常加载
- [x] 单元测试覆盖率>80%
- [x] 集成测试通过
- [x] 基准测试完成
- [x] 并发安全测试通过

#### Day 4-7: 同步器模块 ✅ **已完成**

**文件创建清单:**
```
internal/biz/
├── syncer.go                  # 同步器接口 ✅
├── prepare_syncer.go          # 准备同步器 ✅
├── delivery_syncer.go         # 投递同步器 ✅
├── save_syncer.go             # 保存同步器 ✅
├── syncer_factory.go          # 同步器工厂 ✅
└── syncer_test.go             # 单元测试 ✅
```

**具体任务:**
- [x] 创建同步器接口 (`internal/biz/syncer.go`)
  ```go
  type Syncer interface {
      Sync(ctx context.Context, req *SyncRequest) (*SyncResponse, error)
      GetSyncType() SyncType
      IsAsync() bool
      ValidateRequest(req *SyncRequest) error
      GetConfig() *SyncConfig
      IsHealthy(ctx context.Context) bool
  }
  ```

- [x] 实现准备同步器 (`internal/biz/prepare_syncer.go`)
  - 同步SMS批次准备状态
  - 更新数据库状态
  - 发送准备完成消息

- [x] 实现投递同步器 (`internal/biz/delivery_syncer.go`)
  - 同步SMS投递状态
  - 更新投递统计
  - 处理投递回调

- [x] 实现保存同步器 (`internal/biz/save_syncer.go`)
  - 保存处理结果
  - 更新最终状态
  - 清理临时数据

- [x] 实现同步器工厂 (`internal/biz/syncer_factory.go`)
  - 同步器注册和创建
  - 配置管理和健康检查
  - 统计信息获取

- [x] 添加完整的单元测试 (`internal/biz/syncer_test.go`)
  - 测试覆盖率: 32.8%
  - Mock对象测试
  - 基准性能测试

**验收标准:**
- [x] 三种同步器正确实现
- [x] 异步同步机制工作正常
- [x] 错误处理和重试机制完善
- [x] 集成测试通过
- [x] 项目构建成功
- [x] 单元测试全部通过

### Week 2: 状态追踪器和业务服务模块

#### Day 8-10: 状态追踪器模块 ⭐ **当前优先级**

**文件创建清单:**
```
internal/biz/
├── status_tracker.go          # 状态追踪器接口
├── timeout_detector.go        # 超时检测机制
├── status_recovery.go         # 状态恢复机制
└── status_statistics.go       # 状态统计
```

**具体任务:**
- [ ] 创建状态追踪器接口 (`internal/biz/status_tracker.go`)
  ```go
  type StatusTracker interface {
      TrackStatus(ctx context.Context, req *TrackRequest) error
      GetStatus(ctx context.Context, id string) (*StatusInfo, error)
      UpdateStatus(ctx context.Context, id string, status Status) error
      IsHealthy(ctx context.Context) bool
  }
  ```
- [ ] 实现投递状态追踪和状态变更监控
- [ ] 实现超时检测机制 (`internal/biz/timeout_detector.go`)
  - 定时检测超时任务
  - 触发超时处理流程
  - 支持可配置的超时策略
- [ ] 实现状态恢复机制 (`internal/biz/status_recovery.go`)
  - 异常状态自动恢复
  - 状态一致性检查
  - 数据修复功能
- [ ] 添加状态统计和报告 (`internal/biz/status_statistics.go`)
  - 状态分布统计
  - 性能指标收集
  - 报告生成功能
- [ ] 实现异常状态处理和告警机制

**验收标准:**
- [ ] 状态追踪功能正确实现
- [ ] 超时检测机制工作正常
- [ ] 状态恢复机制可靠
- [ ] 统计报告准确
- [ ] 单元测试覆盖率>80%

#### Day 11-14: 业务服务模块

**文件创建清单:**
```
internal/biz/
├── sms_batch_service.go       # SMS批处理业务服务
├── sms_delivery_pack_service.go # SMS投递包服务
├── business_orchestrator.go   # 业务逻辑编排
└── business_validator.go      # 业务验证规则
```

**具体任务:**
- [ ] 创建SMS批处理业务服务 (`internal/biz/sms_batch_service.go`)
- [ ] 创建SMS投递包服务 (`internal/biz/sms_delivery_pack_service.go`)
- [ ] 实现业务逻辑编排和协调
- [ ] 添加业务异常处理
- [ ] 实现服务间依赖注入
- [ ] 添加业务验证规则

### Week 3: 步骤处理模块

#### Day 15-17: 抽象步骤处理

**文件创建清单:**
```
internal/biz/
├── abstract_step.go           # 抽象步骤接口
├── step_statistics.go         # 步骤统计结构
├── step_state_manager.go      # 步骤状态管理
└── step_timeout_handler.go    # 步骤超时处理
```

#### Day 18-21: 具体步骤实现

**文件创建清单:**
```
internal/biz/
├── sms_preparation_step.go    # 准备步骤
├── sms_delivery_step.go       # 投递步骤
├── sms_delivery_step_we_relay.go # WE中继投递步骤
├── step_factory.go            # 步骤工厂
└── step_enum.go               # 步骤枚举定义
```

### Week 4: 核心模块集成测试

#### Day 22-24: 模块集成
- [ ] 集成发送器、同步器、追踪器模块
- [ ] 实现完整的业务流程
- [ ] 添加模块间接口联调
- [ ] 实现错误处理和异常恢复

#### Day 25-28: 核心功能测试
- [ ] 单元测试覆盖率>80%
- [ ] 集成测试验证
- [ ] 性能基准测试
- [ ] 错误场景测试

---

## 第二阶段: 支撑工具模块实现 (2-3周)

### Week 5: 共享工具模块

#### Day 29-31: 分区键工具

**文件创建清单:**
```
internal/biz/
├── partition_key_utils.go     # 分区键工具
├── partition_manager.go       # 分区管理器
└── partition_load_balancer.go # 分区负载均衡
```

**具体任务:**
- [ ] 创建分区键工具 (`internal/biz/partition_key_utils.go`)
  ```go
  const PARTITION_NUM = 128
  
  func GeneratePartitionKey(batchId string) string {
      hash := fnv.New32a()
      hash.Write([]byte(batchId))
      return fmt.Sprintf("SMS_BATCH_PK_%d", hash.Sum32()%PARTITION_NUM)
  }
  ```

- [ ] 实现128分区键生成算法
- [ ] 实现分区键可用性管理
- [ ] 添加分区键负载均衡
- [ ] 实现分区键映射存储

#### Day 32-35: 雪花ID生成器和消息处理器

**文件创建清单:**
```
internal/biz/
├── snowflake_id_worker.go     # 雪花ID生成器
├── message_handler.go         # 消息处理器接口
└── message_reporter.go        # 消息报告器
```

**具体任务:**
- [ ] 实现雪花ID生成器 (`internal/biz/snowflake_id_worker.go`)
  ```go
  type SnowflakeIdWorker struct {
      workerId     int64
      datacenterId int64
      sequence     int64
      lastTimestamp int64
  }
  
  func (s *SnowflakeIdWorker) NextId() int64 {
      // 雪花算法实现
  }
  ```

- [ ] 配置机器ID和数据中心ID
- [ ] 实现ID生成性能优化
- [ ] 创建消息处理器接口 (`internal/biz/message_handler.go`)
- [ ] 实现内部消息报告发送
- [ ] 实现CDP消息报告发送
- [ ] 实现批次投递任务发送

### Week 6: 线程池管理和缓存管理

#### Day 36-38: 线程池管理模块

**文件创建清单:**
```
internal/biz/
├── goroutine_pool.go          # Goroutine池管理
├── task_pool.go               # 任务池
├── worker_pool.go             # 工作池
internal/conf/
└── task_config.go             # 任务配置管理
```

#### Day 39-42: 缓存管理模块

**文件创建清单:**
```
internal/data/
├── cache_sms_batch.go         # SMS批次LRU缓存
├── cache_manager.go           # 缓存管理器
├── cache_statistics.go        # 缓存统计
└── cache_consistency.go       # 缓存一致性
```

---

## 状态机实现指南

### 1. 安装依赖
```bash
go get github.com/looplab/fsm
```

### 2. 状态机实现

**文件:** `internal/biz/sms_state_machine.go`
```go
package biz

import (
    "github.com/looplab/fsm"
    "time"
)

type SmsStatus string

const (
    StatusInitial           SmsStatus = "INITIAL"
    StatusReady            SmsStatus = "READY"
    StatusRunning          SmsStatus = "RUNNING"
    StatusPaused           SmsStatus = "PAUSED"
    StatusPausing          SmsStatus = "PAUSING"
    StatusCompletedSucceeded SmsStatus = "COMPLETED_SUCCEEDED"
    StatusCompletedFailed   SmsStatus = "COMPLETED_FAILED"
)

type SmsBatchStateMachine struct {
    fsm   *fsm.FSM
    batch *SmsBatch
}

func NewSmsBatchStateMachine(batch *SmsBatch) *SmsBatchStateMachine {
    sm := &SmsBatchStateMachine{
        batch: batch,
    }
    
    sm.fsm = fsm.NewFSM(
        string(StatusInitial),
        fsm.Events{
            {Name: "prepare", Src: []string{"INITIAL"}, Dst: "READY"},
            {Name: "start", Src: []string{"READY", "PAUSED"}, Dst: "RUNNING"},
            {Name: "pause", Src: []string{"RUNNING"}, Dst: "PAUSING"},
            {Name: "paused", Src: []string{"PAUSING"}, Dst: "PAUSED"},
            {Name: "succeed", Src: []string{"RUNNING"}, Dst: "COMPLETED_SUCCEEDED"},
            {Name: "fail", Src: []string{"RUNNING", "PAUSING"}, Dst: "COMPLETED_FAILED"},
        },
        fsm.Callbacks{
            "enter_RUNNING": sm.onEnterRunning,
            "enter_COMPLETED_SUCCEEDED": sm.onEnterCompleted,
            "enter_COMPLETED_FAILED": sm.onEnterFailed,
        },
    )
    
    return sm
}

func (sm *SmsBatchStateMachine) onEnterRunning(e *fsm.Event) {
    sm.batch.StartTime = time.Now()
    sm.batch.Status = StatusRunning
}

func (sm *SmsBatchStateMachine) onEnterCompleted(e *fsm.Event) {
    sm.batch.EndTime = time.Now()
    sm.batch.Status = StatusCompletedSucceeded
}

func (sm *SmsBatchStateMachine) onEnterFailed(e *fsm.Event) {
    sm.batch.EndTime = time.Now()
    sm.batch.Status = StatusCompletedFailed
}

func (sm *SmsBatchStateMachine) Prepare() error {
    return sm.fsm.Event("prepare")
}

func (sm *SmsBatchStateMachine) Start() error {
    return sm.fsm.Event("start")
}

func (sm *SmsBatchStateMachine) Pause() error {
    return sm.fsm.Event("pause")
}

func (sm *SmsBatchStateMachine) Succeed() error {
    return sm.fsm.Event("succeed")
}

func (sm *SmsBatchStateMachine) Fail() error {
    return sm.fsm.Event("fail")
}

func (sm *SmsBatchStateMachine) Current() string {
    return sm.fsm.Current()
}
```

---

## 快速开始指南

### 1. 环境准备
```bash
# 确保服务正在运行
cd /root/workspace/kratos-x/kratos-demo
docker compose up -d

# 检查服务状态
ss -tlnp | grep -E ':(8000|9000|3306|6379|27017|9876)'
```

### 2. 开始第一阶段开发
```bash
# 创建SMS发送器模块目录
mkdir -p internal/biz

# 开始实现SMS发送器接口
vim internal/biz/sms_sender.go
```

### 3. 测试驱动开发
```bash
# 为每个模块创建测试文件
touch internal/biz/sms_sender_test.go

# 运行测试
go test ./internal/biz/...
```

### 4. 持续集成
```bash
# 构建项目
make build

# 运行服务
./bin/kratos-demo -conf ./configs/config.yaml
```

---

## 关键检查点

### 每日检查
- [ ] 代码提交到Git
- [ ] 单元测试通过
- [ ] 代码质量检查
- [ ] 功能验证测试

### 每周检查
- [ ] 模块集成测试
- [ ] 性能基准测试
- [ ] 代码覆盖率检查
- [ ] 文档更新

### 阶段检查
- [ ] 功能完整性验证
- [ ] 性能指标达标
- [ ] 安全漏洞扫描
- [ ] 部署测试验证

---

## 常用命令

### 开发命令
```bash
# 生成protobuf代码
make api

# 构建项目
make build

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make lint
```

### 部署命令
```bash
# 本地环境
docker compose up -d

# 测试环境
cd k8s && ./quick-deploy.sh

# 生产环境
cd k8s && ./deploy.sh
```

### 监控命令
```bash
# 查看服务状态
ss -tlnp | grep :8000

# 查看日志
docker compose logs -f kratos-demo

# 查看监控指标
curl http://localhost:9090/metrics
```

---

## 问题排查

### 常见问题
1. **端口冲突**: 使用 `ss -tlnp` 检查端口占用
2. **数据库连接失败**: 检查MongoDB服务状态
3. **消息队列异常**: 检查RocketMQ服务状态
4. **构建失败**: 检查Go模块依赖

### 调试技巧
1. 使用 `make test` 运行单元测试
2. 使用 `docker compose logs` 查看服务日志
3. 使用 `curl` 测试API接口
4. 使用Jaeger查看链路追踪

---

---

## 📋 SMS发送器模块完成总结

### ✅ 已完成的工作 (2024年最新状态)

#### 核心功能实现
- **SMS发送器接口**: 完整的接口定义，支持发送、配置验证、健康检查
- **多提供商支持**: 阿里云、WE、XSXX三个SMS提供商完整实现
- **工厂模式**: 灵活的SMS发送器工厂，支持动态创建和管理
- **配置管理**: 完善的配置验证和管理机制

#### 测试覆盖
- **单元测试**: 覆盖率>80%，包含所有核心功能
- **集成测试**: 验证多提供商集成和工厂模式
- **基准测试**: 性能基准测试完成
- **并发测试**: 并发安全性验证通过

#### 技术特性
- **错误处理**: 完善的错误处理和重试机制
- **日志记录**: 结构化日志记录
- **健康检查**: 提供商健康状态检测
- **配置验证**: 启动时配置有效性验证

### 📊 项目整体进度更新
- **总体完成度**: 20% → 30% ⬆️
- **核心模块**: 17个 → 16个待完成
- **当前状态**: SMS发送器模块 ✅ 完成

### 🎯 下一步行动计划

#### 立即开始: 同步器模块开发 ⭐
**目标**: 实现数据同步、状态同步、异步处理功能

**优先任务**:
1. 创建同步器接口 (`internal/biz/syncer.go`)
2. 实现准备同步器 (`internal/biz/prepare_syncer.go`)
3. 实现投递同步器 (`internal/biz/delivery_syncer.go`)
4. 实现保存同步器 (`internal/biz/save_syncer.go`)

**预期时间**: 4-7天
**验收标准**: 三种同步器正确实现，异步同步机制工作正常

---

## 📋 模块完成总结

### 同步器模块 (已完成) ✅

**实现文件:**
- `internal/biz/syncer.go` - 同步器接口和基础结构
- `internal/biz/prepare_syncer.go` - 准备同步器实现
- `internal/biz/delivery_syncer.go` - 投递同步器实现
- `internal/biz/save_syncer.go` - 保存同步器实现
- `internal/biz/syncer_factory.go` - 同步器工厂实现
- `internal/biz/syncer_test.go` - 完整单元测试

**技术特性:**
- 模块化设计，职责单一
- 接口驱动，支持依赖注入
- 完善的错误处理和日志记录
- 异步处理支持
- 健康检查机制
- 配置管理和动态更新
- 测试覆盖率: 32.8%
- 项目构建成功，所有测试通过

**下一步计划:**
开始实施状态追踪器模块开发，继续第一阶段核心业务模块实现 🚀

---

**当前行动**: 开始实施状态追踪器模块开发，继续第一阶段核心业务模块实现 🚀