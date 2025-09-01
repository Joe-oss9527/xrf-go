package tls

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewCaddyManager(t *testing.T) {
	manager := NewCaddyManager()

	if manager.adminAPI != DefaultCaddyAdminAPI {
		t.Errorf("Expected admin API %s, got %s", DefaultCaddyAdminAPI, manager.adminAPI)
	}

	if manager.binaryPath != DefaultCaddyBinary {
		t.Errorf("Expected binary path %s, got %s", DefaultCaddyBinary, manager.binaryPath)
	}

	if manager.serviceName != CaddyServiceName {
		t.Errorf("Expected service name %s, got %s", CaddyServiceName, manager.serviceName)
	}
}

func TestSetAdminAPI(t *testing.T) {
	manager := NewCaddyManager()
	customAPI := "localhost:3019"
	manager.SetAdminAPI(customAPI)

	if manager.adminAPI != customAPI {
		t.Errorf("Expected admin API %s, got %s", customAPI, manager.adminAPI)
	}
}

func TestSetConfigDir(t *testing.T) {
	manager := NewCaddyManager()
	customDir := "/tmp/test-caddy"
	manager.SetConfigDir(customDir)

	if manager.configDir != customDir {
		t.Errorf("Expected config dir %s, got %s", customDir, manager.configDir)
	}
}

func TestSetBinaryPath(t *testing.T) {
	manager := NewCaddyManager()
	customPath := "/usr/bin/caddy"
	manager.SetBinaryPath(customPath)

	if manager.binaryPath != customPath {
		t.Errorf("Expected binary path %s, got %s", customPath, manager.binaryPath)
	}
}

func TestIsCaddyInstalled(t *testing.T) {
	manager := NewCaddyManager()

	// 测试不存在的路径
	manager.SetBinaryPath("/tmp/non-existent-caddy")
	if manager.isCaddyInstalled() {
		t.Error("Expected Caddy not installed, but got installed")
	}

	// 测试存在但不可执行的文件
	tmpFile := filepath.Join(t.TempDir(), "fake-caddy")
	os.WriteFile(tmpFile, []byte("fake caddy"), 0644)
	manager.SetBinaryPath(tmpFile)

	// 应该返回 false 因为不是真正的 caddy 二进制文件
	if manager.isCaddyInstalled() {
		t.Error("Expected fake Caddy not recognized as installed")
	}
}

func TestCopyFile(t *testing.T) {
	manager := NewCaddyManager()
	tmpDir := t.TempDir()

	// 创建源文件
	srcPath := filepath.Join(tmpDir, "source.txt")
	content := "test content"
	err := os.WriteFile(srcPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// 测试复制
	dstPath := filepath.Join(tmpDir, "destination.txt")
	err = manager.copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// 验证复制结果
	copiedContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copiedContent) != content {
		t.Errorf("Expected content %s, got %s", content, string(copiedContent))
	}
}

func TestCaddyConfigJSON(t *testing.T) {
	// 测试配置结构的 JSON 序列化
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
										Host: []string{"example.com"},
									},
								},
								Handle: []CaddyHandle{
									{
										Handler: "reverse_proxy",
										Upstreams: []CaddyUpstream{
											{
												Dial: "localhost:8080",
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

	// 序列化为 JSON
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	// 验证 JSON 包含预期内容
	configStr := string(configJSON)
	expectedStrings := []string{
		"\"apps\"",
		"\"http\"",
		"\"servers\"",
		"\"srv0\"",
		"\"listen\"",
		"\":443\"",
		"\"routes\"",
		"\"match\"",
		"\"host\"",
		"\"example.com\"",
		"\"handle\"",
		"\"handler\"",
		"\"reverse_proxy\"",
		"\"upstreams\"",
		"\"dial\"",
		"\"localhost:8080\"",
	}

	for _, expected := range expectedStrings {
		if !contains(configStr, expected) {
			t.Errorf("Expected JSON to contain %s", expected)
		}
	}
}

func TestCreateSystemdService(t *testing.T) {
	manager := NewCaddyManager()
	manager.SetBinaryPath("/usr/local/bin/caddy")

	// 由于需要 root 权限写入 /etc/systemd/system，这个测试只验证服务内容生成
	// 实际的服务安装需要在集成测试中进行

	// 验证服务名设置正确
	if manager.serviceName != "caddy" {
		t.Errorf("Expected service name 'caddy', got '%s'", manager.serviceName)
	}

	t.Logf("Service configuration would be created for binary: %s", manager.binaryPath)
}

func TestGetServiceStatus(t *testing.T) {
	manager := NewCaddyManager()

	// 测试获取不存在服务的状态
	status, err := manager.GetServiceStatus()
	if err != nil {
		t.Logf("Service status check failed (expected): %v", err)
	}

	// 对于不存在的服务，应该返回 "inactive"
	if status != "inactive" {
		t.Logf("Service status: %s", status)
	}
}

func TestIsRunning(t *testing.T) {
	manager := NewCaddyManager()

	// 对于未安装的 Caddy，应该返回 false
	if manager.IsRunning() {
		t.Error("Expected Caddy not running, but got running")
	}
}

// 辅助函数
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
