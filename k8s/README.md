# Kubernetes 部署指南

本目录包含了使用 Kubernetes 部署 Kratos Demo 应用的所有配置文件和脚本。

## 前置要求

- [Docker](https://docs.docker.com/get-docker/)
- [kind](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

### 安装 kind

```bash
# 使用 Go 安装
go install sigs.k8s.io/kind@v0.20.0

# 或者下载二进制文件
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind
```

## 快速开始

### 一键部署

```bash
cd k8s
./deploy.sh
```

这个脚本会自动完成以下操作：
1. 创建 kind 集群
2. 构建应用 Docker 镜像
3. 部署所有依赖服务（MySQL、Redis、Jaeger、Prometheus、Grafana）
4. 部署 Kratos 应用
5. 等待所有服务就绪

### 手动部署

如果你想手动控制部署过程：

#### 1. 创建 Kind 集群

```bash
kind create cluster --config=kind-config.yaml
```

#### 2. 构建并加载应用镜像

```bash
# 构建镜像
docker build -t kratos-demo:latest ..

# 加载到 kind 集群
kind load docker-image kratos-demo:latest --name kratos-demo
```

#### 3. 部署依赖服务

```bash
kubectl apply -f dependencies.yaml
```

#### 4. 等待依赖服务就绪

```bash
kubectl wait --for=condition=available --timeout=300s deployment/mysql
kubectl wait --for=condition=available --timeout=300s deployment/redis
kubectl wait --for=condition=available --timeout=300s deployment/jaeger
kubectl wait --for=condition=available --timeout=300s deployment/prometheus
kubectl wait --for=condition=available --timeout=300s deployment/grafana
```

#### 5. 部署应用

```bash
# 部署配置
kubectl apply -f configmap.yaml

# 部署应用
kubectl apply -f deployment.yaml

# 等待应用就绪
kubectl wait --for=condition=available --timeout=300s deployment/kratos-demo
```

## 访问服务

部署完成后，可以通过以下地址访问各个服务：

- **Kratos HTTP API**: http://localhost:8080
- **Jaeger UI**: http://localhost:16686
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (用户名/密码: admin/admin)

## 测试应用

```bash
# 测试 HTTP API
curl http://localhost:8080/helloworld/kratos

# 查看应用日志
kubectl logs -f deployment/kratos-demo

# 查看所有 Pod 状态
kubectl get pods

# 查看服务状态
kubectl get svc
```

## 文件说明

- `kind-config.yaml`: Kind 集群配置，定义了端口映射
- `dependencies.yaml`: 依赖服务部署配置（MySQL、Redis、Jaeger、Prometheus、Grafana）
- `configmap.yaml`: 应用配置文件
- `deployment.yaml`: Kratos 应用部署配置
- `deploy.sh`: 一键部署脚本

## 架构说明

### 服务组件

- **kratos-demo**: 主应用，2个副本
- **mysql**: MySQL 8.0 数据库
- **redis**: Redis 7 缓存
- **jaeger**: 分布式链路追踪
- **prometheus**: 监控指标收集
- **grafana**: 监控数据可视化

### 网络配置

- HTTP 服务通过 NodePort 30080 暴露到主机 8080 端口
- Jaeger UI 通过 NodePort 30686 暴露到主机 16686 端口
- Prometheus 通过 NodePort 30090 暴露到主机 9090 端口
- Grafana 通过 NodePort 30000 暴露到主机 3000 端口

### 存储

所有服务使用 `emptyDir` 卷进行临时存储，集群删除后数据会丢失。生产环境建议使用持久化存储。

## 故障排查

### 查看 Pod 状态

```bash
kubectl get pods
kubectl describe pod <pod-name>
```

### 查看应用日志

```bash
kubectl logs -f deployment/kratos-demo
kubectl logs -f deployment/mysql
kubectl logs -f deployment/redis
```

### 进入容器调试

```bash
kubectl exec -it deployment/kratos-demo -- /bin/sh
```

### 检查服务连通性

```bash
# 在应用容器中测试数据库连接
kubectl exec -it deployment/kratos-demo -- nc -zv mysql 3306

# 测试 Redis 连接
kubectl exec -it deployment/kratos-demo -- nc -zv redis 6379
```

## 清理资源

```bash
# 删除应用
kubectl delete -f deployment.yaml
kubectl delete -f configmap.yaml

# 删除依赖服务
kubectl delete -f dependencies.yaml

# 删除整个集群
kind delete cluster --name kratos-demo
```

## 生产环境注意事项

1. **持久化存储**: 使用 PersistentVolume 替代 emptyDir
2. **资源限制**: 根据实际需求调整 CPU 和内存限制
3. **安全配置**: 配置网络策略、RBAC 等安全措施
4. **高可用**: 增加副本数量，配置反亲和性
5. **监控告警**: 配置 Grafana 告警规则
6. **备份策略**: 定期备份数据库和配置

## 扩展功能

### 水平扩展

```bash
# 扩展应用副本
kubectl scale deployment kratos-demo --replicas=5

# 查看扩展状态
kubectl get pods -l app=kratos-demo
```

### 滚动更新

```bash
# 更新镜像
kubectl set image deployment/kratos-demo kratos-demo=kratos-demo:v2

# 查看更新状态
kubectl rollout status deployment/kratos-demo

# 回滚更新
kubectl rollout undo deployment/kratos-demo
```