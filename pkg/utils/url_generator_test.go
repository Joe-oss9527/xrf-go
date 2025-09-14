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
