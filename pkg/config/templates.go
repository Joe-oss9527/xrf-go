package config

import (
	"bytes"
	"fmt"
	"text/template"
)

// 基础配置模板
const BaseConfigTemplate = `{
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
}`

// DNS 配置模板
const DNSConfigTemplate = `{
  "dns": {
    "servers": [
      {
        "address": "223.5.5.5"
      },
      {
        "address": "8.8.8.8"
      }
    ]
  }
}`

// VLESS-REALITY 入站模板
const VLESSRealityInboundTemplate = `{
  "inbounds": [
    {
      "tag": "{{.Tag}}",
      "port": {{.Port}},
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
          "fingerprint": "chrome"
        },
        "sockopt": {
          "tcpKeepAliveIdle": 300,
          "tcpUserTimeout": 10000
        }
      }
    }
  ]
}`

// VLESS-Encryption 入站模板（后量子加密）
const VLESSEncryptionInboundTemplate = `{
  "inbounds": [
    {
      "tag": "{{.Tag}}",
      "port": {{.Port}},
      "protocol": "vless",
      "settings": {
        "clients": [
          {
            "id": "{{.UUID}}",
            {{if .Flow}}
            "flow": "{{.Flow}}",
            {{end}}
            "level": 0
          }
        ],
        "decryption": "{{.Decryption}}"
      },
      "streamSettings": {
        "network": "tcp",
        "sockopt": {
          "tcpKeepAliveIdle": 300,
          "tcpUserTimeout": 10000
        }
      }
    }
  ]
}`

// VLESS-WebSocket-TLS 入站模板
const VLESSWSInboundTemplate = `{
  "inbounds": [
    {
      "tag": "{{.Tag}}",
      "port": {{.Port}},
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
        "network": "ws",
        "security": "{{.Security}}",
        {{if eq .Security "tls"}}
        "tlsSettings": {
          "certificates": [
            {
              "certificateFile": "{{.CertFile}}",
              "keyFile": "{{.KeyFile}}"
            }
          ]
        },
        {{end}}
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
}`

// VMess-WebSocket 入站模板
const VMessWSInboundTemplate = `{
  "inbounds": [
    {
      "tag": "{{.Tag}}",
      "port": {{.Port}},
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
        "security": "{{.Security}}",
        {{if eq .Security "tls"}}
        "tlsSettings": {
          "certificates": [
            {
              "certificateFile": "{{.CertFile}}",
              "keyFile": "{{.KeyFile}}"
            }
          ]
        },
        {{end}}
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
}`

// VLESS-HTTPUpgrade 入站模板
const VLESSHTTPUpgradeInboundTemplate = `{
  "inbounds": [
    {
      "tag": "{{.Tag}}",
      "port": {{.Port}},
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
        },
        "sockopt": {
          "tcpKeepAliveIdle": 300,
          "tcpUserTimeout": 10000
        }
      }
    }
  ]
}`

// Trojan-WebSocket-TLS 入站模板
const TrojanWSInboundTemplate = `{
  "inbounds": [
    {
      "tag": "{{.Tag}}",
      "port": {{.Port}},
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
        "network": "ws",
        "security": "tls",
        "tlsSettings": {
          "certificates": [
            {
              "certificateFile": "{{.CertFile}}",
              "keyFile": "{{.KeyFile}}"
            }
          ]
        },
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
}`

// Shadowsocks 入站模板
const ShadowsocksInboundTemplate = `{
  "inbounds": [
    {
      "tag": "{{.Tag}}",
      "port": {{.Port}},
      "protocol": "shadowsocks",
      "settings": {
        "method": "{{.Method}}",
        "password": "{{.Password}}",
        "level": 0
      },
      "sockopt": {
        "tcpKeepAliveIdle": 300,
        "tcpUserTimeout": 10000
      }
    }
  ]
}`

// Shadowsocks-2022 入站模板
const Shadowsocks2022InboundTemplate = `{
  "inbounds": [
    {
      "tag": "{{.Tag}}",
      "port": {{.Port}},
      "protocol": "shadowsocks",
      "settings": {
        "method": "{{.Method}}",
        "password": "{{.Password}}",
        "level": 0
      },
      "sockopt": {
        "tcpKeepAliveIdle": 300,
        "tcpUserTimeout": 10000
      }
    }
  ]
}`

// 直连出站模板
const DirectOutboundTemplate = `{
  "outbounds": [
    {
      "tag": "direct",
      "protocol": "freedom",
      "settings": {
        "domainStrategy": "UseIPv4"
      }
    }
  ]
}`

// 阻断出站模板
const BlockOutboundTemplate = `{
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
}`

// 基础路由模板
const BasicRoutingTemplate = `{
  "routing": {
    "domainStrategy": "IPIfNonMatch",
    "rules": [
      {
        "type": "field",
        "ip": ["127.0.0.1/32", "10.0.0.0/8", "fc00::/7", "fe80::/10"],
        "outboundTag": "direct"
      }
    ]
  }
}`

// 尾部路由模板 - 添加到配置末尾
const TailRoutingTemplate = `{
  "routing": {
    "rules": [
      {
        "type": "field",
        "network": "tcp,udp",
        "outboundTag": "direct"
      }
    ]
  }
}`

// 模板数据结构
type TemplateData struct {
	Tag        string
	Port       int
	UUID       string
	Password   string
	Method     string
	Path       string
	Host       string
	Security   string
	CertFile   string
	KeyFile    string
	Dest       string
	ServerName string
	PrivateKey string
	ShortId    string
	// VLESS Encryption specific
	Decryption string
	Flow       string
}

// 模板渲染器
type TemplateRenderer struct{}

func NewTemplateRenderer() *TemplateRenderer {
	return &TemplateRenderer{}
}

func (t *TemplateRenderer) Render(templateStr string, data TemplateData) (string, error) {
	tmpl, err := template.New("config").Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// 获取模板映射
func GetTemplateMap() map[string]string {
	return map[string]string{
		"base":              BaseConfigTemplate,
		"dns":               DNSConfigTemplate,
		"vless-reality":     VLESSRealityInboundTemplate,
		"vless-encryption":  VLESSEncryptionInboundTemplate,
		"vless-ws":          VLESSWSInboundTemplate,
		"vmess-ws":          VMessWSInboundTemplate,
		"vless-httpupgrade": VLESSHTTPUpgradeInboundTemplate,
		"trojan-ws":         TrojanWSInboundTemplate,
		"shadowsocks":       ShadowsocksInboundTemplate,
		"shadowsocks-2022":  Shadowsocks2022InboundTemplate,
		"direct-outbound":   DirectOutboundTemplate,
		"block-outbound":    BlockOutboundTemplate,
		"basic-routing":     BasicRoutingTemplate,
		"tail-routing":      TailRoutingTemplate,
	}
}

// 根据协议类型获取模板
func GetProtocolTemplate(protocolType string) (string, bool) {
	templates := GetTemplateMap()
	tmpl, exists := templates[protocolType]
	return tmpl, exists
}
