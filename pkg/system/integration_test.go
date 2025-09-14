package system

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Joe-oss9527/xrf-go/pkg/api"
)

// TestSystemIntegration 测试跨模块集成功能
func TestSystemIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Parallel()

	t.Run("DetectorServiceIntegration", func(t *testing.T) {
		t.Parallel()

		start := time.Now()

		// 创建检测器
		detector := NewDetector()
		if detector == nil {
			t.Fatal("NewDetector() returned nil")
		}

		// 检测系统
		info, err := detector.DetectSystem()
		if err != nil {
			t.Fatalf("DetectSystem() error = %v", err)
		}

		// 创建服务管理器
		serviceManager := NewServiceManager(detector)
		if serviceManager == nil {
			t.Fatal("NewServiceManager() returned nil")
		}

		// 验证集成工作流
		supported, reason := detector.IsSupported()
		if !supported && info.OS == "linux" && (info.Architecture == "amd64" || info.Architecture == "arm64") {
			t.Errorf("System should be supported: %s", reason)
		}

		elapsed := time.Since(start)
		if elapsed > 1000*time.Millisecond {
			t.Errorf("DetectorServiceIntegration took %v, expected < 1000ms (system detection + service setup)", elapsed)
		}
	})

	t.Run("DetectorInstallerIntegration", func(t *testing.T) {
		t.Parallel()

		start := time.Now()

		detector := NewDetector()
		installer := NewInstaller(detector)

		if installer == nil {
			t.Fatal("NewInstaller() returned nil")
		}

		// 测试依赖检查集成
		missing := detector.CheckDependencies()

		if len(missing) > 0 {
			// 如果有缺失依赖，验证安装命令生成
			packages := []string{"curl", "tar"}
			if runtime.GOOS == "linux" {
				cmd, err := detector.GetInstallCommand(packages)
				if err != nil {
					t.Errorf("GetInstallCommand() error = %v", err)
				} else if cmd == "" {
					t.Error("GetInstallCommand() should return non-empty command")
				}
			}
		}

		// 验证二进制名称生成
		binaryName, err := detector.GetXrayBinaryName()
		if err != nil {
			t.Errorf("GetXrayBinaryName() error = %v", err)
		} else if binaryName == "" {
			t.Error("GetXrayBinaryName() should return non-empty name")
		}

		// 验证安装状态检查
		isInstalled := installer.IsInstalled()
		_ = isInstalled // 不论是否安装都是有效状态

		elapsed := time.Since(start)
		if elapsed > 200*time.Millisecond {
			t.Errorf("DetectorInstallerIntegration took %v, expected < 200ms (dependency checks + binary name)", elapsed)
		}
	})

	t.Run("ServiceInstallerIntegration", func(t *testing.T) {
		t.Parallel()

		start := time.Now()

		detector := NewDetector()
		installer := NewInstaller(detector)
		serviceManager := NewServiceManager(detector)

		// 验证服务文件生成
		serviceContent := serviceManager.generateServiceFile()
		if serviceContent == "" {
			t.Error("generateServiceFile() should return non-empty content")
		}

		// 验证服务文件包含正确的路径
		if !strings.Contains(serviceContent, XrayBinaryPath) {
			t.Errorf("Service file should contain binary path: %s", XrayBinaryPath)
		}

		if !strings.Contains(serviceContent, XrayConfsDir) {
			t.Errorf("Service file should contain config dir: %s", XrayConfsDir)
		}

		// 验证安装状态一致性
		xrayInstalled := installer.IsInstalled()
		serviceInstalled := serviceManager.IsServiceInstalled()

		// 如果 Xray 已安装但服务未安装，或反之，记录状态
		if xrayInstalled != serviceInstalled {
			t.Logf("Installation state mismatch - Xray: %v, Service: %v", xrayInstalled, serviceInstalled)
		}

		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("ServiceInstallerIntegration took %v, expected < 100ms (service file generation + status checks)", elapsed)
		}
	})
}

// TestRealSystemEnvironmentSimulation 测试真实系统环境模拟
func TestRealSystemEnvironmentSimulation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real system simulation in short mode")
	}

	t.Parallel()

	t.Run("LinuxSystemWorkflow", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Linux-specific test")
		}

		t.Parallel()

		start := time.Now()

		// 模拟完整的Linux系统检测工作流
		detector := NewDetector()
		info, err := detector.DetectSystem()
		if err != nil {
			t.Fatalf("DetectSystem() error = %v", err)
		}

		// 验证Linux特定信息
		if info.OS != "linux" {
			t.Errorf("Expected OS linux, got %s", info.OS)
		}

		if info.Distribution == "" {
			t.Error("Linux distribution should be detected")
		}

		if info.PackageManager == "unknown" {
			t.Error("Linux should have a known package manager")
		}

		// 验证systemd检测
		if info.HasSystemd {
			// 如果检测到systemd，验证相关功能
			serviceManager := NewServiceManager(detector)
			status, err := serviceManager.GetServiceStatus()
			if err != nil {
				t.Logf("GetServiceStatus() error (expected if service not installed): %v", err)
			} else if status == nil {
				t.Error("GetServiceStatus() should return status object")
			}
		}

		elapsed := time.Since(start)
		if elapsed > 400*time.Millisecond {
			t.Errorf("LinuxSystemWorkflow took %v, expected < 400ms (full Linux detection + systemd check)", elapsed)
		}
	})

	t.Run("PermissionAwareWorkflow", func(t *testing.T) {
		t.Parallel()

		start := time.Now()

		detector := NewDetector()
		installer := NewInstaller(detector)
		serviceManager := NewServiceManager(detector)

		isRoot := detector.IsRoot()

		// 验证权限敏感操作的行为
		rootRequiredOps := []struct {
			name string
			fn   func() error
		}{
			{"InstallXray", installer.InstallXray},
			{"InstallService", serviceManager.InstallService},
			{"StartService", serviceManager.StartService},
		}

		for _, op := range rootRequiredOps {
			err := op.fn()
			if !isRoot && err == nil {
				t.Errorf("%s() should require root permissions", op.name)
			}
			if !isRoot && !strings.Contains(err.Error(), "root 权限") && !strings.Contains(err.Error(), "systemd") {
				t.Errorf("%s() should return appropriate permission error", op.name)
			}
		}

		elapsed := time.Since(start)
		if elapsed > 300*time.Millisecond {
			t.Errorf("PermissionAwareWorkflow took %v, expected < 300ms (permission checks + system calls)", elapsed)
		}
	})
}

// TestEndToEndWorkflow 测试端到端工作流
func TestEndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping end-to-end test in short mode")
	}

	t.Parallel()

	t.Run("CompleteSetupWorkflow", func(t *testing.T) {
		t.Parallel()

		start := time.Now()

		// 1. 系统检测阶段
		detector := NewDetector()

		systemInfo, err := detector.DetectSystem()
		if err != nil {
			t.Fatalf("System detection failed: %v", err)
		}

		supported, reason := detector.IsSupported()
		if !supported {
			t.Logf("System not supported: %s", reason)
		}

		// 2. 依赖检查阶段
		missing := detector.CheckDependencies()
		if len(missing) > 0 {
			t.Logf("Missing dependencies: %v", missing)
		}

		// 3. 安装器准备阶段
		installer := NewInstaller(detector)
		installer.SetVerbose(false) // 测试模式下关闭详细输出

		isInstalled := installer.IsInstalled()
		if isInstalled {
			version, err := installer.GetInstalledVersion()
			if err != nil {
				t.Logf("Failed to get version: %v", err)
			} else {
				t.Logf("Current version: %s", version)
			}
		}

		// 4. 服务管理器准备阶段
		serviceManager := NewServiceManager(detector)

		isServiceInstalled := serviceManager.IsServiceInstalled()
		if isServiceInstalled {
			status, err := serviceManager.GetServiceStatus()
			if err != nil {
				t.Logf("Failed to get service status: %v", err)
			} else {
				t.Logf("Service status: %+v", status)
			}
		}

		// 5. API客户端集成
		apiClient := api.NewAPIClient("127.0.0.1:10085")
		if apiClient == nil {
			t.Error("Failed to create API client")
		}

		apiClient.SetTimeout(5 * time.Second)
		if !apiClient.IsConnected() {
			// 尝试连接（预期失败，除非实际有服务运行）
			err := apiClient.Connect()
			if err != nil {
				t.Logf("API connection failed (expected): %v", err)
			}
		}

		// 验证整体工作流的一致性
		if systemInfo.OS == "linux" && (systemInfo.Architecture == "amd64" || systemInfo.Architecture == "arm64") {
			if !supported {
				t.Error("Linux amd64/arm64 should be supported")
			}
		}

		elapsed := time.Since(start)
		if elapsed > 900*time.Millisecond {
			t.Errorf("CompleteSetupWorkflow took %v, expected < 900ms (complete end-to-end workflow)", elapsed)
		}
	})

	t.Run("ErrorRecoveryWorkflow", func(t *testing.T) {
		t.Parallel()

		start := time.Now()

		// 测试错误恢复机制
		detector := NewDetector()
		installer := NewInstaller(detector)

		// 模拟各种错误情况的处理
		if !detector.IsRoot() {
			// 非root用户下的错误处理
			err := installer.InstallXray()
			if err == nil {
				t.Error("InstallXray() should fail for non-root user")
			}

			if !strings.Contains(err.Error(), "root 权限") {
				t.Errorf("InstallXray() should return root permission error")
			}
		}

		// 测试文件操作的错误处理
		tempDir := "/nonexistent/directory/that/should/not/exist"
		srcFile := "/etc/passwd" // 这个文件应该存在
		dstFile := tempDir + "/test"

		err := installer.copyFile(srcFile, dstFile)
		if err == nil {
			t.Error("copyFile() should fail when destination directory doesn't exist")
		}

		elapsed := time.Since(start)
		if elapsed > 200*time.Millisecond {
			t.Errorf("ErrorRecoveryWorkflow took %v, expected < 200ms (error handling + file operations)", elapsed)
		}
	})
}

// TestConcurrentOperations 测试并发操作
func TestConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent operations test in short mode")
	}

	t.Parallel()

	t.Run("ConcurrentSystemDetection", func(t *testing.T) {
		const numGoroutines = 10

		start := time.Now()

		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				detector := NewDetector()
				_, err := detector.DetectSystem()
				results <- err
			}()
		}

		// 收集结果
		for i := 0; i < numGoroutines; i++ {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent DetectSystem() failed: %v", err)
			}
		}

		elapsed := time.Since(start)
		if elapsed > 1500*time.Millisecond {
			t.Errorf("ConcurrentSystemDetection took %v, expected < 1500ms (10 concurrent detections)", elapsed)
		}
	})

	t.Run("ConcurrentAPIClients", func(t *testing.T) {
		const numClients = 5

		start := time.Now()

		clients := make([]*api.APIClient, numClients)
		for i := 0; i < numClients; i++ {
			clients[i] = api.NewAPIClient(fmt.Sprintf("127.0.0.1:%d", 10085+i))
			if clients[i] == nil {
				t.Errorf("Failed to create API client %d", i)
			}
		}

		// 并发测试客户端操作
		results := make(chan error, numClients)

		for i, client := range clients {
			go func(c *api.APIClient, id int) {
				// 测试基本操作
				c.SetTimeout(time.Second)

				config := &api.InboundConfig{
					Tag:      fmt.Sprintf("test-%d", id),
					Port:     8080 + id,
					Protocol: "vless",
				}

				err := c.AddInbound(config)
				// 预期连接失败，但不应该有其他错误
				if err != nil && !strings.Contains(err.Error(), "not connected") && !strings.Contains(err.Error(), "implementation pending") {
					results <- fmt.Errorf("unexpected error from client %d: %v", id, err)
					return
				}

				results <- nil
			}(client, i)
		}

		// 收集结果
		for i := 0; i < numClients; i++ {
			err := <-results
			if err != nil {
				t.Error(err)
			}
		}

		elapsed := time.Since(start)
		if elapsed > 300*time.Millisecond {
			t.Errorf("ConcurrentAPIClients took %v, expected < 300ms (5 concurrent API clients)", elapsed)
		}
	})
}

// TestResourceCleanup 测试资源清理
func TestResourceCleanup(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource cleanup test in short mode")
	}

	t.Parallel()

	t.Run("APIClientCleanup", func(t *testing.T) {
		start := time.Now()

		client := api.NewAPIClient("127.0.0.1:10085")

		// 测试多次关闭
		err1 := client.Close()
		err2 := client.Close()

		if err1 != nil {
			t.Errorf("First Close() should not error: %v", err1)
		}

		if err2 != nil {
			t.Errorf("Second Close() should not error: %v", err2)
		}

		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			t.Errorf("APIClientCleanup took %v, expected < 50ms (double close operations)", elapsed)
		}
	})

	t.Run("TemporaryFileCleanup", func(t *testing.T) {
		start := time.Now()

		// 创建临时文件进行测试
		tempFile, err := os.CreateTemp("", "xrf-test-")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}

		tempPath := tempFile.Name()
		tempFile.Close()

		// 确保文件存在
		if _, err := os.Stat(tempPath); os.IsNotExist(err) {
			t.Fatalf("Temp file should exist: %s", tempPath)
		}

		// 清理文件
		err = os.Remove(tempPath)
		if err != nil {
			t.Errorf("Failed to remove temp file: %v", err)
		}

		// 验证文件已删除
		if _, err := os.Stat(tempPath); !os.IsNotExist(err) {
			t.Error("Temp file should be deleted")
		}

		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("TemporaryFileCleanup took %v, expected < 100ms (file create + cleanup)", elapsed)
		}
	})
}

// BenchmarkIntegrationWorkflow 基准测试集成工作流
func BenchmarkIntegrationWorkflow(b *testing.B) {
	b.Run("SystemDetectionWorkflow", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			detector := NewDetector()
			info, err := detector.DetectSystem()
			if err != nil {
				b.Errorf("DetectSystem() error = %v", err)
			}
			if info == nil {
				b.Error("DetectSystem() returned nil info")
			}
		}
	})

	b.Run("ServiceManagerWorkflow", func(b *testing.B) {
		detector := NewDetector()
		detector.DetectSystem() // 预热

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			serviceManager := NewServiceManager(detector)
			_ = serviceManager.generateServiceFile()
			_ = serviceManager.IsServiceInstalled()
		}
	})

	b.Run("APIClientWorkflow", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			client := api.NewAPIClient("127.0.0.1:10085")
			client.SetTimeout(5 * time.Second)

			config := &api.InboundConfig{
				Tag:      "benchmark-test",
				Port:     8080,
				Protocol: "vless",
			}

			_ = client.AddInbound(config) // 预期失败，但测试性能
			_ = client.Close()
		}
	})
}
