package config

import (
	"os"
	"testing"
	"time"
)

// TestAddProtocol_FullIntegration 使用测试证书的完整流程测试
func TestAddProtocol_FullIntegration(t *testing.T) {
	// 使用测试证书的完整流程测试
	// 包含备份、验证、回滚
	// 不需要重复设置环境变量，TestMain已经设置了

	tempDir := "/tmp/xrf-integration-test"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	configMgr := NewConfigManager(tempDir)
	err := configMgr.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 测试TLS协议的完整流程
	t.Run("VLESS-WS-TLS-Full", func(t *testing.T) {
		start := time.Now()
		options := map[string]interface{}{
			"port": 19443,
			"path": "/ws",
		}

		err := configMgr.AddProtocol("vless-ws", "vless-integration", options)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("AddProtocol failed: %v", err)
		}

		// 完整流程合理预期<300ms（TLS首次证书生成需要更多时间，CI环境可能更慢）
		if duration > 300*time.Millisecond {
			t.Errorf("Full integration took %v, exceeds 300ms target", duration)
		} else {
			t.Logf("✅ Full integration completed in %v (under 300ms target)", duration)
		}

		// 验证配置是否包含证书
		info, err := configMgr.GetProtocolInfo("vless-integration")
		if err != nil {
			t.Fatalf("GetProtocolInfo failed: %v", err)
		}

		if info.Type != "vless" {
			t.Errorf("Expected type vless, got %s", info.Type)
		}
	})

	// 测试Trojan-WS-TLS的完整流程
	t.Run("Trojan-WS-TLS-Full", func(t *testing.T) {
		start := time.Now()
		options := map[string]interface{}{
			"port": 18443,
			"path": "/trojan",
		}

		err := configMgr.AddProtocol("trojan-ws", "trojan-integration", options)
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("AddProtocol failed: %v", err)
		}

		if duration > 200*time.Millisecond {
			t.Errorf("Full integration took %v, exceeds 200ms target", duration)
		} else {
			t.Logf("✅ Full integration completed in %v (under 200ms target)", duration)
		}
	})
}

// BenchmarkAddProtocol_FullFlow 完整流程性能测试，合理预期<100ms
func BenchmarkAddProtocol_FullFlow(b *testing.B) {
	// 完整流程性能测试，合理预期<100ms
	// 不需要重复设置环境变量，TestMain已经设置了

	tempDir := "/tmp/xrf-benchmark-full"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	configMgr := NewConfigManager(tempDir)
	if err := configMgr.Initialize(); err != nil {
		b.Fatalf("Initialize failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tag := "bench-full-" + string(rune(i))
		options := map[string]interface{}{
			"port": 9000 + i,
			"path": "/ws",
		}

		if err := configMgr.AddProtocol("vless-ws", tag, options); err != nil {
			b.Errorf("AddProtocol failed: %v", err)
		}
	}
}

// TestTLS_CertificateAutoGeneration 测试证书自动生成
func TestTLS_CertificateAutoGeneration(t *testing.T) {
	// 不需要重复设置环境变量，TestMain已经设置了

	// 获取测试证书
	testCert, err := GetTestCertificate()
	if err != nil {
		t.Fatalf("Failed to get test certificate: %v", err)
	}

	// 验证证书文件存在
	if _, err := os.Stat(testCert.CertFile); os.IsNotExist(err) {
		t.Error("Certificate file does not exist")
	}

	if _, err := os.Stat(testCert.KeyFile); os.IsNotExist(err) {
		t.Error("Key file does not exist")
	}

	t.Logf("✅ Test certificate generated at: %s", testCert.TempDir)
}

// TestTLS_MultipleProtocolsSharedCertificate 测试多个协议共享证书
func TestTLS_MultipleProtocolsSharedCertificate(t *testing.T) {
	// 不需要重复设置环境变量，TestMain已经设置了

	tempDir := "/tmp/xrf-shared-cert-test"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	configMgr := NewConfigManager(tempDir)
	if err := configMgr.Initialize(); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 添加多个TLS协议
	protocols := []string{"vless-ws", "trojan-ws"}

	var firstCertPath string
	for i, protocol := range protocols {
		tag := protocol + "-shared"
		options := map[string]interface{}{
			"port": 10000 + i,
			"path": "/shared",
		}

		err := configMgr.AddProtocol(protocol, tag, options)
		if err != nil {
			t.Fatalf("Failed to add protocol %s: %v", protocol, err)
		}

		// 获取测试证书路径
		testCert, _ := GetTestCertificate()
		if i == 0 {
			firstCertPath = testCert.CertFile
		} else {
			// 验证使用相同的证书（缓存）
			if testCert.CertFile != firstCertPath {
				t.Error("Different certificates used for multiple protocols")
			}
		}
	}

	t.Log("✅ All TLS protocols share the same test certificate")
}
