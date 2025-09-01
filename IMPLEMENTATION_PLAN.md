# XRF-Go 实施计划

## 项目概述
实现一个简洁高效的 Xray 安装配置工具，核心理念是"高效率，超快速，极易用"，充分利用 Xray 的 confdir 多配置文件特性。

## 实施阶段

### 第一阶段：项目基础架构搭建（1-2天）

#### 1.1 初始化项目结构
- 创建 Go 项目，初始化 go.mod
- 建立目录结构：cmd/xrf/, pkg/{config,system,tls,api,utils}/
- 配置基本的构建和测试环境

#### 1.2 实现核心工具模块 (pkg/utils/)
- **logger.go**: 简单日志系统，支持不同级别
- **colors.go**: 终端彩色输出
- **validator.go**: 配置验证器
- **http.go**: HTTP 工具函数
- **crypto.go**: UUID、密码生成工具

### 第二阶段：配置管理核心（2-3天）

#### 2.1 配置模板系统 (pkg/config/templates.go)
- 将所有 JSON 配置模板作为 Go 常量内嵌
- 实现模板渲染系统，支持变量替换
- 包含所有协议模板：VLESS-REALITY, VMess-WS, Trojan, HTTPUpgrade 等

#### 2.2 协议定义 (pkg/config/protocols.go)
```go
- 定义 Protocol 结构体
- 实现协议别名映射（vr→VLESS-REALITY）
- 默认端口和配置管理
```

#### 2.3 多配置文件管理器 (pkg/config/manager.go)
- 实现 ConfigManager 核心功能
- 文件命名策略（00-99优先级系统）
- 配置文件 CRUD 操作
- 利用 Xray confdir 特性的配置合并逻辑

### 第三阶段：系统管理模块（2天）

#### 3.1 系统检测器 (pkg/system/detector.go)
- OS 类型和版本检测
- 系统架构检测
- 依赖检查（systemd, firewall等）

#### 3.2 Xray 安装器 (pkg/system/installer.go)
- 自动下载正确版本的 Xray
- 二进制文件安装和权限设置
- 目录结构创建（/etc/xray/confs/）

#### 3.3 服务管理 (pkg/system/service.go)
- systemd 服务文件生成（使用 confdir 模式）
- 服务控制：start/stop/restart/status
- 开机自启配置

#### 3.4 系统优化
- BBR 拥塞控制自动配置
- 防火墙规则管理
- 系统参数优化

### 第四阶段：TLS 和证书管理（1-2天）

#### 4.1 ACME 证书管理 (pkg/tls/acme.go)
- Let's Encrypt 证书自动申请
- 证书续期管理
- 证书文件管理

#### 4.2 Caddy 集成 (pkg/tls/caddy.go)
- Caddy 作为前置代理配置
- 自动 HTTPS 和伪装网站

### 第五阶段：Xray API 集成（1天）

#### 5.1 API 客户端 (pkg/api/client.go)
- Xray gRPC API 连接
- 动态添加/删除入站
- 流量统计获取
- 配置热重载

### 第六阶段：CLI 命令实现（2-3天）

#### 6.1 主命令入口 (cmd/xrf/main.go)
使用 Cobra 框架实现所有命令：

**安装命令**
- `xrf install` - 一键安装
- 支持 --protocol, --domain, --port 等参数

**协议管理**
- `xrf add [protocol]` - 添加协议（支持别名）
- `xrf list` - 列出所有协议
- `xrf info [name]` - 查看配置详情
- `xrf remove [name]` - 删除协议
- `xrf change [name] [key] [value]` - 修改配置

**服务控制**
- `xrf start/stop/restart/status` - 服务管理
- `xrf reload` - 热重载

**实用工具**
- `xrf generate [type]` - 生成密码/UUID/密钥
- `xrf url/qr [name]` - 生成分享链接/二维码
- `xrf logs/logerr` - 查看日志
- `xrf backup/restore` - 备份恢复
- `xrf bbr` - 启用 BBR
- `xrf test` - 配置验证

### 第七阶段：高级功能（1-2天）

#### 7.1 配置验证和回滚
- 修改前自动验证（xray test -confdir）
- 失败自动回滚机制
- 配置备份和恢复

#### 7.2 智能端口管理
- 端口占用检测
- 自动分配可用端口
- 端口冲突处理

#### 7.3 热重载支持
- 文件监控机制
- USR1 信号发送
- 无中断配置更新

### 第八阶段：测试和优化（2天）

#### 8.1 单元测试
- 各模块单元测试
- 配置合并逻辑测试
- 模板渲染测试

#### 8.2 集成测试
- 完整安装流程测试
- 多协议配置测试
- 服务管理测试

#### 8.3 性能优化
- 配置文件操作优化
- 启动速度优化（目标：添加配置<1秒）

### 第九阶段：部署和文档（1天）

#### 9.1 构建和发布
- 多平台交叉编译
- 单二进制文件打包
- 一键安装脚本 (scripts/install.sh)

#### 9.2 文档编写
- README.md 使用说明
- API 文档
- 配置示例

## 技术要点

### 核心设计原则
1. **单二进制文件** - 所有模板内嵌，无外部依赖
2. **多配置文件** - 充分利用 Xray confdir 特性
3. **快速操作** - 添加配置必须<1秒完成
4. **智能默认值** - 自动选择最优配置
5. **容错设计** - 失败自动回滚

### 配置文件命名策略
- 00-09: 基础配置（log, api, dns）
- 10-19: 入站协议配置
- 20-29: 出站配置
- 90-99: 路由规则（支持 tail）

### 关键实现细节
1. 所有配置模板以 Go 常量形式内嵌
2. 运行时动态生成到 /etc/xray/confs/
3. 严格遵循 Xray 配置合并规则
4. 支持 2025 年最新协议和优化

## 时间线
- **总工期**: 约 12-15 个工作日
- **第一个可用版本**: 第 6 阶段完成后（约 10 天）
- **完整版本**: 全部阶段完成（约 15 天）

## 风险管理
1. **Xray API 兼容性** - 需要测试不同版本
2. **系统兼容性** - 需支持主流 Linux 发行版
3. **证书管理** - ACME 限制和错误处理
4. **性能目标** - 确保 1 秒内完成配置添加

## 实施顺序建议

### 快速原型（3-5天）
优先实现核心功能，快速验证设计：
1. 基础项目结构
2. 配置模板和管理器
3. 基本 CLI 命令
4. 简单的安装和添加协议功能

### 功能完善（5-7天）
1. 系统管理完整功能
2. TLS/证书管理
3. API 集成
4. 所有 CLI 命令

### 优化和测试（3-5天）
1. 性能优化
2. 错误处理完善
3. 测试覆盖
4. 文档编写

这个计划完全对齐 DESIGN.md 的架构设计，实现所有 requirements.md 中的需求。