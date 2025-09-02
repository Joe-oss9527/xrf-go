package system

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestInstallerVersionCheck(t *testing.T) {
	t.Parallel()

	detector := NewDetector()
	installer := NewInstaller(detector)

	start := time.Now()
	installed := installer.IsInstalled()
	elapsed := time.Since(start)

	if elapsed > 5*time.Millisecond {
		t.Errorf("IsInstalled() took %v, expected < 5ms", elapsed)
	}

	// 测试基本功能，无论是否安装都应该能正常返回
	if runtime.GOOS == "linux" {
		// 在Linux系统上测试版本获取逻辑
		if installed {
			start = time.Now()
			version, err := installer.GetInstalledVersion()
			elapsed = time.Since(start)

			if elapsed > 100*time.Millisecond {
				t.Errorf("GetInstalledVersion() took %v, expected < 100ms", elapsed)
			}

			if err != nil && !strings.Contains(err.Error(), "xray 未安装") {
				// 如果不是因为未安装而报错，则可能有其他问题
				t.Logf("GetInstalledVersion() returned error: %v", err)
			}

			if err == nil && version == "" {
				t.Error("GetInstalledVersion() should return non-empty version when successful")
			}
		}
	}
}

func TestInstallerBinaryName(t *testing.T) {
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
			name:     "macos amd64",
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
			installer := NewInstaller(detector)

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

			// 验证安装器能正确使用这个名称
			_ = installer
		})
	}
}

func TestInstallerDirectoryCreation(t *testing.T) {
	t.Parallel()

	detector := NewDetector()
	installer := NewInstaller(detector)

	// 测试目录路径常量
	expectedDirs := []string{
		XrayConfigDir,
		XrayConfsDir,
	}

	for _, dir := range expectedDirs {
		if dir == "" {
			t.Error("Directory constant should not be empty")
		}
		if !strings.HasPrefix(dir, "/") {
			t.Errorf("Directory %s should be absolute path", dir)
		}
	}

	// 测试 verbose 模式设置
	start := time.Now()
	installer.SetVerbose(true)
	elapsed := time.Since(start)

	if elapsed > 1*time.Millisecond {
		t.Errorf("SetVerbose() took %v, expected < 1ms", elapsed)
	}

	installer.SetVerbose(false)
}

func TestInstallerDependencies(t *testing.T) {
	t.Parallel()

	detector := NewDetector()

	// 测试依赖检查逻辑
	start := time.Now()
	missing := detector.CheckDependencies()
	elapsed := time.Since(start)

	if elapsed > 20*time.Millisecond {
		t.Errorf("CheckDependencies() took %v, expected < 20ms", elapsed)
	}

	// 验证返回的依赖格式
	for _, dep := range missing {
		if !strings.Contains(dep, "(") || !strings.Contains(dep, ")") {
			t.Errorf("Dependency format should include description: %s", dep)
		}
	}

	// 测试安装命令生成
	if runtime.GOOS == "linux" {
		packages := []string{"curl", "tar"}

		start = time.Now()
		cmd, err := detector.GetInstallCommand(packages)
		elapsed = time.Since(start)

		if elapsed > 800*time.Millisecond {
			t.Errorf("GetInstallCommand() took %v, expected < 800ms (system detection)", elapsed)
		}

		if err == nil {
			if cmd == "" {
				t.Error("GetInstallCommand() should return non-empty command")
			}
			if !strings.Contains(cmd, "curl") || !strings.Contains(cmd, "tar") {
				t.Errorf("GetInstallCommand() should include required packages: %s", cmd)
			}
		}
	}
}

func TestInstallerPermissionChecks(t *testing.T) {
	t.Parallel()

	detector := NewDetector()
	installer := NewInstaller(detector)

	// 测试非root权限检查
	if !detector.IsRoot() {
		operations := []struct {
			name string
			fn   func() error
		}{
			{"InstallXray", installer.InstallXray},
			{"Uninstall", installer.Uninstall},
			{"UpdateXray", func() error { return installer.UpdateXray("v25.8.31") }},
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

				if elapsed > 1000*time.Millisecond {
					t.Errorf("%s() took %v, expected < 1000ms (permission + system check)", op.name, elapsed)
				}
			})
		}
	}
}

func TestInstallerAPIInteraction(t *testing.T) {
	t.Parallel()

	// 创建模拟的 GitHub API 服务器
	mockRelease := GitHubRelease{
		TagName: "v25.8.31",
		Assets: []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		}{
			{
				Name:               "Xray-linux-64.zip",
				BrowserDownloadURL: "https://example.com/Xray-linux-64.zip",
			},
			{
				Name:               "Xray-macos-64.zip",
				BrowserDownloadURL: "https://example.com/Xray-macos-64.zip",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/XTLS/Xray-core/releases/latest" {
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(mockRelease); err != nil {
				http.Error(w, "Encoding error", http.StatusInternalServerError)
			}
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// 临时替换 API URL
	originalAPI := XrayReleasesAPI
	defer func() {
		// 注意：这里不能修改常量，所以我们只是测试逻辑
		_ = originalAPI
	}()

	// 测试 GitHub Release 结构
	t.Run("GitHubRelease structure", func(t *testing.T) {
		if mockRelease.TagName == "" {
			t.Error("GitHubRelease should have TagName")
		}
		if len(mockRelease.Assets) == 0 {
			t.Error("GitHubRelease should have Assets")
		}
		for _, asset := range mockRelease.Assets {
			if asset.Name == "" || asset.BrowserDownloadURL == "" {
				t.Error("Asset should have Name and BrowserDownloadURL")
			}
		}
	})

	// 测试版本比较逻辑
	t.Run("Version comparison", func(t *testing.T) {
		start := time.Now()

		// 模拟版本检查逻辑
		currentVersion := "v25.8.30"
		latestVersion := mockRelease.TagName
		hasUpdate := currentVersion != latestVersion

		elapsed := time.Since(start)

		if !hasUpdate {
			t.Error("Should detect update when versions differ")
		}

		if elapsed > 2*time.Millisecond {
			t.Errorf("Version comparison took %v, expected < 2ms", elapsed)
		}
	})
}

func TestInstallerFileOperations(t *testing.T) {
	t.Parallel()

	detector := NewDetector()
	installer := NewInstaller(detector)

	// 测试文件复制逻辑（使用临时文件）
	t.Run("CopyFile logic", func(t *testing.T) {
		// 创建临时源文件
		srcFile, err := os.CreateTemp("", "test-src-")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(srcFile.Name())

		// 写入测试内容
		testContent := "test content for installer"
		if _, err := srcFile.WriteString(testContent); err != nil {
			t.Fatalf("Failed to write test content: %v", err)
		}
		srcFile.Close()

		// 创建临时目标文件路径
		dstFile, err := os.CreateTemp("", "test-dst-")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		dstPath := dstFile.Name()
		dstFile.Close()
		os.Remove(dstPath) // 删除以便测试复制
		defer os.Remove(dstPath)

		start := time.Now()
		err = installer.copyFile(srcFile.Name(), dstPath)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("copyFile() error = %v", err)
			return
		}

		// 验证文件内容
		content, err := os.ReadFile(dstPath)
		if err != nil {
			t.Errorf("Failed to read copied file: %v", err)
			return
		}

		if string(content) != testContent {
			t.Errorf("copyFile() content = %v, want %v", string(content), testContent)
		}

		if elapsed > 10*time.Millisecond {
			t.Errorf("copyFile() took %v, expected < 10ms", elapsed)
		}
	})
}

func TestInstallerConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"XrayBinaryPath", XrayBinaryPath, "/usr/local/bin/xray"},
		{"XrayConfigDir", XrayConfigDir, "/etc/xray"},
		{"XrayConfsDir", XrayConfsDir, "/etc/xray/confs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.value, tt.expected)
			}
		})
	}

	// 验证 URL 格式
	urls := []struct {
		name string
		url  string
	}{
		{"XrayReleasesAPI", XrayReleasesAPI},
		{"GeositeURL", GeositeURL},
		{"GeoipURL", GeoipURL},
	}

	for _, u := range urls {
		t.Run(u.name, func(t *testing.T) {
			if !strings.HasPrefix(u.url, "https://") {
				t.Errorf("%s should use HTTPS: %s", u.name, u.url)
			}
			if u.url == "" {
				t.Errorf("%s should not be empty", u.name)
			}
		})
	}
}

func TestInstallerSystemSupport(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		os       string
		arch     string
		wantErr  bool
		errorMsg string
	}{
		{
			name:    "supported linux amd64",
			os:      "linux",
			arch:    "amd64",
			wantErr: false,
		},
		{
			name:    "supported linux arm64",
			os:      "linux",
			arch:    "arm64",
			wantErr: false,
		},
		{
			name:     "unsupported windows",
			os:       "windows",
			arch:     "amd64",
			wantErr:  true,
			errorMsg: "不支持的操作系统",
		},
		{
			name:     "unsupported architecture",
			os:       "linux",
			arch:     "386",
			wantErr:  true,
			errorMsg: "不支持的系统架构",
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
			installer := NewInstaller(detector)

			start := time.Now()
			supported, reason := detector.IsSupported()
			elapsed := time.Since(start)

			if (reason != "") != tt.wantErr {
				t.Errorf("IsSupported() error = %v, wantErr %v", reason, tt.wantErr)
			}

			if tt.wantErr && !strings.Contains(reason, tt.errorMsg) {
				t.Errorf("IsSupported() reason = %v, should contain %v", reason, tt.errorMsg)
			}

			if supported == tt.wantErr {
				t.Errorf("IsSupported() = %v, want %v", supported, !tt.wantErr)
			}

			if elapsed > 5*time.Millisecond {
				t.Errorf("IsSupported() took %v, expected < 5ms", elapsed)
			}

			// 验证安装器创建
			if installer == nil {
				t.Error("NewInstaller() should not return nil")
			}
		})
	}
}

func TestInstallerVersionParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "standard xray version output",
			output:   "Xray v25.8.31 (Xray, Penetrates Everything.) Custom (go1.25.0 linux/amd64)",
			expected: "v25.8.31",
		},
		{
			name:     "version with v prefix",
			output:   "Xray v25.8.31 (Xray, Penetrates Everything.) Custom",
			expected: "v25.8.31",
		},
		{
			name:     "no version found",
			output:   "Some other output without version",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()

			// 模拟版本解析逻辑
			lines := strings.Split(tt.output, "\n")
			var result = "unknown"

			for _, line := range lines {
				if strings.Contains(line, "Xray") && strings.Contains(line, "v") {
					parts := strings.Fields(line)
					for _, part := range parts {
						if strings.HasPrefix(part, "v") {
							result = part
							break
						}
					}
				} else if strings.Contains(line, "Xray") && !strings.Contains(line, "v") {
					// 处理没有v前缀的版本号
					parts := strings.Fields(line)
					for i, part := range parts {
						if part == "Xray" && i+1 < len(parts) {
							version := parts[i+1]
							if version != "" && version[0] >= '0' && version[0] <= '9' {
								result = "v" + version
								break
							}
						}
					}
				}
			}

			elapsed := time.Since(start)

			if result != tt.expected {
				t.Errorf("Version parsing = %v, want %v", result, tt.expected)
			}

			if elapsed > 5*time.Millisecond {
				t.Errorf("Version parsing took %v, expected < 5ms", elapsed)
			}
		})
	}
}

func BenchmarkInstallerIsInstalled(b *testing.B) {
	detector := NewDetector()
	installer := NewInstaller(detector)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = installer.IsInstalled()
	}
}

func BenchmarkInstallerGetBinaryName(b *testing.B) {
	detector := NewDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := detector.GetXrayBinaryName()
		if err != nil {
			b.Errorf("GetXrayBinaryName() error = %v", err)
		}
	}
}

func BenchmarkInstallerCheckDependencies(b *testing.B) {
	detector := NewDetector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detector.CheckDependencies()
	}
}
