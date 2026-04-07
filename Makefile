.PHONY: build run test clean docker-build web-install web-dev web-build

# 变量
BINARY_NAME=datacollector
VERSION=1.0.0
BUILD_DIR=dist
MAIN_PATH=./cmd/server
WEB_DIR=web

# === 前端相关 ===

# 安装前端依赖
web-install:
	cd $(WEB_DIR) && npm install

# 启动前端开发服务器
web-dev:
	cd $(WEB_DIR) && npm run dev

# 构建前端并复制到 Go embed 目录
web-build:
	cd $(WEB_DIR) && npm run build
	rm -rf internal/web/dist
	cp -r $(WEB_DIR)/dist internal/web/dist

# === Go 后端相关 ===

# 默认构建（先构建前端再构建后端）
build: web-build
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# 仅构建后端（跳过前端，用于开发调试）
build-go:
	go build -ldflags "-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# 运行
run:
	go run $(MAIN_PATH)

# 测试
test:
	go test ./... -v

# 清理
clean:
	rm -rf $(BUILD_DIR)
	rm -rf internal/web/dist

# Docker 构建
docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

# 多平台构建
build-all: clean
	./scripts/build.sh
