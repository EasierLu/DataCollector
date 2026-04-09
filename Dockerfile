# 阶段1：构建前端
FROM node:20-alpine AS frontend-builder

WORKDIR /app/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

# 阶段2：构建后端
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 复制前端构建产物到 Go embed 目录
COPY --from=frontend-builder /app/web/dist ./internal/web/dist/

# CGO 需要启用（SQLite 需要）
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags "-s -w" -o datacollector ./cmd/server

# 阶段3：运行
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/datacollector .
COPY --from=builder /app/configs/config.yaml ./configs/

# 创建数据和日志目录
RUN mkdir -p /app/data /app/logs && chown -R appuser:appgroup /app/data /app/logs

EXPOSE 8080

# 使用环境变量覆盖默认配置
ENV DB_DRIVER=sqlite
ENV DB_SQLITE_PATH=/app/data/datacollector.db
ENV LOG_OUTPUT=file
ENV LOG_FILE_PATH=/app/logs/datacollector.log

USER appuser

ENTRYPOINT ["./datacollector"]
