package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

    "github.com/Joe-oss9527/xrf-go/pkg/utils"
)

const (
	ServiceName        = "xray"
	SystemdServicePath = "/etc/systemd/system/xray.service"
	ServiceUser        = "xray" // 专用服务用户
	ServiceGroup       = "xray" // 专用服务组
	ServiceHome        = "/var/lib/xray"
	ServiceShell       = "/usr/sbin/nologin"
)

// ServiceManager 服务管理器
type ServiceManager struct {
	detector *Detector
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	Enabled   bool   `json:"enabled"`
	Running   bool   `json:"running"`
	Status    string `json:"status"`
	StartTime string `json:"start_time,omitempty"`
	Memory    string `json:"memory,omitempty"`
	CPU       string `json:"cpu,omitempty"`
}

// NewServiceManager 创建服务管理器
func NewServiceManager(detector *Detector) *ServiceManager {
	return &ServiceManager{
		detector: detector,
	}
}

// InstallService 安装 systemd 服务
func (sm *ServiceManager) InstallService() error {
	if !sm.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能安装服务")
	}

	sysInfo := sm.detector.GetSystemInfo()
	if sysInfo != nil && !sysInfo.HasSystemd {
		return fmt.Errorf("系统不支持 systemd")
	}

	utils.PrintInfo("安装 Xray 服务...")

	// 确保服务用户和组存在
	if err := sm.ensureServiceUserAndGroup(); err != nil {
		utils.PrintWarning("创建专用用户失败，将使用降级方案: %v", err)
	}

	// 生成服务文件内容（使用动态用户选择）
	serviceContent := sm.generateServiceFile()

	// 写入服务文件
	if err := os.WriteFile(SystemdServicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("写入服务文件失败: %w", err)
	}

	// 设置配置目录权限
	if err := sm.setDirectoryPermissions(); err != nil {
		utils.PrintWarning("设置目录权限失败: %v", err)
	}

	// 重新加载 systemd
	if err := sm.reloadSystemd(); err != nil {
		return fmt.Errorf("重新加载 systemd 失败: %w", err)
	}

	// 启用服务
	if err := sm.EnableService(); err != nil {
		return fmt.Errorf("启用服务失败: %w", err)
	}

	utils.PrintSuccess("Xray 服务安装完成")
	return nil
}

// generateServiceFile 生成 systemd 服务文件内容
func (sm *ServiceManager) generateServiceFile() string {
	user, group := sm.getSystemUserGroup()

	return fmt.Sprintf(`[Unit]
Description=Xray Service
Documentation=https://github.com/XTLS/Xray-core
After=network.target nss-lookup.target

[Service]
Type=simple
User=%s
Group=%s
ExecStart=%s run -confdir %s
Restart=on-failure
RestartPreventExitStatus=1
RestartSec=5
LimitNOFILE=1000000

# Security settings
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=%s /var/lib/xray
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
CapabilityBoundingSet=CAP_NET_BIND_SERVICE
AmbientCapabilities=CAP_NET_BIND_SERVICE

# Network settings
PrivateDevices=true
RestrictSUIDSGID=true
RestrictRealtime=true
RestrictNamespaces=true

[Install]
WantedBy=multi-user.target
`, user, group, XrayBinaryPath, XrayConfsDir, filepath.Dir(XrayConfsDir))
}

// StartService 启动服务
func (sm *ServiceManager) StartService() error {
	if !sm.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能启动服务")
	}

	utils.PrintInfo("启动 Xray 服务...")

	cmd := exec.Command("systemctl", "start", ServiceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("启动服务失败: %w\n输出: %s", err, string(output))
	}

	// 等待服务启动
	time.Sleep(2 * time.Second)

	// 检查服务状态
	if status, err := sm.GetServiceStatus(); err != nil {
		return fmt.Errorf("检查服务状态失败: %w", err)
	} else if !status.Active {
		return fmt.Errorf("服务启动失败，状态: %s", status.Status)
	}

	utils.PrintSuccess("Xray 服务已启动")
	return nil
}

// StopService 停止服务
func (sm *ServiceManager) StopService() error {
	if !sm.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能停止服务")
	}

	utils.PrintInfo("停止 Xray 服务...")

	cmd := exec.Command("systemctl", "stop", ServiceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("停止服务失败: %w\n输出: %s", err, string(output))
	}

	utils.PrintSuccess("Xray 服务已停止")
	return nil
}

// RestartService 重启服务
func (sm *ServiceManager) RestartService() error {
	if !sm.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能重启服务")
	}

	utils.PrintInfo("重启 Xray 服务...")

	cmd := exec.Command("systemctl", "restart", ServiceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("重启服务失败: %w\n输出: %s", err, string(output))
	}

	// 等待服务重启
	time.Sleep(3 * time.Second)

	// 检查服务状态
	if status, err := sm.GetServiceStatus(); err != nil {
		return fmt.Errorf("检查服务状态失败: %w", err)
	} else if !status.Active {
		return fmt.Errorf("服务重启失败，状态: %s", status.Status)
	}

	utils.PrintSuccess("Xray 服务已重启")
	return nil
}

// ReloadService 重载服务 (发送 USR1 信号)
func (sm *ServiceManager) ReloadService() error {
	if !sm.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能重载服务")
	}

	utils.PrintInfo("重载 Xray 配置...")

	// 发送 USR1 信号给 Xray 进程进行热重载
	cmd := exec.Command("systemctl", "reload", ServiceName)
	if err := cmd.Run(); err != nil {
		// 如果 systemctl reload 不支持，尝试直接发送信号
		return sm.sendSignalToXray("USR1")
	}

	utils.PrintSuccess("Xray 配置已重载")
	return nil
}

// sendSignalToXray 向 Xray 进程发送信号
func (sm *ServiceManager) sendSignalToXray(signal string) error {
	// 获取 Xray 进程 PID
	cmd := exec.Command("pgrep", "-f", "xray.*confdir")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("未找到 Xray 进程")
	}

	pid := strings.TrimSpace(string(output))
	if pid == "" {
		return fmt.Errorf("未找到 Xray 进程")
	}

	// 发送信号
	killCmd := exec.Command("kill", "-"+signal, pid)
	if err := killCmd.Run(); err != nil {
		return fmt.Errorf("发送信号失败: %w", err)
	}

	return nil
}

// EnableService 启用服务开机自启
func (sm *ServiceManager) EnableService() error {
	if !sm.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能启用服务")
	}

	cmd := exec.Command("systemctl", "enable", ServiceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("启用服务失败: %w\n输出: %s", err, string(output))
	}

	utils.PrintSuccess("Xray 服务已设置为开机自启")
	return nil
}

// DisableService 禁用服务开机自启
func (sm *ServiceManager) DisableService() error {
	if !sm.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能禁用服务")
	}

	cmd := exec.Command("systemctl", "disable", ServiceName)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("禁用服务失败: %w\n输出: %s", err, string(output))
	}

	utils.PrintSuccess("Xray 服务开机自启已禁用")
	return nil
}

// GetServiceStatus 获取服务状态
func (sm *ServiceManager) GetServiceStatus() (*ServiceStatus, error) {
	status := &ServiceStatus{
		Name: ServiceName,
	}

	// 获取基本状态信息
	if err := sm.getBasicStatus(status); err != nil {
		return status, err
	}

	// 获取详细信息
	sm.getDetailedStatus(status)

	return status, nil
}

// getBasicStatus 获取基本状态
func (sm *ServiceManager) getBasicStatus(status *ServiceStatus) error {
	// 检查服务是否激活
	cmd := exec.Command("systemctl", "is-active", ServiceName)
	output, _ := cmd.Output()
	activeStatus := strings.TrimSpace(string(output))
	status.Active = activeStatus == "active"
	status.Running = status.Active

	// 检查服务是否启用
	cmd = exec.Command("systemctl", "is-enabled", ServiceName)
	output, _ = cmd.Output()
	enabledStatus := strings.TrimSpace(string(output))
	status.Enabled = enabledStatus == "enabled"

	// 获取状态描述
	cmd = exec.Command("systemctl", "show", ServiceName, "--property=SubState", "--value")
	output, _ = cmd.Output()
	status.Status = strings.TrimSpace(string(output))
	if status.Status == "" {
		if status.Active {
			status.Status = "running"
		} else {
			status.Status = "stopped"
		}
	}

	return nil
}

// getDetailedStatus 获取详细状态信息
func (sm *ServiceManager) getDetailedStatus(status *ServiceStatus) {
	// 获取启动时间
	cmd := exec.Command("systemctl", "show", ServiceName, "--property=ActiveEnterTimestamp", "--value")
	if output, err := cmd.Output(); err == nil {
		timestamp := strings.TrimSpace(string(output))
		if timestamp != "" && timestamp != "n/a" {
			status.StartTime = timestamp
		}
	}

	// 获取内存使用情况
	cmd = exec.Command("systemctl", "show", ServiceName, "--property=MemoryCurrent", "--value")
	if output, err := cmd.Output(); err == nil {
		memory := strings.TrimSpace(string(output))
		if memory != "" && memory != "[not set]" && memory != "0" {
			if memBytes, err := parseBytes(memory); err == nil {
				status.Memory = formatBytes(memBytes)
			}
		}
	}
}

// parseBytes 解析字节数
func parseBytes(s string) (int64, error) {
	// 简单实现，假设输入是纯数字（字节）
	var bytes int64
	if _, err := fmt.Sscanf(s, "%d", &bytes); err != nil {
		return 0, err
	}
	return bytes, nil
}

// formatBytes 格式化字节数显示
func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/1024/1024)
	} else {
		return fmt.Sprintf("%.1f GB", float64(bytes)/1024/1024/1024)
	}
}

// PrintServiceStatus 打印服务状态
func (sm *ServiceManager) PrintServiceStatus() error {
	status, err := sm.GetServiceStatus()
	if err != nil {
		return err
	}

	utils.PrintSection("Xray 服务状态")

	// 状态指示器
	var statusIcon string
	var statusColor func(...interface{}) string
	if status.Active {
		statusIcon = "●"
		statusColor = utils.BoldGreen
	} else {
		statusIcon = "●"
		statusColor = utils.BoldRed
	}

	fmt.Printf("%s %s %s - %s\n",
		statusColor(statusIcon),
		utils.BoldWhite(status.Name+".service"),
		utils.BoldWhite("Xray Service"),
		statusColor(status.Status))

	// 详细信息
	fmt.Printf("   Active: %s\n", formatActiveStatus(status))
	fmt.Printf("   Enabled: %s\n", formatEnabledStatus(status.Enabled))

	if status.StartTime != "" {
		fmt.Printf("   Started: %s\n", status.StartTime)
	}

	if status.Memory != "" {
		fmt.Printf("   Memory: %s\n", status.Memory)
	}

	return nil
}

// formatActiveStatus 格式化激活状态
func formatActiveStatus(status *ServiceStatus) string {
	if status.Active {
		return utils.BoldGreen(fmt.Sprintf("active (%s)", status.Status))
	} else {
		return utils.BoldRed("inactive (dead)")
	}
}

// formatEnabledStatus 格式化启用状态
func formatEnabledStatus(enabled bool) string {
	if enabled {
		return utils.BoldGreen("enabled")
	} else {
		return utils.BoldRed("disabled")
	}
}

// IsServiceInstalled 检查服务是否已安装
func (sm *ServiceManager) IsServiceInstalled() bool {
	_, err := os.Stat(SystemdServicePath)
	return err == nil
}

// UninstallService 卸载服务
func (sm *ServiceManager) UninstallService() error {
	return sm.UninstallServiceWithOptions(false)
}

// UninstallServiceWithOptions 卸载服务（带选项）
func (sm *ServiceManager) UninstallServiceWithOptions(purgeUser bool) error {
	if !sm.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能卸载服务")
	}

	utils.PrintInfo("卸载 Xray 服务...")

	// 停止服务
	sm.StopService() // 忽略错误

	// 禁用服务
	sm.DisableService() // 忽略错误

	// 删除服务文件
	if err := os.Remove(SystemdServicePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除服务文件失败: %w", err)
	}

	// 重新加载 systemd
	if err := sm.reloadSystemd(); err != nil {
		utils.PrintWarning("重新加载 systemd 失败: %v", err)
	}

	// 可选：删除服务用户和组
	if purgeUser && ServiceUser == "xray" {
		sm.removeServiceUser()
	}

	utils.PrintSuccess("Xray 服务已卸载")
	return nil
}

// removeServiceUser 删除服务用户和组
func (sm *ServiceManager) removeServiceUser() {
	// 删除用户
	if exec.Command("getent", "passwd", ServiceUser).Run() == nil {
		if err := exec.Command("userdel", ServiceUser).Run(); err != nil {
			utils.PrintWarning("删除用户 %s 失败: %v", ServiceUser, err)
		} else {
			utils.PrintInfo("已删除用户: %s", ServiceUser)
		}
	}

	// 删除组
	if exec.Command("getent", "group", ServiceGroup).Run() == nil {
		if err := exec.Command("groupdel", ServiceGroup).Run(); err != nil {
			utils.PrintWarning("删除组 %s 失败: %v", ServiceGroup, err)
		} else {
			utils.PrintInfo("已删除组: %s", ServiceGroup)
		}
	}
}

// reloadSystemd 重新加载 systemd 配置
func (sm *ServiceManager) reloadSystemd() error {
	cmd := exec.Command("systemctl", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("systemctl daemon-reload 失败: %w\n输出: %s", err, string(output))
	}
	return nil
}

// GetServiceLogs 获取服务日志
func (sm *ServiceManager) GetServiceLogs(lines int, follow bool) error {
	args := []string{"journalctl", "-u", ServiceName, "--no-pager"}

	if lines > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", lines))
	}

	if follow {
		args = append(args, "-f")
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ValidateConfig 验证配置文件
func (sm *ServiceManager) ValidateConfig() error {
	utils.PrintInfo("验证 Xray 配置...")

	cmd := exec.Command(XrayBinaryPath, "run", "-confdir", XrayConfsDir, "-test")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("配置验证失败: %w\n输出: %s", err, string(output))
	}

	utils.PrintSuccess("配置验证通过")
	return nil
}

// ensureServiceUserAndGroup 确保服务用户和组存在
func (sm *ServiceManager) ensureServiceUserAndGroup() error {
	// 1. 检查并创建组
	if err := exec.Command("getent", "group", ServiceGroup).Run(); err != nil {
		// 创建系统组
		if err := exec.Command("groupadd", "--system", ServiceGroup).Run(); err != nil {
			// 尝试不同的命令格式（兼容不同发行版）
			if err := exec.Command("groupadd", "-r", ServiceGroup).Run(); err != nil {
				return fmt.Errorf("创建组 %s 失败: %w", ServiceGroup, err)
			}
		}
		utils.PrintInfo("创建系统组: %s", ServiceGroup)
	}

	// 2. 检查并创建用户
	if err := exec.Command("getent", "passwd", ServiceUser).Run(); err != nil {
		// 检测 shell 路径
		shell := ServiceShell
		if _, err := os.Stat(shell); err != nil {
			// 尝试其他常见路径
			for _, altShell := range []string{"/sbin/nologin", "/bin/false"} {
				if _, err := os.Stat(altShell); err == nil {
					shell = altShell
					break
				}
			}
		}

		// 创建系统用户
		cmd := exec.Command("useradd",
			"--system",
			"--gid", ServiceGroup,
			"--home-dir", ServiceHome,
			"--no-create-home",
			"--shell", shell,
			"--comment", "Xray proxy service user",
			ServiceUser)

		if err := cmd.Run(); err != nil {
			// 尝试简化版本（兼容旧版本或不同发行版）
			cmd = exec.Command("useradd", "-r", "-g", ServiceGroup, "-s", shell, ServiceUser)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("创建用户 %s 失败: %w", ServiceUser, err)
			}
		}
		utils.PrintInfo("创建系统用户: %s", ServiceUser)
	}

	return nil
}

// getSystemUserGroup 根据系统类型获取合适的用户和组
func (sm *ServiceManager) getSystemUserGroup() (user, group string) {
	// 优先使用专用用户
	if exec.Command("getent", "passwd", "xray").Run() == nil {
		return "xray", "xray"
	}

	// 基于发行版选择合适的降级方案
	sysInfo := sm.detector.GetSystemInfo()
	if sysInfo != nil {
		switch sysInfo.Distribution {
		case "rocky", "rhel", "centos", "fedora", "almalinux":
			// RHEL系列使用 nobody/nobody
			if exec.Command("getent", "group", "nobody").Run() == nil {
				return "nobody", "nobody"
			}
		case "ubuntu", "debian", "raspbian":
			// Debian系列使用 nobody/nogroup
			if exec.Command("getent", "group", "nogroup").Run() == nil {
				return "nobody", "nogroup"
			}
		case "alpine":
			// Alpine Linux
			return "nobody", "nobody"
		}
	}

	// 默认创建专用用户
	if err := sm.ensureServiceUserAndGroup(); err == nil {
		return ServiceUser, ServiceGroup
	}

	// 最后的降级选项
	return "nobody", "nobody"
}

// setDirectoryPermissions 设置目录权限
func (sm *ServiceManager) setDirectoryPermissions() error {
	// 获取用户和组的 UID/GID
	cmd := exec.Command("id", "-u", ServiceUser)
	uidOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("获取用户 UID 失败: %w", err)
	}

	cmd = exec.Command("id", "-g", ServiceGroup)
	gidOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("获取组 GID 失败: %w", err)
	}

	uid := 0
	gid := 0
	if _, err := fmt.Sscanf(strings.TrimSpace(string(uidOutput)), "%d", &uid); err != nil {
		return fmt.Errorf("解析用户 UID 失败: %w", err)
	}
	if _, err := fmt.Sscanf(strings.TrimSpace(string(gidOutput)), "%d", &gid); err != nil {
		return fmt.Errorf("解析组 GID 失败: %w", err)
	}

	// 设置目录权限
	directories := []string{
		XrayConfigDir,
		XrayConfsDir,
		ServiceHome,
		"/var/log/xray",
	}

	for _, dir := range directories {
		// 创建目录（如果不存在）
		if err := os.MkdirAll(dir, 0755); err != nil {
			utils.PrintWarning("创建目录失败 %s: %v", dir, err)
			continue
		}

		// 设置权限
		if err := os.Chown(dir, uid, gid); err != nil {
			utils.PrintWarning("设置目录权限失败 %s: %v", dir, err)
		}
	}

	return nil
}

// ConfigureUser 配置服务用户（保留兼容性）
func (sm *ServiceManager) ConfigureUser() error {
	if !sm.detector.IsRoot() {
		return fmt.Errorf("需要 root 权限才能配置用户")
	}

	// 使用新的用户管理方法
	if err := sm.ensureServiceUserAndGroup(); err != nil {
		return err
	}

	// 设置目录权限
	return sm.setDirectoryPermissions()
}
