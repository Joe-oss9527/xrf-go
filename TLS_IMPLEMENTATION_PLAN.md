# TLS 模块实施计划 - 对齐 DESIGN.md (简化版)

## 📋 实施目标
根据 DESIGN.md 要求，完成 TLS 模块的两个核心组件：
1. **acme.go** - Let's Encrypt 自动证书申请和管理
2. **caddy.go** - Caddy 作为前置代理配置和伪装网站

## 🎯 设计理念
- **纯自动化** - 移除手动证书管理，全自动申请和续期
- **零配置** - 用户只需提供域名，其他全部自动化
- **简洁高效** - 避免冗余设计，符合 233boy 极简理念

## 🏗️ 架构设计

### 1. ACME 模块 (pkg/tls/acme.go)

#### 核心功能
- 自动申请 Let's Encrypt 证书
- 证书续期管理（30天内自动续期）
- DNS-01 和 HTTP-01 挑战支持
- 多域名证书支持

#### 技术选型
- 使用 **github.com/go-acme/lego/v4** 库（最新版本 v4.25.2，2025年8月更新）
- **移除 TLSFileManager 依赖** - 直接管理证书文件存储

#### 主要接口设计
```go
type ACMEManager struct {
    email     string
    caURL     string
    certDir   string  // 直接管理证书目录
    client    *lego.Client
}

// 核心方法
func (am *ACMEManager) ObtainCertificate(domains []string) error
func (am *ACMEManager) RenewCertificate(domain string) error
func (am *ACMEManager) SetupAutoRenewal() error
func (am *ACMEManager) CheckAndRenew() error
func (am *ACMEManager) saveCertificate(domain string, cert, key []byte) error
```

### 2. Caddy 集成模块 (pkg/tls/caddy.go)

#### 核心功能
- 作为 Xray 的前置反向代理
- 自动 HTTPS 和证书管理
- 伪装网站配置（返回真实网站内容）
- WebSocket 和 gRPC 协议支持

#### 技术选型
- 使用 Caddy REST API 进行动态配置
- 通过 systemd 管理 Caddy 服务
- JSON 配置模式（更适合程序化管理）

#### 主要接口设计
```go
type CaddyManager struct {
    adminAPI   string
    configDir  string
    httpClient *http.Client
}

// 核心方法
func (cm *CaddyManager) InstallCaddy() error
func (cm *CaddyManager) ConfigureReverseProxy(upstream string, domain string) error
func (cm *CaddyManager) AddWebsiteMasquerade(domain string, maskSite string) error
func (cm *CaddyManager) EnableAutoHTTPS(domain string) error
func (cm *CaddyManager) ReloadConfig() error
```

## 📝 实施步骤

### 第一阶段：ACME 模块实现（2-3天）

#### 1.1 基础结构搭建
- 创建 `pkg/tls/acme.go` 文件
- 定义 ACMEManager 结构体和接口
- 集成 lego 库依赖

#### 1.2 用户账户管理
```go
// 实现 ACME 用户接口
type ACMEUser struct {
    Email        string
    Registration *registration.Resource
    key          crypto.PrivateKey
}

func (u *ACMEUser) GetEmail() string
func (u *ACMEUser) GetRegistration() *registration.Resource
func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey
```

#### 1.3 证书申请功能
- 实现 HTTP-01 挑战（端口 80）
- 实现 DNS-01 挑战（支持主流 DNS 提供商）
- 证书申请和验证流程
- **直接文件存储** - 简化的证书保存逻辑

#### 1.4 自动续期机制
- 定时检查证书到期时间
- 30天内自动续期
- 续期失败告警机制

### 第二阶段：Caddy 集成模块（2-3天）

#### 2.1 Caddy 安装和管理
- 自动下载和安装 Caddy
- systemd 服务配置
- 基础配置文件生成

#### 2.2 反向代理配置
```json
{
  "apps": {
    "http": {
      "servers": {
        "srv0": {
          "listen": [":443"],
          "routes": [{
            "match": [{"host": ["example.com"]}],
            "handle": [{
              "@type": "reverse_proxy",
              "upstreams": [{"dial": "localhost:8080"}]
            }]
          }]
        }
      }
    }
  }
}
```

#### 2.3 伪装网站功能
- 配置真实网站反向代理
- 路径分流（特定路径转发到 Xray）
- 错误页面自定义

#### 2.4 协议支持
- WebSocket 路径转发
- gRPC 服务代理
- HTTP/2 支持

### 第三阶段：集成和测试（1-2天）

#### 3.1 与现有系统集成
- 修改 ConfigManager 支持 TLS 配置
- 更新协议模板支持 Caddy 前置代理
- CLI 命令扩展

#### 3.2 简化的 CLI 命令
```bash
# 自动化协议添加（内置证书申请）
xrf add vless --domain example.com        # 自动申请证书
xrf add trojan --domain test.com          # 自动申请证书

# 可选的证书管理命令
xrf tls status                            # 查看证书状态
xrf tls renew                             # 手动续期（通常自动）

# Caddy 相关命令
xrf caddy install                         # 安装 Caddy
xrf caddy config --domain example.com     # 配置反向代理
xrf caddy mask --site google.com          # 设置伪装网站
xrf caddy status                          # 查看 Caddy 状态
```

#### 3.3 测试验证
- 单元测试覆盖核心功能
- 集成测试验证端到端流程
- 性能测试确保不影响现有 <1ms 目标

## 🔧 技术细节

### ACME 实现要点
1. **账户持久化**：存储在 `/etc/xray/acme/account.json`
2. **证书存储**：直接存储在 `/etc/xray/certs/`，简化文件管理
3. **挑战端口**：HTTP-01 使用 80 端口，需要临时占用
4. **速率限制**：遵守 Let's Encrypt 限制（每周 50 个证书/域名）
5. **文件权限**：证书 0644，私钥 0600

### Caddy 集成要点
1. **配置方式**：使用 REST API 而非配置文件，便于动态管理
2. **服务管理**：独立 systemd 服务，与 Xray 服务分离
3. **端口分配**：Caddy 监听 443，Xray 监听其他端口（如 8080）
4. **日志集成**：Caddy 日志输出到 systemd journal

## 🚀 性能和兼容性

### 性能保证
- 证书申请异步执行，不阻塞主流程
- Caddy 配置热重载，无停机时间
- 维持现有 <1ms 配置添加性能

### 兼容性考虑
- **移除手动证书管理** - 专注自动化，避免复杂性
- RequiresTLS 协议自动触发证书申请
- Caddy 为可选组件，不强制依赖

## 📊 风险管理

### 潜在风险
1. **证书申请失败**：提供清晰的错误信息和解决建议
2. **端口冲突**：智能端口检测和分配
3. **Caddy 依赖**：设计为可选功能，不影响核心功能
4. **域名验证失败**：检查 DNS 记录和防火墙设置

### 缓解措施
- 完善的错误处理和回滚机制
- 详细的日志记录
- 用户友好的错误提示和解决建议

## 📅 时间线
- **第1-3天**：ACME 模块开发和测试
- **第4-6天**：Caddy 集成开发和测试
- **第7-8天**：系统集成、文档更新、全面测试

## ✅ 交付标准
1. 两个模块完全对齐 DESIGN.md 设计
2. **移除手动证书管理复杂性**，实现纯自动化
3. 保持"简洁高效"的设计理念
4. 单元测试覆盖率 >80%
5. 性能不低于现有水平
6. 完整的使用文档和示例

## 🗂️ 文件清理
### 需要移除的文件
- `pkg/tls/file_manager.go` - 手动证书管理（已不需要）
- `pkg/tls/file_manager_test.go` - 相关测试文件

## 🛠️ 依赖项

### Go 依赖
```go
// go.mod 需要添加
require (
    github.com/go-acme/lego/v4 v4.25.2
)
```

### 系统依赖
- Caddy v2.8+ (通过程序自动下载安装)
- systemd (用于服务管理)
- 端口 80, 443 (用于 HTTP-01 挑战和 HTTPS)

## 📚 参考资源

### 官方文档
- [Lego ACME Library](https://go-acme.github.io/lego/)
- [Caddy API Documentation](https://caddyserver.com/docs/api)
- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)

### 代码示例
- [Lego Library Usage](https://go-acme.github.io/lego/usage/library/)
- [Caddy JSON Config](https://caddyserver.com/docs/json/)
- [Caddy Reverse Proxy](https://caddyserver.com/docs/quick-starts/reverse-proxy)

## 📋 实施优势

### 简化带来的益处
1. **减少代码量** - 移除 400+ 行冗余代码
2. **降低复杂度** - 用户不需要理解证书文件管理
3. **提升体验** - 零配置，一键申请
4. **更少错误** - 减少手动操作的错误可能
5. **符合理念** - 完全对齐"简洁高效"设计原则

### 用户体验改进
```bash
# 之前（复杂）
xrf tls add --cert /path/to/cert.pem --key /path/to/key.pem
xrf add vless --domain example.com --cert-name mycert

# 现在（简单）
xrf add vless --domain example.com  # 自动申请证书
```