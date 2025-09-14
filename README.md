# XRF-Go

[![Go Version](https://img.shields.io/github/go-mod/go-version/Joe-oss9527/xrf-go)](https://golang.org/)
[![Release](https://img.shields.io/github/v/release/Joe-oss9527/xrf-go)](https://github.com/Joe-oss9527/xrf-go/releases)
[![CI](https://github.com/Joe-oss9527/xrf-go/workflows/CI/badge.svg)](https://github.com/Joe-oss9527/xrf-go/actions)
[![License](https://img.shields.io/github/license/Joe-oss9527/xrf-go)](LICENSE)

XRF-Go 是一个高效、简洁的 Xray 安装配置工具，专为简化 Xray 部署和管理而设计。

## 🚀 核心特性

- **🔧 一键安装**: 自动化安装 Xray 和 XRF-Go
- **⚡ 超快配置**: 协议添加速度达到 10-50ms 级别
- **🛡️ 多协议支持**: VLESS-REALITY、VMess、Trojan、Shadowsocks 等
- **🔐 自动化 TLS**: 集成 ACME 和 Caddy，自动申请和续期证书
- **🎯 智能管理**: 配置验证、自动备份、故障回滚
- **📊 状态监控**: 服务状态检查、日志查看、性能监控

## 📦 快速安装

### 方式 1：一键安装脚本（推荐）

```bash
curl -fsSL https://github.com/Joe-oss9527/xrf-go/releases/latest/download/install.sh | bash
```

### 方式 2：手动下载

#### Linux AMD64 (x86_64)
```bash
wget https://github.com/Joe-oss9527/xrf-go/releases/latest/download/xrf-linux-amd64.tar.gz
tar -xzf xrf-linux-amd64.tar.gz
sudo mv xrf-linux-amd64 /usr/local/bin/xrf
sudo chmod +x /usr/local/bin/xrf
```

#### Linux ARM64
```bash
wget https://github.com/Joe-oss9527/xrf-go/releases/latest/download/xrf-linux-arm64.tar.gz
tar -xzf xrf-linux-arm64.tar.gz
sudo mv xrf-linux-arm64 /usr/local/bin/xrf
sudo chmod +x /usr/local/bin/xrf
```

## 🎯 快速开始

### 1. 验证安装
```bash
xrf --version
```

### 2. 查看帮助
```bash
xrf --help
```

### 3. 添加协议配置
```bash
# 添加 VLESS-REALITY（推荐）
xrf add vr --port 443

# 添加 VLESS-Encryption（后量子加密）
xrf add ve --port 443 --auth mlkem768

# 添加 VMess-WebSocket-TLS
xrf add vmess --port 443 --domain example.com

# 添加 Shadowsocks
xrf add ss --port 8388 --password your-password
```

### 4. 查看配置
```bash
xrf list
```

### 5. 获取客户端连接信息
```bash
xrf url <tag>
```

### 6. 管理服务
```bash
# 检查状态
xrf status

# 查看日志
xrf logs

# 重载配置
xrf reload
```

## 📋 支持的协议

| 协议别名 | 协议全名 | 特点 | 推荐度 |
|---------|---------|------|--------|
| `vr` | VLESS-REALITY | 抗封锁、高性能 | ⭐⭐⭐⭐⭐ |
| `ve` | VLESS-Encryption | 后量子加密、抗量子攻击 | ⭐⭐⭐⭐⭐ |
| `vw` | VLESS-WebSocket-TLS | 通用性好 | ⭐⭐⭐⭐ |
| `vmess` | VMess-WebSocket-TLS | 传统稳定 | ⭐⭐⭐ |
| `tw` | Trojan-WebSocket-TLS | 伪装性好 | ⭐⭐⭐⭐ |
| `ss` | Shadowsocks | 轻量简单 | ⭐⭐⭐ |
| `ss2022` | Shadowsocks-2022 | 新版本SS | ⭐⭐⭐⭐ |
| `hu` | VLESS-HTTPUpgrade | HTTP升级 | ⭐⭐⭐ |

## 🔧 高级功能

### 配置管理
```bash
# 修改配置
xrf change <tag> <key> <value>

# 删除配置
xrf remove <tag>

# 测试配置
xrf test

# 生成随机值
xrf generate uuid
xrf generate password
xrf generate key
```

### TLS 证书管理
```bash
# 申请证书
xrf cert get --domain example.com

# 查看证书状态
xrf cert status

# 续期证书
xrf cert renew
```

### 系统管理
```bash
# 检查端口占用
xrf check-port

# 获取公网IP
xrf ip

# 系统信息
xrf info system
```

## 🏗️ 架构设计

XRF-Go 采用模块化架构设计：

```
xrf-go/
├── cmd/xrf/           # CLI 入口点
├── pkg/
│   ├── config/        # 配置管理
│   ├── system/        # 系统检测和服务管理
│   ├── tls/          # TLS 证书管理
│   ├── api/          # Xray gRPC 客户端
│   └── utils/        # 工具函数
└── scripts/
    ├── common.sh      # 共享工具函数
    ├── install.sh     # 安装脚本
    └── build.sh       # 构建脚本
```

## 📊 性能指标

- **协议添加速度**: 10-50ms（生产环境，包含备份和验证）
- **内存占用**: <20MB
- **二进制大小**: <10MB
- **配置操作吞吐**: >40操作/秒
- **启动时间**: <1秒

## 🛠️ 开发

### 环境要求
- Go 1.25+
- Linux 系统（支持 amd64/arm64）
- systemd 支持

### 构建
```bash
# 克隆项目
git clone https://github.com/Joe-oss9527/xrf-go.git
cd xrf-go

# 安装依赖
go mod download

# 构建当前平台
./scripts/build.sh build-current

# 构建所有平台
./scripts/build.sh build-all
```

### 测试
```bash
# 运行所有测试
go test ./...

# 运行带覆盖率的测试
go test -cover ./...

# 代码质量检查
./scripts/dev-verify.sh
```

### 代码质量
项目严格遵循 Go 最佳实践：

```bash
# 格式化
go fmt ./...

# 静态分析
go vet ./...

# Lint 检查
golangci-lint run

# 依赖整理
go mod tidy
```

## 📝 配置文件

XRF-Go 使用 `/etc/xray/confs/` 目录存储配置：

```
/etc/xray/confs/
├── 00-log.json         # 日志配置
├── 01-dns.json         # DNS 配置
├── 10-inbound-*.json   # 入站协议
├── 20-outbound-*.json  # 出站配置
└── 90-routing.json     # 路由规则
```

## 🔐 权限要求

XRF-Go 需要管理员权限来执行以下操作：

- **端口绑定**: 绑定特权端口（80, 443）
- **系统文件**: 写入 `/usr/local/bin`, `/etc/xray`
- **系统服务**: 管理 systemd 服务
- **系统优化**: 配置 BBR、文件描述符限制

支持两种运行方式：
- **root 用户**: 直接运行
- **普通用户**: 通过 sudo 运行

## 🤝 贡献指南

我们欢迎各种形式的贡献！

### 提交问题
- [Bug 报告](https://github.com/Joe-oss9527/xrf-go/issues/new?template=bug_report.md)
- [功能请求](https://github.com/Joe-oss9527/xrf-go/issues/new?template=feature_request.md)

### 提交代码
1. Fork 本项目
2. 创建特性分支：`git checkout -b feature/amazing-feature`
3. 提交更改：`git commit -m 'Add amazing feature'`
4. 推送分支：`git push origin feature/amazing-feature`
5. 提交 Pull Request

### 代码规范
- 遵循 Go 官方代码风格
- 提交前运行 `./scripts/dev-verify.sh`
- 编写测试覆盖新功能
- 更新相关文档

## 📄 许可证

本项目采用 [MIT 许可证](LICENSE)。

## 🔗 相关链接

- **官方文档**: [Xray 项目](https://xtls.github.io/)
- **问题反馈**: [GitHub Issues](https://github.com/Joe-oss9527/xrf-go/issues)
- **更新日志**: [CHANGELOG.md](CHANGELOG.md)
- **开发文档**: [CLAUDE.md](CLAUDE.md)

## 🙏 致谢

感谢以下优秀的开源项目：

- [Xray-core](https://github.com/XTLS/Xray-core) - 高性能代理核心
- [Cobra](https://github.com/spf13/cobra) - CLI 框架
- [Lego](https://github.com/go-acme/lego) - ACME 客户端
- [UUID](https://github.com/google/uuid) - UUID 生成器

---

<div align="center">

**如果这个项目对您有帮助，请给个 ⭐ Star！**

</div>