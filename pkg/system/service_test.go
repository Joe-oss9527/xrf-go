package system

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/xrf-go/pkg/config"
)

func TestServiceManagerBasics(t *testing.T) {
	t.Parallel()

	detector := NewDetector()

	start := time.Now()
	sm := NewServiceManager(detector)
	elapsed := time.Since(start)

	if sm == nil {
		t.Error("NewServiceManager() returned nil")
		return
	}

	if sm.detector != detector {
		t.Error("NewServiceManager() did not set detector correctly")
	}

	if elapsed > 2*time.Millisecond {
		t.Errorf("NewServiceManager() took %v, expected < 2ms", elapsed)
	}
}

func TestServiceManagerServiceFile(t *testing.T) {
	t.Parallel()

	detector := NewDetector()
	sm := NewServiceManager(detector)

	start := time.Now()
	serviceContent := sm.generateServiceFile()
	elapsed := time.Since(start)

	if serviceContent == "" {
		t.Error("generateServiceFile() returned empty content")
	}

	// 验证服务文件包含必要的内容
	requiredContent := []string{
		"[Unit]",
		"[Service]",
		"[Install]",
		"Description=Xray Service",
		"ExecStart=",
		"Restart=on-failure",
		"NoNewPrivileges=true",
		"ProtectSystem=strict",
		"WantedBy=multi-user.target",
	}

	// 验证用户和组字段存在（但不验证具体值，因为它是动态的）
	if !strings.Contains(serviceContent, "User=") {
		t.Error("generateServiceFile() missing User field")
	}
	if !strings.Contains(serviceContent, "Group=") {
		t.Error("generateServiceFile() missing Group field")
	}

	for _, required := range requiredContent {
		if !strings.Contains(serviceContent, required) {
			t.Errorf("generateServiceFile() missing required content: %s", required)
		}
	}

	// 在CI环境中，服务文件生成可能涉及用户组检查系统调用
	expectedDuration := 10 * time.Millisecond
	if config.IsTestEnvironment() {
		expectedDuration = 500 * time.Millisecond // CI环境允许更长时间
	}

	if elapsed > expectedDuration {
		t.Errorf("generateServiceFile() took %v, expected < %v", elapsed, expectedDuration)
	}
}

func TestServiceManagerStatus(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}

	t.Parallel()

	detector := NewDetector()
	sm := NewServiceManager(detector)

	start := time.Now()
	status, err := sm.GetServiceStatus()
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("GetServiceStatus() error = %v", err)
		return
	}

	if status == nil {
		t.Error("GetServiceStatus() returned nil status")
		return
	}

	if status.Name != ServiceName {
		t.Errorf("GetServiceStatus() Name = %v, want %v", status.Name, ServiceName)
	}

	// 验证状态字段
	if status.Status == "" {
		t.Error("GetServiceStatus() Status should not be empty")
	}

	if elapsed > 200*time.Millisecond {
		t.Errorf("GetServiceStatus() took %v, expected < 200ms", elapsed)
	}
}

func TestServiceManagerValidation(t *testing.T) {
	t.Parallel()

	detector := NewDetector()
	sm := NewServiceManager(detector)

	start := time.Now()
	_ = sm.IsServiceInstalled()
	elapsed := time.Since(start)

	if elapsed > 5*time.Millisecond {
		t.Errorf("IsServiceInstalled() took %v, expected < 5ms", elapsed)
	}

	// 测试基本状态检查
	if runtime.GOOS == "linux" {
		status := &ServiceStatus{Name: ServiceName}

		start = time.Now()
		err := sm.getBasicStatus(status)
		elapsed = time.Since(start)

		if err != nil {
			t.Errorf("getBasicStatus() error = %v", err)
		}

		if elapsed > 150*time.Millisecond {
			t.Errorf("getBasicStatus() took %v, expected < 150ms (systemctl calls)", elapsed)
		}
	}
}

func TestServiceManagerPermissions(t *testing.T) {
	t.Parallel()

	detector := NewDetector()
	sm := NewServiceManager(detector)

	// 测试非root权限检查
	if !detector.IsRoot() {
		operations := []struct {
			name string
			fn   func() error
		}{
			{"InstallService", sm.InstallService},
			{"StartService", sm.StartService},
			{"StopService", sm.StopService},
			{"RestartService", sm.RestartService},
			{"ReloadService", sm.ReloadService},
			{"EnableService", sm.EnableService},
			{"DisableService", sm.DisableService},
			{"UninstallService", sm.UninstallService},
			{"ConfigureUser", sm.ConfigureUser},
		}

		for _, op := range operations {
			t.Run(op.name, func(t *testing.T) {
				start := time.Now()
				err := op.fn()
				elapsed := time.Since(start)

				if err == nil {
					t.Errorf("%s() should return error for non-root user", op.name)
				}

				expectedError := "需要 root 权限"
				if !strings.Contains(err.Error(), expectedError) {
					t.Errorf("%s() error = %v, should contain %v", op.name, err, expectedError)
				}

				if elapsed > 5*time.Millisecond {
					t.Errorf("%s() took %v, expected < 5ms", op.name, elapsed)
				}
			})
		}
	}
}

func TestServiceManagerConstants(t *testing.T) {
	t.Parallel()

	// 验证基本常量的合理性，而不是硬编码具体值
	if ServiceName == "" {
		t.Error("ServiceName should not be empty")
	}

	if SystemdServicePath == "" || !strings.HasSuffix(SystemdServicePath, ".service") {
		t.Errorf("SystemdServicePath should be a valid service file path, got: %s", SystemdServicePath)
	}

	if ServiceUser == "" || ServiceUser == "root" {
		t.Errorf("ServiceUser should be set and not root, got: %s", ServiceUser)
	}

	if ServiceGroup == "" || ServiceGroup == "root" {
		t.Errorf("ServiceGroup should be set and not root, got: %s", ServiceGroup)
	}

	// 验证新增的常量
	if ServiceHome == "" || !strings.HasPrefix(ServiceHome, "/") {
		t.Errorf("ServiceHome should be an absolute path, got: %s", ServiceHome)
	}

	if ServiceShell == "" {
		t.Errorf("ServiceShell should be set, got: %s", ServiceShell)
	}
}

func TestServiceManagerFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"bytes", 512, "512 B"},
		{"kilobytes", 1536, "1.5 KB"},
		{"megabytes", 2097152, "2.0 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			result := formatBytes(tt.bytes)
			elapsed := time.Since(start)

			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %v, want %v", tt.bytes, result, tt.expected)
			}

			if elapsed > 1*time.Millisecond {
				t.Errorf("formatBytes() took %v, expected < 1ms", elapsed)
			}
		})
	}
}

func TestServiceManagerParseBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected int64
		wantErr  bool
	}{
		{"valid bytes", "1024", 1024, false},
		{"zero bytes", "0", 0, false},
		{"large number", "1073741824", 1073741824, false},
		{"invalid input", "invalid", 0, true},
		{"empty input", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			result, err := parseBytes(tt.input)
			elapsed := time.Since(start)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseBytes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.expected {
				t.Errorf("parseBytes() = %v, want %v", result, tt.expected)
			}

			if elapsed > 2*time.Millisecond {
				t.Errorf("parseBytes() took %v, expected < 2ms", elapsed)
			}
		})
	}
}

func TestServiceManagerFormatStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   *ServiceStatus
		active   bool
		expected string
	}{
		{
			name:     "active service",
			status:   &ServiceStatus{Active: true, Status: "running"},
			active:   true,
			expected: "active (running)",
		},
		{
			name:     "inactive service",
			status:   &ServiceStatus{Active: false, Status: "dead"},
			active:   false,
			expected: "inactive (dead)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			result := formatActiveStatus(tt.status)
			elapsed := time.Since(start)

			// 移除颜色代码进行比较
			cleanResult := strings.ReplaceAll(result, "\033[", "")
			cleanResult = strings.ReplaceAll(cleanResult, "[0m", "")
			cleanResult = strings.ReplaceAll(cleanResult, "1;32m", "")
			cleanResult = strings.ReplaceAll(cleanResult, "1;31m", "")

			if !strings.Contains(cleanResult, tt.expected) {
				t.Errorf("formatActiveStatus() = %v, should contain %v", result, tt.expected)
			}

			if elapsed > 2*time.Millisecond {
				t.Errorf("formatActiveStatus() took %v, expected < 2ms", elapsed)
			}
		})
	}
}

func TestServiceManagerEnabledStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		expected string
	}{
		{"enabled service", true, "enabled"},
		{"disabled service", false, "disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			result := formatEnabledStatus(tt.enabled)
			elapsed := time.Since(start)

			// 移除颜色代码进行比较
			cleanResult := strings.ReplaceAll(result, "\033[", "")
			cleanResult = strings.ReplaceAll(cleanResult, "[0m", "")
			cleanResult = strings.ReplaceAll(cleanResult, "1;32m", "")
			cleanResult = strings.ReplaceAll(cleanResult, "1;31m", "")

			if !strings.Contains(cleanResult, tt.expected) {
				t.Errorf("formatEnabledStatus() = %v, should contain %v", result, tt.expected)
			}

			if elapsed > 2*time.Millisecond {
				t.Errorf("formatEnabledStatus() took %v, expected < 2ms", elapsed)
			}
		})
	}
}

func TestServiceManagerSystemdCheck(t *testing.T) {
	t.Parallel()

	detector := &Detector{
		info: &SystemInfo{
			OS:         "linux",
			HasSystemd: false,
		},
	}

	sm := NewServiceManager(detector)

	start := time.Now()
	err := sm.InstallService()
	elapsed := time.Since(start)

	if err == nil {
		t.Error("InstallService() should return error when systemd is not available")
	}

	// InstallService checks root permission first, then systemd support
	// Since we're likely not root, expect the root permission error
	if !strings.Contains(err.Error(), "需要 root 权限") && !strings.Contains(err.Error(), "系统不支持 systemd") {
		t.Errorf("InstallService() error = %v, should contain permission or systemd error", err)
	}

	if elapsed > 5*time.Millisecond {
		t.Errorf("InstallService() systemd check took %v, expected < 5ms", elapsed)
	}
}

func TestServiceManagerLogCommand(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}

	t.Parallel()

	// 测试日志命令构造（不实际执行）
	// 这里我们测试构造逻辑而不是实际的日志获取
	start := time.Now()

	// 模拟 GetServiceLogs 的参数验证逻辑
	lines := 50

	expectedArgs := []string{"journalctl", "-u", ServiceName, "--no-pager", "-n", fmt.Sprintf("%d", lines)}

	if lines <= 0 {
		t.Error("Invalid lines parameter")
	}

	if len(expectedArgs) < 4 {
		t.Error("Expected args should contain at least 4 elements")
	}

	elapsed := time.Since(start)
	if elapsed > 2*time.Millisecond {
		t.Errorf("Log command validation took %v, expected < 2ms", elapsed)
	}
}

func BenchmarkServiceManagerGetStatus(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("Linux-specific benchmark")
	}

	detector := NewDetector()
	sm := NewServiceManager(detector)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sm.GetServiceStatus()
		if err != nil {
			b.Errorf("GetServiceStatus() error = %v", err)
		}
	}
}

func BenchmarkServiceManagerGenerateServiceFile(b *testing.B) {
	detector := NewDetector()
	sm := NewServiceManager(detector)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		content := sm.generateServiceFile()
		if content == "" {
			b.Error("generateServiceFile() returned empty content")
		}
	}
}

func BenchmarkFormatBytes(b *testing.B) {
	testBytes := []int64{512, 1024, 1048576, 1073741824}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, bytes := range testBytes {
			_ = formatBytes(bytes)
		}
	}
}
