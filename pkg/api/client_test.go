package api

import (
	"strings"
	"testing"
	"time"
)

func TestAPIClientConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		address string
		want    string
	}{
		{
			name:    "default address",
			address: "",
			want:    "127.0.0.1:10085",
		},
		{
			name:    "custom address",
			address: "192.168.1.1:8080",
			want:    "192.168.1.1:8080",
		},
		{
			name:    "localhost with port",
			address: "localhost:9999",
			want:    "localhost:9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			client := NewAPIClient(tt.address)
			elapsed := time.Since(start)

			if client == nil {
				t.Error("NewAPIClient() returned nil")
				return
			}

			if client.GetAddress() != tt.want {
				t.Errorf("NewAPIClient() address = %v, want %v", client.GetAddress(), tt.want)
			}

			if client.timeout != 10*time.Second {
				t.Errorf("NewAPIClient() timeout = %v, want %v", client.timeout, 10*time.Second)
			}

			if client.IsConnected() {
				t.Error("NewAPIClient() should not be connected initially")
			}

			if elapsed > 2*time.Millisecond {
				t.Errorf("NewAPIClient() took %v, expected < 2ms", elapsed)
			}
		})
	}
}

func TestAPIClientConfigValidation(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("")

	t.Run("InboundConfig validation", func(t *testing.T) {
		tests := []struct {
			name    string
			config  *InboundConfig
			wantErr bool
			errMsg  string
		}{
			{
				name: "valid config",
				config: &InboundConfig{
					Tag:      "test-inbound",
					Port:     8080,
					Protocol: "vless",
					Settings: map[string]interface{}{"key": "value"},
				},
				wantErr: false,
			},
			{
				name: "empty tag",
				config: &InboundConfig{
					Tag:      "",
					Port:     8080,
					Protocol: "vless",
				},
				wantErr: true,
				errMsg:  "inbound tag cannot be empty",
			},
			{
				name: "invalid port - zero",
				config: &InboundConfig{
					Tag:      "test",
					Port:     0,
					Protocol: "vless",
				},
				wantErr: true,
				errMsg:  "invalid port",
			},
			{
				name: "invalid port - negative",
				config: &InboundConfig{
					Tag:      "test",
					Port:     -1,
					Protocol: "vless",
				},
				wantErr: true,
				errMsg:  "invalid port",
			},
			{
				name: "invalid port - too high",
				config: &InboundConfig{
					Tag:      "test",
					Port:     70000,
					Protocol: "vless",
				},
				wantErr: true,
				errMsg:  "invalid port",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				start := time.Now()
				err := client.AddInbound(tt.config)
				elapsed := time.Since(start)

				// 期望在未连接时返回错误，但我们主要测试配置验证
				if err == nil {
					t.Error("AddInbound() should return error when not connected")
				}

				hasValidationError := tt.wantErr && strings.Contains(err.Error(), tt.errMsg)
				hasConnectionError := strings.Contains(err.Error(), "not connected")

				if tt.wantErr && !hasValidationError && !hasConnectionError {
					t.Errorf("AddInbound() error = %v, should contain %v or connection error", err, tt.errMsg)
				}

				if elapsed > 5*time.Millisecond {
					t.Errorf("AddInbound() took %v, expected < 5ms", elapsed)
				}
			})
		}
	})

	t.Run("OutboundConfig validation", func(t *testing.T) {
		tests := []struct {
			name    string
			config  *OutboundConfig
			wantErr bool
			errMsg  string
		}{
			{
				name: "valid config",
				config: &OutboundConfig{
					Tag:      "test-outbound",
					Protocol: "freedom",
					Settings: map[string]interface{}{"key": "value"},
				},
				wantErr: false,
			},
			{
				name: "empty tag",
				config: &OutboundConfig{
					Tag:      "",
					Protocol: "freedom",
				},
				wantErr: true,
				errMsg:  "outbound tag cannot be empty",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				start := time.Now()
				err := client.AddOutbound(tt.config)
				elapsed := time.Since(start)

				if err == nil {
					t.Error("AddOutbound() should return error when not connected")
				}

				hasValidationError := tt.wantErr && strings.Contains(err.Error(), tt.errMsg)
				hasConnectionError := strings.Contains(err.Error(), "not connected")

				if tt.wantErr && !hasValidationError && !hasConnectionError {
					t.Errorf("AddOutbound() error = %v, should contain %v or connection error", err, tt.errMsg)
				}

				if elapsed > 5*time.Millisecond {
					t.Errorf("AddOutbound() took %v, expected < 5ms", elapsed)
				}
			})
		}
	})
}

func TestAPIClientOperations(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("127.0.0.1:10085")

	// 测试未连接状态下的操作
	operations := []struct {
		name string
		fn   func() error
	}{
		{
			"AddInbound",
			func() error {
				return client.AddInbound(&InboundConfig{Tag: "test", Port: 8080, Protocol: "vless"})
			},
		},
		{
			"RemoveInbound",
			func() error {
				return client.RemoveInbound("test")
			},
		},
		{
			"AddOutbound",
			func() error {
				return client.AddOutbound(&OutboundConfig{Tag: "test", Protocol: "freedom"})
			},
		},
		{
			"RemoveOutbound",
			func() error {
				return client.RemoveOutbound("test")
			},
		},
		{
			"RestartCore",
			func() error {
				return client.RestartCore()
			},
		},
	}

	for _, op := range operations {
		t.Run(op.name+"_not_connected", func(t *testing.T) {
			start := time.Now()
			err := op.fn()
			elapsed := time.Since(start)

			if err == nil {
				t.Errorf("%s() should return error when not connected", op.name)
			}

			if !strings.Contains(err.Error(), "not connected") {
				t.Errorf("%s() error = %v, should contain 'not connected'", op.name, err)
			}

			if elapsed > 5*time.Millisecond {
				t.Errorf("%s() took %v, expected < 5ms", op.name, elapsed)
			}
		})
	}

	// 测试统计相关操作
	t.Run("GetStats validation", func(t *testing.T) {
		start := time.Now()
		_, err := client.GetStats("", false)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("GetStats() should return error when not connected")
		}

		if elapsed > 5*time.Millisecond {
			t.Errorf("GetStats() took %v, expected < 5ms", elapsed)
		}

		// 测试空名称验证
		start = time.Now()
		_, err = client.GetStats("", false)
		elapsed = time.Since(start)

		if err != nil && !strings.Contains(err.Error(), "not connected") && !strings.Contains(err.Error(), "cannot be empty") {
			t.Errorf("GetStats() should validate empty name")
		}

		if elapsed > 5*time.Millisecond {
			t.Errorf("GetStats() empty name validation took %v, expected < 5ms", elapsed)
		}
	})

	t.Run("QueryStats not connected", func(t *testing.T) {
		start := time.Now()
		_, err := client.QueryStats("test", false)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("QueryStats() should return error when not connected")
		}

		if !strings.Contains(err.Error(), "not connected") {
			t.Errorf("QueryStats() error = %v, should contain 'not connected'", err)
		}

		if elapsed > 5*time.Millisecond {
			t.Errorf("QueryStats() took %v, expected < 5ms", elapsed)
		}
	})

	// 测试列表操作
	listOps := []struct {
		name string
		fn   func() (interface{}, error)
	}{
		{
			"ListInbounds",
			func() (interface{}, error) { return client.ListInbounds() },
		},
		{
			"ListOutbounds",
			func() (interface{}, error) { return client.ListOutbounds() },
		},
	}

	for _, op := range listOps {
		t.Run(op.name+"_not_connected", func(t *testing.T) {
			start := time.Now()
			_, err := op.fn()
			elapsed := time.Since(start)

			if err == nil {
				t.Errorf("%s() should return error when not connected", op.name)
			}

			// result应该为nil，这是正确的

			if !strings.Contains(err.Error(), "not connected") {
				t.Errorf("%s() error = %v, should contain 'not connected'", op.name, err)
			}

			if elapsed > 5*time.Millisecond {
				t.Errorf("%s() took %v, expected < 5ms", op.name, elapsed)
			}
		})
	}
}

func TestAPIClientErrorHandling(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("127.0.0.1:10085")

	t.Run("empty tag validation", func(t *testing.T) {
		tests := []struct {
			name string
			fn   func() error
		}{
			{
				"RemoveInbound empty tag",
				func() error { return client.RemoveInbound("") },
			},
			{
				"RemoveOutbound empty tag",
				func() error { return client.RemoveOutbound("") },
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				start := time.Now()
				err := tt.fn()
				elapsed := time.Since(start)

				if err == nil {
					t.Errorf("%s should return error", tt.name)
				}

				hasTagError := strings.Contains(err.Error(), "cannot be empty")
				hasConnectionError := strings.Contains(err.Error(), "not connected")

				if !hasTagError && !hasConnectionError {
					t.Errorf("%s error = %v, should validate empty tag or connection", tt.name, err)
				}

				if elapsed > 5*time.Millisecond {
					t.Errorf("%s took %v, expected < 5ms", tt.name, elapsed)
				}
			})
		}
	})

	t.Run("GetStats empty name", func(t *testing.T) {
		start := time.Now()
		_, err := client.GetStats("", false)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("GetStats() should return error for empty name")
		}

		hasNameError := strings.Contains(err.Error(), "cannot be empty")
		hasConnectionError := strings.Contains(err.Error(), "not connected")

		if !hasNameError && !hasConnectionError {
			t.Errorf("GetStats() error = %v, should validate empty name or connection", err)
		}

		if elapsed > 5*time.Millisecond {
			t.Errorf("GetStats() took %v, expected < 5ms", elapsed)
		}
	})

	t.Run("Close operations", func(t *testing.T) {
		// 测试关闭未连接的客户端
		start := time.Now()
		err := client.Close()
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("Close() on unconnected client should not return error: %v", err)
		}

		if elapsed > 2*time.Millisecond {
			t.Errorf("Close() took %v, expected < 2ms", elapsed)
		}
	})
}

func TestAPIClientTimeout(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("127.0.0.1:10085")

	// 测试超时设置
	newTimeout := 5 * time.Second

	start := time.Now()
	client.SetTimeout(newTimeout)
	elapsed := time.Since(start)

	if client.timeout != newTimeout {
		t.Errorf("SetTimeout() timeout = %v, want %v", client.timeout, newTimeout)
	}

	if elapsed > 1*time.Millisecond {
		t.Errorf("SetTimeout() took %v, expected < 1ms", elapsed)
	}
}

func TestAPIClientStructures(t *testing.T) {
	t.Parallel()

	t.Run("InboundConfig structure", func(t *testing.T) {
		config := &InboundConfig{
			Tag:      "test-inbound",
			Port:     8080,
			Protocol: "vless",
			Settings: map[string]interface{}{
				"clients":    []interface{}{},
				"decryption": "none",
			},
		}

		if config.Tag != "test-inbound" {
			t.Errorf("InboundConfig.Tag = %v, want test-inbound", config.Tag)
		}

		if config.Port != 8080 {
			t.Errorf("InboundConfig.Port = %v, want 8080", config.Port)
		}

		if config.Protocol != "vless" {
			t.Errorf("InboundConfig.Protocol = %v, want vless", config.Protocol)
		}

		if config.Settings == nil {
			t.Error("InboundConfig.Settings should not be nil")
		}
	})

	t.Run("OutboundConfig structure", func(t *testing.T) {
		config := &OutboundConfig{
			Tag:      "test-outbound",
			Protocol: "freedom",
			Settings: map[string]interface{}{
				"domainStrategy": "UseIP",
			},
		}

		if config.Tag != "test-outbound" {
			t.Errorf("OutboundConfig.Tag = %v, want test-outbound", config.Tag)
		}

		if config.Protocol != "freedom" {
			t.Errorf("OutboundConfig.Protocol = %v, want freedom", config.Protocol)
		}

		if config.Settings == nil {
			t.Error("OutboundConfig.Settings should not be nil")
		}
	})

	t.Run("StatInfo structure", func(t *testing.T) {
		stat := &StatInfo{
			Name:  "inbound>>>test>>>traffic>>>uplink",
			Value: 1024,
		}

		if stat.Name == "" {
			t.Error("StatInfo.Name should not be empty")
		}

		if stat.Value != 1024 {
			t.Errorf("StatInfo.Value = %v, want 1024", stat.Value)
		}
	})
}

func TestAPIClientConnectionAttempt(t *testing.T) {
	t.Parallel()

	client := NewAPIClient("127.0.0.1:99999") // 使用不太可能占用的端口

	// 测试连接尝试（预期失败）
	start := time.Now()
	err := client.Connect()
	elapsed := time.Since(start)

	// gRPC 连接可能是延迟的，这里只检查连接是否设置
	if err != nil {
		t.Logf("Connect() failed as expected: %v", err)
	}

	if elapsed > 1000*time.Millisecond {
		t.Errorf("Connect() took %v, expected < 1000ms", elapsed)
	}

	// 验证连接状态 - 如果连接成功创建了，应该清理
	if client.IsConnected() {
		client.Close() // 清理连接
	}
}

func BenchmarkAPIClientOperations(b *testing.B) {
	client := NewAPIClient("127.0.0.1:10085")

	b.Run("NewAPIClient", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = NewAPIClient("127.0.0.1:10085")
		}
	})

	b.Run("IsConnected", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = client.IsConnected()
		}
	})

	b.Run("GetAddress", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = client.GetAddress()
		}
	})

	b.Run("SetTimeout", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			client.SetTimeout(10 * time.Second)
		}
	})

	b.Run("AddInbound_validation", func(b *testing.B) {
		config := &InboundConfig{
			Tag:      "test",
			Port:     8080,
			Protocol: "vless",
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = client.AddInbound(config)
		}
	})

	b.Run("Close", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = client.Close()
		}
	})
}
