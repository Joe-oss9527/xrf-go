# XRF-Go

> **高效率，超快速，极易用** - 简洁高效的 Xray 安装配置工具

XRF-Go 是一个专为 Xray 设计的现代化配置管理工具，继承 233boy 项目的设计理念，以**多配置同时运行**为核心设计，专门优化了添加、更改、查看、删除四项常用功能。

## ✨ 特性

### 🚀 极速体验
- **<1秒添加配置** - 优化的多配置文件策略
- **零学习成本** - 直观的命令设计
- **智能默认值** - 自动选择最优配置

### 🛡️ 现代协议支持
- **VLESS-REALITY** - 最新抗审查技术（推荐）
- **VLESS-WebSocket-TLS** - 经典 WebSocket 传输
- **VMess-WebSocket-TLS** - 兼容性最佳
- **VLESS-HTTPUpgrade** - HTTP/2 升级传输
- **Trojan-WebSocket-TLS** - 高伪装性
- **Shadowsocks/2022** - 轻量级代理

### 🔧 智能管理
- **多配置文件** - 充分利用 Xray `-confdir` 特性
- **热重载支持** - 无中断配置更新
- **自动端口分配** - 智能避免端口冲突
- **配置验证** - 修改前自动验证，失败自动回滚

### 📦 单文件分发
- **无外部依赖** - 所有配置模板内嵌二进制
- **跨平台支持** - Linux/macOS/Windows
- **一键安装** - 自动检测系统并优化配置

## 🚀 快速开始

### 一键安装

```bash
curl -fsSL https://get.xrf.sh | bash
```

### 手动安装

```bash
# 下载最新版本
wget https://github.com/yourusername/xrf-go/releases/latest/download/xrf-linux-amd64
sudo install xrf-linux-amd64 /usr/local/bin/xrf

# 初始化配置
xrf install
```

## 📖 使用指南

### 基本命令

```bash
# 🔧 安装和初始化
xrf install                                    # 默认安装 VLESS-REALITY
xrf install --protocol vw --domain example.com # 指定协议和域名

# ➕ 添加协议（支持别名）
xrf add vr --port 443 --domain example.com    # VLESS-REALITY
xrf add vw --port 443 --domain example.com    # VLESS-WebSocket-TLS
xrf add vmess --port 80 --path /ws            # VMess-WebSocket
xrf add tw --port 443 --domain example.com    # Trojan-WebSocket-TLS
xrf add ss --port 8388                        # Shadowsocks
xrf add ss2022 --method 2022-blake3-aes-256-gcm # Shadowsocks-2022
xrf add hu --port 8080 --path /upgrade        # VLESS-HTTPUpgrade

# 📋 管理配置
xrf list                                       # 列出所有协议
xrf info [tag]                                # 查看详细配置
xrf remove [tag]                              # 删除协议
xrf change [tag] [key] [value]                # 修改配置

# 🔄 服务控制
xrf start                                     # 启动服务
xrf stop                                      # 停止服务
xrf restart                                   # 重启服务
xrf status                                    # 查看状态
xrf reload                                    # 热重载配置

# 🛠️ 实用工具
xrf generate password                         # 生成随机密码
xrf generate uuid                             # 生成 UUID
xrf generate ss2022                           # 生成 SS2022 密钥
xrf generate keypair                          # 生成 X25519 密钥对
xrf url [tag]                                 # 生成分享链接
xrf qr [tag]                                  # 显示二维码

# ✅ 配置管理
xrf test                                      # 验证配置
xrf backup                                    # 备份配置
xrf restore [backup-file]                    # 恢复配置
```

### 协议别名快速参考

| 别名 | 完整协议名 | 描述 |
|------|-----------|------|
| `vr` | VLESS-REALITY | 最新抗审查技术，**强烈推荐** |
| `vw` | VLESS-WebSocket-TLS | 需要域名和 TLS 证书 |
| `vmess` | VMess-WebSocket-TLS | 兼容性最好，广泛支持 |
| `tw` | Trojan-WebSocket-TLS | 高伪装性，需要域名证书 |
| `ss` | Shadowsocks | 轻量级，适合移动端 |
| `ss2022` | Shadowsocks-2022 | 增强安全性的新版本 |
| `hu` | VLESS-HTTPUpgrade | HTTP/2 升级传输 |

## 🏗️ 架构设计

### 多配置文件策略

XRF-Go 充分利用 Xray 的 `-confdir` 特性，采用模块化配置管理：

```
/etc/xray/confs/
├── 00-base.json              # 基础配置 (log, api, stats)
├── 01-dns.json               # DNS 配置
├── 10-inbound-vless.json     # VLESS-REALITY 入站
├── 11-inbound-vmess.json     # VMess 入站
├── 20-outbound-direct.json   # 直连出站
├── 21-outbound-block.json    # 阻断出站
├── 90-routing-basic.json     # 基础路由规则
└── 99-routing-tail.json      # 最终路由规则
```

### 配置合并规则

- **00-09**: 基础配置（log, api, dns）
- **10-19**: 入站配置（自动追加到数组）
- **20-29**: 出站配置（自动插入到开头）
- **90-99**: 路由配置（支持 tail 模式）

### 性能优化

所有协议配置都包含现代网络优化参数：

```json
{
  "sockopt": {
    "tcpKeepAliveIdle": 300,
    "tcpUserTimeout": 10000
  }
}
```

## 🔧 高级用法

### 自定义配置

```bash
# 指定自定义配置目录
xrf --confdir /path/to/custom/confs list

# 详细输出模式
xrf -v add vr --port 443

# 禁用彩色输出
xrf --no-color list
```

### 批量操作

```bash
# 一次安装多个协议
xrf install --protocols vr,vw,vmess --domain example.com

# 批量生成工具
for i in {1..5}; do xrf generate uuid; done
```

### REALITY 配置示例

```bash
# 添加 VLESS-REALITY（推荐配置）
xrf add vr \
  --port 443 \
  --domain example.com \
  --dest www.microsoft.com \
  --server-name www.microsoft.com
```

## 🛠️ 开发

### 构建要求

- Go 1.23+
- Linux/macOS/Windows

### 编译

```bash
# 克隆源码
git clone https://github.com/yourusername/xrf-go.git
cd xrf-go

# 编译
go build -o xrf cmd/xrf/main.go

# 优化编译
go build -ldflags="-s -w" -o xrf cmd/xrf/main.go

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o xrf-linux-amd64 cmd/xrf/main.go
GOOS=darwin GOARCH=arm64 go build -o xrf-darwin-arm64 cmd/xrf/main.go
```

### 测试

```bash
# 运行测试
go test ./...

# 测试覆盖率
go test -cover ./...

# 验证配置
xray test -confdir /etc/xray/confs
```

## 📁 项目结构

```
xrf-go/
├── cmd/xrf/                  # CLI 主程序
│   └── main.go
├── pkg/
│   ├── config/              # 配置管理核心
│   │   ├── manager.go       # 多配置文件管理器
│   │   ├── templates.go     # 内置配置模板
│   │   └── protocols.go     # 协议定义和支持
│   ├── system/              # 系统操作
│   │   ├── detector.go      # 系统检测
│   │   ├── installer.go     # 安装器
│   │   └── service.go       # 服务管理
│   ├── tls/                 # TLS/证书管理
│   │   └── acme.go          # 自动证书
│   ├── api/                 # Xray API 接口
│   │   └── client.go        # API 客户端
│   └── utils/               # 工具函数
│       ├── logger.go        # 日志
│       ├── colors.go        # 彩色输出
│       ├── crypto.go        # 加密工具
│       ├── validator.go     # 配置验证
│       └── http.go          # HTTP 工具
├── scripts/
│   └── install.sh           # 一键安装脚本
├── DESIGN.md                # 设计文档
├── requirements.md          # 需求文档
└── CLAUDE.md                # AI 开发指南
```

## 🤝 贡献

我们欢迎各种形式的贡献！

1. **Fork** 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送分支 (`git push origin feature/AmazingFeature`)
5. 开启 **Pull Request**

### 开发指南

- 遵循 Go 代码规范
- 添加适当的测试
- 更新相关文档
- 确保所有测试通过

## 📄 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

- [Xray-core](https://github.com/XTLS/Xray-core) - 强大的代理平台
- [233boy/Xray](https://github.com/233boy/Xray) - 设计理念来源
- [Project X](https://xtls.github.io/) - 技术文档和社区支持

## 📞 支持

- 🐛 [报告问题](https://github.com/yourusername/xrf-go/issues)
- 💬 [讨论区](https://github.com/yourusername/xrf-go/discussions)
- 📖 [官方文档](https://xtls.github.io/)
- 📧 [联系我们](mailto:support@example.com)

---

<div align="center">

**[⭐ Star](https://github.com/yourusername/xrf-go)** • **[🔄 Fork](https://github.com/yourusername/xrf-go/fork)** • **[📢 反馈](https://github.com/yourusername/xrf-go/issues)**

Made with ❤️ by XRF-Go Team

</div>