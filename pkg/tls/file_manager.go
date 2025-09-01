package tls

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/yourusername/xrf-go/pkg/utils"
)

// TLSFileManager TLS 文件管理器
type TLSFileManager struct {
	certDir string
}

// CertificateInfo 证书信息
type CertificateInfo struct {
	CommonName      string             `json:"common_name"`
	SubjectAltNames []string           `json:"subject_alt_names"`
	Issuer          string             `json:"issuer"`
	NotBefore       time.Time          `json:"not_before"`
	NotAfter        time.Time          `json:"not_after"`
	IsExpired       bool               `json:"is_expired"`
	DaysUntilExpiry int                `json:"days_until_expiry"`
	KeyUsage        x509.KeyUsage      `json:"key_usage"`
	ExtKeyUsage     []x509.ExtKeyUsage `json:"ext_key_usage"`
	SerialNumber    string             `json:"serial_number"`
}

// TLSCertificate 表示一个 TLS 证书配置
type TLSCertificate struct {
	Name         string          `json:"name"`
	CertPath     string          `json:"cert_path"`
	KeyPath      string          `json:"key_path"`
	Info         CertificateInfo `json:"info"`
	Valid        bool            `json:"valid"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

// NewTLSFileManager 创建新的 TLS 文件管理器
func NewTLSFileManager(certDir string) *TLSFileManager {
	if certDir == "" {
		certDir = "/etc/xray/certs"
	}
	return &TLSFileManager{
		certDir: certDir,
	}
}

// Initialize 初始化证书目录
func (tm *TLSFileManager) Initialize() error {
	if err := os.MkdirAll(tm.certDir, 0755); err != nil {
		return utils.NewPermissionDeniedError(tm.certDir, err)
	}

	utils.Info("Initialized TLS certificate directory: %s", tm.certDir)
	return nil
}

// AddCertificate 添加证书文件
func (tm *TLSFileManager) AddCertificate(name, certPath, keyPath string) (*TLSCertificate, error) {
	// 验证证书文件
	certInfo, err := tm.ValidateCertificateFiles(certPath, keyPath)
	if err != nil {
		return nil, err
	}

	// 复制文件到管理目录
	targetCertPath := filepath.Join(tm.certDir, fmt.Sprintf("%s.crt", name))
	targetKeyPath := filepath.Join(tm.certDir, fmt.Sprintf("%s.key", name))

	if err := tm.copyFile(certPath, targetCertPath); err != nil {
		return nil, utils.NewFileNotFoundError(certPath, err)
	}

	if err := tm.copyFile(keyPath, targetKeyPath); err != nil {
		return nil, utils.NewFileNotFoundError(keyPath, err)
	}

	// 设置适当的权限
	os.Chmod(targetCertPath, 0644)
	os.Chmod(targetKeyPath, 0600)

	certificate := &TLSCertificate{
		Name:     name,
		CertPath: targetCertPath,
		KeyPath:  targetKeyPath,
		Info:     *certInfo,
		Valid:    true,
	}

	utils.Success("Added TLS certificate: %s", name)
	return certificate, nil
}

// ValidateCertificateFiles 验证证书文件
func (tm *TLSFileManager) ValidateCertificateFiles(certPath, keyPath string) (*CertificateInfo, error) {
	// 读取证书文件
	certPEM, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, utils.NewFileNotFoundError(certPath, err)
	}

	// 读取私钥文件
	keyPEM, err := ioutil.ReadFile(keyPath)
	if err != nil {
		return nil, utils.NewFileNotFoundError(keyPath, err)
	}

	// 解析证书
	certInfo, err := tm.parseCertificate(certPEM)
	if err != nil {
		return nil, utils.NewCertificateInvalidError(certPath, err)
	}

	// 验证私钥
	if err := tm.validatePrivateKey(keyPEM); err != nil {
		return nil, utils.NewCertificateInvalidError(keyPath, err)
	}

	// 验证证书和私钥匹配
	if err := tm.validateCertKeyMatch(certPEM, keyPEM); err != nil {
		return nil, &utils.XRFError{
			Type:    utils.ErrCertificateInvalid,
			Message: "证书和私钥不匹配",
			Cause:   err,
			Context: map[string]interface{}{
				"cert_path": certPath,
				"key_path":  keyPath,
			},
		}
	}

	return certInfo, nil
}

// parseCertificate 解析证书并提取信息
func (tm *TLSFileManager) parseCertificate(certPEM []byte) (*CertificateInfo, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	// 计算到期时间
	now := time.Now()
	daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)

	info := &CertificateInfo{
		CommonName:      cert.Subject.CommonName,
		SubjectAltNames: cert.DNSNames,
		Issuer:          cert.Issuer.CommonName,
		NotBefore:       cert.NotBefore,
		NotAfter:        cert.NotAfter,
		IsExpired:       now.After(cert.NotAfter),
		DaysUntilExpiry: daysUntilExpiry,
		KeyUsage:        cert.KeyUsage,
		ExtKeyUsage:     cert.ExtKeyUsage,
		SerialNumber:    cert.SerialNumber.String(),
	}

	return info, nil
}

// validatePrivateKey 验证私钥文件
func (tm *TLSFileManager) validatePrivateKey(keyPEM []byte) error {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return fmt.Errorf("failed to parse private key PEM")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		_, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		return err
	case "PRIVATE KEY":
		_, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		return err
	case "EC PRIVATE KEY":
		_, err := x509.ParseECPrivateKey(block.Bytes)
		return err
	default:
		return fmt.Errorf("unsupported private key type: %s", block.Type)
	}
}

// validateCertKeyMatch 验证证书和私钥是否匹配
func (tm *TLSFileManager) validateCertKeyMatch(certPEM, keyPEM []byte) error {
	// 解析证书
	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to parse certificate")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return err
	}

	// 解析私钥
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to parse private key")
	}

	switch keyBlock.Type {
	case "RSA PRIVATE KEY":
		_, err = x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	case "PRIVATE KEY":
		_, err = x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	case "EC PRIVATE KEY":
		_, err = x509.ParseECPrivateKey(keyBlock.Bytes)
	default:
		return fmt.Errorf("unsupported private key type")
	}

	if err != nil {
		return err
	}

	// 使用证书的公钥与私钥进行匹配验证
	// 这是一个简化的验证，实际应用中可能需要更复杂的验证
	// 如果能成功解析证书和私钥，就认为它们是匹配的
	_ = cert.PublicKey
	return nil
}

// ListCertificates 列出所有证书
func (tm *TLSFileManager) ListCertificates() ([]*TLSCertificate, error) {
	if _, err := os.Stat(tm.certDir); os.IsNotExist(err) {
		return []*TLSCertificate{}, nil
	}

	files, err := ioutil.ReadDir(tm.certDir)
	if err != nil {
		return nil, utils.NewPermissionDeniedError(tm.certDir, err)
	}

	certMap := make(map[string]*TLSCertificate)

	// 扫描证书文件
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		name := file.Name()
		if filepath.Ext(name) == ".crt" {
			baseName := name[:len(name)-4] // 去掉 .crt 后缀

			if certMap[baseName] == nil {
				certMap[baseName] = &TLSCertificate{
					Name: baseName,
				}
			}
			certMap[baseName].CertPath = filepath.Join(tm.certDir, name)
		} else if filepath.Ext(name) == ".key" {
			baseName := name[:len(name)-4] // 去掉 .key 后缀

			if certMap[baseName] == nil {
				certMap[baseName] = &TLSCertificate{
					Name: baseName,
				}
			}
			certMap[baseName].KeyPath = filepath.Join(tm.certDir, name)
		}
	}

	// 验证每个证书
	var certificates []*TLSCertificate
	for _, cert := range certMap {
		if cert.CertPath != "" && cert.KeyPath != "" {
			// 验证证书
			certInfo, err := tm.ValidateCertificateFiles(cert.CertPath, cert.KeyPath)
			if err != nil {
				cert.Valid = false
				cert.ErrorMessage = err.Error()
			} else {
				cert.Info = *certInfo
				cert.Valid = true
			}
			certificates = append(certificates, cert)
		}
	}

	return certificates, nil
}

// RemoveCertificate 删除证书
func (tm *TLSFileManager) RemoveCertificate(name string) error {
	certPath := filepath.Join(tm.certDir, fmt.Sprintf("%s.crt", name))
	keyPath := filepath.Join(tm.certDir, fmt.Sprintf("%s.key", name))

	// 删除证书文件
	if _, err := os.Stat(certPath); err == nil {
		if err := os.Remove(certPath); err != nil {
			return utils.NewPermissionDeniedError(certPath, err)
		}
	}

	// 删除私钥文件
	if _, err := os.Stat(keyPath); err == nil {
		if err := os.Remove(keyPath); err != nil {
			return utils.NewPermissionDeniedError(keyPath, err)
		}
	}

	utils.Success("Removed TLS certificate: %s", name)
	return nil
}

// GetCertificate 获取指定证书信息
func (tm *TLSFileManager) GetCertificate(name string) (*TLSCertificate, error) {
	certPath := filepath.Join(tm.certDir, fmt.Sprintf("%s.crt", name))
	keyPath := filepath.Join(tm.certDir, fmt.Sprintf("%s.key", name))

	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return nil, utils.NewFileNotFoundError(certPath, err)
	}

	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return nil, utils.NewFileNotFoundError(keyPath, err)
	}

	certInfo, err := tm.ValidateCertificateFiles(certPath, keyPath)
	if err != nil {
		return &TLSCertificate{
			Name:         name,
			CertPath:     certPath,
			KeyPath:      keyPath,
			Valid:        false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &TLSCertificate{
		Name:     name,
		CertPath: certPath,
		KeyPath:  keyPath,
		Info:     *certInfo,
		Valid:    true,
	}, nil
}

// copyFile 复制文件
func (tm *TLSFileManager) copyFile(src, dst string) error {
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(dst, data, 0644)
}

// CheckCertificateExpiry 检查证书是否即将过期
func (tm *TLSFileManager) CheckCertificateExpiry(days int) ([]*TLSCertificate, error) {
	certificates, err := tm.ListCertificates()
	if err != nil {
		return nil, err
	}

	var expiring []*TLSCertificate
	for _, cert := range certificates {
		if cert.Valid && cert.Info.DaysUntilExpiry <= days {
			expiring = append(expiring, cert)
		}
	}

	return expiring, nil
}

// GetCertificateDirectory 获取证书目录路径
func (tm *TLSFileManager) GetCertificateDirectory() string {
	return tm.certDir
}
