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
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

func FindAvailablePort(startPort int) int {
	for port := startPort; port <= 65535; port++ {
		if IsPortAvailable(port) {
			return port
		}
	}
	return 0
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