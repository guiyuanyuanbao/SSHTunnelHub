package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var (
	secretKey     []byte
	secretKeyOnce sync.Once
)

// InitSecretKey initializes or loads a persistent 32-byte AES key.
func InitSecretKey(dataDir string) error {
	var initErr error
	secretKeyOnce.Do(func() {
		envKey := os.Getenv("APP_SECRET_KEY")
		if len(envKey) >= 32 {
			secretKey = []byte(envKey[:32])
			return
		}

		keyFilePath := filepath.Join(dataDir, ".secret.key")
		if data, err := os.ReadFile(keyFilePath); err == nil && len(data) == 32 {
			secretKey = data
			return
		}

		// Generate a new 32-byte random key
		key := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			initErr = err
			return
		}

		if err := os.MkdirAll(dataDir, 0700); err != nil {
			initErr = err
			return
		}

		if err := os.WriteFile(keyFilePath, key, 0600); err != nil {
			initErr = err
			return
		}

		secretKey = key
	})
	return initErr
}

// Encrypt encrypts plain text using AES-256-GCM and returns base64 string.
func Encrypt(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}
	if len(secretKey) != 32 {
		return "", errors.New("secret key not initialized or invalid length")
	}

	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

// Decrypt decrypts base64 encoded ciphertext using AES-256-GCM.
func Decrypt(cipherTextBase64 string) (string, error) {
	if cipherTextBase64 == "" {
		return "", nil
	}
	if len(secretKey) != 32 {
		return "", errors.New("secret key not initialized or invalid length")
	}

	data, err := base64.StdEncoding.DecodeString(cipherTextBase64)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(secretKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("malformed ciphertext")
	}

	nonce, cipherText := data[:nonceSize], data[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
