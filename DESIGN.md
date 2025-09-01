# XRF-Go: 简洁高效的Xray安装配置工具

## 设计理念

**高效率，超快速，极易用** - 专注核心功能，避免过度工程化

设计理念继承自233boy项目，以**多配置同时运行**为核心设计，专门优化了添加、更改、查看、删除四项常用功能。

- **一键操作**: 复杂任务通过单个命令完成，添加配置仅需不到1秒
- **零学习成本**: 直观的命令设计，强大的快捷参数  
- **模块化配置**: 充分利用Xray多配置文件特性
- **自动化TLS**: 自动配置证书和伪装网站
- **API操作**: 使用Xray API进行配置管理

## 项目结构 (简化版)

```
xrf-go/
├── cmd/xrf/
│   └── main.go                 # 主入口，包含所有命令
├── pkg/
│   ├── config/                 # 配置管理 (核心)
│   │   ├── manager.go         # 多配置文件管理器
│   │   ├── templates.go       # 内置配置模板
│   │   └── protocols.go       # 协议定义和支持列表
│   ├── system/                 # 系统操作
│   │   ├── detector.go        # 系统检测
│   │   ├── installer.go       # 安装器
│   │   └── service.go         # 服务管理
│   ├── tls/                    # TLS/证书管理
│   │   ├── acme.go           # 自动证书申请
│   │   └── caddy.go          # Caddy集成
│   ├── api/                    # Xray API接口
│   │   └── client.go         # API客户端
│   └── utils/
│       ├── logger.go          # 简单日志
│       ├── http.go            # HTTP工具
│       ├── colors.go          # 终端颜色输出
│       └── validator.go       # 配置验证
├── scripts/
│   └── install.sh             # 一键安装脚本
├── go.mod
├── go.sum
└── README.md
```

## 核心接口 (最小化)

### 配置管理器
```go
type ConfigManager struct {
    baseDir string
}

// 核心方法
func (c *ConfigManager) Install(protocols []string, domain string) error
func (c *ConfigManager) AddProtocol(protocolType, tag string, options map[string]interface{}) error
func (c *ConfigManager) RemoveProtocol(tag string) error
func (c *ConfigManager) ListProtocols() []ProtocolInfo
func (c *ConfigManager) UpdateProtocol(tag string, options map[string]interface{}) error

// 配置文件操作 (利用Xray多配置特性)
func (c *ConfigManager) createConfigFile(priority int, module, name string, config interface{}) error
func (c *ConfigManager) removeConfigFile(filename string) error
```

### 系统管理器
```go
type SystemManager struct{}

func (s *SystemManager) DetectOS() (string, error)
func (s *SystemManager) InstallXray(version string) error
func (s *SystemManager) InstallService() error
func (s *SystemManager) ManageService(action string) error // start/stop/restart/status
func (s *SystemManager) ConfigureFirewall(ports []int) error
func (s *SystemManager) EnableBBR() error
func (s *SystemManager) CheckSystemRequirements() error
```

### API管理器
```go
type APIManager struct {
    client *api.Client
}

func (a *APIManager) AddInbound(config *InboundConfig) error
func (a *APIManager) RemoveInbound(tag string) error
func (a *APIManager) GetStats(tag string) (*Stats, error)
func (a *APIManager) RestartCore() error
```

### 协议定义
```go
type Protocol struct {
    Name        string
    Aliases     []string  // 快捷别名，如 "vr" for VLESS-REALITY
    DefaultPort int
    RequiresTLS bool
    SupportedTransports []string
}

var SupportedProtocols = []Protocol{
    {Name: "VLESS-REALITY", Aliases: []string{"vr", "vless"}, DefaultPort: 443},
    {Name: "VLESS-WS-TLS", Aliases: []string{"vw"}, DefaultPort: 443, RequiresTLS: true},
    {Name: "VMess-WS-TLS", Aliases: []string{"vmess", "mw"}, DefaultPort: 80},
    {Name: "Trojan-WS-TLS", Aliases: []string{"tw", "trojan"}, DefaultPort: 443, RequiresTLS: true},
    {Name: "Shadowsocks", Aliases: []string{"ss"}, DefaultPort: 8388},
    {Name: "Shadowsocks-2022", Aliases: []string{"ss2022"}, DefaultPort: 8388},
    {Name: "VLESS-HTTPUpgrade", Aliases: []string{"httpupgrade", "hu"}, DefaultPort: 8080},
}
```

## 主要命令 (简化)

### 1. 安装命令
```bash
# 基本安装 - 一键完成
xrf install                                    # 默认VLESS-REALITY
xrf install --protocol vmess                   # 指定协议
xrf install --domain example.com --protocols vless,vmess,trojan  # 多协议

# 安装选项
xrf install --port 443 --domain example.com --enable-bbr --auto-firewall
```

### 2. 协议管理
```bash
# 添加协议 (自动处理多配置文件，瞬间完成)
xrf add vless --port 8443 --domain example.com   # 添加VLESS-REALITY
xrf add vmess --port 80 --path /ws               # 添加VMess-WebSocket
xrf add trojan --port 443 --domain example.com  # 添加Trojan-TLS
xrf add ss --method aes-256-gcm --password auto  # 添加Shadowsocks
xrf add httpupgrade --port 8080 --path /upgrade  # 添加VLESS-HTTPUpgrade

# 快捷添加 (使用协议别名)
xrf add vr                      # VLESS-REALITY (默认)
xrf add vw                      # VLESS-WebSocket-TLS  
xrf add tw                      # Trojan-WebSocket-TLS
xrf add ss2022                  # Shadowsocks 2022

# 管理协议
xrf list                        # 显示所有协议配置
xrf info vless-reality          # 查看指定配置详情
xrf remove vmess-ws            # 删除协议配置
xrf change vless-reality port 8080    # 修改端口
xrf change vless-reality domain new.com # 修改域名
```

### 3. 服务管理
```bash
xrf start     # 启动服务
xrf stop      # 停止服务
xrf restart   # 重启服务
xrf status    # 服务状态
xrf reload    # 热重载配置
```

### 4. 实用工具
```bash
# 生成工具
xrf generate password          # 生成随机密码
xrf generate uuid             # 生成UUID  
xrf generate ss2022           # 生成SS2022密码
xrf pbk                       # 生成X25519密钥对
xrf get-port                  # 获取可用端口

# 配置管理
xrf config show               # 显示当前配置
xrf config backup             # 备份配置
xrf config restore            # 恢复配置
xrf url <name>                # 获取分享链接
xrf qr <name>                 # 显示二维码

# 系统工具
xrf logs                      # 查看运行日志
xrf logerr                    # 查看错误日志  
xrf test                      # 测试配置
xrf ip                        # 显示服务器IP
xrf bbr                       # 启用BBR加速
```

## 多配置文件实现 (核心特性)

### 充分利用 -confdir 特性的文件结构
```
/etc/xray/confs/
├── 00-base.json              # 基础配置 (log, api, stats)
├── 01-dns.json               # DNS 配置模块
├── 10-inbound-vless.json     # VLESS-REALITY 入站 (tag: vless-in)
├── 11-inbound-vmess.json     # VMess WebSocket 入站 (tag: vmess-in)  
├── 12-inbound-vless-httpupgrade.json  # VLESS HTTPUpgrade 入站
├── 13-inbound-trojan.json    # Trojan 入站 (tag: trojan-in)
├── 20-outbound-direct.json   # 直连出站 (tag: direct)
├── 21-outbound-block.json    # 阻断出站 (tag: block)
├── 90-routing.json           # 路由规则 (添加到前面)
└── 99-routing-tail.json      # 最终路由规则 (添加到末尾)
```

### 配置合并规则实现
```go
// 严格按照Xray多配置文件合并规则实现
func (c *ConfigManager) AddProtocol(protocolType, tag string, options map[string]interface{}) error {
    config := map[string]interface{}{
        "inbounds": []map[string]interface{}{
            c.generateInboundConfig(protocolType, tag, options),
        },
    }
    
    // 入站配置文件：自动添加到数组末尾
    filename := fmt.Sprintf("1%d-inbound-%s.json", c.getNextInboundID(), tag)
    return c.saveConfigFile(filename, config)
}

func (c *ConfigManager) AddOutbound(outboundType, tag string, options map[string]interface{}) error {
    config := map[string]interface{}{
        "outbounds": []map[string]interface{}{
            c.generateOutboundConfig(outboundType, tag, options),
        },
    }
    
    // 出站配置文件：根据类型决定添加位置
    var filename string
    if strings.Contains(outboundType, "tail") {
        // 包含"tail"的添加到末尾
        filename = fmt.Sprintf("9%d-outbound-%s-tail.json", c.getNextOutboundID(), tag)
    } else {
        // 默认添加到开头
        filename = fmt.Sprintf("2%d-outbound-%s.json", c.getNextOutboundID(), tag)
    }
    return c.saveConfigFile(filename, config)
}

// 利用tag替换机制更新配置
func (c *ConfigManager) UpdateProtocol(tag string, options map[string]interface{}) error {
    // 直接覆盖具有相同tag的配置文件
    // Xray会自动根据tag进行配置替换
    return c.updateConfigFileByTag(tag, options)
}

// 配置预览 - 模拟Xray的配置合并过程
func (c *ConfigManager) PreviewMergedConfig() (*Config, error) {
    files := c.getConfigFilesInOrder()
    merged := &Config{}
    
    for _, file := range files {
        config := c.loadConfigFile(file)
        merged = c.mergeConfigs(merged, config) // 按Xray规则合并
    }
    return merged, nil
}
```

### 配置文件操作策略
```go
// 配置文件命名约定 (严格遵循加载顺序)
type ConfigFile struct {
    Priority   int    // 00-99 控制加载顺序
    Type       string // base, dns, routing, inbound, outbound  
    Name       string // 协议或功能名称
    Tail       bool   // 是否添加到末尾 (仅outbound有效)
}

func (cf *ConfigFile) Filename() string {
    if cf.Tail && cf.Type == "outbound" {
        return fmt.Sprintf("%02d-%s-%s-tail.json", cf.Priority, cf.Type, cf.Name)
    }
    return fmt.Sprintf("%02d-%s-%s.json", cf.Priority, cf.Type, cf.Name)
}

// 智能配置管理
func (c *ConfigManager) smartConfigManagement() {
    // 1. 基础配置 (00-09)：系统级配置，很少修改
    // 2. 入站配置 (10-19)：按需添加，支持热更新
    // 3. 出站配置 (20-29)：策略配置，支持优先级
    // 4. 路由配置 (90-99)：规则配置，支持tail模式
}
```

## 内置配置模板 (内嵌到二进制)

### 1. 基础配置模板 (00-base.json)
```json
{
  "log": {
    "level": "warning",
    "dnsLog": false
  },
  "api": {
    "tag": "api",
    "services": ["HandlerService", "LoggerService", "StatsService"]
  },
  "stats": {},
  "policy": {
    "levels": {
      "0": {
        "statsUserUplink": true,
        "statsUserDownlink": true
      }
    }
  }
}
```

### 2. DNS配置模板 (01-dns.json)
```json
{
  "dns": {
    "servers": [
      {
        "address": "223.5.5.5",
        "domains": ["geosite:cn"]
      },
      {
        "address": "8.8.8.8",
        "domains": ["geosite:geolocation-!cn"]
      }
    ]
  }
}
```

### 3. 入站协议模板

#### VLESS-REALITY (10-inbound-vless.json)
```json
{
  "inbounds": [
    {
      "tag": "vless-reality",
      "port": 443,
      "protocol": "vless",
      "settings": {
        "clients": [
          {
            "id": "{{.UUID}}",
            "flow": "xtls-rprx-vision",
            "level": 0
          }
        ],
        "decryption": "none"
      },
      "streamSettings": {
        "network": "tcp",
        "security": "reality",
        "realitySettings": {
          "show": false,
          "dest": "{{.Dest}}:443",
          "serverNames": ["{{.ServerName}}"],
          "privateKey": "{{.PrivateKey}}",
          "shortIds": ["{{.ShortId}}"],
          "minClientVer": "",
          "maxClientVer": "",
          "maxTimeDiff": 0,
          "spiderX": ""
        },
        "sockopt": {
          "tcpKeepAliveIdle": 300,
          "tcpUserTimeout": 10000
        }
      }
    }
  ]
}
```

#### VMess WebSocket (11-inbound-vmess.json)
```json
{
  "inbounds": [
    {
      "tag": "vmess-ws",
      "port": 80,
      "protocol": "vmess",
      "settings": {
        "clients": [
          {
            "id": "{{.UUID}}",
            "alterId": 0,
            "level": 0
          }
        ]
      },
      "streamSettings": {
        "network": "ws",
        "wsSettings": {
          "acceptProxyProtocol": false,
          "path": "{{.Path}}",
          "host": "{{.Host}}",
          "headers": {}
        },
        "sockopt": {
          "tcpKeepAliveIdle": 300,
          "tcpUserTimeout": 10000
        }
      }
    }
  ]
}
```

#### VLESS HTTPUpgrade (12-inbound-vless-httpupgrade.json)  
```json
{
  "inbounds": [
    {
      "tag": "vless-httpupgrade",
      "port": 8080,
      "protocol": "vless",
      "settings": {
        "clients": [
          {
            "id": "{{.UUID}}",
            "level": 0
          }
        ],
        "decryption": "none"
      },
      "streamSettings": {
        "network": "httpupgrade",
        "httpupgradeSettings": {
          "acceptProxyProtocol": false,
          "path": "{{.Path}}",
          "host": "{{.Host}}",
          "headers": {}
        }
      }
    }
  ]
}
```

#### Trojan (13-inbound-trojan.json)
```json
{
  "inbounds": [
    {
      "tag": "trojan",
      "port": 443,
      "protocol": "trojan",
      "settings": {
        "clients": [
          {
            "password": "{{.Password}}",
            "level": 0
          }
        ]
      },
      "streamSettings": {
        "network": "tcp",
        "security": "tls",
        "tlsSettings": {
          "certificates": [
            {
              "certificateFile": "{{.CertFile}}",
              "keyFile": "{{.KeyFile}}"
            }
          ]
        }
      }
    }
  ]
}
```

### 4. 出站配置模板

#### 直连出站 (20-outbound-direct.json)
```json
{
  "outbounds": [
    {
      "tag": "direct",
      "protocol": "freedom",
      "settings": {
        "domainStrategy": "UseIPv4"
      }
    }
  ]
}
```

#### 阻断出站 (21-outbound-block.json)
```json
{
  "outbounds": [
    {
      "tag": "block",
      "protocol": "blackhole",
      "settings": {
        "response": {
          "type": "http"
        }
      }
    }
  ]
}
```

### 5. 路由配置模板

#### 基础路由 (90-routing.json) - 添加到outbounds前面
```json
{
  "routing": {
    "domainStrategy": "IPIfNonMatch",
    "rules": [
      {
        "type": "field",
        "ip": ["geoip:private"],
        "outboundTag": "direct"
      },
      {
        "type": "field",
        "domain": ["geosite:cn"],
        "outboundTag": "direct"
      }
    ]
  }
}
```

#### 最终路由 (99-routing-tail.json) - 添加到outbounds末尾  
```json
{
  "routing": {
    "rules": [
      {
        "type": "field",
        "network": "tcp,udp",
        "outboundTag": "direct"
      }
    ]
  }
}
```

### 配置文件管理策略

#### 利用confdir特性的优势:
1. **模块化管理**: 每个协议一个文件，便于启用/禁用
2. **顺序控制**: 数字前缀严格控制配置加载和合并顺序
3. **tag替换**: 相同tag的配置自动替换，支持协议更新
4. **数组合并**: inbound添加到末尾，outbound添加到开头
5. **tail支持**: 特殊的routing-tail文件添加到数组末尾

#### 配置模板内嵌策略:
- 所有配置模板以Go字符串常量形式内嵌到 `pkg/config/templates.go`
- 运行时根据需要动态生成配置文件到 `/etc/xray/confs/`
- 保持单二进制文件的便携性，无外部依赖
- 支持2025年最新协议: REALITY、HTTPUpgrade、优化的WebSocket配置
- 内置socket优化参数提升网络性能

#### 实际应用示例:
```bash
# Xray启动命令利用confdir
xray run -confdir /etc/xray/confs

# 配置文件会按文件名排序加载：
# 00-base.json → 01-dns.json → 10-inbound-vless.json → ... → 99-routing-tail.json
```

## 实现要点

### 1. 单二进制文件
- 所有配置模板内嵌到二进制中
- 无外部依赖文件
- 一个可执行文件解决所有问题

### 2. 智能默认值
- 自动检测系统并使用最佳默认配置
- 端口冲突自动处理
- 证书自动生成和续期

### 3. 容错设计
- 操作失败自动回滚
- 配置验证防止错误
- 详细的错误提示和修复建议

### 4. 多配置文件策略 (confdir核心实现)
```go
// 服务管理 - 完全基于confdir模式
func (s *SystemManager) InstallService() error {
    serviceContent := `[Unit]
Description=Xray Service (confdir mode)
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
WantedBy=multi-user.target`

    return s.writeServiceFile("/etc/systemd/system/xray.service", serviceContent)
}

// 配置热重载 - 利用confdir的文件监控
func (c *ConfigManager) EnableHotReload() error {
    // 监控 /etc/xray/confs/ 目录变化
    // 文件变化时发送 USR1 信号给 xray 进程重载配置
    return c.watchConfigDir("/etc/xray/confs")
}

// 配置验证 - 在修改前验证整个confdir
func (c *ConfigManager) ValidateConfDir() error {
    // 使用 xray test -confdir /etc/xray/confs 验证
    cmd := exec.Command("xray", "test", "-confdir", "/etc/xray/confs")
    return cmd.Run()
}
```

### 5. 配置文件生成策略
```go
// 根据协议类型自动选择合适的优先级和配置结构
func (c *ConfigManager) getConfigPriority(configType, protocolType string) int {
    priorityMap := map[string]map[string]int{
        "base":     {"system": 0},
        "dns":      {"default": 1},
        "inbound":  {"vless": 10, "vmess": 11, "httpupgrade": 12, "trojan": 13, "ss": 14},
        "outbound": {"direct": 20, "block": 21, "proxy": 25},
        "routing":  {"basic": 90, "china": 91, "tail": 99},
    }
    
    if typeMap, exists := priorityMap[configType]; exists {
        if priority, exists := typeMap[protocolType]; exists {
            return priority
        }
        if priority, exists := typeMap["default"]; exists {
            return priority
        }
    }
    return 50 // 默认优先级
}

// 配置文件命名严格遵循confdir加载规则
func (c *ConfigManager) generateFilename(configType, name string, isTail bool) string {
    priority := c.getConfigPriority(configType, name)
    
    if isTail {
        return fmt.Sprintf("%02d-%s-%s-tail.json", priority, configType, name)
    }
    return fmt.Sprintf("%02d-%s-%s.json", priority, configType, name)
}
```

## 使用示例 (简化后)

### 典型工作流
```bash
# 1. 一键安装
curl -fsSL https://get.xrf.sh | bash -s -- --protocol vless-reality --domain example.com

# 2. 添加更多协议
xrf add vmess --port 80 --path /ws
xrf add trojan --port 8443

# 3. 管理服务
xrf restart  # 重启生效
xrf status   # 检查状态

# 4. 查看配置
xrf list     # 列出所有协议
xrf config   # 显示完整配置
```

### 高级操作
```bash
# 协议快速切换
xrf switch vless-reality  # 快速切换到单协议模式
xrf enable-all           # 启用所有协议

# 系统维护
xrf update               # 更新Xray版本
xrf clean                # 清理日志和临时文件
xrf backup               # 备份配置
```

这个简化设计保持了233boy项目的简洁性，同时充分利用了Xray的多配置文件特性，避免了过度工程化的问题。

## 参考链接

### 官方文档
- [Xray官方配置文档](https://xtls.github.io/config/) - Xray配置详细说明
- [Xray多文件配置](https://xtls.github.io/config/features/multiple.html) - confdir特性详解
- [VLESS协议文档](https://xtls.github.io/config/outbounds/vless.html) - VLESS配置参数
- [WebSocket传输配置](https://xtls.github.io/en/config/transports/websocket.html) - WebSocket传输设置
- [REALITY传输配置](https://xtls.github.io/config/transport.html) - 最新REALITY配置

### 参考项目
- [233boy/Xray](https://github.com/233boy/Xray) - 最好用的Xray一键安装脚本，本项目的重要参考
- [XTLS/Xray-core](https://github.com/XTLS/Xray-core) - Xray核心项目
- [Project X](https://xtls.github.io/) - Xray项目官网

### 相关技术
- [Go语言官方文档](https://golang.org/doc/) - Go开发参考
- [Cobra CLI框架](https://github.com/spf13/cobra) - Go命令行工具框架
- [systemd服务配置](https://www.freedesktop.org/software/systemd/man/systemd.service.html) - Linux服务管理

### 设计参考
- [233boy项目文档](https://233boy.com/xray/xray-script/) - 使用教程和设计理念
- [Xray多配置文件最佳实践](https://xtls.github.io/config/features/multiple.html) - 配置文件组织方法