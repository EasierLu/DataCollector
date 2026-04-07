.PHONY: build run test clean docker-build

# 变量
BINARY_NAME=datacollector
VERSION=1.0.0
BUILD_DIR=dist
MAIN_PATH=./cmd/server

# 默认构建（当前平台）
build:
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

# Docker 构建
docker-build:
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

# 多平台构建
build-all: clean
	./scripts/build.sh
