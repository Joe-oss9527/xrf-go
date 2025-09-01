package config

import (
	"fmt"
	"strings"
)

// Protocol 定义协议结构
type Protocol struct {
	Name             string   `json:"name"`
	Aliases          []string `json:"aliases"`
	DefaultPort      int      `json:"default_port"`
	RequiresTLS      bool     `json:"requires_tls"`
	RequiresDomain   bool     `json:"requires_domain"`
	SupportedTransports []string `json:"supported_transports"`
	Template         string   `json:"template"`
	Description      string   `json:"description"`
}

// ProtocolInfo 运行时协议信息
type ProtocolInfo struct {
	Tag         string                 `json:"tag"`
	Type        string                 `json:"type"`
	Port        int                    `json:"port"`
	Status      string                 `json:"status"`
	ConfigFile  string                 `json:"config_file"`
	Settings    map[string]interface{} `json:"settings"`
	ShareURL    string                 `json:"share_url,omitempty"`
}

// 定义所有支持的协议
var SupportedProtocols = []Protocol{
	{
		Name:        "VLESS-REALITY",
		Aliases:     []string{"vr", "vless", "reality"},
		DefaultPort: 443,
		RequiresTLS: false,
		RequiresDomain: true,
		SupportedTransports: []string{"tcp"},
		Template:    "vless-reality",
		Description: "VLESS with REALITY transport (recommended)",
	},
	{
		Name:        "VLESS-WebSocket-TLS",
		Aliases:     []string{"vw", "vless-ws"},
		DefaultPort: 443,
		RequiresTLS: true,
		RequiresDomain: true,
		SupportedTransports: []string{"ws"},
		Template:    "vless-ws",
		Description: "VLESS with WebSocket and TLS",
	},
	{
		Name:        "VMess-WebSocket-TLS",
		Aliases:     []string{"vmess", "mw", "vmess-ws"},
		DefaultPort: 80,
		RequiresTLS: false,
		RequiresDomain: false,
		SupportedTransports: []string{"ws"},
		Template:    "vmess-ws",
		Description: "VMess with WebSocket transport",
	},
	{
		Name:        "Trojan-WebSocket-TLS",
		Aliases:     []string{"tw", "trojan", "trojan-ws"},
		DefaultPort: 443,
		RequiresTLS: true,
		RequiresDomain: true,
		SupportedTransports: []string{"ws"},
		Template:    "trojan-ws",
		Description: "Trojan with WebSocket and TLS",
	},
	{
		Name:        "Shadowsocks",
		Aliases:     []string{"ss"},
		DefaultPort: 8388,
		RequiresTLS: false,
		RequiresDomain: false,
		SupportedTransports: []string{"tcp", "udp"},
		Template:    "shadowsocks",
		Description: "Shadowsocks protocol",
	},
	{
		Name:        "Shadowsocks-2022",
		Aliases:     []string{"ss2022"},
		DefaultPort: 8388,
		RequiresTLS: false,
		RequiresDomain: false,
		SupportedTransports: []string{"tcp", "udp"},
		Template:    "shadowsocks-2022",
		Description: "Shadowsocks 2022 with improved security",
	},
	{
		Name:        "VLESS-HTTPUpgrade",
		Aliases:     []string{"httpupgrade", "hu", "vless-hu"},
		DefaultPort: 8080,
		RequiresTLS: false,
		RequiresDomain: false,
		SupportedTransports: []string{"httpupgrade"},
		Template:    "vless-httpupgrade",
		Description: "VLESS with HTTP Upgrade transport",
	},
}

// ProtocolManager 协议管理器
type ProtocolManager struct {
	protocols map[string]Protocol
	aliases   map[string]string
}

func NewProtocolManager() *ProtocolManager {
	pm := &ProtocolManager{
		protocols: make(map[string]Protocol),
		aliases:   make(map[string]string),
	}
	
	// 初始化协议映射
	for _, protocol := range SupportedProtocols {
		protocolKey := strings.ToLower(protocol.Name)
		pm.protocols[protocolKey] = protocol
		
		// 建立别名映射
		for _, alias := range protocol.Aliases {
			pm.aliases[strings.ToLower(alias)] = protocolKey
		}
	}
	
	return pm
}

// GetProtocol 根据名称或别名获取协议
func (pm *ProtocolManager) GetProtocol(nameOrAlias string) (Protocol, error) {
	nameOrAlias = strings.ToLower(nameOrAlias)
	
	// 直接查找协议名
	if protocol, exists := pm.protocols[nameOrAlias]; exists {
		return protocol, nil
	}
	
	// 通过别名查找
	if protocolName, exists := pm.aliases[nameOrAlias]; exists {
		return pm.protocols[protocolName], nil
	}
	
	return Protocol{}, fmt.Errorf("unsupported protocol: %s", nameOrAlias)
}

// IsProtocolSupported 检查协议是否支持
func (pm *ProtocolManager) IsProtocolSupported(nameOrAlias string) bool {
	_, err := pm.GetProtocol(nameOrAlias)
	return err == nil
}

// GetAllProtocols 获取所有支持的协议
func (pm *ProtocolManager) GetAllProtocols() []Protocol {
	protocols := make([]Protocol, 0, len(pm.protocols))
	for _, protocol := range pm.protocols {
		protocols = append(protocols, protocol)
	}
	return protocols
}

// GetProtocolByTemplate 根据模板名获取协议
func (pm *ProtocolManager) GetProtocolByTemplate(template string) (Protocol, error) {
	for _, protocol := range pm.protocols {
		if protocol.Template == template {
			return protocol, nil
		}
	}
	return Protocol{}, fmt.Errorf("protocol not found for template: %s", template)
}

// GetDefaultSettings 获取协议的默认设置
func (pm *ProtocolManager) GetDefaultSettings(protocolName string) map[string]interface{} {
	protocol, err := pm.GetProtocol(protocolName)
	if err != nil {
		return nil
	}
	
	settings := map[string]interface{}{
		"port": protocol.DefaultPort,
		"requiresTLS": protocol.RequiresTLS,
		"requiresDomain": protocol.RequiresDomain,
		"supportedTransports": protocol.SupportedTransports,
	}
	
	// 根据协议类型添加特定设置
	switch strings.ToLower(protocol.Name) {
	case "vless-reality":
		settings["transport"] = "tcp"
		settings["security"] = "reality"
		settings["dest"] = "www.microsoft.com"
		settings["serverName"] = "www.microsoft.com"
		
	case "vless-websocket-tls", "vmess-websocket-tls", "trojan-websocket-tls":
		settings["transport"] = "ws"
		settings["path"] = "/ws"
		if protocol.RequiresTLS {
			settings["security"] = "tls"
		}
		
	case "vless-httpupgrade":
		settings["transport"] = "httpupgrade"
		settings["path"] = "/upgrade"
		
	case "shadowsocks":
		settings["method"] = "chacha20-poly1305"
		
	case "shadowsocks-2022":
		settings["method"] = "2022-blake3-aes-256-gcm"
	}
	
	return settings
}

// GetProtocolAliases 获取协议的所有别名
func (pm *ProtocolManager) GetProtocolAliases(protocolName string) []string {
	protocol, err := pm.GetProtocol(protocolName)
	if err != nil {
		return nil
	}
	return protocol.Aliases
}

// GetProtocolDescription 获取协议描述
func (pm *ProtocolManager) GetProtocolDescription(protocolName string) string {
	protocol, err := pm.GetProtocol(protocolName)
	if err != nil {
		return ""
	}
	return protocol.Description
}

// ValidateProtocolSettings 验证协议设置
func (pm *ProtocolManager) ValidateProtocolSettings(protocolName string, settings map[string]interface{}) error {
	protocol, err := pm.GetProtocol(protocolName)
	if err != nil {
		return err
	}
	
	// 验证端口
	if port, exists := settings["port"]; exists {
		if portInt, ok := port.(int); ok {
			if portInt < 1 || portInt > 65535 {
				return fmt.Errorf("invalid port: %d", portInt)
			}
		}
	}
	
	// 验证域名要求
	if protocol.RequiresDomain {
		if domain, exists := settings["domain"]; !exists || domain == "" {
			return fmt.Errorf("protocol %s requires domain", protocol.Name)
		}
	}
	
	// 验证传输方式
	if transport, exists := settings["transport"]; exists {
		if transportStr, ok := transport.(string); ok {
			supported := false
			for _, supportedTransport := range protocol.SupportedTransports {
				if transportStr == supportedTransport {
					supported = true
					break
				}
			}
			if !supported {
				return fmt.Errorf("transport %s not supported by protocol %s", transportStr, protocol.Name)
			}
		}
	}
	
	return nil
}

// GetRecommendedProtocols 获取推荐的协议列表
func (pm *ProtocolManager) GetRecommendedProtocols() []string {
	return []string{
		"VLESS-REALITY",      // 最推荐
		"VLESS-WebSocket-TLS",
		"VMess-WebSocket-TLS",
		"VLESS-HTTPUpgrade",
		"Trojan-WebSocket-TLS",
		"Shadowsocks-2022",
		"Shadowsocks",
	}
}

// SearchProtocols 搜索协议
func (pm *ProtocolManager) SearchProtocols(query string) []Protocol {
	query = strings.ToLower(query)
	var results []Protocol
	
	for _, protocol := range pm.protocols {
		// 搜索协议名
		if strings.Contains(strings.ToLower(protocol.Name), query) {
			results = append(results, protocol)
			continue
		}
		
		// 搜索别名
		for _, alias := range protocol.Aliases {
			if strings.Contains(strings.ToLower(alias), query) {
				results = append(results, protocol)
				break
			}
		}
		
		// 搜索描述
		if strings.Contains(strings.ToLower(protocol.Description), query) {
			results = append(results, protocol)
		}
	}
	
	return results
}

// 全局协议管理器实例
var DefaultProtocolManager = NewProtocolManager()