package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// 生成测试证书和私钥
func generateTestCertificate(t *testing.T) (certPEM, keyPEM []byte) {
	// 生成私钥
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// 创建证书模板
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "test.example.com",
			Organization: []string{"Test Company"},
		},
		DNSNames:              []string{"test.example.com", "*.test.example.com"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// 生成证书
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// 编码证书为PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// 编码私钥为PEM
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}

	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKeyDER,
	})

	return certPEM, keyPEM
}

func TestTLSFileManager(t *testing.T) {
	// 创建临时目录
	tempDir := "/tmp/xrf-tls-test"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	// 创建TLS管理器
	tlsManager := NewTLSFileManager(tempDir)

	// 测试初始化
	t.Run("Initialize", func(t *testing.T) {
		err := tlsManager.Initialize()
		if err != nil {
			t.Fatalf("Initialize failed: %v", err)
		}

		// 检查目录是否创建
		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			t.Error("TLS directory was not created")
		}
	})

	// 生成测试证书
	certPEM, keyPEM := generateTestCertificate(t)

	// 创建临时证书文件
	certPath := filepath.Join(tempDir, "test-cert.pem")
	keyPath := filepath.Join(tempDir, "test-key.pem")

	if err := os.WriteFile(certPath, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write test certificate: %v", err)
	}

	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		t.Fatalf("Failed to write test private key: %v", err)
	}

	// 测试添加证书
	t.Run("AddCertificate", func(t *testing.T) {
		certificate, err := tlsManager.AddCertificate("test", certPath, keyPath)
		if err != nil {
			t.Fatalf("AddCertificate failed: %v", err)
		}

		if certificate.Name != "test" {
			t.Errorf("Expected certificate name 'test', got '%s'", certificate.Name)
		}

		if !certificate.Valid {
			t.Errorf("Certificate should be valid, got invalid: %s", certificate.ErrorMessage)
		}

		if certificate.Info.CommonName != "test.example.com" {
			t.Errorf("Expected CommonName 'test.example.com', got '%s'", certificate.Info.CommonName)
		}

		// 检查文件是否复制到正确位置
		expectedCertPath := filepath.Join(tempDir, "test.crt")
		expectedKeyPath := filepath.Join(tempDir, "test.key")

		if _, err := os.Stat(expectedCertPath); os.IsNotExist(err) {
			t.Error("Certificate file was not created in TLS directory")
		}

		if _, err := os.Stat(expectedKeyPath); os.IsNotExist(err) {
			t.Error("Private key file was not created in TLS directory")
		}
	})

	// 测试列出证书
	t.Run("ListCertificates", func(t *testing.T) {
		certificates, err := tlsManager.ListCertificates()
		if err != nil {
			t.Fatalf("ListCertificates failed: %v", err)
		}

		if len(certificates) != 1 {
			t.Errorf("Expected 1 certificate, got %d", len(certificates))
		}

		if certificates[0].Name != "test" {
			t.Errorf("Expected certificate name 'test', got '%s'", certificates[0].Name)
		}

		if !certificates[0].Valid {
			t.Error("Certificate should be valid")
		}
	})

	// 测试获取特定证书
	t.Run("GetCertificate", func(t *testing.T) {
		certificate, err := tlsManager.GetCertificate("test")
		if err != nil {
			t.Fatalf("GetCertificate failed: %v", err)
		}

		if certificate.Name != "test" {
			t.Errorf("Expected certificate name 'test', got '%s'", certificate.Name)
		}

		if !certificate.Valid {
			t.Error("Certificate should be valid")
		}

		// 检查证书信息
		if certificate.Info.CommonName != "test.example.com" {
			t.Errorf("Expected CommonName 'test.example.com', got '%s'", certificate.Info.CommonName)
		}

		if len(certificate.Info.SubjectAltNames) != 2 {
			t.Errorf("Expected 2 SAN entries, got %d", len(certificate.Info.SubjectAltNames))
		}

		if certificate.Info.IsExpired {
			t.Error("Certificate should not be expired")
		}
	})

	// 测试证书过期检查
	t.Run("CheckCertificateExpiry", func(t *testing.T) {
		// 检查30天内过期的证书（应该没有）
		expiring, err := tlsManager.CheckCertificateExpiry(30)
		if err != nil {
			t.Fatalf("CheckCertificateExpiry failed: %v", err)
		}

		if len(expiring) != 0 {
			t.Errorf("Expected 0 expiring certificates, got %d", len(expiring))
		}

		// 检查365天内过期的证书（应该有1个）
		expiring, err = tlsManager.CheckCertificateExpiry(365)
		if err != nil {
			t.Fatalf("CheckCertificateExpiry failed: %v", err)
		}

		if len(expiring) != 1 {
			t.Errorf("Expected 1 expiring certificate, got %d", len(expiring))
		}
	})

	// 测试删除证书
	t.Run("RemoveCertificate", func(t *testing.T) {
		err := tlsManager.RemoveCertificate("test")
		if err != nil {
			t.Fatalf("RemoveCertificate failed: %v", err)
		}

		// 验证文件已被删除
		expectedCertPath := filepath.Join(tempDir, "test.crt")
		expectedKeyPath := filepath.Join(tempDir, "test.key")

		if _, err := os.Stat(expectedCertPath); !os.IsNotExist(err) {
			t.Error("Certificate file should be deleted")
		}

		if _, err := os.Stat(expectedKeyPath); !os.IsNotExist(err) {
			t.Error("Private key file should be deleted")
		}

		// 验证证书列表为空
		certificates, err := tlsManager.ListCertificates()
		if err != nil {
			t.Fatalf("ListCertificates failed: %v", err)
		}

		if len(certificates) != 0 {
			t.Errorf("Expected 0 certificates after removal, got %d", len(certificates))
		}
	})
}

func TestCertificateValidation(t *testing.T) {
	tempDir := "/tmp/xrf-tls-validation-test"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	tlsManager := NewTLSFileManager(tempDir)
	tlsManager.Initialize()

	t.Run("ValidCertificate", func(t *testing.T) {
		certPEM, keyPEM := generateTestCertificate(t)

		// 创建临时文件
		certPath := filepath.Join(tempDir, "valid-cert.pem")
		keyPath := filepath.Join(tempDir, "valid-key.pem")

		os.WriteFile(certPath, certPEM, 0644)
		os.WriteFile(keyPath, keyPEM, 0600)

		// 验证证书
		certInfo, err := tlsManager.ValidateCertificateFiles(certPath, keyPath)
		if err != nil {
			t.Fatalf("Valid certificate validation failed: %v", err)
		}

		if certInfo.CommonName != "test.example.com" {
			t.Errorf("Expected CommonName 'test.example.com', got '%s'", certInfo.CommonName)
		}
	})

	t.Run("InvalidCertificate", func(t *testing.T) {
		// 创建无效的证书文件
		invalidCertPath := filepath.Join(tempDir, "invalid-cert.pem")
		invalidKeyPath := filepath.Join(tempDir, "invalid-key.pem")

		os.WriteFile(invalidCertPath, []byte("not a certificate"), 0644)
		os.WriteFile(invalidKeyPath, []byte("not a key"), 0600)

		// 验证应该失败
		_, err := tlsManager.ValidateCertificateFiles(invalidCertPath, invalidKeyPath)
		if err == nil {
			t.Error("Invalid certificate should fail validation")
		}
	})

	t.Run("NonexistentFiles", func(t *testing.T) {
		nonexistentCert := filepath.Join(tempDir, "nonexistent-cert.pem")
		nonexistentKey := filepath.Join(tempDir, "nonexistent-key.pem")

		_, err := tlsManager.ValidateCertificateFiles(nonexistentCert, nonexistentKey)
		if err == nil {
			t.Error("Nonexistent certificate files should fail validation")
		}
	})
}

func TestErrorHandling(t *testing.T) {
	tempDir := "/tmp/xrf-tls-error-test"
	os.RemoveAll(tempDir)
	defer os.RemoveAll(tempDir)

	tlsManager := NewTLSFileManager(tempDir)

	t.Run("GetNonexistentCertificate", func(t *testing.T) {
		_, err := tlsManager.GetCertificate("nonexistent")
		if err == nil {
			t.Error("Getting nonexistent certificate should return error")
		}
	})

	t.Run("RemoveNonexistentCertificate", func(t *testing.T) {
		// 删除不存在的证书应该成功（幂等操作）
		err := tlsManager.RemoveCertificate("nonexistent")
		if err != nil {
			t.Errorf("Removing nonexistent certificate should not fail: %v", err)
		}
	})

	t.Run("AddCertificateWithInvalidFiles", func(t *testing.T) {
		tlsManager.Initialize()

		// 创建无效文件
		invalidCertPath := filepath.Join(tempDir, "invalid.crt")
		invalidKeyPath := filepath.Join(tempDir, "invalid.key")

		os.WriteFile(invalidCertPath, []byte("invalid"), 0644)
		os.WriteFile(invalidKeyPath, []byte("invalid"), 0600)

		_, err := tlsManager.AddCertificate("invalid", invalidCertPath, invalidKeyPath)
		if err == nil {
			t.Error("Adding invalid certificate should return error")
		}
	})
}
