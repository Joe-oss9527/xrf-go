#!/bin/bash

# XRF-Go 构建脚本
# 支持多平台交叉编译

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 日志函数
info() {
    echo -e "${BLUE}[BUILD]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# 项目信息
PROJECT_NAME="XRF-Go"
BINARY_NAME="xrf"
VERSION=${VERSION:-"v1.0.0"}
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT=${GIT_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}

# 构建标志
BUILD_FLAGS="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}"

# 支持的平台（专注于主流Linux服务器平台）
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
)

# 检查环境
check_environment() {
    info "检查构建环境..."
    
    # 检查 Go 版本
    if ! command -v go &> /dev/null; then
        error "Go 环境未安装，请先安装 Go 1.21+"
    fi
    
    local go_version
    go_version=$(go version | grep -o 'go[0-9.]*' | sed 's/go//')
    local major
    major=$(echo "$go_version" | cut -d. -f1)
    local minor
    minor=$(echo "$go_version" | cut -d. -f2)
    
    if [[ $major -lt 1 ]] || [[ $major -eq 1 && $minor -lt 21 ]]; then
        error "需要 Go 1.21+，当前版本: $go_version"
    fi
    
    info "Go 版本: $go_version ✓"
    
    # 检查项目结构
    if [[ ! -f "cmd/xrf/main.go" ]]; then
        error "项目结构不正确，请在项目根目录运行此脚本"
    fi
    
    # 检查依赖
    info "检查项目依赖..."
    go mod tidy
    go mod verify
    success "依赖检查完成"
}

# 运行测试
run_tests() {
    if [[ "${SKIP_TESTS}" == "true" ]]; then
        warning "跳过测试"
        return
    fi
    
    info "运行测试..."
    
    # 运行单元测试
    go test -v ./... || {
        warning "部分测试失败，但继续构建"
    }
    
    success "测试完成"
}

# 构建单个平台
build_single() {
    local platform="$1"
    local goos
    goos=$(echo "$platform" | cut -d'/' -f1)
    local goarch
    goarch=$(echo "$platform" | cut -d'/' -f2)
    
    # 架构处理 (仅支持 amd64 和 arm64)
    
    local output_name="${BINARY_NAME}-${goos}-${goarch}"
    if [[ $goos == "windows" ]]; then
        output_name="${output_name}.exe"
    fi
    
    local output_path="dist/${output_name}"
    
    info "构建 ${goos}/${goarch}..."
    
    # 设置环境变量并构建
    env GOOS="$goos" GOARCH="$goarch" \
        go build -trimpath -ldflags="$BUILD_FLAGS" \
        -o "$output_path" \
        cmd/xrf/main.go
    
    # 计算文件大小
    local size
    size=$(ls -lh "$output_path" | awk '{print $5}')
    success "构建完成: ${output_name} (${size})"
    
    # 生成校验和
    if command -v sha256sum &> /dev/null; then
        (cd dist && sha256sum "$(basename "$output_path")") >> dist/checksums.txt
    fi
}

# 构建所有平台
build_all() {
    info "开始多平台构建..."
    
    # 创建输出目录
    mkdir -p dist
    rm -f dist/checksums.txt
    
    # 构建信息文件
    cat > dist/build-info.txt << EOF
${PROJECT_NAME} 构建信息
========================

版本: ${VERSION}
构建时间: ${BUILD_TIME}
Git 提交: ${GIT_COMMIT}
Go 版本: $(go version)
构建平台: $(uname -s)/$(uname -m)

支持平台:
EOF
    
    # 构建每个平台
    for platform in "${PLATFORMS[@]}"; do
        if ! build_single "$platform"; then
            warning "平台 $platform 构建失败，跳过"
            continue
        fi
        echo "- $platform" >> dist/build-info.txt
    done
    
    success "所有平台构建完成！"
    
    # 显示结果
    info "构建结果:"
    ls -la dist/
}

# 构建当前平台
build_current() {
    local goos=$(go env GOOS)
    local goarch=$(go env GOARCH)
    local platform="${goos}/${goarch}"
    
    info "构建当前平台: $platform"
    
    mkdir -p dist
    build_single "$platform"
}

# 清理构建文件
clean() {
    info "清理构建文件..."
    rm -rf dist/
    go clean -cache
    success "清理完成"
}

# 创建发布包
create_release() {
    if [[ ! -d "dist" ]] || [[ -z "$(ls -A dist/)" ]]; then
        error "没有找到构建文件，请先运行构建"
    fi
    
    info "创建发布包..."
    
    local release_dir="dist/release"
    mkdir -p "$release_dir"
    
    # 为每个二进制文件创建压缩包
    for binary in dist/xrf-*; do
        if [[ -f "$binary" && ! "$binary" =~ \.(txt|md)$ ]]; then
            local basename
            basename=$(basename "$binary")
            local archive_name
            archive_name="${basename}.tar.gz"
            
            # 复制到临时目录
            local temp_dir
            temp_dir=$(mktemp -d)
            cp "$binary" "$temp_dir/xrf"
            cp README.md "$temp_dir/" 2>/dev/null || true
            cp LICENSE "$temp_dir/" 2>/dev/null || true
            
            # 创建压缩包
            (cd "$temp_dir" && tar -czf "$archive_name" *)
            mv "$temp_dir/$archive_name" "$release_dir/"
            
            rm -rf "$temp_dir"
            info "创建发布包: $archive_name"
        fi
    done
    
    # 复制校验和文件
    if [[ -f "dist/checksums.txt" ]]; then
        cp dist/checksums.txt "$release_dir/"
    fi
    
    success "发布包创建完成: $release_dir"
}

# 显示帮助信息
show_help() {
    echo "XRF-Go 构建脚本"
    echo
    echo "用法: $0 [选项]"
    echo
    echo "选项:"
    echo "  build-all      构建所有支持的平台"
    echo "  build-current  构建当前平台"
    echo "  test          运行测试"
    echo "  clean         清理构建文件"
    echo "  release       创建发布包"
    echo "  help          显示此帮助信息"
    echo
    echo "环境变量:"
    echo "  VERSION       设置版本号 (默认: v1.0.0)"
    echo "  SKIP_TESTS    跳过测试 (true/false)"
    echo "  GIT_COMMIT    设置 Git 提交哈希"
    echo
    echo "示例:"
    echo "  $0 build-all                    # 构建所有平台"
    echo "  VERSION=v1.1.0 $0 build-all   # 指定版本构建"
    echo "  SKIP_TESTS=true $0 build-all  # 跳过测试构建"
}

# 主函数
main() {
    case "${1:-build-current}" in
        "build-all")
            check_environment
            run_tests
            build_all
            ;;
        "build-current")
            check_environment
            run_tests
            build_current
            ;;
        "test")
            check_environment
            run_tests
            ;;
        "clean")
            clean
            ;;
        "release")
            create_release
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            error "未知命令: $1\n运行 '$0 help' 查看帮助"
            ;;
    esac
}

# 运行主函数
main "$@"
