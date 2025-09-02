package system

import (
	"strings"
	"testing"
	"time"

	"github.com/yourusername/xrf-go/pkg/config"
)

// 纯逻辑单元测试 - 不涉及真实系统调用

func TestDetector_ParseOSRelease_Logic(t *testing.T) {
	t.Parallel()

	// 验证测试环境
	if !config.IsTestEnvironment() {
		t.Skip("Not in test environment")
	}

	tests := []struct {
		name           string
		content        string
		expectedDistro string
		expectedVer    string
		wantErr        bool
	}{
		{
			name: "Ubuntu format",
			content: `NAME="Ubuntu"
ID=ubuntu
VERSION_ID="20.04"`,
			expectedDistro: "ubuntu",
			expectedVer:    "20.04",
			wantErr:        false,
		},
		{
			name: "CentOS format",
			content: `NAME="CentOS Linux"
ID="centos"  
VERSION_ID="8"`,
			expectedDistro: "centos",
			expectedVer:    "8",
			wantErr:        false,
		},
		{
			name:           "Empty content",
			content:        "",
			expectedDistro: "unknown",
			expectedVer:    "unknown",
			wantErr:        false,
		},
		{
			name:           "Malformed content",
			content:        "INVALID_FORMAT_NO_EQUALS",
			expectedDistro: "unknown",
			expectedVer:    "unknown",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector := NewDetector()
			info := &SystemInfo{}

			start := time.Now()
			err := detector.parseOSRelease(tt.content, info)
			elapsed := time.Since(start)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseOSRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if info.Distribution != tt.expectedDistro {
				t.Errorf("parseOSRelease() Distribution = %v, want %v", info.Distribution, tt.expectedDistro)
			}

			if info.Version != tt.expectedVer {
				t.Errorf("parseOSRelease() Version = %v, want %v", info.Version, tt.expectedVer)
			}

			// 纯逻辑测试应该很快
			if elapsed > 20*time.Millisecond {
				t.Errorf("parseOSRelease() took %v, expected < 20ms for pure logic", elapsed)
			}
		})
	}
}

func TestDetector_InferDistroFromPrettyName_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prettyName string
		expected   string
	}{
		{
			name:       "Ubuntu detection",
			prettyName: "Ubuntu 20.04.3 LTS",
			expected:   "ubuntu",
		},
		{
			name:       "CentOS detection",
			prettyName: "CentOS Linux 8 (Core)",
			expected:   "centos",
		},
		{
			name:       "Debian detection",
			prettyName: "Debian GNU/Linux 11 (bullseye)",
			expected:   "debian",
		},
		{
			name:       "Case insensitive",
			prettyName: "UBUNTU 18.04 LTS",
			expected:   "ubuntu",
		},
		{
			name:       "Unknown distro",
			prettyName: "Some Unknown Linux Distribution",
			expected:   "unknown",
		},
		{
			name:       "Empty input",
			prettyName: "",
			expected:   "unknown",
		},
	}

	detector := NewDetector()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			result := detector.inferDistroFromPrettyName(tt.prettyName)
			elapsed := time.Since(start)

			if result != tt.expected {
				t.Errorf("inferDistroFromPrettyName() = %v, want %v", result, tt.expected)
			}

			// 字符串处理应该非常快
			if elapsed > 5*time.Millisecond {
				t.Errorf("inferDistroFromPrettyName() took %v, expected < 5ms for string processing", elapsed)
			}
		})
	}
}

func TestDetector_GetXrayBinaryName_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		os       string
		arch     string
		expected string
		wantErr  bool
	}{
		{
			name:     "Linux AMD64",
			os:       "linux",
			arch:     "amd64",
			expected: "Xray-linux-64",
			wantErr:  false,
		},
		{
			name:     "Linux ARM64",
			os:       "linux",
			arch:     "arm64",
			expected: "Xray-linux-arm64-v8a",
			wantErr:  false,
		},
		{
			name:     "Darwin AMD64",
			os:       "darwin",
			arch:     "amd64",
			expected: "Xray-macos-64",
			wantErr:  false,
		},
		{
			name:     "Windows AMD64",
			os:       "windows",
			arch:     "amd64",
			expected: "Xray-windows-64",
			wantErr:  false,
		},
		{
			name:     "Linux 32bit",
			os:       "linux",
			arch:     "386",
			expected: "Xray-linux-32",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// 使用预设的系统信息进行纯逻辑测试
			detector := &Detector{
				info: &SystemInfo{
					OS:           tt.os,
					Architecture: tt.arch,
				},
			}

			start := time.Now()
			result, err := detector.GetXrayBinaryName()
			elapsed := time.Since(start)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetXrayBinaryName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.expected {
				t.Errorf("GetXrayBinaryName() = %v, want %v", result, tt.expected)
			}

			// 字符串格式化应该很快
			if elapsed > 5*time.Millisecond {
				t.Errorf("GetXrayBinaryName() took %v, expected < 5ms for string formatting", elapsed)
			}
		})
	}
}

func TestDetector_GetInstallCommand_Logic(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		packageManager string
		packages       []string
		expectedCmd    string
		wantErr        bool
	}{
		{
			name:           "APT packages",
			packageManager: "apt",
			packages:       []string{"curl", "tar"},
			expectedCmd:    "apt update && apt install -y curl tar",
			wantErr:        false,
		},
		{
			name:           "YUM packages",
			packageManager: "yum",
			packages:       []string{"wget", "unzip"},
			expectedCmd:    "yum install -y wget unzip",
			wantErr:        false,
		},
		{
			name:           "DNF packages",
			packageManager: "dnf",
			packages:       []string{"git"},
			expectedCmd:    "dnf install -y git",
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
			name:           "Empty packages",
			packageManager: "apt",
			packages:       []string{},
			expectedCmd:    "apt update && apt install -y ",
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// 使用预设的包管理器信息进行纯逻辑测试
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

			// 字符串拼接应该很快
			if elapsed > 5*time.Millisecond {
				t.Errorf("GetInstallCommand() took %v, expected < 5ms for string joining", elapsed)
			}
		})
	}
}

func TestDetector_IsSupported_Logic(t *testing.T) {
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
			name:       "Unsupported Architecture",
			os:         "linux",
			arch:       "386",
			wantResult: false,
			wantReason: "不支持的系统架构",
		},
		{
			name:       "Unsupported Darwin",
			os:         "darwin",
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
				t.Errorf("IsSupported() result = %v, want %v", supported, tt.wantResult)
			}

			if tt.wantReason != "" && !strings.Contains(reason, tt.wantReason) {
				t.Errorf("IsSupported() reason = %v, should contain %v", reason, tt.wantReason)
			}

			// 简单逻辑判断应该很快
			if elapsed > 5*time.Millisecond {
				t.Errorf("IsSupported() took %v, expected < 5ms for simple logic", elapsed)
			}
		})
	}
}
