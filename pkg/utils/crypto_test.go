package utils

import (
	"strings"
	"testing"
)

// TestUUIDGeneration 测试UUID生成功能
func TestUUIDGeneration(t *testing.T) {
	t.Run("GenerateUUID", func(t *testing.T) {
		uuid := GenerateUUID()

		// 验证UUID格式
		if len(uuid) != 36 {
			t.Errorf("UUID length should be 36, got %d", len(uuid))
		}

		// 验证UUID格式 (8-4-4-4-12)
		parts := strings.Split(uuid, "-")
		if len(parts) != 5 {
			t.Errorf("UUID should have 5 parts separated by hyphens, got %d", len(parts))
		}

		expectedLengths := []int{8, 4, 4, 4, 12}
		for i, part := range parts {
			if len(part) != expectedLengths[i] {
				t.Errorf("UUID part %d should have length %d, got %d",
					i, expectedLengths[i], len(part))
			}
		}
	})

	t.Run("UniqueUUIDs", func(t *testing.T) {
		uuids := make(map[string]bool)

		// 生成100个UUID并验证唯一性
		for i := 0; i < 100; i++ {
			uuid := GenerateUUID()
			if uuids[uuid] {
				t.Errorf("Duplicate UUID generated: %s", uuid)
			}
			uuids[uuid] = true
		}
	})
}

// TestPasswordGeneration 测试密码生成功能
func TestPasswordGeneration(t *testing.T) {
	t.Run("GeneratePassword", func(t *testing.T) {
		password := GeneratePassword(16)

		if len(password) != 16 {
			t.Errorf("Password length should be 16, got %d", len(password))
		}

		// 验证密码只包含预期字符（与crypto.go中的charset一致）
		validChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
		for _, char := range password {
			if !strings.ContainsRune(validChars, char) {
				t.Errorf("Password contains invalid character: %c", char)
			}
		}
	})

	t.Run("PasswordLengthVariations", func(t *testing.T) {
		lengths := []int{8, 12, 16, 24, 32}

		for _, length := range lengths {
			password := GeneratePassword(length)
			if len(password) != length {
				t.Errorf("Password length should be %d, got %d", length, len(password))
			}
		}
	})

	t.Run("UniquePasswords", func(t *testing.T) {
		passwords := make(map[string]bool)

		// 生成50个密码并验证唯一性
		for i := 0; i < 50; i++ {
			password := GeneratePassword(16)
			if passwords[password] {
				t.Errorf("Duplicate password generated: %s", password)
			}
			passwords[password] = true
		}
	})
}

// TestShadowsocks2022Key 测试Shadowsocks 2022密钥生成
func TestShadowsocks2022Key(t *testing.T) {
	t.Run("GenerateShadowsocks2022Key", func(t *testing.T) {
		key, err := GenerateShadowsocks2022Key("2022-blake3-aes-256-gcm")
		if err != nil {
			t.Fatalf("GenerateShadowsocks2022Key failed: %v", err)
		}

		// Shadowsocks 2022密钥应该是32字节的base64编码
		// Base64编码后的长度应该是44字符 (32 * 4/3 向上取整)
		if len(key) != 44 {
			t.Errorf("SS2022 key length should be 44, got %d", len(key))
		}

		// 验证是否是有效的base64
		if !ValidateBase64(key) {
			t.Errorf("SS2022 key should be valid base64: %s", key)
		}
	})

	t.Run("UniqueShadowsocks2022Keys", func(t *testing.T) {
		keys := make(map[string]bool)

		// 生成20个密钥并验证唯一性
		for i := 0; i < 20; i++ {
			key, err := GenerateShadowsocks2022Key("2022-blake3-aes-256-gcm")
			if err != nil {
				t.Errorf("GenerateShadowsocks2022Key failed: %v", err)
				continue
			}
			if keys[key] {
				t.Errorf("Duplicate SS2022 key generated: %s", key)
			}
			keys[key] = true
		}
	})
}

// TestX25519KeyPair 测试X25519密钥对生成
func TestX25519KeyPair(t *testing.T) {
	t.Run("GenerateX25519KeyPair", func(t *testing.T) {
		privateKey, publicKey, err := GenerateX25519KeyPair()
		if err != nil {
			t.Fatalf("GenerateX25519KeyPair failed: %v", err)
		}

		// 验证私钥长度 (32字节的base64编码 = 43字符，RawURL编码)
		if len(privateKey) != 43 {
			t.Errorf("Private key length should be 43, got %d", len(privateKey))
		}

		// 验证公钥长度 (32字节的base64编码 = 43字符，RawURL编码)
		if len(publicKey) != 43 {
			t.Errorf("Public key length should be 43, got %d", len(publicKey))
		}

		// 私钥和公钥不应该相同
		if privateKey == publicKey {
			t.Error("Private key and public key should be different")
		}
	})

	t.Run("UniqueKeyPairs", func(t *testing.T) {
		privateKeys := make(map[string]bool)
		publicKeys := make(map[string]bool)

		// 生成10个密钥对并验证唯一性
		for i := 0; i < 10; i++ {
			privateKey, publicKey, err := GenerateX25519KeyPair()
			if err != nil {
				t.Errorf("GenerateX25519KeyPair failed: %v", err)
				continue
			}

			if privateKeys[privateKey] {
				t.Errorf("Duplicate private key generated: %s", privateKey)
			}
			if publicKeys[publicKey] {
				t.Errorf("Duplicate public key generated: %s", publicKey)
			}

			privateKeys[privateKey] = true
			publicKeys[publicKey] = true
		}
	})
}

func TestDeriveX25519Public(t *testing.T) {
    t.Run("Derive from private", func(t *testing.T) {
        priv, pub, err := GenerateX25519KeyPair()
        if err != nil {
            t.Fatalf("GenerateX25519KeyPair failed: %v", err)
        }
        derived, err := DeriveX25519Public(priv)
        if err != nil {
            t.Fatalf("DeriveX25519Public failed: %v", err)
        }
        if derived != pub {
            t.Errorf("Derived public key mismatch\n got: %s\nwant: %s", derived, pub)
        }
    })
}

// TestShortID 测试REALITY短ID生成
func TestShortID(t *testing.T) {
	t.Run("GenerateShortID", func(t *testing.T) {
		shortID := GenerateShortID(8)

		// REALITY短ID应该是指定长度的十六进制字符串
		if len(shortID) != 8 {
			t.Errorf("Short ID length should be 8, got %d", len(shortID))
		}

		// 验证是否是有效的十六进制
		for _, char := range shortID {
			if (char < '0' || char > '9') && (char < 'a' || char > 'f') && (char < 'A' || char > 'F') {
				t.Errorf("Short ID contains invalid hex character: %c", char)
			}
		}
	})

	t.Run("UniqueShortIDs", func(t *testing.T) {
		shortIDs := make(map[string]bool)

		// 生成50个短ID并验证唯一性
		for i := 0; i < 50; i++ {
			shortID := GenerateShortID(8)
			if shortIDs[shortID] {
				t.Errorf("Duplicate short ID generated: %s", shortID)
			}
			shortIDs[shortID] = true
		}
	})
}

// TestBase64Operations 测试Base64验证
func TestBase64Operations(t *testing.T) {
	t.Run("ValidateBase64", func(t *testing.T) {
		testData := "SGVsbG8sIFhSRi1HbyE=" // "Hello, XRF-Go!" 的base64编码

		// 验证有效的base64
		if !ValidateBase64(testData) {
			t.Error("Valid base64 string should pass validation")
		}
	})

	t.Run("InvalidBase64", func(t *testing.T) {
		invalidBase64 := "invalid-base64-string!"

		if ValidateBase64(invalidBase64) {
			t.Error("Invalid base64 string should fail validation")
		}
	})
}

// TestCryptoPerformance 测试加密操作性能
func TestCryptoPerformance(t *testing.T) {
	t.Run("UUIDGenerationPerformance", func(t *testing.T) {
		count := 1000

		for i := 0; i < count; i++ {
			GenerateUUID()
		}

		t.Logf("Generated %d UUIDs", count)
	})

	t.Run("PasswordGenerationPerformance", func(t *testing.T) {
		count := 1000

		for i := 0; i < count; i++ {
			GeneratePassword(16)
		}

		t.Logf("Generated %d passwords", count)
	})

	t.Run("X25519KeyPairPerformance", func(t *testing.T) {
		count := 100

		for i := 0; i < count; i++ {
			if _, _, err := GenerateX25519KeyPair(); err != nil {
				t.Fatal(err)
			}
		}

		t.Logf("Generated %d X25519 key pairs", count)
	})
}
