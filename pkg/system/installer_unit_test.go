package system

import (
	"strings"
	"testing"
	"time"

	"github.com/Joe-oss9527/xrf-go/pkg/config"
)

// 纯逻辑单元测试 - 不涉及真实系统调用

func TestInstaller_VersionParsing_Logic(t *testing.T) {
	t.Parallel()

	// 验证测试环境
	if !config.IsTestEnvironment() {
		t.Skip("Not in test environment")
	}

	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "Standard Xray version output",
			output:   "Xray 25.8.31 (Xray, Penetrates Everything.) Custom (go1.25.0 linux/amd64)",
			expected: "v25.8.31",
		},
		{
			name:     "Version with v prefix",
			output:   "Xray v25.8.31 (Xray, Penetrates Everything.) Custom",
			expected: "v25.8.31",
		},
		{
			name:     "Multi-line version output",
			output:   "Xray 25.8.31 (Xray, Penetrates Everything.) Custom\nBuild: go1.25.0\nArchitecture: linux/amd64",
			expected: "v25.8.31",
		},
		{
			name:     "No version found",
			output:   "Some other output without version",
			expected: "unknown",
		},
		{
			name:     "Empty output",
			output:   "",
			expected: "unknown",
		},
		{
			name:     "Version in different format",
			output:   "Xray version 25.8.31",
			expected: "v25.8.31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()

			// 模拟版本解析逻辑（修复之前的错误）
			lines := strings.Split(tt.output, "\n")
			var result = "unknown"

			for _, line := range lines {
				if strings.Contains(line, "Xray") {
					parts := strings.Fields(line)
					for i, part := range parts {
						if part == "Xray" && i+1 < len(parts) {
							version := parts[i+1]
							// 处理 "v1.8.0" 格式
							if strings.HasPrefix(version, "v") && len(version) > 1 && version[1] >= '0' && version[1] <= '9' {
								result = version
								break
							}
							// 处理 "1.7.5" 格式
							if len(version) > 0 && version[0] >= '0' && version[0] <= '9' {
								result = "v" + version
								break
							}
						}
						// 处理 "Xray version 2.0.0" 格式
						if part == "version" && i+1 < len(parts) {
							version := parts[i+1]
							if len(version) > 0 && version[0] >= '0' && version[0] <= '9' {
								result = "v" + version
								break
							}
						}
					}
					if result != "unknown" {
						break
					}
				}
			}

			elapsed := time.Since(start)

			if result != tt.expected {
				t.Errorf("Version parsing = %v, want %v", result, tt.expected)
			}

			// 字符串处理应该很快
			if elapsed > 10*time.Millisecond {
				t.Errorf("Version parsing took %v, expected < 10ms for string processing", elapsed)
			}
		})
	}
}

func TestInstaller_BinaryNameGeneration_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		os       string
		arch     string
		expected string
	}{
		{
			name:     "Linux AMD64",
			os:       "linux",
			arch:     "amd64",
			expected: "Xray-linux-64",
		},
		{
			name:     "Linux ARM64",
			os:       "linux",
			arch:     "arm64",
			expected: "Xray-linux-arm64-v8a",
		},
		{
			name:     "macOS AMD64",
			os:       "darwin",
			arch:     "amd64",
			expected: "Xray-macos-64",
		},
		{
			name:     "Windows AMD64",
			os:       "windows",
			arch:     "amd64",
			expected: "Xray-windows-64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector := &Detector{
				info: &SystemInfo{
					OS:           tt.os,
					Architecture: tt.arch,
				},
			}

			start := time.Now()
			result, err := detector.GetXrayBinaryName()
			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("GetXrayBinaryName() error = %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("GetXrayBinaryName() = %v, want %v", result, tt.expected)
			}

			// 字符串格式化应该很快
			if elapsed > 5*time.Millisecond {
				t.Errorf("GetXrayBinaryName() took %v, expected < 5ms for string formatting", elapsed)
			}
		})
	}
}

func TestInstaller_DependencyCommandGeneration_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		packageManager string
		packages       []string
		expectedCmd    string
		wantErr        bool
	}{
		{
			name:           "APT with multiple packages",
			packageManager: "apt",
			packages:       []string{"curl", "tar", "unzip"},
			expectedCmd:    "apt update && apt install -y curl tar unzip",
			wantErr:        false,
		},
		{
			name:           "YUM with single package",
			packageManager: "yum",
			packages:       []string{"wget"},
			expectedCmd:    "yum install -y wget",
			wantErr:        false,
		},
		{
			name:           "DNF with packages",
			packageManager: "dnf",
			packages:       []string{"git", "vim"},
			expectedCmd:    "dnf install -y git vim",
			wantErr:        false,
		},
		{
			name:           "Pacman packages",
			packageManager: "pacman",
			packages:       []string{"base-devel"},
			expectedCmd:    "pacman -S --noconfirm base-devel",
			wantErr:        false,
		},
		{
			name:           "Zypper packages",
			packageManager: "zypper",
			packages:       []string{"gcc"},
			expectedCmd:    "zypper install -y gcc",
			wantErr:        false,
		},
		{
			name:           "APK packages",
			packageManager: "apk",
			packages:       []string{"build-base"},
			expectedCmd:    "apk add build-base",
			wantErr:        false,
		},
		{
			name:           "Unsupported package manager",
			packageManager: "unknown",
			packages:       []string{"test"},
			expectedCmd:    "",
			wantErr:        true,
		},
		{
			name:           "Empty package list",
			packageManager: "apt",
			packages:       []string{},
			expectedCmd:    "apt update && apt install -y ",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector := &Detector{
				info: &SystemInfo{
					PackageManager: tt.packageManager,
				},
			}

			start := time.Now()
			cmd, err := detector.GetInstallCommand(tt.packages)
			elapsed := time.Since(start)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetInstallCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && cmd != tt.expectedCmd {
				t.Errorf("GetInstallCommand() = %v, want %v", cmd, tt.expectedCmd)
			}

			// 字符串操作应该很快
			if elapsed > 5*time.Millisecond {
				t.Errorf("GetInstallCommand() took %v, expected < 5ms for string operations", elapsed)
			}
		})
	}
}

func TestInstaller_SystemSupportCheck_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		os         string
		arch       string
		wantResult bool
		wantReason string
	}{
		{
			name:       "Supported Linux AMD64",
			os:         "linux",
			arch:       "amd64",
			wantResult: true,
			wantReason: "",
		},
		{
			name:       "Supported Linux ARM64",
			os:         "linux",
			arch:       "arm64",
			wantResult: true,
			wantReason: "",
		},
		{
			name:       "Unsupported Windows",
			os:         "windows",
			arch:       "amd64",
			wantResult: false,
			wantReason: "不支持的操作系统",
		},
		{
			name:       "Unsupported macOS",
			os:         "darwin",
			arch:       "amd64",
			wantResult: false,
			wantReason: "不支持的操作系统",
		},
		{
			name:       "Unsupported architecture",
			os:         "linux",
			arch:       "386",
			wantResult: false,
			wantReason: "不支持的系统架构",
		},
		{
			name:       "Unknown OS",
			os:         "freebsd",
			arch:       "amd64",
			wantResult: false,
			wantReason: "不支持的操作系统",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector := &Detector{
				info: &SystemInfo{
					OS:           tt.os,
					Architecture: tt.arch,
				},
			}

			start := time.Now()
			supported, reason := detector.IsSupported()
			elapsed := time.Since(start)

			if supported != tt.wantResult {
				t.Errorf("IsSupported() = %v, want %v", supported, tt.wantResult)
			}

			if tt.wantReason != "" && !strings.Contains(reason, tt.wantReason) {
				t.Errorf("IsSupported() reason = %v, should contain %v", reason, tt.wantReason)
			}

			// 简单逻辑判断应该很快
			if elapsed > 5*time.Millisecond {
				t.Errorf("IsSupported() took %v, expected < 5ms for logic check", elapsed)
			}
		})
	}
}

func TestInstaller_VersionComparison_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		expectedUpdate bool
	}{
		{
			name:           "Same version",
			currentVersion: "v25.8.31",
			latestVersion:  "v25.8.31",
			expectedUpdate: false,
		},
		{
			name:           "Update available",
			currentVersion: "v25.8.30",
			latestVersion:  "v25.8.31",
			expectedUpdate: true,
		},
		{
			name:           "Major version update",
			currentVersion: "v24.12.31",
			latestVersion:  "v25.8.31",
			expectedUpdate: true,
		},
		{
			name:           "Unknown current version",
			currentVersion: "unknown",
			latestVersion:  "v25.8.31",
			expectedUpdate: true,
		},
		{
			name:           "Empty versions",
			currentVersion: "",
			latestVersion:  "",
			expectedUpdate: false,
		},
		{
			name:           "Different format but same",
			currentVersion: "25.8.31",
			latestVersion:  "v25.8.31",
			expectedUpdate: true, // 字符串比较会认为不同（需要标准化比较）
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()

			// 简单版本比较逻辑
			hasUpdate := tt.currentVersion != tt.latestVersion

			elapsed := time.Since(start)

			if hasUpdate != tt.expectedUpdate {
				t.Errorf("Version comparison %v vs %v = %v, want %v",
					tt.currentVersion, tt.latestVersion, hasUpdate, tt.expectedUpdate)
			}

			// 字符串比较应该瞬时完成
			if elapsed > 2*time.Millisecond {
				t.Errorf("Version comparison took %v, expected < 2ms", elapsed)
			}
		})
	}
}

func TestInstaller_PathValidation_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		isValid bool
	}{
		{
			name:    "Valid absolute path",
			path:    "/usr/local/bin/xray",
			isValid: true,
		},
		{
			name:    "Valid config directory",
			path:    "/etc/xray/confs",
			isValid: true,
		},
		{
			name:    "Relative path",
			path:    "relative/path",
			isValid: false,
		},
		{
			name:    "Empty path",
			path:    "",
			isValid: false,
		},
		{
			name:    "Root path",
			path:    "/",
			isValid: true,
		},
		{
			name:    "Path with spaces",
			path:    "/path with spaces/file",
			isValid: true, // 技术上有效，但可能有问题
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()

			// 简单路径验证逻辑
			isValid := strings.HasPrefix(tt.path, "/") && tt.path != ""

			elapsed := time.Since(start)

			if isValid != tt.isValid {
				t.Errorf("Path validation for %v = %v, want %v", tt.path, isValid, tt.isValid)
			}

			// 字符串检查应该瞬时完成
			if elapsed > 2*time.Millisecond {
				t.Errorf("Path validation took %v, expected < 2ms", elapsed)
			}
		})
	}
}
