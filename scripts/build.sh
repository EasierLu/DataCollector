#!/bin/bash

# DataCollector 多平台构建脚本

set -e

VERSION="${VERSION:-1.0.0}"
BINARY_NAME="datacollector"
BUILD_DIR="dist"
MAIN_PATH="./cmd/server"
LDFLAGS="-s -w -X main.version=${VERSION}"

# 清理
rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

echo "=== Building DataCollector v${VERSION} ==="

# 定义目标平台
PLATFORMS=(
    "windows/amd64/.exe"
    "windows/arm64/.exe"
    "darwin/amd64/"
    "darwin/arm64/"
    "linux/amd64/"
    "linux/arm64/"
    "linux/arm/v7"
)

for PLATFORM in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH EXTRA <<< "${PLATFORM}"
    
    OUTPUT_NAME="${BINARY_NAME}-v${VERSION}-${GOOS}-${GOARCH}"
    
    # 处理 ARM v7
    GOARM=""
    if [ "${EXTRA}" = "v7" ]; then
        GOARM=7
        OUTPUT_NAME="${BINARY_NAME}-v${VERSION}-${GOOS}-armv7"
    fi
    
    # 处理 Windows 后缀
    if [ "${GOOS}" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi
    
    echo "Building ${OUTPUT_NAME}..."
    
    # SQLite 需要 CGO，交叉编译时需要对应的 C 编译器
    # 对于非 Linux/amd64 平台，禁用 CGO（将使用纯 Go SQLite 驱动，如 modernc.org/sqlite）
    CGO_FLAG=0
    if [ "${GOOS}" = "linux" ] && [ "${GOARCH}" = "amd64" ]; then
        CGO_FLAG=1
    fi
    
    CGO_ENABLED=${CGO_FLAG} GOOS=${GOOS} GOARCH=${GOARCH} GOARM=${GOARM} \
        go build -ldflags "${LDFLAGS}" -o "${BUILD_DIR}/${OUTPUT_NAME}" ${MAIN_PATH}
done

echo ""
echo "=== Build complete ==="
ls -lh "${BUILD_DIR}/"
echo ""
echo "All binaries are in ${BUILD_DIR}/"
