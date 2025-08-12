#!/bin/bash

set -e

echo "🚀 开始部署 Kratos Demo 到 Kind 集群"

# 检查 kind 是否安装
if ! command -v kind &> /dev/null; then
    echo "❌ kind 未安装，请先安装 kind"
    echo "安装命令: go install sigs.k8s.io/kind@v0.20.0"
    exit 1
fi

# 检查 kubectl 是否安装
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl 未安装，请先安装 kubectl"
    exit 1
fi

# 检查 docker 是否运行
if ! docker info &> /dev/null; then
    echo "❌ Docker 未运行，请启动 Docker"
    exit 1
fi

echo "📋 检查现有集群"
if kind get clusters | grep -q "kratos-demo"; then
    echo "⚠️  发现现有 kratos-demo 集群，是否删除并重新创建? (y/N)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        echo "🗑️  删除现有集群"
        kind delete cluster --name kratos-demo
    else
        echo "📦 使用现有集群"
    fi
fi

# 创建 kind 集群
if ! kind get clusters | grep -q "kratos-demo"; then
    echo "🏗️  创建 Kind 集群"
    kind create cluster --config=kind-config.yaml
fi

# 设置 kubectl 上下文
echo "🔧 设置 kubectl 上下文"
kubectl cluster-info --context kind-kratos-demo

# 构建 Docker 镜像
echo "🐳 构建应用镜像"
cd ..
docker build -t kratos-demo:latest .
cd k8s

# 加载镜像到 kind 集群
echo "📦 加载镜像到 Kind 集群"
kind load docker-image kratos-demo:latest --name kratos-demo

# 部署依赖服务
echo "🗄️  部署依赖服务 (MySQL, Redis, Jaeger, Prometheus, Grafana)"
kubectl apply -f dependencies.yaml

# 等待依赖服务就绪
echo "⏳ 等待依赖服务启动..."
kubectl wait --for=condition=available --timeout=300s deployment/mysql
kubectl wait --for=condition=available --timeout=300s deployment/redis
kubectl wait --for=condition=available --timeout=300s deployment/jaeger
kubectl wait --for=condition=available --timeout=300s deployment/prometheus
kubectl wait --for=condition=available --timeout=300s deployment/grafana

# 部署应用配置
echo "⚙️  部署应用配置"
kubectl apply -f configmap.yaml

# 部署应用
echo "🚀 部署 Kratos 应用"
kubectl apply -f deployment.yaml

# 等待应用就绪
echo "⏳ 等待应用启动..."
kubectl wait --for=condition=available --timeout=300s deployment/kratos-demo

# 显示部署状态
echo "📊 部署状态:"
kubectl get pods -l app=kratos-demo
kubectl get svc

echo ""
echo "✅ 部署完成！"
echo ""
echo "🌐 访问地址:"
echo "  - Kratos HTTP API: http://localhost:8080"
echo "  - Jaeger UI: http://localhost:16686"
echo "  - Prometheus: http://localhost:9090"
echo "  - Grafana: http://localhost:3000 (admin/admin)"
echo ""
echo "🔍 查看应用日志:"
echo "  kubectl logs -f deployment/kratos-demo"
echo ""
echo "🧹 清理集群:"
echo "  kind delete cluster --name kratos-demo"
echo ""