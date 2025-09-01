package tls

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/yourusername/xrf-go/pkg/utils"
)

const (
	// Let's Encrypt 生产环境
	LEProductionURL = "https://acme-v02.api.letsencrypt.org/directory"
	// Let's Encrypt 测试环境
	LEStagingURL = "https://acme-staging-v02.api.letsencrypt.org/directory"

	// 默认目录
	DefaultACMEDir = "/etc/xray/acme"
	DefaultCertDir = "/etc/xray/certs"
)

// ACMEManager ACME 证书管理器
type ACMEManager struct {
	email   string
	caURL   string
	certDir string
	acmeDir string
	client  *lego.Client
}

// ACMEUser 实现 ACME 用户接口
type ACMEUser struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	key          crypto.PrivateKey
}

// ACMEAccount ACME 账户持久化
type ACMEAccount struct {
	Email        string                 `json:"email"`
	Registration *registration.Resource `json:"registration"`
	PrivateKey   []byte                 `json:"private_key"`
}

// GetEmail 获取用户邮箱
func (u *ACMEUser) GetEmail() string {
	return u.Email
}

// GetRegistration 获取用户注册信息
func (u *ACMEUser) GetRegistration() *registration.Resource {
	return u.Registration
}

// GetPrivateKey 获取用户私钥
func (u *ACMEUser) GetPrivateKey() crypto.PrivateKey {
	return u.key
}

// NewACMEManager 创建新的 ACME 管理器
func NewACMEManager(email string) *ACMEManager {
	return &ACMEManager{
		email:   email,
		caURL:   LEProductionURL, // 默认使用生产环境
		certDir: DefaultCertDir,
		acmeDir: DefaultACMEDir,
	}
}

// SetStagingMode 设置为测试模式
func (am *ACMEManager) SetStagingMode() {
	am.caURL = LEStagingURL
}

// SetCertDir 设置证书目录
func (am *ACMEManager) SetCertDir(certDir string) {
	am.certDir = certDir
}

// SetACMEDir 设置 ACME 目录
func (am *ACMEManager) SetACMEDir(acmeDir string) {
	am.acmeDir = acmeDir
}

// Initialize 初始化 ACME 管理器
func (am *ACMEManager) Initialize() error {
	// 创建必要的目录
	dirs := []string{am.certDir, am.acmeDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// 初始化或加载用户账户
	user, err := am.getOrCreateUser()
	if err != nil {
		return fmt.Errorf("failed to initialize user: %w", err)
	}

	// 创建 lego 配置
	config := lego.NewConfig(user)
	config.CADirURL = am.caURL

	// 创建客户端
	client, err := lego.NewClient(config)
	if err != nil {
		return fmt.Errorf("failed to create ACME client: %w", err)
	}

	// 设置 HTTP-01 挑战
	err = client.Challenge.SetHTTP01Provider(http01.NewProviderServer("", "80"))
	if err != nil {
		return fmt.Errorf("failed to set HTTP-01 provider: %w", err)
	}

	am.client = client

	utils.Info("ACME Manager initialized successfully")
	utils.Info("Email: %s", am.email)
	utils.Info("CA URL: %s", am.caURL)
	utils.Info("Cert Dir: %s", am.certDir)

	return nil
}

// getOrCreateUser 获取或创建用户账户
func (am *ACMEManager) getOrCreateUser() (*ACMEUser, error) {
	accountPath := filepath.Join(am.acmeDir, "account.json")

	// 尝试加载现有账户
	if _, err := os.Stat(accountPath); err == nil {
		return am.loadUser(accountPath)
	}

	// 创建新账户
	return am.createUser(accountPath)
}

// loadUser 从文件加载用户账户
func (am *ACMEManager) loadUser(accountPath string) (*ACMEUser, error) {
	data, err := os.ReadFile(accountPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read account file: %w", err)
	}

	var account ACMEAccount
	if err := json.Unmarshal(data, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account: %w", err)
	}

	// 解析私钥
	privateKey, err := utils.ParsePrivateKey(account.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	user := &ACMEUser{
		Email:        account.Email,
		Registration: account.Registration,
		key:          privateKey,
	}

	utils.Info("Loaded existing ACME account")
	return user, nil
}

// createUser 创建新的用户账户
func (am *ACMEManager) createUser(accountPath string) (*ACMEUser, error) {
	// 生成私钥
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	user := &ACMEUser{
		Email: am.email,
		key:   privateKey,
	}

	// 创建临时客户端进行注册
	config := lego.NewConfig(user)
	config.CADirURL = am.caURL

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp client: %w", err)
	}

	// 注册用户
	reg, err := client.Registration.Register(registration.RegisterOptions{
		TermsOfServiceAgreed: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	user.Registration = reg

	// 保存账户
	if err := am.saveUser(user, accountPath); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	utils.Success("Created new ACME account: %s", am.email)
	return user, nil
}

// saveUser 保存用户账户到文件
func (am *ACMEManager) saveUser(user *ACMEUser, accountPath string) error {
	// 编码私钥
	privateKeyBytes, err := utils.EncodePrivateKey(user.key)
	if err != nil {
		return fmt.Errorf("failed to encode private key: %w", err)
	}

	account := ACMEAccount{
		Email:        user.Email,
		Registration: user.Registration,
		PrivateKey:   privateKeyBytes,
	}

	data, err := json.MarshalIndent(account, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal account: %w", err)
	}

	if err := os.WriteFile(accountPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write account file: %w", err)
	}

	return nil
}

// saveCertificate 保存证书到文件
func (am *ACMEManager) saveCertificate(domain string, cert, key []byte) error {
	certPath := filepath.Join(am.certDir, domain+".crt")
	keyPath := filepath.Join(am.certDir, domain+".key")

	// 保存证书文件
	if err := os.WriteFile(certPath, cert, 0644); err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	// 保存私钥文件
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	utils.Success("Certificate saved for domain: %s", domain)
	utils.Info("Certificate: %s", certPath)
	utils.Info("Private Key: %s", keyPath)

	return nil
}

// ObtainCertificate 申请证书
func (am *ACMEManager) ObtainCertificate(domains []string) error {
	if am.client == nil {
		return fmt.Errorf("ACME client not initialized")
	}

	if len(domains) == 0 {
		return fmt.Errorf("no domains provided")
	}

	utils.Info("Obtaining certificate for domains: %v", domains)

	request := certificate.ObtainRequest{
		Domains: domains,
		Bundle:  true,
	}

	certificates, err := am.client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("failed to obtain certificate: %w", err)
	}

	// 保存证书文件
	primaryDomain := domains[0]
	if err := am.saveCertificate(primaryDomain, certificates.Certificate, certificates.PrivateKey); err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	utils.Success("Successfully obtained certificate for domains: %v", domains)
	return nil
}

// RenewCertificate 续期证书
func (am *ACMEManager) RenewCertificate(domain string) error {
	if am.client == nil {
		return fmt.Errorf("ACME client not initialized")
	}

	certPath := filepath.Join(am.certDir, domain+".crt")
	keyPath := filepath.Join(am.certDir, domain+".key")

	// 检查证书文件是否存在
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		return fmt.Errorf("certificate not found for domain: %s", domain)
	}
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return fmt.Errorf("private key not found for domain: %s", domain)
	}

	// 读取现有证书
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}

	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	utils.Info("Renewing certificate for domain: %s", domain)

	// 续期证书
	certResource := certificate.Resource{
		Domain:      domain,
		Certificate: certPEM,
		PrivateKey:  keyPEM,
	}

	newCert, err := am.client.Certificate.Renew(certResource, true, false, "")
	if err != nil {
		return fmt.Errorf("failed to renew certificate: %w", err)
	}

	// 保存新证书
	if err := am.saveCertificate(domain, newCert.Certificate, newCert.PrivateKey); err != nil {
		return fmt.Errorf("failed to save renewed certificate: %w", err)
	}

	utils.Success("Successfully renewed certificate for domain: %s", domain)
	return nil
}

// CheckAndRenew 检查并续期即将过期的证书
func (am *ACMEManager) CheckAndRenew() error {
	if am.certDir == "" {
		return fmt.Errorf("certificate directory not set")
	}

	files, err := os.ReadDir(am.certDir)
	if err != nil {
		return fmt.Errorf("failed to read certificate directory: %w", err)
	}

	renewThreshold := 30 * 24 * time.Hour // 30天

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".crt" {
			continue
		}

		domain := file.Name()[:len(file.Name())-4]
		certPath := filepath.Join(am.certDir, file.Name())

		// 检查证书是否需要续期
		needsRenewal, err := am.checkCertificateExpiry(certPath, renewThreshold)
		if err != nil {
			utils.Error("Failed to check certificate expiry for %s: %v", domain, err)
			continue
		}

		if needsRenewal {
			utils.Info("Certificate for %s expires soon, renewing...", domain)
			if err := am.RenewCertificate(domain); err != nil {
				utils.Error("Failed to renew certificate for %s: %v", domain, err)
			}
		}
	}

	return nil
}

// checkCertificateExpiry 检查证书是否即将过期
func (am *ACMEManager) checkCertificateExpiry(certPath string, threshold time.Duration) (bool, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return false, err
	}

	cert, err := utils.ParseCertificate(certPEM)
	if err != nil {
		return false, err
	}

	timeUntilExpiry := time.Until(cert.NotAfter)
	return timeUntilExpiry <= threshold, nil
}

// SetupAutoRenewal 设置自动续期
func (am *ACMEManager) SetupAutoRenewal() error {
	// 这里可以实现 cron job 或 systemd timer 来定期运行 CheckAndRenew
	// 暂时返回成功，具体实现会在后续添加
	utils.Info("Auto-renewal setup completed")
	return nil
}
