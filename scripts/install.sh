#!/bin/bash

# XRF-Go 一键安装脚本
# Usage: curl -fsSL https://get.xrf.sh | bash

set -e

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

# 检查是否为 root 用户
check_root() {
    if [[ $EUID -eq 0 ]]; then
        error "请不要使用 root 用户运行此脚本，使用 sudo 权限即可"
    fi
    
    # 检查 sudo 权限
    if ! sudo -n true 2>/dev/null; then
        warning "此脚本需要 sudo 权限来安装系统文件"
        echo "请输入您的密码："
        sudo -v
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
        armv7l)
            ARCH="armv7"
            ;;
        *)
            error "不支持的架构: $ARCH"
            ;;
    esac
    
    info "检测到系统: $OS $VER ($ARCH)"
}

# 检查依赖
check_dependencies() {
    info "检查系统依赖..."
    
    local deps=("curl" "unzip" "systemctl")
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
    
    local xray_version="v25.8.3"  # 可以动态获取最新版本
    local download_url="https://github.com/XTLS/Xray-core/releases/download/${xray_version}/Xray-linux-${ARCH}.zip"
    local temp_dir=$(mktemp -d)
    
    info "下载 Xray ${xray_version} for ${ARCH}..."
    curl -fsSL -o "${temp_dir}/xray.zip" "$download_url"
    
    cd "$temp_dir"
    unzip -q xray.zip
    
    sudo install -m 755 xray /usr/local/bin/
    sudo install -m 644 geoip.dat /usr/local/bin/ 2>/dev/null || true
    sudo install -m 644 geosite.dat /usr/local/bin/ 2>/dev/null || true
    
    rm -rf "$temp_dir"
    
    success "Xray 安装完成: $(xray version | head -1)"
}

# 下载并安装 XRF-Go
install_xrf_go() {
    info "正在安装 XRF-Go..."
    
    # 这里应该从 GitHub Releases 下载预编译的二进制文件
    # 目前使用本地编译的方式（演示）
    local xrf_version="v1.0.0"
    local download_url="https://github.com/yourusername/xrf-go/releases/download/${xrf_version}/xrf-linux-${ARCH}"
    local temp_file=$(mktemp)
    
    # 如果 release 不存在，则尝试从源码编译
    if ! curl -fsSL -o "$temp_file" "$download_url" 2>/dev/null; then
        warning "预编译版本不可用，将从源码编译..."
        compile_from_source
        return
    fi
    
    sudo install -m 755 "$temp_file" /usr/local/bin/xrf
    rm -f "$temp_file"
    
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
    git clone https://github.com/yourusername/xrf-go.git || error "克隆源码失败"
    
    cd xrf-go
    info "编译中..."
    go build -ldflags="-s -w" -o xrf cmd/xrf/main.go
    
    sudo install -m 755 xrf /usr/local/bin/
    
    rm -rf "$temp_dir"
    success "XRF-Go 编译安装完成"
}

# 创建配置目录
setup_config() {
    info "设置配置目录..."
    
    sudo mkdir -p /etc/xray/confs
    sudo chown $(whoami):$(whoami) /etc/xray/confs
    
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
    
    echo "$service_content" | sudo tee /etc/systemd/system/xray.service >/dev/null
    
    sudo systemctl daemon-reload
    sudo systemctl enable xray.service
    
    success "Xray 服务配置完成"
}

# 初始化配置
init_config() {
    info "初始化 XRF-Go 配置..."
    
    xrf install --no-start 2>/dev/null || {
        # 如果没有 --no-start 选项，使用基本的初始化
        xrf --confdir /etc/xray/confs >/dev/null 2>&1 || true
    }
    
    success "配置初始化完成"
}

# 优化系统
optimize_system() {
    info "优化系统设置..."
    
    # 启用 BBR（如果支持）
    if [[ -f /proc/sys/net/ipv4/tcp_congestion_control ]]; then
        if ! sysctl net.ipv4.tcp_congestion_control | grep -q bbr; then
            echo 'net.core.default_qdisc = fq' | sudo tee -a /etc/sysctl.conf
            echo 'net.ipv4.tcp_congestion_control = bbr' | sudo tee -a /etc/sysctl.conf
            info "BBR 拥塞控制已配置（重启后生效）"
        else
            info "BBR 拥塞控制已启用"
        fi
    fi
    
    # 优化文件描述符限制
    echo '* soft nofile 1048576' | sudo tee -a /etc/security/limits.conf
    echo '* hard nofile 1048576' | sudo tee -a /etc/security/limits.conf
    
    success "系统优化完成"
}

# 显示安装完成信息
show_completion() {
    echo
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN} XRF-Go 安装完成！${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo
    echo -e "${BLUE}快速开始:${NC}"
    echo "  1. 添加协议:    xrf add vr --port 443 --domain your.domain.com"
    echo "  2. 列出协议:    xrf list"
    echo "  3. 启动服务:    sudo systemctl start xray"
    echo "  4. 查看状态:    xrf status"
    echo "  5. 获取帮助:    xrf --help"
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
    echo "  • GitHub: https://github.com/yourusername/xrf-go"
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