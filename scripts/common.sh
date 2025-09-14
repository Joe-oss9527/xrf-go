#!/bin/bash

# XRF-Go 共享工具函数
# 用于统一 JSON 解析、版本管理和系统安装逻辑

# 设置错误处理
set -euo pipefail

# 颜色定义
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly YELLOW='\033[1;33m'
readonly BLUE='\033[0;34m'
readonly NC='\033[0m'

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# 轻量级空白裁剪
_trim() {
    local s="${1:-}"
    # shellcheck disable=SC2001
    echo "${s}" | sed 's/^\s\+//;s/\s\+$//'
}

# 校验是否为合理的 tag 值（而非 URL/空字符串）
# 允许: 字母数字、点、短横线、下划线，且不包含斜杠或冒号
_is_valid_tag() {
    local tag
    tag="$(_trim "${1:-}")"
    [[ -n "$tag" ]] || return 1
    [[ ! "$tag" =~ ^https?:// ]] || return 1
    [[ ! "$tag" =~ / ]] || return 1
    [[ ! "$tag" =~ : ]] || return 1
    [[ "$tag" =~ ^[A-Za-z0-9._-]+$ ]] || return 1
}

# 从 GitHub Release JSON 中提取 tag_name
# 参数: $1=release_json
extract_tag_name() {
    local release_json="$1"

    if [[ -z "$release_json" ]]; then
        return 1
    fi

    # 优先使用 jq（专业 JSON 解析器）
    if command -v jq >/dev/null 2>&1; then
        echo "$release_json" | jq -r '.tag_name'
    else
        # Fallback: 使用 awk 进行更清晰的解析
        echo "$release_json" | grep '"tag_name":' | awk -F'"' '{print $4}'
    fi
}

# 获取 GitHub Release 的最新版本
# 参数: $1=repo (格式: owner/repo)
# 参数: $2=user_agent (可选)
get_github_latest_version() {
    local repo="$1"
    local user_agent="${2:-xrf-go-tools}"

    local curl_opts=(
        -fsSL
        -H "Accept: application/vnd.github+json"
        -H "User-Agent: $user_agent"
    )

    # 如果设置了 GITHUB_TOKEN，添加认证头
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

    # Fallback: 通过 latest 页面重定向解析 tag，避免 API 限流或缺失 jq
    # 使用 -I 获取头部并打印最终URL；某些环境需要 -L 以跟随 302
    local effective
    effective=$(curl -fsSLI -o /dev/null -w '%{url_effective}' -H "User-Agent: $user_agent" "https://github.com/${repo}/releases/latest" 2>/dev/null || echo "")
    if [[ -n "$effective" ]]; then
        # 期望形如 .../releases/tag/<TAG>
        local tag_from_url
        tag_from_url=$(echo "$effective" | sed -n 's#.*/releases/tag/\([^/?]*\).*#\1#p')
        if _is_valid_tag "$tag_from_url"; then
            echo "$tag_from_url"
            return 0
        fi
    fi

    return 1
}

# 获取 Xray 最新版本
get_xray_latest_version() {
    get_github_latest_version "XTLS/Xray-core" "xrf-go-installer"
}

# 获取 XRF-Go 最新版本
get_xrf_latest_version() {
    get_github_latest_version "Joe-oss9527/xrf-go" "xrf-go-installer"
}

# 从 GitHub Release JSON 中按名称正则选择资产下载地址
# 参数: $1=release_json $2=name_regex (grep -E 风格)
select_asset_url() {
    local release_json="$1"
    local name_regex="$2"

    # 展平 JSON 并定位 assets 数组
    local flat
    flat=$(echo "$release_json" | tr -d '\n')
    local assets
    assets=$(echo "$flat" | sed -n 's/.*"assets":[[]\(.*\)[]].*/\1/p')
    if [[ -z "$assets" ]]; then
        return 1
    fi

    # 按资产对象切分后逐一匹配名称并取其下载链接
    echo "$assets" | sed 's/},[[:space:]]*{/\n/g' | while IFS= read -r block; do
        local name
        name=$(echo "$block" | sed -n 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        local url
        url=$(echo "$block" | sed -n 's/.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')
        if [[ -n "$name" && -n "$url" ]]; then
            if echo "$name" | grep -Eiq "$name_regex"; then
                echo "$url"
                break
            fi
        fi
    done
}

# 获取系统架构（标准化）
get_system_arch() {
    local arch
    arch=$(uname -m)
    case $arch in
        x86_64)
            echo "amd64"
            ;;
        aarch64)
            echo "arm64"
            ;;
        *)
            log_error "不支持的架构: $arch (仅支持 x86_64 和 aarch64)"
            return 1
            ;;
    esac
}

# 安装 Xray（用于 CI/CD 和自动化脚本）
# 参数: $1=version (可选，默认获取最新版本)
# 参数: $2=fallback_version (可选，默认 v25.9.11)
install_xray_for_automation() {
    local version="${1:-}"
    local fallback_version="${2:-v25.9.11}"
    local arch

    # 获取系统架构
    arch=$(get_system_arch) || return 1

    # 获取版本
    if [[ -z "$version" ]]; then
        log_info "获取 Xray 最新版本..."
        version=$(get_xray_latest_version || echo "")
    fi

    if [[ -z "$version" ]]; then
        log_warning "无法获取 Xray 最新版本，使用 fallback: $fallback_version"
        version="$fallback_version"
    fi

    log_info "准备安装 Xray $version"

    # 构造可能的下载URL（处理不同的命名约定）
    local download_urls=()
    if [[ "$arch" == "amd64" ]]; then
        download_urls=(
            "https://github.com/XTLS/Xray-core/releases/download/${version}/Xray-linux-64.zip"
            "https://github.com/XTLS/Xray-core/releases/download/${version}/Xray-linux-amd64.zip"
        )
    else
        download_urls=(
            "https://github.com/XTLS/Xray-core/releases/download/${version}/Xray-linux-arm64-v8a.zip"
            "https://github.com/XTLS/Xray-core/releases/download/${version}/Xray-linux-arm64.zip"
        )
    fi

    # 尝试下载
    local temp_file="/tmp/xray-${version}.zip"
    local downloaded=false

    for url in "${download_urls[@]}"; do
        log_info "尝试下载: $(basename "$url")"
        if curl -fsSL -o "$temp_file" "$url" 2>/dev/null; then
            downloaded=true
            log_success "下载成功: $(basename "$url")"
            break
        else
            log_warning "下载失败: $(basename "$url")"
        fi
    done

    if [[ "$downloaded" != "true" ]]; then
        log_error "无法从任何已知URL下载 Xray $version"
        return 1
    fi

    # 解压和安装
    local temp_dir="/tmp/xray-install-$$"
    mkdir -p "$temp_dir"

    if ! unzip -q -d "$temp_dir" "$temp_file"; then
        rm -rf "$temp_dir" "$temp_file"
        log_error "解压 Xray 失败"
        return 1
    fi

    # 安装二进制文件
    if [[ -f "$temp_dir/xray" ]]; then
        if command -v sudo >/dev/null 2>&1 && [[ $EUID -ne 0 ]]; then
            sudo install -m 755 "$temp_dir/xray" /usr/local/bin/xray
        else
            install -m 755 "$temp_dir/xray" /usr/local/bin/xray
        fi

        # 可选文件（可能不存在）
        for optional_file in geoip.dat geosite.dat; do
            if [[ -f "$temp_dir/$optional_file" ]]; then
                if command -v sudo >/dev/null 2>&1 && [[ $EUID -ne 0 ]]; then
                    sudo install -m 644 "$temp_dir/$optional_file" /usr/local/bin/ 2>/dev/null || true
                else
                    install -m 644 "$temp_dir/$optional_file" /usr/local/bin/ 2>/dev/null || true
                fi
            fi
        done
    else
        rm -rf "$temp_dir" "$temp_file"
        log_error "解压后未找到 xray 二进制文件"
        return 1
    fi

    # 清理临时文件
    rm -rf "$temp_dir" "$temp_file"

    # 验证安装
    if xray version >/dev/null 2>&1; then
        log_success "Xray 安装完成: $(xray version | head -1)"
    else
        log_error "Xray 安装验证失败"
        return 1
    fi
}

# 检查工具是否可用
check_tool() {
    local tool="$1"
    local install_hint="${2:-请手动安装 $tool}"

    if ! command -v "$tool" >/dev/null 2>&1; then
        log_error "缺少必需工具: $tool"
        log_info "$install_hint"
        return 1
    fi
}

# 检查基本依赖
check_basic_dependencies() {
    log_info "检查基本依赖..."

    local deps=("curl" "unzip" "tar")
    for dep in "${deps[@]}"; do
        check_tool "$dep" || return 1
    done

    # 检查 systemctl（仅在 Linux 上）
    if [[ "$(uname)" == "Linux" ]]; then
        check_tool "systemctl" "需要 systemd 支持" || return 1
    fi

    log_success "基本依赖检查通过"
}

# 导出函数（使其在 source 后可用）
if [[ "${BASH_SOURCE[0]}" != "${0}" ]]; then
    # 脚本被 source，导出函数
    export -f extract_tag_name
    export -f get_github_latest_version
    export -f get_xray_latest_version
    export -f get_xrf_latest_version
    export -f select_asset_url
    export -f get_system_arch
    export -f install_xray_for_automation
    export -f check_tool
    export -f check_basic_dependencies
    export -f log_info
    export -f log_success
    export -f log_warning
    export -f log_error
fi
