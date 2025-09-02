package system

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/yourusername/xrf-go/pkg/config"
)

// 纯逻辑单元测试 - 不涉及真实系统调用

func TestServiceManager_GenerateServiceFile_Logic(t *testing.T) {
	t.Parallel()

	// 验证测试环境
	if !config.IsTestEnvironment() {
		t.Skip("Not in test environment")
	}

	detector := NewDetector()
	sm := NewServiceManager(detector)

	start := time.Now()
	serviceContent := sm.generateServiceFile()
	elapsed := time.Since(start)

	// 验证服务文件内容完整性
	requiredSections := []string{
		"[Unit]",
		"[Service]",
		"[Install]",
	}

	for _, section := range requiredSections {
		if !strings.Contains(serviceContent, section) {
			t.Errorf("generateServiceFile() missing section: %s", section)
		}
	}

	// 验证关键配置项（不包含动态用户/组）
	requiredConfigs := []string{
		"Description=Xray Service",
		"Type=simple",
		"Restart=on-failure",
		"NoNewPrivileges=true",
		"ProtectSystem=strict",
		"WantedBy=multi-user.target",
	}

	// 单独验证用户和组字段存在
	if !strings.Contains(serviceContent, "User=") {
		t.Error("generateServiceFile() missing User field")
	}
	if !strings.Contains(serviceContent, "Group=") {
		t.Error("generateServiceFile() missing Group field")
	}

	for _, config := range requiredConfigs {
		if !strings.Contains(serviceContent, config) {
			t.Errorf("generateServiceFile() missing config: %s", config)
		}
	}

	// 验证路径配置
	if !strings.Contains(serviceContent, XrayBinaryPath) {
		t.Errorf("generateServiceFile() should contain XrayBinaryPath: %s", XrayBinaryPath)
	}

	if !strings.Contains(serviceContent, XrayConfsDir) {
		t.Errorf("generateServiceFile() should contain XrayConfsDir: %s", XrayConfsDir)
	}

	// 字符串模板生成应该很快，但在CI环境中可能涉及系统调用
	// CI环境中的用户组检查可能较慢
	expectedDuration := 10 * time.Millisecond
	if config.IsTestEnvironment() {
		expectedDuration = 500 * time.Millisecond // CI环境允许更长时间
	}
	
	if elapsed > expectedDuration {
		t.Errorf("generateServiceFile() took %v, expected < %v", elapsed, expectedDuration)
	}
}

func TestServiceManager_FormatBytes_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "Bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "Kilobytes",
			bytes:    1536, // 1.5KB
			expected: "1.5 KB",
		},
		{
			name:     "Megabytes",
			bytes:    2097152, // 2MB
			expected: "2.0 MB",
		},
		{
			name:     "Gigabytes",
			bytes:    1073741824, // 1GB
			expected: "1.0 GB",
		},
		{
			name:     "Zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "Exact 1KB",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "Large GB value",
			bytes:    5368709120, // 5GB
			expected: "5.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			result := formatBytes(tt.bytes)
			elapsed := time.Since(start)

			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %v, want %v", tt.bytes, result, tt.expected)
			}

			// 数学计算应该非常快
			if elapsed > 2*time.Millisecond {
				t.Errorf("formatBytes() took %v, expected < 2ms for math calculation", elapsed)
			}
		})
	}
}

func TestServiceManager_ParseBytes_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected int64
		wantErr  bool
	}{
		{
			name:     "Valid positive number",
			input:    "1024",
			expected: 1024,
			wantErr:  false,
		},
		{
			name:     "Zero",
			input:    "0",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "Large number",
			input:    "1073741824",
			expected: 1073741824,
			wantErr:  false,
		},
		{
			name:     "Invalid string",
			input:    "not-a-number",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "Negative number",
			input:    "-1024",
			expected: -1024,
			wantErr:  false,
		},
		{
			name:     "Float-like string",
			input:    "1024.5",
			expected: 1024, // fmt.Sscanf parses the integer part
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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

			// 字符串解析应该很快
			if elapsed > 2*time.Millisecond {
				t.Errorf("parseBytes() took %v, expected < 2ms for string parsing", elapsed)
			}
		})
	}
}

func TestServiceManager_FormatActiveStatus_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		status   *ServiceStatus
		expected []string // 期望包含的字符串片段
	}{
		{
			name: "Active service",
			status: &ServiceStatus{
				Active: true,
				Status: "running",
			},
			expected: []string{"active", "running"},
		},
		{
			name: "Inactive service",
			status: &ServiceStatus{
				Active: false,
				Status: "dead",
			},
			expected: []string{"inactive", "dead"},
		},
		{
			name: "Active but different status",
			status: &ServiceStatus{
				Active: true,
				Status: "starting",
			},
			expected: []string{"active", "starting"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			result := formatActiveStatus(tt.status)
			elapsed := time.Since(start)

			// 清理ANSI颜色代码进行检查
			cleanResult := strings.ReplaceAll(result, "\033[", "")
			for i := 0; i < 10; i++ {
				cleanResult = strings.ReplaceAll(cleanResult, fmt.Sprintf("%dm", i), "")
				cleanResult = strings.ReplaceAll(cleanResult, fmt.Sprintf("1;3%dm", i), "")
			}

			for _, expected := range tt.expected {
				if !strings.Contains(cleanResult, expected) {
					t.Errorf("formatActiveStatus() = %v, should contain %v", cleanResult, expected)
				}
			}

			// 字符串格式化应该很快
			if elapsed > 5*time.Millisecond {
				t.Errorf("formatActiveStatus() took %v, expected < 5ms for string formatting", elapsed)
			}
		})
	}
}

func TestServiceManager_FormatEnabledStatus_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		enabled  bool
		expected string
	}{
		{
			name:     "Enabled service",
			enabled:  true,
			expected: "enabled",
		},
		{
			name:     "Disabled service",
			enabled:  false,
			expected: "disabled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			result := formatEnabledStatus(tt.enabled)
			elapsed := time.Since(start)

			// 清理ANSI颜色代码
			cleanResult := strings.ReplaceAll(result, "\033[", "")
			for i := 0; i < 10; i++ {
				cleanResult = strings.ReplaceAll(cleanResult, fmt.Sprintf("%dm", i), "")
				cleanResult = strings.ReplaceAll(cleanResult, fmt.Sprintf("1;3%dm", i), "")
			}

			if !strings.Contains(cleanResult, tt.expected) {
				t.Errorf("formatEnabledStatus() = %v, should contain %v", cleanResult, tt.expected)
			}

			// 简单条件格式化应该很快
			if elapsed > 5*time.Millisecond {
				t.Errorf("formatEnabledStatus() took %v, expected < 5ms for conditional formatting", elapsed)
			}
		})
	}
}

func TestServiceManager_Constants_Logic(t *testing.T) {
	t.Parallel()

	// 验证服务常量的合理性
	tests := []struct {
		name     string
		value    string
		validate func(string) bool
	}{
		{
			name:  "ServiceName should be valid",
			value: ServiceName,
			validate: func(s string) bool {
				return s != "" && !strings.Contains(s, " ") && len(s) < 50
			},
		},
		{
			name:  "SystemdServicePath should be absolute",
			value: SystemdServicePath,
			validate: func(s string) bool {
				return strings.HasPrefix(s, "/") && strings.HasSuffix(s, ".service")
			},
		},
		{
			name:  "ServiceUser should be safe",
			value: ServiceUser,
			validate: func(s string) bool {
				return s == "xray" && !strings.Contains(s, " ")
			},
		},
		{
			name:  "ServiceGroup should be safe",
			value: ServiceGroup,
			validate: func(s string) bool {
				return s == "xray" && !strings.Contains(s, " ")
			},
		},
		{
			name:  "ServiceHome should be absolute path",
			value: ServiceHome,
			validate: func(s string) bool {
				return strings.HasPrefix(s, "/var/lib/") && s != ""
			},
		},
		{
			name:  "ServiceShell should be nologin",
			value: ServiceShell,
			validate: func(s string) bool {
				return strings.Contains(s, "nologin") || strings.Contains(s, "false")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			isValid := tt.validate(tt.value)
			elapsed := time.Since(start)

			if !isValid {
				t.Errorf("%s failed validation: %v", tt.name, tt.value)
			}

			// 简单验证应该瞬时完成
			if elapsed > 2*time.Millisecond {
				t.Errorf("Constant validation took %v, expected < 2ms", elapsed)
			}
		})
	}
}

// TestServiceManager_GetSystemUserGroup_Logic 测试用户组选择逻辑
func TestServiceManager_GetSystemUserGroup_Logic(t *testing.T) {
	t.Parallel()

	// 验证测试环境
	if !config.IsTestEnvironment() {
		t.Skip("Not in test environment")
	}

	detector := NewDetector()
	sm := NewServiceManager(detector)

	start := time.Now()
	user, group := sm.getSystemUserGroup()
	elapsed := time.Since(start)

	// 验证返回值不为空
	if user == "" || group == "" {
		t.Error("getSystemUserGroup() returned empty user or group")
	}

	// 验证用户组合理性
	validUsers := []string{"xray", "nobody"}
	validGroups := []string{"xray", "nobody", "nogroup"}

	userValid := false
	for _, v := range validUsers {
		if user == v {
			userValid = true
			break
		}
	}

	groupValid := false
	for _, v := range validGroups {
		if group == v {
			groupValid = true
			break
		}
	}

	if !userValid {
		t.Errorf("Invalid user: %s", user)
	}
	if !groupValid {
		t.Errorf("Invalid group: %s", group)
	}

	// 用户组选择应该很快，但CI环境中的getent命令可能较慢
	expectedDuration := 100 * time.Millisecond
	if config.IsTestEnvironment() {
		expectedDuration = 500 * time.Millisecond // CI环境允许更长时间
	}
	
	if elapsed > expectedDuration {
		t.Errorf("getSystemUserGroup() took %v, expected < %v", elapsed, expectedDuration)
	}
}

// TestServiceManager_Constants_Updated 测试更新后的常量值
func TestServiceManager_Constants_Updated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"ServiceUser should be xray", ServiceUser, "xray"},
		{"ServiceGroup should be xray", ServiceGroup, "xray"},
		{"ServiceHome should be /var/lib/xray", ServiceHome, "/var/lib/xray"},
		{"ServiceShell should contain nologin", ServiceShell, "/usr/sbin/nologin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.value, tt.expected)
			}
		})
	}
}
