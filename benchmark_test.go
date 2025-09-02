package main

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/yourusername/xrf-go/pkg/config"
)

// BenchmarkAddProtocol 测试添加协议的性能
func BenchmarkAddProtocol(b *testing.B) {
	// 创建临时配置目录
	tempDir := "/tmp/xrf-benchmark-test"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	// 创建配置管理器
	configMgr := config.NewConfigManager(tempDir)

	// 初始化配置目录
	if err := configMgr.Initialize(); err != nil {
		b.Fatalf("Failed to initialize config manager: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tag := fmt.Sprintf("benchmark_test_%d", i)
		options := map[string]interface{}{
			"port": 10000 + i,
		}

		start := time.Now()
		err := configMgr.AddProtocol("vless-ws", tag, options)
		duration := time.Since(start)

		if err != nil {
			b.Errorf("AddProtocol failed: %v", err)
		}

		// 验证是否超过80毫秒目标（包含备份和验证）
		if duration > 80*time.Millisecond {
			b.Errorf("AddProtocol took %v, exceeds 80ms target", duration)
		}

		b.Logf("AddProtocol for %s took %v", tag, duration)
	}
}

// BenchmarkConfigOperations 测试完整的配置操作流程
func BenchmarkConfigOperations(b *testing.B) {
	tempDir := "/tmp/xrf-benchmark-ops"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	configMgr := config.NewConfigManager(tempDir)
	if err := configMgr.Initialize(); err != nil {
		b.Fatalf("Failed to initialize: %v", err)
	}

	protocols := []string{"vless-ws", "vmess", "ss", "vless-reality"}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		protocol := protocols[i%len(protocols)]
		tag := fmt.Sprintf("bench_%d", i)

		// 测试添加
		start := time.Now()
		options := map[string]interface{}{
			"port": 20000 + i,
		}

		err := configMgr.AddProtocol(protocol, tag, options)
		addDuration := time.Since(start)

		if err != nil {
			b.Errorf("AddProtocol failed: %v", err)
			continue
		}

		// 测试信息查询
		start = time.Now()
		_, err = configMgr.GetProtocolInfo(tag)
		infoDuration := time.Since(start)

		if err != nil {
			b.Errorf("GetProtocolInfo failed: %v", err)
		}

		// 测试删除
		start = time.Now()
		err = configMgr.RemoveProtocol(tag)
		removeDuration := time.Since(start)

		if err != nil {
			b.Errorf("RemoveProtocol failed: %v", err)
		}

		totalDuration := addDuration + infoDuration + removeDuration

		// 验证添加操作时间
		if addDuration > 80*time.Millisecond {
			b.Errorf("AddProtocol took %v, exceeds 80ms target", addDuration)
		}

		b.Logf("Operations for %s: add=%v, info=%v, remove=%v, total=%v",
			protocol, addDuration, infoDuration, removeDuration, totalDuration)
	}
}

// TestPerformanceTarget 测试性能目标
func TestPerformanceTarget(t *testing.T) {
	tempDir := "/tmp/xrf-performance-test"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	configMgr := config.NewConfigManager(tempDir)
	if err := configMgr.Initialize(); err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	// 测试不同协议的性能
	protocols := []struct {
		name    string
		options map[string]interface{}
	}{
		{"vless-ws", map[string]interface{}{"port": 8080}},
		{"vmess", map[string]interface{}{"port": 8081}},
		{"ss", map[string]interface{}{"port": 8082}},
		{"vless-reality", map[string]interface{}{"port": 8083}},
		{"trojan-ws", map[string]interface{}{"port": 8084}},
	}

	for i, protocol := range protocols {
		tag := fmt.Sprintf("perf_test_%d", i)

		start := time.Now()
		err := configMgr.AddProtocol(protocol.name, tag, protocol.options)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("AddProtocol failed for %s: %v", protocol.name, err)
			continue
		}

		t.Logf("Protocol %s: %v", protocol.name, duration)

		// 验证性能目标 - 为CI环境调整了更宽松的目标
		targetDuration := 80 * time.Millisecond // 基础协议目标调整到80ms
		if protocol.name == "vless-ws" || protocol.name == "trojan-ws" {
			targetDuration = 400 * time.Millisecond // TLS协议首次需要证书生成，CI环境调整到400ms
		}

		if duration > targetDuration {
			t.Errorf("Protocol %s took %v, exceeds %v target", protocol.name, duration, targetDuration)
		} else {
			t.Logf("✅ Protocol %s meets %v target (%v)", protocol.name, targetDuration, duration)
		}
	}
}

// BenchmarkWithHotReload 测试包含热重载的完整流程性能
func BenchmarkWithHotReload(b *testing.B) {
	tempDir := "/tmp/xrf-benchmark-reload"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	configMgr := config.NewConfigManager(tempDir)
	if err := configMgr.Initialize(); err != nil {
		b.Fatalf("Failed to initialize: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tag := fmt.Sprintf("reload_test_%d", i)

		// 测试添加协议
		start := time.Now()
		options := map[string]interface{}{
			"port": 30000 + i,
		}

		err := configMgr.AddProtocol("vless-ws", tag, options)
		if err != nil {
			b.Errorf("AddProtocol failed: %v", err)
			continue
		}
		addDuration := time.Since(start)

		// 测试配置验证（热重载的前置步骤）
		start = time.Now()
		err = configMgr.ValidateConfig()
		validateDuration := time.Since(start)

		if err != nil {
			// 在基准测试中，配置验证可能失败（因为没有实际的xray二进制），这是正常的
			b.Logf("Config validation failed (expected in test): %v", err)
		}

		totalDuration := addDuration + validateDuration

		b.Logf("Add+Validate for %s: add=%v, validate=%v, total=%v",
			tag, addDuration, validateDuration, totalDuration)

		// 验证添加操作本身是否满足80毫秒目标
		if addDuration > 80*time.Millisecond {
			b.Errorf("AddProtocol took %v, exceeds 80ms target", addDuration)
		}
	}
}
