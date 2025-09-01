package tls

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewACMEManager(t *testing.T) {
	email := "test@example.com"
	manager := NewACMEManager(email)

	if manager.email != email {
		t.Errorf("Expected email %s, got %s", email, manager.email)
	}

	if manager.caURL != LEProductionURL {
		t.Errorf("Expected CA URL %s, got %s", LEProductionURL, manager.caURL)
	}
}

func TestSetStagingMode(t *testing.T) {
	manager := NewACMEManager("test@example.com")
	manager.SetStagingMode()

	if manager.caURL != LEStagingURL {
		t.Errorf("Expected staging URL %s, got %s", LEStagingURL, manager.caURL)
	}
}

func TestSetCertDir(t *testing.T) {
	manager := NewACMEManager("test@example.com")
	customDir := "/tmp/test-certs"
	manager.SetCertDir(customDir)

	if manager.certDir != customDir {
		t.Errorf("Expected cert dir %s, got %s", customDir, manager.certDir)
	}
}

func TestSetACMEDir(t *testing.T) {
	manager := NewACMEManager("test@example.com")
	customDir := "/tmp/test-acme"
	manager.SetACMEDir(customDir)

	if manager.acmeDir != customDir {
		t.Errorf("Expected ACME dir %s, got %s", customDir, manager.acmeDir)
	}
}

func TestInitialize(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()
	certDir := filepath.Join(tmpDir, "certs")
	acmeDir := filepath.Join(tmpDir, "acme")

	manager := NewACMEManager("test@example.com")
	manager.SetStagingMode() // 使用测试环境
	manager.SetCertDir(certDir)
	manager.SetACMEDir(acmeDir)

	// 初始化 (注意：这会尝试连接 Let's Encrypt，在测试中可能失败)
	err := manager.Initialize()
	
	// 至少验证目录创建成功
	if _, err := os.Stat(certDir); os.IsNotExist(err) {
		t.Errorf("Certificate directory was not created: %s", certDir)
	}

	if _, err := os.Stat(acmeDir); os.IsNotExist(err) {
		t.Errorf("ACME directory was not created: %s", acmeDir)
	}

	// 如果网络连接失败，我们跳过客户端初始化的测试
	if err != nil {
		t.Logf("ACME client initialization failed (expected in test environment): %v", err)
	}
}

func TestSaveCertificate(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewACMEManager("test@example.com")
	manager.SetCertDir(tmpDir)

	domain := "test.example.com"
	cert := []byte("-----BEGIN CERTIFICATE-----\ntest cert\n-----END CERTIFICATE-----")
	key := []byte("-----BEGIN PRIVATE KEY-----\ntest key\n-----END PRIVATE KEY-----")

	err := manager.saveCertificate(domain, cert, key)
	if err != nil {
		t.Fatalf("Failed to save certificate: %v", err)
	}

	// 验证文件存在
	certPath := filepath.Join(tmpDir, domain+".crt")
	keyPath := filepath.Join(tmpDir, domain+".key")

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		t.Errorf("Certificate file was not created: %s", certPath)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Errorf("Key file was not created: %s", keyPath)
	}

	// 验证文件内容
	savedCert, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("Failed to read saved certificate: %v", err)
	}

	if string(savedCert) != string(cert) {
		t.Errorf("Saved certificate content doesn't match")
	}
}

func TestCheckCertificateExpiry(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewACMEManager("test@example.com")
	manager.SetCertDir(tmpDir)

	// 创建一个简单的测试证书文件
	// 注意：这只是一个格式测试，不是真正的证书
	testCert := `-----BEGIN CERTIFICATE-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4f3lnUqTG1GVCKxfYoVx
test certificate content for testing
-----END CERTIFICATE-----`

	certPath := filepath.Join(tmpDir, "test.crt")
	err := os.WriteFile(certPath, []byte(testCert), 0644)
	if err != nil {
		t.Fatalf("Failed to write test certificate: %v", err)
	}

	// 测试解析（预期会失败，因为不是真正的证书）
	_, err = manager.checkCertificateExpiry(certPath, 30*24*time.Hour)
	if err == nil {
		t.Error("Expected error when parsing invalid certificate, got nil")
	}
}

func TestACMEUser(t *testing.T) {
	email := "test@example.com"
	user := &ACMEUser{
		Email: email,
	}

	if user.GetEmail() != email {
		t.Errorf("Expected email %s, got %s", email, user.GetEmail())
	}

	if user.GetRegistration() != nil {
		t.Error("Expected nil registration for new user")
	}

	if user.GetPrivateKey() != nil {
		t.Error("Expected nil private key for new user")
	}
}