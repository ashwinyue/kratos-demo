FROM golang:1.23 AS builder

COPY . /src
WORKDIR /src

RUN GOPROXY=https://goproxy.cn go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o bin/kratos-demo ./cmd/kratos-demo

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/bin /app

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
EXPOSE 9090
VOLUME /data/conf

CMD ["./kratos-demo", "-conf", "/data/conf"]
