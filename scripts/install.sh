#!/bin/bash

# XRF-Go 一键安装脚本
# Usage: curl -fsSL https://github.com/Joe-oss9527/xrf-go/releases/latest/download/install.sh | bash

set -euo pipefail

# 检查并加载共享工具函数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
COMMON_SH="${SCRIPT_DIR}/common.sh"

# 如果存在本地 common.sh，则加载它（开发环境）
if [[ -f "$COMMON_SH" ]]; then
    source "$COMMON_SH"
    # 统一本脚本的日志接口
    info() { log_info "$1"; }
    success() { log_success "$1"; }
    warning() { log_warning "$1"; }
    error() { log_error "$1"; exit 1; }
else
    # 生产环境：内嵌必要的工具函数

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

    # 内嵌核心工具函数
    _trim() {
        local s="${1:-}"
        echo "${s}" | sed 's/^\s\+//;s/\s\+$//'
    }

    _is_valid_tag() {
        local tag
        tag="$(_trim "${1:-}")"
        [[ -n "$tag" ]] || return 1
        [[ ! "$tag" =~ ^https?:// ]] || return 1
        [[ ! "$tag" =~ / ]] || return 1
        [[ ! "$tag" =~ : ]] || return 1
        [[ "$tag" =~ ^[A-Za-z0-9._-]+$ ]] || return 1
    }

    extract_tag_name() {
        local release_json="$1"
        if [[ -z "$release_json" ]]; then
            return 1
        fi

        if command -v jq >/dev/null 2>&1; then
            echo "$release_json" | jq -r '.tag_name'
        else
            echo "$release_json" | grep '"tag_name":' | awk -F'"' '{print $4}'
        fi
    }

    get_github_latest_version() {
        local repo="$1"
        local user_agent="${2:-xrf-go-installer}"

        local curl_opts=(
            -fsSL
            -H "Accept: application/vnd.github+json"
            -H "User-Agent: $user_agent"
        )

        if [[ -n "${GITHUB_TOKEN:-}" ]]; then
            curl_opts+=( -H "Authorization: Bearer ${GITHUB_TOKEN}" )
        fi

        local release_json
        release_json=$(curl "${curl_opts[@]}" "https://api.github.com/repos/$repo/releases/latest" 2>/dev/null || echo "")

        if [[ -n "$release_json" ]]; then
            local tag
            tag=$(extract_tag_name "$release_json" || echo "")
            if _is_valid_tag "$tag"; then
                echo "$tag"
                return 0
            fi
        fi

        # fallback: 通过重定向解析最新 tag
        local effective
        effective=$(curl -fsSLI -o /dev/null -w '%{url_effective}' -H "User-Agent: $user_agent" "https://github.com/${repo}/releases/latest" 2>/dev/null || echo "")
        if [[ -n "$effective" ]]; then
            local tag_from_url
            tag_from_url=$(echo "$effective" | sed -n 's#.*/releases/tag/\([^/?]*\).*#\1#p')
            if _is_valid_tag "$tag_from_url"; then
                echo "$tag_from_url"
                return 0
            fi
        fi

        return 1
    }
fi

# 额外保护：本脚本内部也做一次 tag 校验（独立于 common.sh）
is_valid_tag() {
    local tag="$1"
    [[ -n "$tag" ]] && [[ ! "$tag" =~ ^https?:// ]] && [[ ! "$tag" =~ / ]] && [[ ! "$tag" =~ : ]] && [[ "$tag" =~ ^[A-Za-z0-9._-]+$ ]]
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

# 这些函数现在已经在 common.sh 中定义或内嵌在上方

# 从 GitHub Release JSON 中按名称正则选择资产下载地址
# 参数: $1=release_json $2=name_regex (grep -E 风格)
select_asset_url() {
    local release_json="$1"
    local name_regex="$2"

    # 展平 JSON 并定位 assets 数组
    local flat=$(echo "$release_json" | tr -d '\n')
    local assets=$(echo "$flat" | sed -n 's/.*"assets":[[]\(.*\)[]].*/\1/p')
    if [[ -z "$assets" ]]; then
        echo ""; return 1
    fi

    # 按资产对象切分后逐一匹配名称并取其下载链接
    echo "$assets" | sed 's/},[[:space:]]*{/\n/g' | while IFS= read -r block; do
        local name=$(echo "$block" | sed -n 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        local url=$(echo "$block" | sed -n 's/.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        if [[ -n "$name" && -n "$url" ]]; then
            if echo "$name" | grep -Eiq "$name_regex"; then
                echo "$url"
                break
            fi
        fi
    done
}

# 下载并安装 Xray
install_xray() {
    info "正在安装 Xray..."
    
    # 版本选择策略：
    # 1) 若设置了 XRAY_VERSION，则优先使用（可为具体 tag，如 v26.0.0；或设置为 latest）
    # 2) 否则尝试通过 GitHub API 获取最新版本
    # 3) 若仍失败，则直接报错并退出（不做静默回退）

    local xray_version="${XRAY_VERSION:-}"
    if [[ -z "${xray_version}" || "${xray_version}" == "latest" ]]; then
        info "获取 Xray 最新版本..."
        local curl_opts=( -fsSL -H "Accept: application/vnd.github+json" -H "User-Agent: xrf-go-installer" )
        if [[ -n "${GITHUB_TOKEN:-}" ]]; then
            curl_opts+=( -H "Authorization: Bearer ${GITHUB_TOKEN}" )
        fi
        xray_version=$(get_github_latest_version "XTLS/Xray-core" "xrf-go-installer" || true)
        if ! is_valid_tag "${xray_version:-}"; then
            error "获取到的 Xray 版本号不合法：${xray_version:-<empty>}\n请检查网络/令牌，或显式设置 XRAY_VERSION=vX.Y.Z 再试。"
        fi
        if [[ -z "$xray_version" ]]; then
            error "无法获取 Xray 最新版本。请检查网络或：\n  - 设置 GITHUB_TOKEN 以避免 GitHub API 限流\n  - 或显式指定版本：XRAY_VERSION=v26.x.y bash install.sh"
        fi
    else
        info "使用指定的 Xray 版本: ${xray_version}"
    fi
    
    local temp_dir
    temp_dir=$(mktemp -d)

    # 通过 GitHub API 获取指定 tag 的 release（包含 assets）
    local curl_opts=( -fsSL -H "Accept: application/vnd.github+json" -H "User-Agent: xrf-go-installer" )
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        curl_opts+=( -H "Authorization: Bearer ${GITHUB_TOKEN}" )
    fi
    local release_api="https://api.github.com/repos/XTLS/Xray-core/releases/tags/${xray_version}"
    local release_json
    release_json=$(curl "${curl_opts[@]}" "$release_api" || true)
    if [[ -z "$release_json" ]]; then
        error "获取 Xray Release 资产失败：${release_api}\n建议：\n  • 检查网络连通性或稍后重试\n  • 设置 GITHUB_TOKEN 以避免 GitHub API 限流\n  • 手动查看发布页确认资产是否存在：https://github.com/XTLS/Xray-core/releases/tag/${xray_version}\n  • 或显式指定版本：XRAY_VERSION=v26.x.y bash install.sh"
    fi

    # 动态选择与架构匹配的资产
    local name_regex=""
    if [[ "$ARCH" == "amd64" ]]; then
        name_regex='^Xray-.*linux.*(64|amd64).*\.zip$'
    else
        name_regex='^Xray-.*linux.*(arm64|arm64-v8a).*\.zip$'
    fi

    local download_url
    download_url=$(select_asset_url "$release_json" "$name_regex")
    if [[ -z "$download_url" ]]; then
        error "未找到匹配架构(${ARCH})的 Xray 资产。\n已使用匹配规则: ${name_regex}\n发布页：https://github.com/XTLS/Xray-core/releases/tag/${xray_version}\n请在发布页确认是否存在相应资产，或设置 XRAY_VERSION 指定其他版本。"
    fi

    local fname
    fname=$(basename "$download_url")
    info "下载 Xray ${xray_version} (${fname}) for ${ARCH}..."
    curl -fsSL -o "${temp_dir}/xray.zip" "$download_url" || error "下载 Xray 失败：${download_url}\n建议：\n  • 检查网络连通性/代理设置\n  • 稍后重试或手动从发布页下载：https://github.com/XTLS/Xray-core/releases/tag/${xray_version}"
    
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
    
    # 获取最新版本
    local xrf_version
    xrf_version=$(get_github_latest_version "Joe-oss9527/xrf-go" "xrf-go-installer" || true)
    if [[ -z "$xrf_version" ]]; then
        error "无法获取 XRF-Go 最新版本。\n建议：\n  • 检查网络连通性或稍后重试\n  • 设置 GITHUB_TOKEN 以避免 GitHub API 限流\n  • 手动查看发布页：https://github.com/Joe-oss9527/xrf-go/releases/latest"
    fi
    if ! is_valid_tag "${xrf_version:-}"; then
        error "获取到的 XRF-Go 版本号不合法：${xrf_version:-<empty>}\n请检查网络/令牌，或手动指定版本。"
    fi

    # 获取 Release JSON 用于资产选择
    local curl_opts=( -fsSL -H "Accept: application/vnd.github+json" -H "User-Agent: xrf-go-installer" )
    if [[ -n "${GITHUB_TOKEN:-}" ]]; then
        curl_opts+=( -H "Authorization: Bearer ${GITHUB_TOKEN}" )
    fi
    local xrf_json
    xrf_json=$(curl "${curl_opts[@]}" "https://api.github.com/repos/Joe-oss9527/xrf-go/releases/latest" || true)
    if [[ -z "$xrf_json" ]]; then
        error "无法获取 XRF-Go Release 信息用于资产下载。\n手动查看发布页：https://github.com/Joe-oss9527/xrf-go/releases/latest"
    fi

    local temp_dir
    temp_dir=$(mktemp -d)
    info "下载 XRF-Go ${xrf_version} for ${ARCH}..."

    # 解析资产并选择与架构匹配的 tar.gz
    local name_regex="^xrf-.*linux-${ARCH}.*\.(tar\.gz|tgz)$"
    local xrf_url
    xrf_url=$(select_asset_url "$xrf_json" "$name_regex")
    if [[ -z "$xrf_url" ]]; then
        rm -rf "$temp_dir"
        error "未找到与架构(${ARCH})匹配的 XRF-Go 预编译归档。\n匹配规则: ${name_regex}\n发布页：https://github.com/Joe-oss9527/xrf-go/releases/tag/${xrf_version}\n请在发布页确认是否存在相应产物。"
    fi

    local downloaded
    downloaded=$(basename "$xrf_url")
    curl -fsSL -o "${temp_dir}/${downloaded}" "$xrf_url" || {
        rm -rf "$temp_dir"
        error "下载 XRF-Go 预编译归档失败：${xrf_url}\n建议：\n  • 检查网络连通性/代理设置\n  • 稍后重试或手动从发布页下载：https://github.com/Joe-oss9527/xrf-go/releases/tag/${xrf_version}"
    }
    
    # 解压或直接安装
    # 解压并安装
    tar -xzf "${temp_dir}/${downloaded}" -C "$temp_dir"
    $SUDO_CMD install -m 755 "${temp_dir}/xrf-linux-${ARCH}" /usr/local/bin/xrf
    
    rm -rf "$temp_dir"
    success "XRF-Go 安装完成: $(xrf version | grep 'XRF-Go 版本')"
}

## 编译回退逻辑已移除：若预编译资产不可用或下载失败，将直接报错退出

# 创建配置目录
setup_config() {
    info "设置配置目录..."
    
    $SUDO_CMD mkdir -p /etc/xray/confs
    $SUDO_CMD chown "$(whoami)":"$(whoami)" /etc/xray/confs
    
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
