package utils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os/exec"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/curve25519"
)

func GenerateUUID() string {
	return uuid.New().String()
}

func GeneratePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

func GenerateShadowsocks2022Key(method string) (string, error) {
	var keySize int
	switch method {
	case "2022-blake3-aes-128-gcm":
		keySize = 16
	case "2022-blake3-aes-256-gcm":
		keySize = 32
	case "2022-blake3-chacha20-poly1305":
		keySize = 32
	default:
		return "", fmt.Errorf("unsupported method: %s", method)
	}

	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(key), nil
}

func GenerateX25519KeyPair() (privateKey, publicKey string, err error) {
	var priv, pub [32]byte

	if _, err := rand.Read(priv[:]); err != nil {
		return "", "", err
	}

	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64

	curve25519.ScalarBaseMult(&pub, &priv)

	privateKey = base64.RawURLEncoding.EncodeToString(priv[:])
	publicKey = base64.RawURLEncoding.EncodeToString(pub[:])

	return privateKey, publicKey, nil
}

func GenerateShortID(length int) string {
	if length <= 0 || length > 16 {
		length = 8
	}

	b := make([]byte, length/2+1)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("0", length)
	}

	shortID := hex.EncodeToString(b)
	if len(shortID) > length {
		shortID = shortID[:length]
	}

	return shortID
}

func GenerateRandomHex(length int) string {
	bytes := make([]byte, (length+1)/2)
	if _, err := rand.Read(bytes); err != nil {
		return ""
	}
	hex := hex.EncodeToString(bytes)
	if len(hex) > length {
		hex = hex[:length]
	}
	return hex
}

func GenerateBase64Key(byteLength int) string {
	key := make([]byte, byteLength)
	if _, err := rand.Read(key); err != nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(key)
}

func ValidateUUID(u string) bool {
	_, err := uuid.Parse(u)
	return err == nil
}

func ValidateBase64(s string) bool {
	_, err := base64.StdEncoding.DecodeString(s)
	return err == nil
}

func ParseCertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

// ParsePrivateKey 解析私钥
func ParsePrivateKey(keyData []byte) (crypto.PrivateKey, error) {
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key PEM")
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		return x509.ParsePKCS8PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	default:
		return nil, fmt.Errorf("unsupported private key type: %s", block.Type)
	}
}

// EncodePrivateKey 编码私钥为 PEM 格式
func EncodePrivateKey(privateKey crypto.PrivateKey) ([]byte, error) {
	switch key := privateKey.(type) {
	case *rsa.PrivateKey:
		keyBytes := x509.MarshalPKCS1PrivateKey(key)
		block := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyBytes,
		}
		return pem.EncodeToMemory(block), nil

	case *ecdsa.PrivateKey:
		keyBytes, err := x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, err
		}
		block := &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: keyBytes,
		}
		return pem.EncodeToMemory(block), nil

	default:
		keyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return nil, err
		}
		block := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: keyBytes,
		}
		return pem.EncodeToMemory(block), nil
	}
}

// ExecuteCommand 执行系统命令
func ExecuteCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.Output()
}
