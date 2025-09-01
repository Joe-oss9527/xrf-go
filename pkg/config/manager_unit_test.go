package config

import (
	"os"
	"testing"
	"time"
)

// TestAddProtocol_ConfigGeneration 测试配置生成逻辑，不执行I/O和外部验证
func TestAddProtocol_ConfigGeneration(t *testing.T) {
	// 测试配置生成逻辑，不执行I/O和外部验证
	// 使用临时目录，跳过Xray验证
	os.Setenv("XRF_SKIP_VALIDATION", "1")
	defer os.Unsetenv("XRF_SKIP_VALIDATION")

	tempDir := "/tmp/xrf-unit-test"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	configMgr := NewConfigManager(tempDir)
	err := configMgr.Initialize()
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// 测试不同协议的配置生成速度
	protocols := []struct {
		name string
		tag  string
	}{
		{"vless-ws", "vless-unit"},
		{"vmess", "vmess-unit"},
		{"ss", "ss-unit"},
	}

	for _, protocol := range protocols {
		t.Run(protocol.name, func(t *testing.T) {
			start := time.Now()
			options := map[string]interface{}{
				"port": 8000,
			}

			err := configMgr.AddProtocol(protocol.name, protocol.tag, options)
			duration := time.Since(start)

			if err != nil {
				t.Errorf("AddProtocol failed: %v", err)
			}

			// 单元测试目标：包含I/O但跳过验证，调整到20ms
			targetDuration := 20 * time.Millisecond
			if protocol.name == "vless-ws" {
				targetDuration = 80 * time.Millisecond // 首次需要生成证书，调整到80ms
			}

			if duration > targetDuration {
				t.Errorf("%s took %v, exceeds %v target", protocol.name, duration, targetDuration)
			} else {
				t.Logf("✅ %s completed in %v (under %v target)", protocol.name, duration, targetDuration)
			}
		})
	}
}

// BenchmarkAddProtocol_ConfigOnly 纯配置生成性能测试，目标<1ms
func BenchmarkAddProtocol_ConfigOnly(b *testing.B) {
	// 纯配置生成性能测试，目标<1ms
	// 不包含文件操作和验证
	os.Setenv("XRF_SKIP_VALIDATION", "1")
	defer os.Unsetenv("XRF_SKIP_VALIDATION")

	tempDir := "/tmp/xrf-benchmark-test"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	configMgr := NewConfigManager(tempDir)
	if err := configMgr.Initialize(); err != nil {
		b.Fatalf("Initialize failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tag := "bench-" + string(rune(i))
		options := map[string]interface{}{
			"port": 8000 + i,
		}

		if err := configMgr.AddProtocol("vless-ws", tag, options); err != nil {
			b.Errorf("AddProtocol failed: %v", err)
		}
	}
}
