package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// ShareURL 生成分享链接
type ShareURL struct {
	Protocol string
	Config   map[string]interface{}
}

// GenerateVLESSURL 生成 VLESS 分享链接
func GenerateVLESSURL(uuid, host string, port int, params map[string]string) (string, error) {
	if uuid == "" || host == "" || port == 0 {
		return "", fmt.Errorf("uuid, host, and port are required")
	}

	// 构建基础 URL
	u := &url.URL{
		Scheme: "vless",
		User:   url.User(uuid),
		Host:   fmt.Sprintf("%s:%d", host, port),
	}

	// 添加查询参数；将 remarks 作为 URL 片段（#remark），而非查询参数
	query := u.Query()
	if remark, ok := params["remarks"]; ok && remark != "" {
		// 作为 fragment，避免非标准的 remarks 查询参数
		u.Fragment = remark
		delete(params, "remarks")
	}
	for key, value := range params {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// GenerateVMessURL 生成 VMess 分享链接 (v2rayN 格式)
func GenerateVMessURL(config map[string]interface{}) (string, error) {
	vmessConfig := map[string]interface{}{
		"v":    "2",
		"ps":   getStringValue(config, "remark", "vmess"),
		"add":  getStringValue(config, "host", ""),
		"port": getIntValue(config, "port", 80),
		"id":   getStringValue(config, "uuid", ""),
		"aid":  getIntValue(config, "alterId", 0),
		"net":  getStringValue(config, "network", "tcp"),
		"type": getStringValue(config, "type", "none"),
		"host": getStringValue(config, "host", ""),
		"path": getStringValue(config, "path", ""),
		"tls":  getStringValue(config, "security", "none"),
	}

	// 可选：SNI 与 ALPN（部分客户端支持）
	if sni := getStringValue(config, "sni", getStringValue(config, "host", "")); sni != "" {
		vmessConfig["sni"] = sni
	}
	if alpn := getStringValue(config, "alpn", ""); alpn != "" {
		vmessConfig["alpn"] = alpn
	}

	jsonBytes, err := json.Marshal(vmessConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal vmess config: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(jsonBytes)
	return "vmess://" + encoded, nil
}

// GenerateTrojanURL 生成 Trojan 分享链接
func GenerateTrojanURL(password, host string, port int, params map[string]string) (string, error) {
	if password == "" || host == "" || port == 0 {
		return "", fmt.Errorf("password, host, and port are required")
	}

	// 构建基础 URL
	u := &url.URL{
		Scheme: "trojan",
		User:   url.User(password),
		Host:   fmt.Sprintf("%s:%d", host, port),
	}

	// 添加查询参数；将 remarks 作为 URL 片段（#remark）
	query := u.Query()
	if remark, ok := params["remarks"]; ok && remark != "" {
		u.Fragment = remark
		delete(params, "remarks")
	}
	for key, value := range params {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()

	return u.String(), nil
}

// GenerateShadowsocksURL 生成 Shadowsocks 分享链接 (SIP002 标准)
func GenerateShadowsocksURL(method, password, host string, port int, remark string) (string, error) {
	if method == "" || password == "" || host == "" || port == 0 {
		return "", fmt.Errorf("method, password, host, and port are required")
	}

	// 构建 user info: method:password
	userInfo := fmt.Sprintf("%s:%s", method, password)

	// SIP002 标准: 对于 Stream 和 AEAD 推荐使用 Base64URL 编码，对于 AEAD-2022 必须不编码
	var userInfoEncoded string
	if strings.HasPrefix(method, "2022-") {
		// AEAD-2022: 必须进行 percent 编码，不能使用 Base64
		userInfoEncoded = url.QueryEscape(userInfo)
	} else {
		// Stream 和 AEAD: 推荐使用 Base64URL 编码
		userInfoEncoded = base64.URLEncoding.EncodeToString([]byte(userInfo))
	}

	// 构建 URL
	baseURL := fmt.Sprintf("ss://%s@%s:%d", userInfoEncoded, host, port)

	if remark != "" {
		baseURL += "#" + url.QueryEscape(remark)
	}

	return baseURL, nil
}

// GenerateProtocolURL 根据协议类型生成分享链接
func GenerateProtocolURL(protocolType, tag string, config map[string]interface{}) (string, error) {
	protocolType = strings.ToLower(protocolType)

	// 提取通用信息
	host := getStringValue(config, "host", "")
	if host == "" {
		host = getStringValue(config, "domain", "")
	}
	port := getIntValue(config, "port", 0)

	switch {
	case strings.Contains(protocolType, "vless") || protocolType == "ve" || protocolType == "hu":
		uuid := getStringValue(config, "uuid", "")
		if uuid == "" {
			// 尝试从 clients 中获取
			if clients := getClientsFromConfig(config); len(clients) > 0 {
				uuid = getStringValue(clients[0], "id", "")
			}
		}

		params := make(map[string]string)
		params["type"] = getStringValue(config, "network", "tcp")
		// VLESS 协议按照官方要求，encryption 固定为 none
		params["encryption"] = "none"

		// 根据协议类型或安全类型添加特定参数
		if strings.Contains(protocolType, "encryption") || strings.EqualFold(protocolType, "ve") {
			// VLESS-Encryption (Post-Quantum) 协议处理
			params["security"] = "none" // 通常不需要额外的传输层安全
			params["type"] = "tcp"

			// VLESS-Encryption 的关键特征：使用真实的 encryption 而不是 "none"
			if encValue := getStringValue(config, "encryption", ""); encValue != "" && encValue != "none" {
				params["encryption"] = encValue
			} else {
				// 如果没有明确的 encryption，尝试从 decryption 推导
				// 先尝试从顶层读取，然后从 settings 中读取
				decValue := getStringValue(config, "decryption", "")
				if decValue == "" {
					// 尝试从 settings.decryption 读取
					if settings, exists := config["settings"].(map[string]interface{}); exists {
						decValue = getStringValue(settings, "decryption", "")
					}
				}

				if decValue != "" && decValue != "none" {
					// 对于VLESS-Encryption，直接使用服务端的decryption作为客户端的encryption
					// 这是因为在VLESS-Encryption中，服务端的decryption参数实际上包含了客户端需要的encryption信息
					// 根据官方文档，客户端和服务端共享相同的参数结构
					if strings.Contains(decValue, "mlkem768x25519plus") {
						// 将服务端的decryption转换为客户端的encryption格式
						// 替换RTT设置为客户端推荐的0rtt
						clientEnc := strings.Replace(decValue, ".600s.", ".0rtt.", 1)
						params["encryption"] = clientEnc
					} else {
						// 其他格式的加密，直接使用
						params["encryption"] = decValue
					}
				} else {
					params["encryption"] = "none"
				}
			}

			// Flow 参数（VLESS-Encryption 推荐使用 XTLS）
			flow := getStringValue(config, "flow", "")
			if flow == "" {
				flow = "xtls-rprx-vision"
			}
			params["flow"] = flow

		} else if strings.Contains(protocolType, "reality") || strings.ToLower(getStringValue(config, "security", "")) == "reality" {
			// VLESS REALITY (Vision) 推荐参数
			params["security"] = "reality"
			params["encryption"] = "none"
			// 优先使用配置中的 flow；若无，使用 xtls-rprx-vision
			flow := getStringValue(config, "flow", "")
			if flow == "" {
				flow = "xtls-rprx-vision"
			}
			params["flow"] = flow

			// REALITY 相关参数
			params["pbk"] = getStringValue(config, "publicKey", "")
			params["fp"] = getStringValue(config, "fingerprint", "chrome")
			// SNI: 优先使用 serverName，然后尝试 serverNames[0]，最后使用 host
			sniValue := getStringValue(config, "serverName", "")
			if sniValue == "" {
				if serverNames := getServerNamesFromConfig(config); len(serverNames) > 0 {
					sniValue = serverNames[0]
				} else {
					sniValue = host
				}
			}
			params["sni"] = sniValue
			params["sid"] = getStringValue(config, "shortId", "")
			// SpiderX (client param key: spx). Default to '/' for best compatibility
			if spx := getStringValue(config, "spiderX", "/"); spx != "" {
				params["spx"] = spx
			}
			// ALPN: many clients use h2 for REALITY; default to h2 when unset
			if alpn := getStringValue(config, "alpn", ""); alpn != "" {
				params["alpn"] = alpn
			} else {
				params["alpn"] = "h2"
			}
			// 明确 headerType=none 以提升兼容性
			if params["type"] == "tcp" {
				params["headerType"] = "none"
			}
		} else if strings.Contains(protocolType, "ws") {
			params["security"] = "tls"
			params["type"] = "ws"
			params["path"] = getStringValue(config, "path", "/")
			params["host"] = host
			// 为 TLS 客户端指定 SNI（多数客户端支持且为推荐做法）
			sniValue := getStringValue(config, "sni", host)
			params["sni"] = sniValue
			if alpn := getStringValue(config, "alpn", ""); alpn != "" {
				params["alpn"] = alpn
			}
		} else if strings.Contains(protocolType, "httpupgrade") || strings.EqualFold(protocolType, "hu") {
			// VLESS-HTTPUpgrade 协议处理
			params["type"] = "httpupgrade"
			params["security"] = "none" // HTTPUpgrade 通常不使用 TLS（由反向代理处理）
			params["path"] = getStringValue(config, "path", "/")

			// HTTPUpgrade 特定参数
			if host != "" {
				params["host"] = host // HTTP Host header
			}

			// 可选：自定义 headers (如果配置中有)
			if headers := getHeadersFromConfig(config); len(headers) > 0 {
				// 将 headers 编码为 URL 参数（简化处理）
				for key, value := range headers {
					params["header_"+key] = value
				}
			}
		}

		if remark := getStringValue(config, "remark", tag); remark != "" {
			params["remarks"] = remark
		}

		return GenerateVLESSURL(uuid, host, port, params)

	case strings.Contains(protocolType, "vmess"):
		return GenerateVMessURL(config)

	case strings.Contains(protocolType, "trojan"):
		password := getStringValue(config, "password", "")
		if password == "" {
			// 尝试从 clients 中获取
			if clients := getClientsFromConfig(config); len(clients) > 0 {
				password = getStringValue(clients[0], "password", "")
			}
		}

		params := make(map[string]string)
		params["security"] = "tls"
		params["type"] = getStringValue(config, "network", "tcp")
		// 为 TLS 客户端指定 SNI（若未明确提供，则使用主机名）
		if _, ok := params["sni"]; !ok {
			params["sni"] = host
		}
		if path := getStringValue(config, "path", ""); path != "" {
			params["path"] = path
			// WebSocket 模式下携带 Host 头
			if params["type"] == "ws" {
				params["host"] = host
				if alpn := getStringValue(config, "alpn", ""); alpn != "" {
					params["alpn"] = alpn
				}
			}
		}
		if remark := getStringValue(config, "remark", tag); remark != "" {
			params["remarks"] = remark
		}

		return GenerateTrojanURL(password, host, port, params)

	case strings.Contains(protocolType, "shadowsocks"):
		method := getStringValue(config, "method", "chacha20-poly1305")
		password := getStringValue(config, "password", "")
		remark := getStringValue(config, "remark", tag)

		return GenerateShadowsocksURL(method, password, host, port, remark)

	default:
		return "", fmt.Errorf("unsupported protocol: %s", protocolType)
	}
}

// 辅助函数

func getStringValue(config map[string]interface{}, key, defaultValue string) string {
	if value, exists := config[key]; exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

func getIntValue(config map[string]interface{}, key string, defaultValue int) int {
	if value, exists := config[key]; exists {
		switch v := value.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case string:
			var intVal int
			if _, err := fmt.Sscanf(v, "%d", &intVal); err == nil {
				return intVal
			}
		}
	}
	return defaultValue
}

// getHeadersFromConfig extracts HTTP headers from config
func getHeadersFromConfig(config map[string]interface{}) map[string]string {
	headers := make(map[string]string)
	if headersVal, exists := config["headers"]; exists {
		if headersMap, ok := headersVal.(map[string]interface{}); ok {
			for key, value := range headersMap {
				if strVal, ok := value.(string); ok {
					headers[key] = strVal
				}
			}
		}
	}
	return headers
}

func getServerNamesFromConfig(config map[string]interface{}) []string {
	if serverNames, exists := config["serverNames"]; exists {
		if serverNameList, ok := serverNames.([]interface{}); ok {
			var result []string
			for _, serverName := range serverNameList {
				if name, ok := serverName.(string); ok {
					result = append(result, name)
				}
			}
			return result
		}
	}
	return nil
}

func getClientsFromConfig(config map[string]interface{}) []map[string]interface{} {
	inbounds, exists := config["inbounds"]
	if !exists {
		return nil
	}

	inboundList, ok := inbounds.([]interface{})
	if !ok || len(inboundList) == 0 {
		return nil
	}

	inbound, ok := inboundList[0].(map[string]interface{})
	if !ok {
		return nil
	}

	settings, exists := inbound["settings"]
	if !exists {
		return nil
	}

	settingsMap, ok := settings.(map[string]interface{})
	if !ok {
		return nil
	}

	clients, exists := settingsMap["clients"]
	if !exists {
		return nil
	}

	clientList, ok := clients.([]interface{})
	if !ok {
		return nil
	}

	var result []map[string]interface{}
	for _, client := range clientList {
		if clientMap, ok := client.(map[string]interface{}); ok {
			result = append(result, clientMap)
		}
	}

	return result
}
