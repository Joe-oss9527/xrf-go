package utils

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	return nil
}

func IsPortAvailable(port int) bool {
	if port < 1 || port > 65535 {
		return false
	}

	// 检查 TCP 端口
	tcpAddr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", tcpAddr)
	if err != nil {
		return false
	}
	listener.Close()

	// 检查 UDP 端口
	udpAddr := fmt.Sprintf(":%d", port)
	conn, err := net.ListenPacket("udp", udpAddr)
	if err != nil {
		return false
	}
	conn.Close()

	return true
}

func FindAvailablePort(startPort, endPort int) (int, error) {
	if endPort == 0 {
		// 兼容旧版本，endPort为0时搜索到65535
		endPort = 65535
	}

	if startPort < 1 || startPort > 65535 || endPort < 1 || endPort > 65535 || startPort > endPort {
		return 0, fmt.Errorf("invalid port range: %d-%d", startPort, endPort)
	}

	for port := startPort; port <= endPort; port++ {
		if IsPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", startPort, endPort)
}

// CheckPortConflict 检查端口冲突
func CheckPortConflict(port int, usedPorts []int) bool {
	for _, usedPort := range usedPorts {
		if port == usedPort {
			return true
		}
	}
	return false
}

// SuggestPort 智能端口建议
func SuggestPort(protocolType string, preferredPort int) (int, error) {
	// 如果指定了端口且可用，直接使用
	if preferredPort != 0 {
		if IsPortAvailable(preferredPort) {
			return preferredPort, nil
		}
		return 0, fmt.Errorf("port %d is not available", preferredPort)
	}

	// 根据协议类型建议默认端口范围
	var startPort, endPort int

	switch protocolType {
	case "vless-reality", "vr", "VLESS-REALITY":
		// REALITY 推荐使用 443 或 80
		if IsPortAvailable(443) {
			return 443, nil
		}
		if IsPortAvailable(80) {
			return 80, nil
		}
		startPort, endPort = 40000, 50000
	case "vmess", "vless-ws", "trojan-ws", "mw", "vw", "tw", "VMess-WebSocket-TLS", "VLESS-WebSocket-TLS", "Trojan-WebSocket-TLS":
		// WebSocket 类推荐使用 80, 443 或高端口
		if IsPortAvailable(80) {
			return 80, nil
		}
		if IsPortAvailable(443) {
			return 443, nil
		}
		startPort, endPort = 30000, 40000
	case "shadowsocks", "ss", "ss2022", "Shadowsocks", "Shadowsocks-2022":
		// Shadowsocks 推荐使用高端口
		startPort, endPort = 50000, 60000
	case "vless-hu", "hu", "VLESS-HTTPUpgrade":
		// HTTPUpgrade 推荐使用 80 或高端口
		if IsPortAvailable(80) {
			return 80, nil
		}
		startPort, endPort = 30000, 40000
	default:
		// 默认高端口范围
		startPort, endPort = 10000, 20000
	}

	return FindAvailablePort(startPort, endPort)
}

// GetPortsByProtocol 根据协议获取推荐端口列表
func GetPortsByProtocol(protocolType string) []int {
	switch protocolType {
	case "vless-reality", "vr", "VLESS-REALITY":
		return []int{443, 80, 8443, 2053, 2083, 2087, 2096}
	case "vmess", "vless-ws", "trojan-ws", "mw", "vw", "tw", "vless-hu", "hu", "VMess-WebSocket-TLS", "VLESS-WebSocket-TLS", "Trojan-WebSocket-TLS", "VLESS-HTTPUpgrade":
		return []int{80, 443, 8080, 8443, 2052, 2053, 2082, 2083, 2086, 2087, 2095, 2096}
	case "shadowsocks", "ss", "ss2022", "Shadowsocks", "Shadowsocks-2022":
		return []int{1080, 8388, 9000, 50000, 51000, 52000}
	default:
		return []int{8080, 8443, 9000, 10000, 20000}
	}
}

func ValidateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)*[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)
	if !domainRegex.MatchString(domain) {
		return fmt.Errorf("invalid domain format: %s", domain)
	}

	return nil
}

func ValidateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP address: %s", ip)
	}
	return nil
}

func ValidateURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return err
	}

	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid URL: missing scheme or host")
	}

	return nil
}

func ValidatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("path must start with /")
	}

	pathRegex := regexp.MustCompile(`^/[a-zA-Z0-9\-._~!$&'()*+,;=:@/%]*$`)
	if !pathRegex.MatchString(path) {
		return fmt.Errorf("invalid path format: %s", path)
	}

	return nil
}

func ValidateEmail(email string) error {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

func ValidateJSON(jsonStr string) error {
	var js interface{}
	return json.Unmarshal([]byte(jsonStr), &js)
}

func ValidateXrayConfig(configPath string) error {
	if configPath == "" {
		return fmt.Errorf("config path cannot be empty")
	}

	xrayPath, err := exec.LookPath("xray")
	if err != nil {
		return fmt.Errorf("xray not found in PATH")
	}

	cmd := exec.Command(xrayPath, "test", "-config", configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("config validation failed: %s", string(output))
	}

	return nil
}

func ValidateXrayConfDir(confDir string) error {
	if confDir == "" {
		return fmt.Errorf("confdir path cannot be empty")
	}

	if _, err := os.Stat(confDir); os.IsNotExist(err) {
		return fmt.Errorf("confdir does not exist: %s", confDir)
	}

	xrayPath, err := exec.LookPath("xray")
	if err != nil {
		return fmt.Errorf("xray not found in PATH")
	}

	cmd := exec.Command(xrayPath, "run", "-confdir", confDir, "-test")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("confdir validation failed: %s", string(output))
	}

	return nil
}

func ValidateProtocolType(protocol string) error {
	validProtocols := []string{
		"vless", "vmess", "trojan", "shadowsocks", "dokodemo-door",
		"http", "socks", "freedom", "blackhole",
	}

	protocol = strings.ToLower(protocol)
	for _, valid := range validProtocols {
		if protocol == valid {
			return nil
		}
	}

	return fmt.Errorf("unsupported protocol: %s", protocol)
}

func ValidateTransportType(transport string) error {
	validTransports := []string{
		"tcp", "kcp", "ws", "websocket", "http", "h2", "quic", "grpc",
		"httpupgrade", "splithttp",
	}

	transport = strings.ToLower(transport)
	for _, valid := range validTransports {
		if transport == valid {
			return nil
		}
	}

	return fmt.Errorf("unsupported transport: %s", transport)
}

func ValidateSecurityType(security string) error {
	validSecurity := []string{
		"none", "tls", "reality", "xtls",
	}

	security = strings.ToLower(security)
	for _, valid := range validSecurity {
		if security == valid {
			return nil
		}
	}

	return fmt.Errorf("unsupported security: %s", security)
}

func ParsePortString(portStr string) (int, error) {
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, fmt.Errorf("invalid port format: %s", portStr)
	}

	if err := ValidatePort(port); err != nil {
		return 0, err
	}

	return port, nil
}

func ValidateFileExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", path)
	}
	return nil
}

func ValidateDirectoryExists(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", path)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	return nil
}

// IsValidUUID 验证UUID格式是否正确
func IsValidUUID(uuid string) bool {
	if uuid == "" {
		return false
	}

	// UUID v4 格式: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	// 其中 y 必须是 8, 9, a, 或 b
	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[4][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	return uuidRegex.MatchString(uuid)
}
