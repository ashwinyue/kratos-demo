# Kratos项目开发规范

## 1. 项目目录结构规范

### 基础目录结构
```
kratos-demo/
├── api/                    # API定义文件（protobuf）
│   └── helloworld/
├── cmd/                    # 应用程序入口
│   └── kratos-demo/
├── configs/                # 配置文件
│   └── config.yaml
├── internal/               # 内部代码，不对外暴露
│   ├── biz/               # 业务逻辑层
│   ├── conf/              # 配置结构定义
│   ├── data/              # 数据访问层
│   ├── server/            # 服务器配置（HTTP/gRPC）
│   └── service/           # 服务层（实现API接口）
├── third_party/           # 第三方proto文件
├── bin/                   # 构建输出目录
├── k8s/                   # Kubernetes部署文件
├── rocketmq/              # RocketMQ配置文件
├── docker-compose.yml     # 本地开发环境
├── Dockerfile            # 容器镜像构建
├── Makefile              # 构建脚本
├── go.mod                # Go模块定义
└── README.md             # 项目说明
```

### 构建规范
- 所有构建文件统一放到 `bin/` 目录
- 使用 `make build` 命令构建项目
- 二进制文件命名格式：`{项目名}-{版本}`

## 2. 端口使用规范

### 查看端口使用
使用 `ss` 命令查看端口占用情况：
```bash
# 查看所有监听端口
ss -tlnp

# 查看特定端口
ss -tlnp | grep :8080

# 查看端口范围
ss -tlnp | grep -E ':(8080|9090|3000)'
```

### 端口分配规范

#### 应用服务端口
- HTTP服务：`8000-8099`
- gRPC服务：`9000-9099`
- 管理接口：`8080, 9090`

#### 基础设施端口
- MySQL: `3306`
- Redis: `6379`
- MongoDB: `27017`
- RocketMQ NameServer: `9876`
- RocketMQ Broker: `10911, 10909`
- RocketMQ Console: `8080`

#### 监控服务端口
- Prometheus: `9091`
- Grafana: `3000`
- Jaeger: `16687`

### 端口冲突处理
1. 使用 `ss -tlnp | grep :{端口}` 检查端口占用
2. 如有冲突，优先调整应用端口，保持基础设施端口不变
3. 更新配置文件和文档中的端口信息

## 3. 环境部署策略

### 本地开发环境
- **使用 Docker Compose**
- 配置文件：`docker-compose.yml`
- 启动命令：`docker compose up -d`
- 包含所有依赖服务：MySQL、Redis、MongoDB、RocketMQ、监控服务

#### 本地开发服务清单
```yaml
# 数据库服务
- MySQL (3306)
- Redis (6379) 
- MongoDB (27017)

# 消息队列
- RocketMQ NameServer (9876)
- RocketMQ Broker (10911, 10909)
- RocketMQ Console (8080)

# 监控服务
- Prometheus (9091)
- Grafana (3000)
- Jaeger (16687)
```

### 线上生产环境
- **使用 Kubernetes**
- 配置文件目录：`k8s/`
- 部署脚本：`k8s/deploy.sh`
- 快速部署：`k8s/quick-deploy.sh`

#### Kubernetes部署文件
```
k8s/
├── configmap.yaml         # 配置映射
├── dependencies.yaml       # 依赖服务部署
├── deployment.yaml         # 应用部署
├── kind-config.yaml        # Kind集群配置
├── deploy.sh              # 部署脚本
└── quick-deploy.sh         # 快速部署脚本
```

### 环境切换规范
1. **开发阶段**：使用 Docker Compose
   ```bash
   docker compose up -d
   ```

2. **测试阶段**：使用 Kind 本地 Kubernetes
   ```bash
   cd k8s && ./quick-deploy.sh
   ```

3. **生产部署**：使用生产 Kubernetes 集群
   ```bash
   cd k8s && ./deploy.sh
   ```

## 4. 开发流程规范

### 重构开发流程
当用户说【开始计划】时，触发以下自动化流程：

#### 4.1 计划检查阶段
1. **检查执行计划**：自动读取 `execution_checklist.md` 和 `complete_refactoring_plan.md`
2. **确认当前任务**：识别当前优先级任务和待完成模块
3. **验证依赖关系**：检查前置任务是否完成
4. **评估资源状态**：确认开发环境和依赖服务状态

#### 4.2 重构执行阶段
1. **创建必要文件**：根据计划创建所需的代码文件
2. **实现核心功能**：按照模块设计实现业务逻辑
3. **编写测试代码**：确保单元测试覆盖率>80%
4. **运行验证测试**：执行测试并验证功能正确性
5. **性能基准测试**：运行基准测试验证性能指标

#### 4.3 计划更新阶段
1. **更新完成状态**：将已完成的任务标记为 ✅
2. **更新进度百分比**：重新计算项目整体完成度
3. **调整优先级**：标记下一个优先级任务为 ⭐
4. **记录完成总结**：添加模块完成总结和技术特性
5. **制定下步计划**：明确下一阶段的具体行动计划

#### 4.4 流程验收标准
- [ ] 执行计划文档已更新
- [ ] 代码实现符合设计规范
- [ ] 测试覆盖率达到要求
- [ ] 项目可正常构建和运行
- [ ] 下一步计划已明确

### 服务启动顺序
1. 启动基础设施服务（数据库、消息队列）
2. 等待服务就绪（健康检查）
3. 启动应用服务
4. 验证服务连通性

### 配置管理
- 本地开发：使用 `configs/config.yaml`
- Kubernetes：使用 ConfigMap 和 Secret
- 环境变量优先级高于配置文件

### 日志规范
- 使用结构化日志（JSON格式）
- 日志级别：DEBUG < INFO < WARN < ERROR
- 生产环境默认 INFO 级别
- 开发环境可使用 DEBUG 级别

## 5. 监控和观测性

### 指标监控
- Prometheus 收集指标
- Grafana 展示仪表板
- 关键指标：QPS、延迟、错误率、资源使用率

### 链路追踪
- 使用 Jaeger 进行分布式链路追踪
- 所有外部调用都要添加 trace

### 健康检查
- HTTP 健康检查端点：`/health`
- Kubernetes 就绪探针和存活探针
- 依赖服务连通性检查