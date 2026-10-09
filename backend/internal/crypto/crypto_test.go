package crypto

import (
	"os"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "sshtunnel_crypto_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := InitSecretKey(tempDir); err != nil {
		t.Fatalf("Failed to init secret key: %v", err)
	}

	testCases := []string{
		"password123!@#",
		"-----BEGIN RSA PRIVATE KEY-----\nMIIEowIBAAKCAQEAz...",
		"my-secret-passphrase",
		"123456",
		"Special Chinese characters 测试密钥 123",
	}

	for _, original := range testCases {
		encrypted, err := Encrypt(original)
		if err != nil {
			t.Fatalf("Failed to encrypt '%s': %v", original, err)
		}
		if encrypted == original {
			t.Fatalf("Encrypted text should not equal original text")
		}

		decrypted, err := Decrypt(encrypted)
		if err != nil {
			t.Fatalf("Failed to decrypt: %v", err)
		}
		if decrypted != original {
			t.Fatalf("Expected '%s', got '%s'", original, decrypted)
		}
	}
}
