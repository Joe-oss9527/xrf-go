package config

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	testCertCache     *TestCertificate
	testCertCacheOnce sync.Once
)

type TestCertificate struct {
	CertFile string
	KeyFile  string
	TempDir  string
}

// GetTestCertificate 获取或生成测试证书 - 仿照Xray-core的MustGenerate
func GetTestCertificate() (*TestCertificate, error) {
	var err error
	testCertCacheOnce.Do(func() {
		testCertCache, err = generateTestCertificate()
	})
	return testCertCache, err
}

func generateTestCertificate() (*TestCertificate, error) {
	// 使用固定的测试证书目录，避免多次创建
	tempDir := filepath.Join(os.TempDir(), "xrf-test-certs")
	os.RemoveAll(tempDir) // 清理旧证书
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, err
	}

	// 生成私钥
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}

	// 证书模板 - 参考Xray-core测试证书
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"XRF-Go Test"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:              []string{"localhost"},
	}

	// 生成自签名证书
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}

	certFile := filepath.Join(tempDir, "test.crt")
	keyFile := filepath.Join(tempDir, "test.key")

	// 写证书文件
	certOut, err := os.Create(certFile)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}

	// 写私钥文件
	keyOut, err := os.Create(keyFile)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}
	defer keyOut.Close()

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privDER}); err != nil {
		os.RemoveAll(tempDir)
		return nil, err
	}

	return &TestCertificate{
		CertFile: certFile,
		KeyFile:  keyFile,
		TempDir:  tempDir,
	}, nil
}

// CleanupTestCertificate 清理测试证书
func CleanupTestCertificate() error {
	if testCertCache != nil && testCertCache.TempDir != "" {
		err := os.RemoveAll(testCertCache.TempDir)
		testCertCache = nil // 重置缓存
		return err
	}
	return nil
}

// IsTestEnvironment 检测测试环境
func IsTestEnvironment() bool {
	// 检测go test环境的多种方式
	for _, arg := range os.Args {
		if strings.Contains(arg, ".test") ||
			strings.Contains(arg, "go-build") {
			return true
		}
	}
	return os.Getenv("XRF_TEST_MODE") == "1"
}
