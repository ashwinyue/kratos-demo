#!/bin/bash

set -e

echo "🚀 快速部署 Kratos Demo 到 Kind 集群"

# 检查必要工具
command -v kind >/dev/null 2>&1 || { echo "❌ 需要安装 kind"; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "❌ 需要安装 kubectl"; exit 1; }
command -v docker >/dev/null 2>&1 || { echo "❌ 需要安装 docker"; exit 1; }

# 创建集群（如果不存在）
if ! kind get clusters | grep -q "kratos-demo"; then
    echo "🏗️  创建 Kind 集群"
    kind create cluster --config=kind-config.yaml
else
    echo "📦 使用现有集群"
fi

# 构建并加载镜像
echo "🐳 构建并加载应用镜像"
cd ..
docker build -t kratos-demo:latest . --quiet
kind load docker-image kratos-demo:latest --name kratos-demo
cd k8s

# 部署所有资源
echo "🚀 部署所有资源"
kubectl apply -f dependencies.yaml
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml

echo "⏳ 等待服务启动（这可能需要几分钟）..."

# 等待关键服务
kubectl wait --for=condition=available --timeout=180s deployment/mysql || echo "⚠️  MySQL 启动超时"
kubectl wait --for=condition=available --timeout=180s deployment/redis || echo "⚠️  Redis 启动超时"
kubectl wait --for=condition=available --timeout=180s deployment/kratos-demo || echo "⚠️  应用启动超时"

echo ""
echo "✅ 部署完成！"
echo ""
echo "🌐 访问地址:"
echo "  - Kratos API: http://localhost:8080"
echo "  - Jaeger: http://localhost:16686"
echo "  - Prometheus: http://localhost:9090"
echo "  - Grafana: http://localhost:3000"
echo ""
echo "🧪 测试命令:"
echo "  curl http://localhost:8080/helloworld/kratos"
echo ""
echo "📊 查看状态:"
kubectl get pods
echo ""
echo "🧹 清理: kind delete cluster --name kratos-demo"