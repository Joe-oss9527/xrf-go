#!/bin/bash

# XRF-Go 一键安装脚本
# Usage: curl -fsSL https://github.com/Joe-oss9527/xrf-go/releases/latest/download/install.sh | bash

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
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

# 检查用户权限 - 支持root用户和sudo权限
check_root() {
    if [[ $EUID -eq 0 ]]; then
        info "检测到 root 用户，直接执行安装"
        SUDO_CMD=""
    elif command -v sudo >/dev/null 2>&1; then
        info "检测到普通用户，将使用 sudo 权限"
        SUDO_CMD="sudo"
        
        # 验证sudo权限
        if ! sudo -n true 2>/dev/null; then
            warning "此脚本需要 sudo 权限来安装系统文件"
            echo "请输入您的密码以继续安装："
            if ! sudo -v; then
                error "无法获取 sudo 权限，安装终止"
            fi
        fi
        success "sudo 权限验证通过"
    else
        error "此安装脚本需要 root 权限或 sudo 命令
        
解决方案：
  1. 使用 root 用户运行: sudo bash install.sh
  2. 或安装 sudo: apt install sudo (Debian/Ubuntu) 或 yum install sudo (CentOS/RHEL)"
    fi
}

# 检测系统信息
detect_system() {
    if [[ -f /etc/os-release ]]; then
        source /etc/os-release
        OS=$ID
        VER=$VERSION_ID
    else
        error "无法检测操作系统版本"
    fi
    
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64)
            ARCH="arm64"
            ;;
        *)
            error "不支持的架构: $ARCH (仅支持 x86_64 和 aarch64)"
            ;;
    esac
    
    info "检测到系统: $OS $VER ($ARCH)"
}

# 检查依赖
check_dependencies() {
    info "检查系统依赖..."
    
    local deps=("curl" "unzip" "tar" "systemctl")
    for dep in "${deps[@]}"; do
        if ! command -v $dep &> /dev/null; then
            error "缺少依赖: $dep，请先安装"
        fi
    done
    
    success "系统依赖检查通过"
}

# 下载并安装 Xray
install_xray() {
    info "正在安装 Xray..."
    
    # 动态获取最新版本
    info "获取 Xray 最新版本..."
    local xray_version=$(curl -fsSL "https://api.github.com/repos/XTLS/Xray-core/releases/latest" | grep '"tag_name":' | cut -d'"' -f4)
    
    if [[ -z "$xray_version" ]]; then
        warning "无法获取最新版本，使用默认版本 v25.8.31"
        xray_version="v25.8.31"
    fi
    
    local base_url="https://github.com/XTLS/Xray-core/releases/download/${xray_version}"
    local temp_dir=$(mktemp -d)
    
    # 根据架构尝试多个可能的资产文件名以避免404
    local candidates=()
    if [[ "$ARCH" == "amd64" ]]; then
        candidates+=("Xray-linux-64.zip" "Xray-linux-amd64.zip")
    else
        candidates+=("Xray-linux-arm64-v8a.zip" "Xray-linux-arm64.zip")
    fi
    
    info "下载 Xray ${xray_version} for ${ARCH}..."
    local downloaded=""
    for fname in "${candidates[@]}"; do
        local url="${base_url}/${fname}"
        if curl -fsSL -o "${temp_dir}/xray.zip" "$url"; then
            downloaded="$fname"
            break
        fi
    done
    
    if [[ -z "$downloaded" ]]; then
        error "下载 Xray 失败，请检查网络连接或稍后重试"
    fi
    
    cd "$temp_dir"
    unzip -q xray.zip
    
    $SUDO_CMD install -m 755 xray /usr/local/bin/
    $SUDO_CMD install -m 644 geoip.dat /usr/local/bin/ 2>/dev/null || true
    $SUDO_CMD install -m 644 geosite.dat /usr/local/bin/ 2>/dev/null || true
    
    rm -rf "$temp_dir"
    
    success "Xray 安装完成: $(xray version | head -1)"
}

# 下载并安装 XRF-Go
install_xrf_go() {
    info "正在安装 XRF-Go..."
    
    # 获取最新 Release 版本并从 GitHub Releases 下载预编译的二进制文件
    local xrf_version=$(curl -fsSL "https://api.github.com/repos/Joe-oss9527/xrf-go/releases/latest" | grep '"tag_name":' | cut -d'"' -f4)
    if [[ -z "$xrf_version" ]]; then
        warning "无法获取 XRF-Go 最新版本，使用默认版本 v1.0.0"
        xrf_version="v1.0.0"
    fi
    
    local base_url="https://github.com/Joe-oss9527/xrf-go/releases/download/${xrf_version}"
    local temp_dir=$(mktemp -d)
    local downloaded=""
    
    # 优先下载 tar.gz 归档，其次尝试裸二进制
    info "下载 XRF-Go ${xrf_version} for ${ARCH}..."
    for fname in "xrf-linux-${ARCH}.tar.gz" "xrf-linux-${ARCH}"; do
        local url="${base_url}/${fname}"
        if curl -fsSL -o "${temp_dir}/${fname}" "$url" 2>/dev/null; then
            downloaded="$fname"
            break
        fi
    done
    
    if [[ -z "$downloaded" ]]; then
        warning "预编译版本不可用，将从源码编译..."
        rm -rf "$temp_dir"
        compile_from_source
        return
    fi
    
    # 解压或直接安装
    if [[ "$downloaded" == *.tar.gz ]]; then
        tar -xzf "${temp_dir}/${downloaded}" -C "$temp_dir"
        # 解包后文件名为 xrf-linux-${ARCH}
        $SUDO_CMD install -m 755 "${temp_dir}/xrf-linux-${ARCH}" /usr/local/bin/xrf
    else
        $SUDO_CMD install -m 755 "${temp_dir}/${downloaded}" /usr/local/bin/xrf
    fi
    
    rm -rf "$temp_dir"
    success "XRF-Go 安装完成: $(xrf version | grep 'XRF-Go 版本')"
}

# 从源码编译（备用方案）
compile_from_source() {
    info "从源码编译 XRF-Go..."
    
    # 检查 Go 环境
    if ! command -v go &> /dev/null; then
        error "需要 Go 1.23+ 环境来编译 XRF-Go"
    fi
    
    local go_version=$(go version | grep -o 'go[0-9.]*' | head -1)
    info "检测到 Go 版本: $go_version"
    
    local temp_dir=$(mktemp -d)
    cd "$temp_dir"
    
    info "克隆源码..."
    git clone https://github.com/Joe-oss9527/xrf-go.git || error "克隆源码失败"
    
    cd xrf-go
    info "编译中..."
    go build -ldflags="-s -w" -o xrf cmd/xrf/main.go
    
    $SUDO_CMD install -m 755 xrf /usr/local/bin/
    
    rm -rf "$temp_dir"
    success "XRF-Go 编译安装完成"
}

# 创建配置目录
setup_config() {
    info "设置配置目录..."
    
    $SUDO_CMD mkdir -p /etc/xray/confs
    $SUDO_CMD chown $(whoami):$(whoami) /etc/xray/confs
    
    success "配置目录创建完成: /etc/xray/confs"
}

# 创建 systemd 服务
setup_service() {
    info "配置 Xray 系统服务..."
    
    local service_content='[Unit]
Description=Xray Service (XRF-Go managed)
Documentation=https://xtls.github.io/
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/xray run -confdir /etc/xray/confs
Restart=on-failure
RestartSec=3
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target'
    
    echo "$service_content" | $SUDO_CMD tee /etc/systemd/system/xray.service >/dev/null
    
    $SUDO_CMD systemctl daemon-reload
    $SUDO_CMD systemctl enable xray.service
    
    success "Xray 服务配置完成"
}

# 初始化配置并启动服务
init_config() {
    info "初始化 XRF-Go 配置并启动服务..."
    
    # 直接使用xrf install，它会自动配置VLESS-REALITY并启动服务
    if xrf install 2>/dev/null; then
        success "XRF-Go 配置完成，服务已启动"
    else
        # 如果install命令失败，尝试基本初始化
        warning "xrf install 失败，尝试手动初始化..."
        xrf --confdir /etc/xray/confs >/dev/null 2>&1 || true
        info "请稍后手动运行 'xrf install' 来配置协议"
    fi
}

# 优化系统
optimize_system() {
    info "优化系统设置..."
    
    # 启用 BBR（如果支持）
    if [[ -f /proc/sys/net/ipv4/tcp_congestion_control ]]; then
        if ! sysctl net.ipv4.tcp_congestion_control | grep -q bbr; then
            echo 'net.core.default_qdisc = fq' | $SUDO_CMD tee -a /etc/sysctl.conf
            echo 'net.ipv4.tcp_congestion_control = bbr' | $SUDO_CMD tee -a /etc/sysctl.conf
            info "BBR 拥塞控制已配置（重启后生效）"
        else
            info "BBR 拥塞控制已启用"
        fi
    fi
    
    # 优化文件描述符限制（避免重复）
    if ! grep -qE '^\*\s+soft\s+nofile\s+1048576$' /etc/security/limits.conf 2>/dev/null; then
        echo '* soft nofile 1048576' | $SUDO_CMD tee -a /etc/security/limits.conf >/dev/null
    fi
    if ! grep -qE '^\*\s+hard\s+nofile\s+1048576$' /etc/security/limits.conf 2>/dev/null; then
        echo '* hard nofile 1048576' | $SUDO_CMD tee -a /etc/security/limits.conf >/dev/null
    fi
    
    success "系统优化完成"
}

# 显示安装完成信息
show_completion() {
    echo
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN} XRF-Go 安装完成！${NC}"
    if [[ $EUID -eq 0 ]]; then
        echo -e "${GREEN} (已使用 root 用户权限安装)${NC}"
    else
        echo -e "${GREEN} (已使用 sudo 权限安装)${NC}"
    fi
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo
    echo -e "${BLUE}快速开始:${NC}"
    echo "  1. 查看配置:    xrf list                 # VLESS-REALITY已配置并启动"
    echo "  2. 获取链接:    xrf url vless_reality    # 生成客户端连接链接"
    echo "  3. 添加更多:    xrf add vr --port 8443   # 添加更多协议"
    echo "  4. 查看状态:    xrf status               # 检查服务状态"
    echo "  5. 查看日志:    xrf logs                 # 查看运行日志"
    echo
    echo -e "${BLUE}常用命令:${NC}"
    echo "  • xrf add [protocol]     - 添加协议配置"
    echo "  • xrf remove [tag]       - 删除协议配置"
    echo "  • xrf list              - 列出所有协议"
    echo "  • xrf generate [type]   - 生成密码/UUID/密钥"
    echo "  • xrf test              - 验证配置"
    echo
    echo -e "${BLUE}支持的协议:${NC}"
    echo "  • vr     - VLESS-REALITY (推荐)"
    echo "  • vw     - VLESS-WebSocket-TLS"
    echo "  • vmess  - VMess-WebSocket-TLS"
    echo "  • tw     - Trojan-WebSocket-TLS"
    echo "  • ss     - Shadowsocks"
    echo "  • ss2022 - Shadowsocks-2022"
    echo "  • hu     - VLESS-HTTPUpgrade"
    echo
    echo -e "${YELLOW}文档和支持:${NC}"
    echo "  • GitHub: https://github.com/Joe-oss9527/xrf-go"
    echo "  • 官方文档: https://xtls.github.io/"
    echo
}

# 主函数
main() {
    echo -e "${BLUE}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo " XRF-Go 一键安装脚本"
    echo " 简洁高效的 Xray 安装配置工具"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "${NC}"
    
    # 检查系统
    check_root
    detect_system
    check_dependencies
    
    # 安装组件
    install_xray
    install_xrf_go
    
    # 配置系统
    setup_config
    setup_service
    init_config
    optimize_system
    
    # 完成
    show_completion
}

# 运行主函数
main "$@"
