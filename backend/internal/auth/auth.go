package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// TokenClaims represents the JWT payload claims.
type TokenClaims struct {
	Iss string `json:"iss"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

type rateLimitEntry struct {
	failures  int
	firstFail time.Time
	lockUntil time.Time
}

// AuthManager manages the authentication lifecycle, secret key precedence,
// JWT token generation/validation, and anti-brute-force rate limiting.
type AuthManager struct {
	mu          sync.RWMutex
	dataDir     string
	hashFile    string
	envKey      string
	masterKey   []byte
	limits      map[string]*rateLimitEntry
	limitsMutex sync.Mutex
}

// NewAuthManager creates and initializes a new AuthManager.
func NewAuthManager(dataDir string, envKey string, masterKey []byte) *AuthManager {
	mgr := &AuthManager{
		dataDir:   dataDir,
		hashFile:  filepath.Join(dataDir, ".auth.hash"),
		envKey:    strings.TrimSpace(envKey),
		masterKey: masterKey,
		limits:    make(map[string]*rateLimitEntry),
	}
	return mgr
}

// IsFromEnv returns true if the secret key is defined via environment variable.
func (a *AuthManager) IsFromEnv() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.envKey != ""
}

// IsInitialized returns true if an auth key is configured either via env or local file.
func (a *AuthManager) IsInitialized() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.envKey != "" {
		return true
	}
	if data, err := os.ReadFile(a.hashFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		return true
	}
	return false
}

// VerifyKey checks if the provided key matches the active secret key.
func (a *AuthManager) VerifyKey(inputKey string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	input := strings.TrimSpace(inputKey)
	if input == "" {
		return false
	}

	// 1. Environment variable has highest priority
	if a.envKey != "" {
		return subtle.ConstantTimeCompare([]byte(input), []byte(a.envKey)) == 1
	}

	// 2. Check local persistent hash
	data, err := os.ReadFile(a.hashFile)
	if err != nil {
		return false
	}
	hash := strings.TrimSpace(string(data))
	if hash == "" {
		return false
	}

	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(input)) == nil
}

// SetKey stores a new secret key into the local persistent hash file.
// If envKey is configured, modification is forbidden.
func (a *AuthManager) SetKey(newKey string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.envKey != "" {
		return errors.New("cannot change secret key: HUB_AUTH_KEY is controlled by environment variable")
	}

	trimmed := strings.TrimSpace(newKey)
	if len(trimmed) < 4 {
		return errors.New("secret key must be at least 4 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(trimmed), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash secret key: %w", err)
	}

	if err := os.MkdirAll(a.dataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	if err := os.WriteFile(a.hashFile, hash, 0600); err != nil {
		return fmt.Errorf("failed to write hash file: %w", err)
	}

	return nil
}

// getSigningKey derives a cryptographic key based on master key and current effective secret.
func (a *AuthManager) getSigningKey() []byte {
	var currentSecret []byte
	if a.envKey != "" {
		currentSecret = []byte(a.envKey)
	} else if data, err := os.ReadFile(a.hashFile); err == nil {
		currentSecret = data
	}

	h := hmac.New(sha256.New, a.masterKey)
	h.Write(currentSecret)
	h.Write([]byte("sshtunnelhub-jwt-signing-v1"))
	return h.Sum(nil)
}

// GenerateToken creates an HMAC-SHA256 signed JWT token valid for the specified duration.
func (a *AuthManager) GenerateToken(duration time.Duration) (string, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)

	now := time.Now()
	claims := TokenClaims{
		Iss: "sshtunnelhub",
		Iat: now.Unix(),
		Exp: now.Add(duration).Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	sig := a.signHMAC([]byte(signingInput), a.getSigningKey())
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return signingInput + "." + sigB64, nil
}

// ValidateToken verifies that the JWT token is valid, properly signed, and not expired.
func (a *AuthManager) ValidateToken(tokenStr string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return false
	}

	headerB64, claimsB64, sigB64 := parts[0], parts[1], parts[2]
	signingInput := headerB64 + "." + claimsB64

	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return false
	}

	expectedSig := a.signHMAC([]byte(signingInput), a.getSigningKey())
	if hmac.Equal(sig, expectedSig) != true {
		return false
	}

	claimsData, err := base64.RawURLEncoding.DecodeString(claimsB64)
	if err != nil {
		return false
	}

	var claims TokenClaims
	if err := json.Unmarshal(claimsData, &claims); err != nil {
		return false
	}

	if claims.Exp <= time.Now().Unix() {
		return false
	}

	return true
}

func (a *AuthManager) signHMAC(data, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// CheckRateLimit checks if an IP is temporarily locked out.
func (a *AuthManager) CheckRateLimit(ip string) (bool, time.Duration) {
	a.limitsMutex.Lock()
	defer a.limitsMutex.Unlock()

	entry, ok := a.limits[ip]
	if !ok {
		return true, 0
	}

	now := time.Now()
	if now.Before(entry.lockUntil) {
		return false, entry.lockUntil.Sub(now)
	}

	// Reset if more than 2 minutes have passed since first failure
	if now.Sub(entry.firstFail) > 2*time.Minute {
		delete(a.limits, ip)
	}

	return true, 0
}

// RecordFailedAttempt registers a failed authentication from an IP.
func (a *AuthManager) RecordFailedAttempt(ip string) {
	a.limitsMutex.Lock()
	defer a.limitsMutex.Unlock()

	now := time.Now()
	entry, ok := a.limits[ip]
	if !ok || now.Sub(entry.firstFail) > 2*time.Minute {
		a.limits[ip] = &rateLimitEntry{
			failures:  1,
			firstFail: now,
		}
		return
	}

	entry.failures++
	if entry.failures >= 5 {
		entry.lockUntil = now.Add(1 * time.Minute)
	}
}

// ResetRateLimit clears failure records for an IP upon successful login.
func (a *AuthManager) ResetRateLimit(ip string) {
	a.limitsMutex.Lock()
	defer a.limitsMutex.Unlock()
	delete(a.limits, ip)
}
