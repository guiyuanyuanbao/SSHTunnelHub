package auth

import (
	"os"
	"testing"
	"time"
)

func TestAuthManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "auth_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	masterKey := []byte("01234567890123456789012345678901")

	// 1. Initialized test without env or file
	mgr := NewAuthManager(tempDir, "", masterKey)
	if mgr.IsInitialized() {
		t.Fatalf("expected uninitialized initially")
	}

	// 2. Set key via init
	if err := mgr.SetKey("MySecret123"); err != nil {
		t.Fatalf("failed to set key: %v", err)
	}
	if !mgr.IsInitialized() {
		t.Fatalf("expected initialized after setting key")
	}

	// 3. Verify key
	if !mgr.VerifyKey("MySecret123") {
		t.Fatalf("expected key verification to succeed")
	}
	if mgr.VerifyKey("WrongKey") {
		t.Fatalf("expected wrong key to fail")
	}

	// 4. Generate and validate token
	token, err := mgr.GenerateToken(1 * time.Hour)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	if !mgr.ValidateToken(token) {
		t.Fatalf("expected token validation to succeed")
	}
	if mgr.ValidateToken("invalid.token.structure") {
		t.Fatalf("expected invalid token to fail")
	}

	// 5. Override via environment variable
	envMgr := NewAuthManager(tempDir, "EnvSecret999", masterKey)
	if !envMgr.IsFromEnv() {
		t.Fatalf("expected is from env")
	}
	if !envMgr.VerifyKey("EnvSecret999") {
		t.Fatalf("expected env key to be valid")
	}
	// The old key should be invalid now because env takes precedence
	if envMgr.VerifyKey("MySecret123") {
		t.Fatalf("expected old file key to be overridden by env key")
	}
	// Old token should be invalid because signing key changes with effective secret
	if envMgr.ValidateToken(token) {
		t.Fatalf("expected old token to be invalidated when key changes")
	}

	// 6. Rate limiting test
	ip := "127.0.0.1"
	for i := 0; i < 4; i++ {
		mgr.RecordFailedAttempt(ip)
		allowed, _ := mgr.CheckRateLimit(ip)
		if !allowed {
			t.Fatalf("expected allowed on %d-th attempt", i+1)
		}
	}
	// 5th failure triggers lockout
	mgr.RecordFailedAttempt(ip)
	allowed, remaining := mgr.CheckRateLimit(ip)
	if allowed {
		t.Fatalf("expected locked out on 5th attempt")
	}
	if remaining <= 0 {
		t.Fatalf("expected remaining time > 0")
	}
	mgr.ResetRateLimit(ip)
	allowedAfterReset, _ := mgr.CheckRateLimit(ip)
	if !allowedAfterReset {
		t.Fatalf("expected allowed after reset")
	}
}
