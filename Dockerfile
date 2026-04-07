# 阶段1：构建
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO 需要启用（SQLite 需要）
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-s -w" -o datacollector ./cmd/server

# 阶段2：运行
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/datacollector .
COPY --from=builder /app/configs/config.yaml ./configs/

# 创建数据和日志目录
RUN mkdir -p /app/data /app/logs

EXPOSE 8080

# 使用环境变量覆盖默认配置
ENV DB_DRIVER=sqlite
ENV DB_SQLITE_PATH=/app/data/datacollector.db
ENV LOG_OUTPUT=file
ENV LOG_FILE_PATH=/app/logs/datacollector.log

ENTRYPOINT ["./datacollector"]
