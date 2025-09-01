# XRF-Go Makefile
# 简洁高效的构建配置

.PHONY: all build test clean install uninstall dev run fmt lint vet tidy check help

# 项目信息
PROJECT_NAME := XRF-Go
BINARY_NAME := xrf
MAIN_PATH := cmd/xrf/main.go

# 版本信息
VERSION ?= v1.0.0
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 构建标志
BUILD_FLAGS := -s -w
BUILD_FLAGS += -X main.Version=$(VERSION)
BUILD_FLAGS += -X main.BuildTime=$(BUILD_TIME) 
BUILD_FLAGS += -X main.GitCommit=$(GIT_COMMIT)

# 目录
DIST_DIR := dist
SCRIPTS_DIR := scripts

# Go 设置
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt

# 默认目标
all: check test build

# 构建当前平台
build:
	@echo "🔨 构建 $(BINARY_NAME)..."
	@mkdir -p $(DIST_DIR)
	$(GOBUILD) -trimpath -ldflags="$(BUILD_FLAGS)" -o $(DIST_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ 构建完成: $(DIST_DIR)/$(BINARY_NAME)"

# 构建所有平台
build-all:
	@echo "🚀 多平台构建..."
	@$(SCRIPTS_DIR)/build.sh build-all

# 运行测试
test:
	@echo "🧪 运行测试..."
	CGO_ENABLED=1 $(GOTEST) -v -race -coverprofile=coverage.out ./... || \
	$(GOTEST) -v -coverprofile=coverage.out ./...
	@echo "✅ 测试完成"

# 基准测试
bench:
	@echo "⚡ 运行基准测试..."
	$(GOTEST) -bench=. -benchmem ./...

# 代码检查
check: fmt vet lint tidy

# 格式化代码
fmt:
	@echo "🎨 格式化代码..."
	$(GOFMT) ./...

# Go vet 检查
vet:
	@echo "🔍 静态分析..."
	$(GOCMD) vet ./...

# 代码规范检查 (如果有 golangci-lint)
lint:
	@echo "📝 代码规范检查..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint 未安装，跳过 lint 检查"; \
	fi

# 整理依赖
tidy:
	@echo "📦 整理依赖..."
	$(GOMOD) tidy
	$(GOMOD) verify

# 开发模式运行
dev: build
	@echo "🏃 开发模式运行..."
	@./$(DIST_DIR)/$(BINARY_NAME) --help

# 直接运行（不构建）
run:
	@echo "🏃 直接运行..."
	$(GOCMD) run $(MAIN_PATH) --help

# 安装到系统
install: build
	@echo "📦 安装到系统..."
	@sudo install -m 755 $(DIST_DIR)/$(BINARY_NAME) /usr/local/bin/
	@echo "✅ 已安装到 /usr/local/bin/$(BINARY_NAME)"

# 从系统卸载
uninstall:
	@echo "🗑️  从系统卸载..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ 卸载完成"

# 清理构建文件
clean:
	@echo "🧹 清理构建文件..."
	@rm -rf $(DIST_DIR)/
	@rm -f coverage.out
	@$(GOCMD) clean -cache
	@echo "✅ 清理完成"

# 创建发布包
release: build-all
	@echo "📦 创建发布包..."
	@$(SCRIPTS_DIR)/build.sh release

# 快速开发循环（格式化+测试+构建）
quick: fmt test build
	@echo "✅ 快速开发循环完成"

# 性能分析
profile:
	@echo "📊 生成性能分析..."
	$(GOTEST) -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...
	@echo "✅ 性能分析完成: cpu.prof, mem.prof"

# 统计代码行数
stats:
	@echo "📈 代码统计..."
	@find . -name '*.go' -not -path './vendor/*' | xargs wc -l | tail -1
	@echo "文件数: $$(find . -name '*.go' -not -path './vendor/*' | wc -l)"

# Docker 构建（如果需要）
docker-build:
	@echo "🐳 Docker 构建..."
	docker build -t $(PROJECT_NAME):$(VERSION) .

# 帮助信息
help:
	@echo "$(PROJECT_NAME) Makefile"
	@echo ""
	@echo "可用目标:"
	@echo "  build        构建当前平台的二进制文件"
	@echo "  build-all    构建所有平台的二进制文件"
	@echo "  test         运行测试"
	@echo "  bench        运行基准测试"
	@echo "  check        代码检查 (fmt + vet + lint + tidy)"
	@echo "  fmt          格式化代码"
	@echo "  vet          Go vet 静态分析"
	@echo "  lint         代码规范检查"
	@echo "  tidy         整理依赖"
	@echo "  dev          开发模式（构建+运行）"
	@echo "  run          直接运行（不构建）"
	@echo "  install      安装到系统"
	@echo "  uninstall    从系统卸载"
	@echo "  clean        清理构建文件"
	@echo "  release      创建发布包"
	@echo "  quick        快速开发循环"
	@echo "  profile      性能分析"
	@echo "  stats        代码统计"
	@echo "  help         显示帮助信息"
	@echo ""
	@echo "环境变量:"
	@echo "  VERSION      版本号 (默认: $(VERSION))"
	@echo ""
	@echo "示例:"
	@echo "  make build                构建当前平台"
	@echo "  make test                 运行测试"
	@echo "  VERSION=v1.1.0 make build 指定版本构建"