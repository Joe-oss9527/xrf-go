package system

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// SystemInfo 系统信息
type SystemInfo struct {
	OS             string `json:"os"`              // 操作系统 (linux, darwin, windows)
	Distribution   string `json:"distribution"`    // 发行版 (ubuntu, centos, debian, etc.)
	Version        string `json:"version"`         // 版本号
	Architecture   string `json:"architecture"`    // 架构 (amd64, arm64, etc.)
	Kernel         string `json:"kernel"`          // 内核版本
	HasSystemd     bool   `json:"has_systemd"`     // 是否支持 systemd
	HasFirewall    bool   `json:"has_firewall"`    // 是否有防火墙
	FirewallType   string `json:"firewall_type"`   // 防火墙类型
	PackageManager string `json:"package_manager"` // 包管理器
}

// Detector 系统检测器
type Detector struct {
	info *SystemInfo
}

// NewDetector 创建系统检测器
func NewDetector() *Detector {
	return &Detector{}
}

// DetectSystem 检测系统信息
func (d *Detector) DetectSystem() (*SystemInfo, error) {
	if d.info != nil {
		return d.info, nil
	}

	info := &SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}

	// 检测Linux发行版信息
	if info.OS == "linux" {
		if err := d.detectLinuxDistribution(info); err != nil {
			return nil, fmt.Errorf("failed to detect Linux distribution: %w", err)
		}
	}

	// 检测内核版本
	if err := d.detectKernelVersion(info); err != nil {
		// 内核版本检测失败不应该终止整个检测过程
		info.Kernel = "unknown"
	}

	// 检测systemd支持
	info.HasSystemd = d.detectSystemd()

	// 检测防火墙
	info.HasFirewall, info.FirewallType = d.detectFirewall()

	// 检测包管理器
	info.PackageManager = d.detectPackageManager()

	d.info = info
	return info, nil
}

// detectLinuxDistribution 检测Linux发行版
func (d *Detector) detectLinuxDistribution(info *SystemInfo) error {
	// 尝试读取 /etc/os-release
	if content, err := os.ReadFile("/etc/os-release"); err == nil {
		return d.parseOSRelease(string(content), info)
	}

	// 尝试读取 /etc/lsb-release
	if content, err := os.ReadFile("/etc/lsb-release"); err == nil {
		return d.parseLSBRelease(string(content), info)
	}

	// 尝试检测具体的发行版文件
	distroFiles := map[string]string{
		"/etc/redhat-release": "redhat",
		"/etc/centos-release": "centos",
		"/etc/fedora-release": "fedora",
		"/etc/debian_version": "debian",
		"/etc/alpine-release": "alpine",
	}

	for file, distro := range distroFiles {
		if content, err := os.ReadFile(file); err == nil {
			info.Distribution = distro
			info.Version = strings.TrimSpace(string(content))
			return nil
		}
	}

	// 如果都检测不到，设置为未知
	info.Distribution = "unknown"
	info.Version = "unknown"
	return nil
}

// parseOSRelease 解析 /etc/os-release 文件
func (d *Detector) parseOSRelease(content string, info *SystemInfo) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)

		switch key {
		case "ID":
			info.Distribution = strings.ToLower(value)
		case "VERSION_ID":
			info.Version = value
		case "PRETTY_NAME":
			if info.Distribution == "" {
				// 如果没有ID字段，尝试从PRETTY_NAME推断
				info.Distribution = d.inferDistroFromPrettyName(value)
			}
		}
	}

	if info.Distribution == "" {
		info.Distribution = "unknown"
	}
	if info.Version == "" {
		info.Version = "unknown"
	}

	return scanner.Err()
}

// parseLSBRelease 解析 /etc/lsb-release 文件
func (d *Detector) parseLSBRelease(content string, info *SystemInfo) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)

		switch key {
		case "DISTRIB_ID":
			info.Distribution = strings.ToLower(value)
		case "DISTRIB_RELEASE":
			info.Version = value
		}
	}

	return scanner.Err()
}

// inferDistroFromPrettyName 从PRETTY_NAME推断发行版
func (d *Detector) inferDistroFromPrettyName(prettyName string) string {
	prettyName = strings.ToLower(prettyName)

	distroKeywords := map[string]string{
		"ubuntu":    "ubuntu",
		"debian":    "debian",
		"centos":    "centos",
		"rhel":      "rhel",
		"fedora":    "fedora",
		"opensuse":  "opensuse",
		"archlinux": "arch",
		"alpine":    "alpine",
	}

	for keyword, distro := range distroKeywords {
		if strings.Contains(prettyName, keyword) {
			return distro
		}
	}

	return "unknown"
}

// detectKernelVersion 检测内核版本
func (d *Detector) detectKernelVersion(info *SystemInfo) error {
	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	info.Kernel = strings.TrimSpace(string(output))
	return nil
}

// detectSystemd 检测systemd支持
func (d *Detector) detectSystemd() bool {
	// 检查 systemctl 命令是否存在
	if _, err := exec.LookPath("systemctl"); err != nil {
		return false
	}

	// 检查 /run/systemd/system 目录是否存在
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		return false
	}

	return true
}

// detectFirewall 检测防火墙
func (d *Detector) detectFirewall() (bool, string) {
	firewallCommands := map[string]string{
		"ufw":          "ufw",
		"firewall-cmd": "firewalld",
		"iptables":     "iptables",
		"nft":          "nftables",
	}

	for cmd, fwType := range firewallCommands {
		if _, err := exec.LookPath(cmd); err == nil {
			// 检查服务是否运行
			if d.isFirewallActive(cmd, fwType) {
				return true, fwType
			}
		}
	}

	return false, "none"
}

// isFirewallActive 检查防火墙是否激活
func (d *Detector) isFirewallActive(cmd, fwType string) bool {
	switch fwType {
	case "ufw":
		if output, err := exec.Command("ufw", "status").Output(); err == nil {
			return strings.Contains(string(output), "Status: active")
		}
	case "firewalld":
		if err := exec.Command("firewall-cmd", "--state").Run(); err == nil {
			return true
		}
	case "iptables":
		if output, err := exec.Command("iptables", "-L").Output(); err == nil {
			// 如果有除了默认策略外的规则，认为是激活的
			lines := strings.Split(string(output), "\n")
			return len(lines) > 10 // 简单启发式判断
		}
	case "nftables":
		if output, err := exec.Command("nft", "list", "tables").Output(); err == nil {
			return strings.TrimSpace(string(output)) != ""
		}
	}
	return false
}

// detectPackageManager 检测包管理器
func (d *Detector) detectPackageManager() string {
	packageManagers := []string{
		"apt",    // Debian/Ubuntu
		"yum",    // RHEL/CentOS 7
		"dnf",    // Fedora/RHEL 8+
		"pacman", // Arch Linux
		"zypper", // openSUSE
		"apk",    // Alpine Linux
		"brew",   // macOS
	}

	for _, pm := range packageManagers {
		if _, err := exec.LookPath(pm); err == nil {
			return pm
		}
	}

	return "unknown"
}

// IsSupported 检查是否支持当前系统
func (d *Detector) IsSupported() (bool, string) {
	info, err := d.DetectSystem()
	if err != nil {
		return false, fmt.Sprintf("系统检测失败: %v", err)
	}

	// 目前主要支持 Linux 系统
	if info.OS != "linux" {
		return false, fmt.Sprintf("不支持的操作系统: %s", info.OS)
	}

	// 检查架构支持
	supportedArch := []string{"amd64", "arm64"}
	for _, arch := range supportedArch {
		if info.Architecture == arch {
			return true, ""
		}
	}

	return false, fmt.Sprintf("不支持的系统架构: %s", info.Architecture)
}

// GetXrayBinaryName 获取适合当前系统的 Xray 二进制文件名
func (d *Detector) GetXrayBinaryName() (string, error) {
	info, err := d.DetectSystem()
	if err != nil {
		return "", err
	}

	// Xray 官方发布的文件名格式
	osName := info.OS
	archName := info.Architecture

	// 架构名称映射
	archMap := map[string]string{
		"amd64": "64",
		"arm64": "arm64-v8a",
	}

	if mappedArch, exists := archMap[archName]; exists {
		archName = mappedArch
	}

	// OS名称映射
	osMap := map[string]string{
		"linux":   "linux",
		"darwin":  "macos",
		"windows": "windows",
	}

	if mappedOS, exists := osMap[osName]; exists {
		osName = mappedOS
	}

	return fmt.Sprintf("Xray-%s-%s", osName, archName), nil
}

// CheckDependencies 检查系统依赖
func (d *Detector) CheckDependencies() []string {
	var missing []string

	requiredCommands := map[string]string{
		"curl": "用于下载 Xray 二进制文件",
		"tar":  "用于解压文件",
	}

	for cmd, desc := range requiredCommands {
		if _, err := exec.LookPath(cmd); err != nil {
			missing = append(missing, fmt.Sprintf("%s (%s)", cmd, desc))
		}
	}

	return missing
}

// GetInstallCommand 获取安装依赖的命令
func (d *Detector) GetInstallCommand(packages []string) (string, error) {
	info, err := d.DetectSystem()
	if err != nil {
		return "", err
	}

	switch info.PackageManager {
	case "apt":
		return fmt.Sprintf("apt update && apt install -y %s", strings.Join(packages, " ")), nil
	case "yum":
		return fmt.Sprintf("yum install -y %s", strings.Join(packages, " ")), nil
	case "dnf":
		return fmt.Sprintf("dnf install -y %s", strings.Join(packages, " ")), nil
	case "pacman":
		return fmt.Sprintf("pacman -S --noconfirm %s", strings.Join(packages, " ")), nil
	case "zypper":
		return fmt.Sprintf("zypper install -y %s", strings.Join(packages, " ")), nil
	case "apk":
		return fmt.Sprintf("apk add %s", strings.Join(packages, " ")), nil
	default:
		return "", fmt.Errorf("不支持的包管理器: %s", info.PackageManager)
	}
}

// IsRoot 检查是否以root权限运行
func (d *Detector) IsRoot() bool {
	return os.Geteuid() == 0
}

// GetSystemInfo 获取系统信息 (已检测或新检测)
func (d *Detector) GetSystemInfo() *SystemInfo {
	if d.info == nil {
		d.DetectSystem() // 忽略错误，返回已有信息
	}
	return d.info
}

// PrintSystemInfo 打印系统信息
func (d *Detector) PrintSystemInfo() error {
	info, err := d.DetectSystem()
	if err != nil {
		return err
	}

	fmt.Printf("系统信息:\n")
	fmt.Printf("  操作系统: %s\n", info.OS)
	fmt.Printf("  发行版: %s %s\n", info.Distribution, info.Version)
	fmt.Printf("  架构: %s\n", info.Architecture)
	fmt.Printf("  内核: %s\n", info.Kernel)
	fmt.Printf("  Systemd 支持: %s\n", boolToYesNo(info.HasSystemd))
	fmt.Printf("  防火墙: %s", boolToYesNo(info.HasFirewall))
	if info.HasFirewall {
		fmt.Printf(" (%s)", info.FirewallType)
	}
	fmt.Printf("\n")
	fmt.Printf("  包管理器: %s\n", info.PackageManager)

	return nil
}

func boolToYesNo(b bool) string {
	if b {
		return "是"
	}
	return "否"
}
