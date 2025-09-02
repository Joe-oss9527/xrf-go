package system

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestDetectorSystemDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		wantOS  string
		wantErr bool
	}{
		{
			name:   "current system detection",
			wantOS: runtime.GOOS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			detector := NewDetector()
			info, err := detector.DetectSystem()
			elapsed := time.Since(start)

			if (err != nil) != tt.wantErr {
				t.Errorf("DetectSystem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if info.OS != tt.wantOS {
					t.Errorf("DetectSystem() OS = %v, want %v", info.OS, tt.wantOS)
				}
				if info.Architecture != runtime.GOARCH {
					t.Errorf("DetectSystem() Architecture = %v, want %v", info.Architecture, runtime.GOARCH)
				}
			}

			if elapsed > 800*time.Millisecond {
				t.Errorf("DetectSystem() took %v, expected < 800ms (integration test)", elapsed)
			}
		})
	}
}

func TestDetectorLinuxDistribution(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}

	t.Parallel()

	detector := NewDetector()
	info := &SystemInfo{OS: "linux", Architecture: "amd64"}

	start := time.Now()
	err := detector.detectLinuxDistribution(info)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("detectLinuxDistribution() error = %v", err)
	}

	if info.Distribution == "" {
		t.Error("detectLinuxDistribution() should set Distribution")
	}

	if elapsed > 50*time.Millisecond {
		t.Errorf("detectLinuxDistribution() took %v, expected < 50ms (file I/O)", elapsed)
	}
}

func TestDetectorSystemSupport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mockOS   string
		mockArch string
		want     bool
	}{
		{
			name:     "supported linux amd64",
			mockOS:   "linux",
			mockArch: "amd64",
			want:     true,
		},
		{
			name:     "supported linux arm64",
			mockOS:   "linux",
			mockArch: "arm64",
			want:     true,
		},
		{
			name:     "unsupported windows",
			mockOS:   "windows",
			mockArch: "amd64",
			want:     false,
		},
		{
			name:     "unsupported architecture",
			mockOS:   "linux",
			mockArch: "386",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector := &Detector{
				info: &SystemInfo{
					OS:           tt.mockOS,
					Architecture: tt.mockArch,
				},
			}

			start := time.Now()
			supported, reason := detector.IsSupported()
			elapsed := time.Since(start)

			if supported != tt.want {
				t.Errorf("IsSupported() = %v, want %v, reason: %s", supported, tt.want, reason)
			}

			if elapsed > 5*time.Millisecond {
				t.Errorf("IsSupported() took %v, expected < 5ms", elapsed)
			}
		})
	}
}

func TestDetectorDependencies(t *testing.T) {
	t.Parallel()

	detector := NewDetector()

	start := time.Now()
	missing := detector.CheckDependencies()
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Errorf("CheckDependencies() took %v, expected < 100ms (command lookup)", elapsed)
	}

	for _, dep := range missing {
		if !strings.Contains(dep, "(") {
			t.Errorf("CheckDependencies() dependency %s should include description", dep)
		}
	}
}

func TestDetectorFirewallDetection(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}

	t.Parallel()

	detector := NewDetector()

	start := time.Now()
	hasFirewall, fwType := detector.detectFirewall()
	elapsed := time.Since(start)

	if elapsed > 800*time.Millisecond {
		t.Errorf("detectFirewall() took %v, expected < 800ms (system calls)", elapsed)
	}

	validTypes := []string{"ufw", "firewalld", "iptables", "nftables", "none"}
	found := false
	for _, validType := range validTypes {
		if fwType == validType {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("detectFirewall() returned invalid firewall type: %s", fwType)
	}

	if hasFirewall && fwType == "none" {
		t.Error("detectFirewall() hasFirewall=true but fwType=none")
	}
}

func TestDetectorPackageManagerDetection(t *testing.T) {
	t.Parallel()

	detector := NewDetector()

	start := time.Now()
	pm := detector.detectPackageManager()
	elapsed := time.Since(start)

	if pm == "" {
		t.Error("detectPackageManager() should not return empty string")
	}

	if elapsed > 100*time.Millisecond {
		t.Errorf("detectPackageManager() took %v, expected < 100ms (command lookup)", elapsed)
	}
}

func TestDetectorXrayBinaryName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		os       string
		arch     string
		expected string
	}{
		{
			name:     "linux amd64",
			os:       "linux",
			arch:     "amd64",
			expected: "Xray-linux-64",
		},
		{
			name:     "linux arm64",
			os:       "linux",
			arch:     "arm64",
			expected: "Xray-linux-arm64-v8a",
		},
		{
			name:     "darwin amd64",
			os:       "darwin",
			arch:     "amd64",
			expected: "Xray-macos-64",
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
			binaryName, err := detector.GetXrayBinaryName()
			elapsed := time.Since(start)

			if err != nil {
				t.Errorf("GetXrayBinaryName() error = %v", err)
				return
			}

			if binaryName != tt.expected {
				t.Errorf("GetXrayBinaryName() = %v, want %v", binaryName, tt.expected)
			}

			if elapsed > 5*time.Millisecond {
				t.Errorf("GetXrayBinaryName() took %v, expected < 5ms", elapsed)
			}
		})
	}
}

func TestDetectorIsRoot(t *testing.T) {
	t.Parallel()

	detector := NewDetector()

	start := time.Now()
	isRoot := detector.IsRoot()
	elapsed := time.Since(start)

	if elapsed > 2*time.Millisecond {
		t.Errorf("IsRoot() took %v, expected < 2ms", elapsed)
	}

	expectedRoot := os.Geteuid() == 0
	if isRoot != expectedRoot {
		t.Errorf("IsRoot() = %v, want %v", isRoot, expectedRoot)
	}
}

func TestDetectorGetInstallCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		pm          string
		packages    []string
		expectedCmd string
		wantErr     bool
	}{
		{
			name:        "apt package manager",
			pm:          "apt",
			packages:    []string{"curl", "tar"},
			expectedCmd: "apt update && apt install -y curl tar",
		},
		{
			name:        "yum package manager",
			pm:          "yum",
			packages:    []string{"curl", "tar"},
			expectedCmd: "yum install -y curl tar",
		},
		{
			name:        "dnf package manager",
			pm:          "dnf",
			packages:    []string{"curl", "tar"},
			expectedCmd: "dnf install -y curl tar",
		},
		{
			name:     "unsupported package manager",
			pm:       "unknown",
			packages: []string{"curl", "tar"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector := &Detector{
				info: &SystemInfo{
					PackageManager: tt.pm,
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

			if elapsed > 5*time.Millisecond {
				t.Errorf("GetInstallCommand() took %v, expected < 5ms", elapsed)
			}
		})
	}
}

func TestParseOSRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		content        string
		expectedDistro string
		expectedVer    string
	}{
		{
			name: "ubuntu os-release",
			content: `NAME="Ubuntu"
VERSION="20.04.3 LTS (Focal Fossa)"
ID=ubuntu
ID_LIKE=debian
PRETTY_NAME="Ubuntu 20.04.3 LTS"
VERSION_ID="20.04"
HOME_URL="https://www.ubuntu.com/"
SUPPORT_URL="https://help.ubuntu.com/"
BUG_REPORT_URL="https://bugs.launchpad.net/ubuntu/"`,
			expectedDistro: "ubuntu",
			expectedVer:    "20.04",
		},
		{
			name: "centos os-release",
			content: `NAME="CentOS Linux"
VERSION="8 (Core)"
ID="centos"
ID_LIKE="rhel fedora"
VERSION_ID="8"
PLATFORM_ID="platform:el8"
PRETTY_NAME="CentOS Linux 8 (Core)"`,
			expectedDistro: "centos",
			expectedVer:    "8",
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

			if err != nil {
				t.Errorf("parseOSRelease() error = %v", err)
				return
			}

			if info.Distribution != tt.expectedDistro {
				t.Errorf("parseOSRelease() Distribution = %v, want %v", info.Distribution, tt.expectedDistro)
			}

			if info.Version != tt.expectedVer {
				t.Errorf("parseOSRelease() Version = %v, want %v", info.Version, tt.expectedVer)
			}

			if elapsed > 5*time.Millisecond {
				t.Errorf("parseOSRelease() took %v, expected < 5ms", elapsed)
			}
		})
	}
}

func TestInferDistroFromPrettyName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		prettyName string
		expected   string
	}{
		{
			name:       "Ubuntu pretty name",
			prettyName: "Ubuntu 20.04.3 LTS",
			expected:   "ubuntu",
		},
		{
			name:       "Debian pretty name",
			prettyName: "Debian GNU/Linux 11 (bullseye)",
			expected:   "debian",
		},
		{
			name:       "CentOS pretty name",
			prettyName: "CentOS Linux 8 (Core)",
			expected:   "centos",
		},
		{
			name:       "Unknown distribution",
			prettyName: "Some Unknown Linux Distribution",
			expected:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			detector := NewDetector()

			start := time.Now()
			result := detector.inferDistroFromPrettyName(tt.prettyName)
			elapsed := time.Since(start)

			if result != tt.expected {
				t.Errorf("inferDistroFromPrettyName() = %v, want %v", result, tt.expected)
			}

			if elapsed > 2*time.Millisecond {
				t.Errorf("inferDistroFromPrettyName() took %v, expected < 2ms", elapsed)
			}
		})
	}
}

func BenchmarkDetectorDetectSystem(b *testing.B) {
	detector := NewDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.info = nil
		_, err := detector.DetectSystem()
		if err != nil {
			b.Errorf("DetectSystem() error = %v", err)
		}
	}
}

func BenchmarkDetectorCheckDependencies(b *testing.B) {
	detector := NewDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.CheckDependencies()
	}
}
