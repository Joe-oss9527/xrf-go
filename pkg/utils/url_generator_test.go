package utils

import (
	"reflect"
	"strings"
	"testing"
)

func TestGenerateVLESSURL(t *testing.T) {
	tests := []struct {
		name     string
		uuid     string
		host     string
		port     int
		params   map[string]string
		wantErr  bool
		contains []string
	}{
		{
			name:    "empty uuid",
			uuid:    "",
			host:    "example.com",
			port:    443,
			wantErr: true,
		},
		{
			name:    "empty host",
			uuid:    "test-uuid",
			host:    "",
			port:    443,
			wantErr: true,
		},
		{
			name:    "zero port",
			uuid:    "test-uuid",
			host:    "example.com",
			port:    0,
			wantErr: true,
		},
		{
			name: "basic URL",
			uuid: "test-uuid",
			host: "example.com",
			port: 443,
			contains: []string{
				"vless://test-uuid@example.com:443",
			},
		},
		{
			name: "with parameters",
			uuid: "test-uuid",
			host: "example.com",
			port: 443,
			params: map[string]string{
				"security": "tls",
				"type":     "ws",
			},
			contains: []string{
				"vless://test-uuid@example.com:443",
				"security=tls",
				"type=ws",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := GenerateVLESSURL(tt.uuid, tt.host, tt.port, tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GenerateVLESSURL() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("GenerateVLESSURL() unexpected error: %v", err)
				return
			}

			for _, contains := range tt.contains {
				if !strings.Contains(result, contains) {
					t.Errorf("GenerateVLESSURL() = %v, should contain %v", result, contains)
				}
			}
		})
	}
}

func TestGetClientsFromConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   map[string]interface{}
		expected []map[string]interface{}
	}{
		{
			name:     "empty config",
			config:   map[string]interface{}{},
			expected: nil,
		},
		{
			name: "no inbounds",
			config: map[string]interface{}{
				"other": "value",
			},
			expected: nil,
		},
		{
			name: "empty inbounds",
			config: map[string]interface{}{
				"inbounds": []interface{}{},
			},
			expected: nil,
		},
		{
			name: "no settings",
			config: map[string]interface{}{
				"inbounds": []interface{}{
					map[string]interface{}{
						"port": 8080,
					},
				},
			},
			expected: nil,
		},
		{
			name: "no clients",
			config: map[string]interface{}{
				"inbounds": []interface{}{
					map[string]interface{}{
						"settings": map[string]interface{}{
							"other": "value",
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "valid clients",
			config: map[string]interface{}{
				"inbounds": []interface{}{
					map[string]interface{}{
						"settings": map[string]interface{}{
							"clients": []interface{}{
								map[string]interface{}{
									"id":    "uuid1",
									"email": "test1@example.com",
								},
								map[string]interface{}{
									"id":    "uuid2",
									"email": "test2@example.com",
								},
							},
						},
					},
				},
			},
			expected: []map[string]interface{}{
				{
					"id":    "uuid1",
					"email": "test1@example.com",
				},
				{
					"id":    "uuid2",
					"email": "test2@example.com",
				},
			},
		},
		{
			name: "mixed valid and invalid clients",
			config: map[string]interface{}{
				"inbounds": []interface{}{
					map[string]interface{}{
						"settings": map[string]interface{}{
							"clients": []interface{}{
								map[string]interface{}{
									"id": "uuid1",
								},
								"invalid_client", // This should be skipped
								map[string]interface{}{
									"id": "uuid2",
								},
							},
						},
					},
				},
			},
			expected: []map[string]interface{}{
				{
					"id": "uuid1",
				},
				{
					"id": "uuid2",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getClientsFromConfig(tt.config)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("getClientsFromConfig() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGenerateProtocolURL_VLESSReality(t *testing.T) {
	cfg := map[string]interface{}{
		"host":        "example.com",
		"port":        443,
		"remark":      "Reality-Test",
		"uuid":        "00000000-0000-0000-0000-000000000001",
		"network":     "tcp",
		"publicKey":   "xL32brlJCeiqgiJ4KOL-22F0bIfoVTyEzmtuTA4Ccjs",
		"fingerprint": "chrome",
		"serverName":  "www.microsoft.com",
		"shortId":     "abcd",
		// no flow provided -> default xtls-rprx-vision expected
	}

	url, err := GenerateProtocolURL("vless-reality", "Reality-Test", cfg)
	if err != nil {
		t.Fatalf("GenerateProtocolURL returned error: %v", err)
	}

	checks := []string{
		"vless://00000000-0000-0000-0000-000000000001@example.com:443",
		"security=reality",
		"encryption=none",
		"flow=xtls-rprx-vision",
		"sni=www.microsoft.com",
		"fp=chrome",
		"pbk=xL32brlJCeiqgiJ4KOL-22F0bIfoVTyEzmtuTA4Ccjs",
		"sid=abcd",
		"type=tcp",
		"headerType=none",
		"#Reality-Test",
	}
	for _, want := range checks {
		if !strings.Contains(url, want) {
			t.Errorf("VLESS Reality share URL missing %q in %q", want, url)
		}
	}
	if strings.Contains(url, "remarks=") {
		t.Errorf("VLESS Reality URL should not include remarks query, got: %s", url)
	}
}

func TestGenerateProtocolURL_VLESSRealityWithServerNames(t *testing.T) {
	cfg := map[string]interface{}{
		"host":        "example.com",
		"port":        443,
		"remark":      "Reality-ServerNames-Test",
		"uuid":        "00000000-0000-0000-0000-000000000002",
		"network":     "tcp",
		"publicKey":   "xL32brlJCeiqgiJ4KOL-22F0bIfoVTyEzmtuTA4Ccjs",
		"fingerprint": "chrome",
		"serverNames": []interface{}{"www.microsoft.com", "www.google.com"}, // 测试 serverNames 数组
		"shortId":     "efgh",
		"spiderX":     "/test",
	}

	url, err := GenerateProtocolURL("vless-reality", "Reality-ServerNames-Test", cfg)
	if err != nil {
		t.Fatalf("GenerateProtocolURL returned error: %v", err)
	}

	checks := []string{
		"vless://00000000-0000-0000-0000-000000000002@example.com:443",
		"security=reality",
		"encryption=none",
		"flow=xtls-rprx-vision",
		"sni=www.microsoft.com", // 应该使用 serverNames[0]
		"fp=chrome",
		"pbk=xL32brlJCeiqgiJ4KOL-22F0bIfoVTyEzmtuTA4Ccjs",
		"sid=efgh",
		"spx=%2Ftest", // spiderX URL encoded
		"#Reality-ServerNames-Test",
	}
	for _, want := range checks {
		if !strings.Contains(url, want) {
			t.Errorf("VLESS Reality share URL missing %q in %q", want, url)
		}
	}
}

func TestGenerateProtocolURL_VLESSWS(t *testing.T) {
	cfg := map[string]interface{}{
		"host":    "example.com",
		"port":    443,
		"remark":  "VLESS-WS",
		"uuid":    "00000000-0000-0000-0000-000000000002",
		"network": "ws",
		"path":    "/ws",
	}

	url, err := GenerateProtocolURL("vless-ws", "VLESS-WS", cfg)
	if err != nil {
		t.Fatalf("GenerateProtocolURL returned error: %v", err)
	}

	checks := []string{
		"vless://00000000-0000-0000-0000-000000000002@example.com:443",
		"security=tls",
		"encryption=none",
		"type=ws",
		"path=%2Fws",
		"host=example.com",
		"sni=example.com",
		"#VLESS-WS",
	}
	for _, want := range checks {
		if !strings.Contains(url, want) {
			t.Errorf("VLESS WS share URL missing %q in %q", want, url)
		}
	}
}

func TestGenerateShadowsocksURL_AEAD2022(t *testing.T) {
	// 测试 AEAD-2022 编码（不使用 Base64）
	url, err := GenerateShadowsocksURL("2022-blake3-aes-256-gcm", "YctPZ6U7xPPcU+gp3u+0tx/tRizJN9K8y+uKlW2qjlI=", "example.com", 8888, "AEAD-2022-Test")
	if err != nil {
		t.Fatalf("GenerateShadowsocksURL returned error: %v", err)
	}

	// 应该使用 percent 编码而不是 Base64
	expected := "ss://2022-blake3-aes-256-gcm%3AYctPZ6U7xPPcU%2Bgp3u%2B0tx%2FtRizJN9K8y%2BuKlW2qjlI%3D@example.com:8888#AEAD-2022-Test"
	if url != expected {
		t.Errorf("AEAD-2022 Shadowsocks URL = %q, want %q", url, expected)
	}
}

func TestGenerateShadowsocksURL_Regular(t *testing.T) {
	// 测试常规方法（使用 Base64URL）
	url, err := GenerateShadowsocksURL("chacha20-ietf-poly1305", "testpassword", "example.com", 8888, "Regular-Test")
	if err != nil {
		t.Fatalf("GenerateShadowsocksURL returned error: %v", err)
	}

	// 应该使用 Base64URL 编码
	if !strings.HasPrefix(url, "ss://") || !strings.Contains(url, "@example.com:8888") || !strings.Contains(url, "#Regular-Test") {
		t.Errorf("Regular Shadowsocks URL format incorrect: %s", url)
	}

	// 验证 Base64URL 编码
	parts := strings.Split(strings.TrimPrefix(url, "ss://"), "@")
	if len(parts) != 2 {
		t.Errorf("URL format incorrect, expected userinfo@host, got: %s", url)
	}
}

func TestGenerateProtocolURL_VLESSEncryption(t *testing.T) {
	cfg := map[string]interface{}{
		"host":       "example.com",
		"port":       443,
		"remark":     "VLESS-Encryption-Test",
		"uuid":       "00000000-0000-0000-0000-000000000003",
		"network":    "tcp",
		"encryption": "mlkem768x25519plus.native.0rtt.testclientkey12345",
		"flow":       "xtls-rprx-vision",
	}

	url, err := GenerateProtocolURL("vless-encryption", "VLESS-Encryption-Test", cfg)
	if err != nil {
		t.Fatalf("GenerateProtocolURL returned error: %v", err)
	}

	checks := []string{
		"vless://00000000-0000-0000-0000-000000000003@example.com:443",
		"security=none",
		"type=tcp",
		"flow=xtls-rprx-vision",
		"encryption=mlkem768x25519plus.native.0rtt.testclientkey12345", // 应该不是 none
		"#VLESS-Encryption-Test",
	}
	for _, want := range checks {
		if !strings.Contains(url, want) {
			t.Errorf("VLESS Encryption share URL missing %q in %q", want, url)
		}
	}

	// 确认不是标准的 encryption=none
	if strings.Contains(url, "encryption=none") {
		t.Errorf("VLESS Encryption URL should not use encryption=none, got: %s", url)
	}
}

func TestGenerateProtocolURL_VLESSHTTPUpgrade(t *testing.T) {
	cfg := map[string]interface{}{
		"host":    "example.com",
		"port":    8080,
		"remark":  "VLESS-HTTPUpgrade-Test",
		"uuid":    "00000000-0000-0000-0000-000000000004",
		"network": "httpupgrade",
		"path":    "/upgrade",
		"headers": map[string]interface{}{
			"User-Agent": "Mozilla/5.0",
			"Accept":     "text/html",
		},
	}

	url, err := GenerateProtocolURL("vless-httpupgrade", "VLESS-HTTPUpgrade-Test", cfg)
	if err != nil {
		t.Fatalf("GenerateProtocolURL returned error: %v", err)
	}

	checks := []string{
		"vless://00000000-0000-0000-0000-000000000004@example.com:8080",
		"security=none",
		"encryption=none",
		"type=httpupgrade",
		"path=%2Fupgrade",
		"host=example.com",
		"#VLESS-HTTPUpgrade-Test",
	}
	for _, want := range checks {
		if !strings.Contains(url, want) {
			t.Errorf("VLESS HTTPUpgrade share URL missing %q in %q", want, url)
		}
	}
}

func TestGenerateProtocolURL_VEAlias(t *testing.T) {
	// 测试 've' 别名是否能正确处理 VLESS-Encryption
	cfg := map[string]interface{}{
		"host":       "example.com",
		"port":       443,
		"remark":     "VE-Alias-Test",
		"uuid":       "00000000-0000-0000-0000-000000000005",
		"network":    "tcp",
		"decryption": "mlkem768x25519plus.native.600s.100-111-1111.privatekey12345", // 服务端配置
		"flow":       "xtls-rprx-vision",
	}

	url, err := GenerateProtocolURL("ve", "VE-Alias-Test", cfg)
	if err != nil {
		t.Fatalf("GenerateProtocolURL returned error: %v", err)
	}

	checks := []string{
		"vless://00000000-0000-0000-0000-000000000005@example.com:443",
		"security=none",
		"type=tcp",
		"flow=xtls-rprx-vision",
		"#VE-Alias-Test",
	}
	for _, want := range checks {
		if !strings.Contains(url, want) {
			t.Errorf("VE alias share URL missing %q in %q", want, url)
		}
	}

	// 应该尝试从 decryption 推导 encryption（虽然这里会使用回退）
	if !strings.Contains(url, "encryption=") {
		t.Errorf("VE URL should contain encryption parameter, got: %s", url)
	}
}

func TestGenerateProtocolURL_HUAlias(t *testing.T) {
	// 测试 'hu' 别名是否能正确处理 VLESS-HTTPUpgrade
	cfg := map[string]interface{}{
		"host":    "example.com",
		"port":    8080,
		"remark":  "HU-Alias-Test",
		"uuid":    "00000000-0000-0000-0000-000000000006",
		"network": "httpupgrade",
		"path":    "/api/upgrade",
	}

	url, err := GenerateProtocolURL("hu", "HU-Alias-Test", cfg)
	if err != nil {
		t.Fatalf("GenerateProtocolURL returned error: %v", err)
	}

	checks := []string{
		"vless://00000000-0000-0000-0000-000000000006@example.com:8080",
		"security=none",
		"encryption=none",
		"type=httpupgrade",
		"path=%2Fapi%2Fupgrade",
		"host=example.com",
		"#HU-Alias-Test",
	}
	for _, want := range checks {
		if !strings.Contains(url, want) {
			t.Errorf("HU alias share URL missing %q in %q", want, url)
		}
	}
}

func TestGenerateProtocolURL_TrojanWS(t *testing.T) {
	cfg := map[string]interface{}{
		"host":     "example.com",
		"port":     443,
		"remark":   "Trojan-WS",
		"password": "testpass",
		"network":  "ws",
		"path":     "/ws",
	}

	url, err := GenerateProtocolURL("trojan-ws", "Trojan-WS", cfg)
	if err != nil {
		t.Fatalf("GenerateProtocolURL returned error: %v", err)
	}

	checks := []string{
		"trojan://testpass@example.com:443",
		"security=tls",
		"type=ws",
		"path=%2Fws",
		"host=example.com",
		"sni=example.com",
		"#Trojan-WS",
	}
	for _, want := range checks {
		if !strings.Contains(url, want) {
			t.Errorf("Trojan WS share URL missing %q in %q", want, url)
		}
	}
	if strings.Contains(url, "remarks=") {
		t.Errorf("Trojan URL should not include remarks query, got: %s", url)
	}
}
