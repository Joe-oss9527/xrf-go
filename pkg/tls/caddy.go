package tls

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

    "github.com/Joe-oss9527/xrf-go/pkg/utils"
)

const (
	// Caddy 默认配置
	DefaultCaddyAdminAPI = "localhost:2019"
	DefaultCaddyBinary   = "/usr/local/bin/caddy"
	CaddyDownloadURL     = "https://github.com/caddyserver/caddy/releases/latest/download/caddy_linux_amd64.tar.gz"
	CaddyServiceName     = "caddy"
)

// CaddyManager Caddy 管理器
type CaddyManager struct {
	adminAPI    string
	configDir   string
	binaryPath  string
	httpClient  *http.Client
	serviceName string
}

// CaddyConfig Caddy JSON 配置结构
type CaddyConfig struct {
	Apps CaddyApps `json:"apps"`
}

type CaddyApps struct {
	HTTP *CaddyHTTP `json:"http,omitempty"`
	TLS  *CaddyTLS  `json:"tls,omitempty"`
}

type CaddyHTTP struct {
	Servers map[string]*CaddyServer `json:"servers"`
}

type CaddyTLS struct {
	Automation *CaddyTLSAutomation `json:"automation,omitempty"`
}

type CaddyTLSAutomation struct {
	Policies []CaddyTLSPolicy `json:"policies,omitempty"`
}

type CaddyTLSPolicy struct {
	Subjects []string      `json:"subjects,omitempty"`
	Issuers  []interface{} `json:"issuers,omitempty"`
}

type CaddyServer struct {
	Listen []string     `json:"listen"`
	Routes []CaddyRoute `json:"routes"`
}

type CaddyRoute struct {
	Match  []CaddyMatch  `json:"match,omitempty"`
	Handle []CaddyHandle `json:"handle"`
}

type CaddyMatch struct {
	Host []string `json:"host,omitempty"`
	Path []string `json:"path,omitempty"`
}

type CaddyHandle struct {
	Handler   string          `json:"handler"`
	Upstreams []CaddyUpstream `json:"upstreams,omitempty"`
	// 反向代理配置
	// 静态文件服务配置
	Root string `json:"root,omitempty"`
	// 模板配置
	Template *CaddyTemplate `json:"template,omitempty"`
	// 重写配置
	URI string `json:"uri,omitempty"`
}

type CaddyUpstream struct {
	Dial string `json:"dial"`
}

type CaddyTemplate struct {
	// 模板配置
}

// NewCaddyManager 创建新的 Caddy 管理器
func NewCaddyManager() *CaddyManager {
	return &CaddyManager{
		adminAPI:    DefaultCaddyAdminAPI,
		configDir:   "/etc/caddy",
		binaryPath:  DefaultCaddyBinary,
		serviceName: CaddyServiceName,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetAdminAPI 设置管理API地址
func (cm *CaddyManager) SetAdminAPI(adminAPI string) {
	cm.adminAPI = adminAPI
}

// SetConfigDir 设置配置目录
func (cm *CaddyManager) SetConfigDir(configDir string) {
	cm.configDir = configDir
}

// SetBinaryPath 设置二进制文件路径
func (cm *CaddyManager) SetBinaryPath(binaryPath string) {
	cm.binaryPath = binaryPath
}

// InstallCaddy 安装 Caddy
func (cm *CaddyManager) InstallCaddy() error {
	// 检查 Caddy 是否已安装
	if cm.isCaddyInstalled() {
		utils.Info("Caddy is already installed at: %s", cm.binaryPath)
		return nil
	}

	utils.Info("Installing Caddy...")

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "caddy-install")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 下载 Caddy
	tarPath := filepath.Join(tmpDir, "caddy.tar.gz")
	if err := cm.downloadFile(CaddyDownloadURL, tarPath); err != nil {
		return fmt.Errorf("failed to download Caddy: %w", err)
	}

	// 解压
	extractDir := filepath.Join(tmpDir, "extract")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	cmd := exec.Command("tar", "-xzf", tarPath, "-C", extractDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract Caddy: %w", err)
	}

	// 复制到目标位置
	srcPath := filepath.Join(extractDir, "caddy")
	if err := cm.copyFile(srcPath, cm.binaryPath); err != nil {
		return fmt.Errorf("failed to copy Caddy binary: %w", err)
	}

	// 设置可执行权限
	if err := os.Chmod(cm.binaryPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permission: %w", err)
	}

	// 创建配置目录
	if err := os.MkdirAll(cm.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// 创建 systemd 服务
	if err := cm.createSystemdService(); err != nil {
		return fmt.Errorf("failed to create systemd service: %w", err)
	}

	utils.Success("Caddy installed successfully")
	return nil
}

// isCaddyInstalled 检查 Caddy 是否已安装
func (cm *CaddyManager) isCaddyInstalled() bool {
	if _, err := os.Stat(cm.binaryPath); err != nil {
		return false
	}

	// 验证可以运行
	cmd := exec.Command(cm.binaryPath, "version")
	return cmd.Run() == nil
}

// downloadFile 下载文件
func (cm *CaddyManager) downloadFile(url, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// copyFile 复制文件
func (cm *CaddyManager) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// createSystemdService 创建 systemd 服务
func (cm *CaddyManager) createSystemdService() error {
	serviceContent := fmt.Sprintf(`[Unit]
Description=Caddy HTTP/2 web server
Documentation=https://caddyserver.com/docs/
After=network.target network-online.target
Requires=network-online.target

[Service]
Type=notify
User=caddy
Group=caddy
ExecStart=%s run --environ
ExecReload=/bin/kill -USR1 $MAINPID
TimeoutStopSec=5s
LimitNOFILE=1048576
LimitNPROC=1048576
PrivateTmp=true
ProtectSystem=full
AmbientCapabilities=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
`, cm.binaryPath)

	servicePath := fmt.Sprintf("/etc/systemd/system/%s.service", cm.serviceName)
	if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return err
	}

	// 创建 caddy 用户
	cmd := exec.Command("useradd", "--system", "--shell", "/bin/false", "--home", "/var/lib/caddy", "caddy")
	cmd.Run() // 忽略错误，用户可能已存在

	// 创建 caddy 用户目录
	os.MkdirAll("/var/lib/caddy", 0755)
	exec.Command("chown", "caddy:caddy", "/var/lib/caddy").Run()

	// 重新加载 systemd
	exec.Command("systemctl", "daemon-reload").Run()

	return nil
}

// ConfigureReverseProxy 配置反向代理
func (cm *CaddyManager) ConfigureReverseProxy(domain, upstream string) error {
	utils.Info("Configuring reverse proxy for %s -> %s", domain, upstream)

	config := CaddyConfig{
		Apps: CaddyApps{
			HTTP: &CaddyHTTP{
				Servers: map[string]*CaddyServer{
					"srv0": {
						Listen: []string{":443"},
						Routes: []CaddyRoute{
							{
								Match: []CaddyMatch{
									{
										Host: []string{domain},
									},
								},
								Handle: []CaddyHandle{
									{
										Handler: "reverse_proxy",
										Upstreams: []CaddyUpstream{
											{
												Dial: upstream,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			TLS: &CaddyTLS{
				Automation: &CaddyTLSAutomation{
					Policies: []CaddyTLSPolicy{
						{
							Subjects: []string{domain},
						},
					},
				},
			},
		},
	}

	return cm.loadConfig(config)
}

// AddWebsiteMasquerade 添加伪装网站
func (cm *CaddyManager) AddWebsiteMasquerade(domain, maskSite string) error {
	utils.Info("Adding website masquerade for %s -> %s", domain, maskSite)

	config := CaddyConfig{
		Apps: CaddyApps{
			HTTP: &CaddyHTTP{
				Servers: map[string]*CaddyServer{
					"srv0": {
						Listen: []string{":443"},
						Routes: []CaddyRoute{
							{
								Match: []CaddyMatch{
									{
										Host: []string{domain},
									},
								},
								Handle: []CaddyHandle{
									{
										Handler: "reverse_proxy",
										Upstreams: []CaddyUpstream{
											{
												Dial: maskSite,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return cm.loadConfig(config)
}

// EnableAutoHTTPS 启用自动 HTTPS
func (cm *CaddyManager) EnableAutoHTTPS(domain string) error {
	utils.Info("Enabling auto HTTPS for domain: %s", domain)

	config := CaddyConfig{
		Apps: CaddyApps{
			TLS: &CaddyTLS{
				Automation: &CaddyTLSAutomation{
					Policies: []CaddyTLSPolicy{
						{
							Subjects: []string{domain},
						},
					},
				},
			},
		},
	}

	return cm.loadConfig(config)
}

// loadConfig 通过 API 加载配置
func (cm *CaddyManager) loadConfig(config CaddyConfig) error {
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	url := fmt.Sprintf("http://%s/load", cm.adminAPI)
	resp, err := cm.httpClient.Post(url, "application/json", bytes.NewBuffer(configJSON))
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	utils.Success("Caddy configuration loaded successfully")
	return nil
}

// ReloadConfig 重新加载配置
func (cm *CaddyManager) ReloadConfig() error {
	utils.Info("Reloading Caddy configuration...")

	// 通过 systemd 重启服务
	cmd := exec.Command("systemctl", "reload", cm.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload Caddy: %w", err)
	}

	utils.Success("Caddy configuration reloaded")
	return nil
}

// StartService 启动 Caddy 服务
func (cm *CaddyManager) StartService() error {
	utils.Info("Starting Caddy service...")

	cmd := exec.Command("systemctl", "start", cm.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Caddy service: %w", err)
	}

	// 启用自动启动
	cmd = exec.Command("systemctl", "enable", cm.serviceName)
	if err := cmd.Run(); err != nil {
		utils.Warning("Failed to enable Caddy service: %v", err)
	}

	utils.Success("Caddy service started")
	return nil
}

// StopService 停止 Caddy 服务
func (cm *CaddyManager) StopService() error {
	utils.Info("Stopping Caddy service...")

	cmd := exec.Command("systemctl", "stop", cm.serviceName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop Caddy service: %w", err)
	}

	utils.Success("Caddy service stopped")
	return nil
}

// GetServiceStatus 获取服务状态
func (cm *CaddyManager) GetServiceStatus() (string, error) {
	cmd := exec.Command("systemctl", "is-active", cm.serviceName)
	output, err := cmd.Output()
	if err != nil {
		return "inactive", nil
	}

	status := strings.TrimSpace(string(output))
	return status, nil
}

// GetConfig 获取当前配置
func (cm *CaddyManager) GetConfig() (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s/config/", cm.adminAPI)
	resp, err := cm.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	return config, nil
}

// IsRunning 检查 Caddy 是否正在运行
func (cm *CaddyManager) IsRunning() bool {
	status, err := cm.GetServiceStatus()
	if err != nil {
		return false
	}
	return status == "active"
}

// TestConfig 测试配置是否有效
func (cm *CaddyManager) TestConfig(configPath string) error {
	cmd := exec.Command(cm.binaryPath, "validate", "--config", configPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("config validation failed: %s", string(output))
	}
	return nil
}
