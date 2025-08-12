# Kratos Project Template

## Install Kratos
```
go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
```
## Create a service
```
# Create a template project
kratos new server

cd server
# Add a proto template
kratos proto add api/server/server.proto
# Generate the proto code
kratos proto client api/server/server.proto
# Generate the source code of service by proto file
kratos proto server api/server/server.proto -t internal/service

go generate ./...
go build -o ./bin/ ./...
./bin/server -conf ./configs
```
## Generate other auxiliary files by Makefile
```
# Download and update dependencies
make init
# Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file
make api
# Generate all files
make all
```
## Automated Initialization (wire)
```
# install wire
go get github.com/google/wire/cmd/wire

# generate wire
cd cmd/server
wire
```

## 部署方式

### Docker Compose 部署

使用 Docker Compose 可以快速启动完整的服务栈：

```bash
# 启动所有服务（MySQL、Redis、Jaeger、Prometheus、Grafana）
docker compose up -d

# 构建并启动应用
go build -o bin/kratos-demo ./cmd/kratos-demo
./bin/kratos-demo -conf ./configs
```

访问地址：
- 应用 HTTP API: http://localhost:8000
- Jaeger UI: http://localhost:16686
- Prometheus: http://localhost:9091
- Grafana: http://localhost:3000 (admin/admin)

### Kubernetes 部署

使用 Kind 进行本地 Kubernetes 部署：

```bash
# 快速部署（推荐）
cd k8s
./quick-deploy.sh

# 或者完整部署
./deploy.sh
```

访问地址：
- 应用 HTTP API: http://localhost:8080
- Jaeger UI: http://localhost:16686
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)

详细的 Kubernetes 部署说明请参考 [k8s/README.md](k8s/README.md)

### Docker 单容器部署

```bash
# build
docker build -t kratos-demo .

# run
docker run --rm -p 8000:8000 -p 9000:9000 -p 9090:9090 -v $(pwd)/configs:/data/conf kratos-demo
```

