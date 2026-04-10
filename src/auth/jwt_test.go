// Package auth_test verifies JWT generation, validation, and claim parsing.
//
// These tests require the following environment variables (set via `make test-backend`):
//
//	JWT_SECRET    — signing key (any non-empty string works for tests)
//	REDIS_HOST    — required by config.init(); Redis does not need to be reachable
//	POSTGRES_HOST — required by config.init(); Postgres does not need to be reachable
package auth_test

import (
	"os"
	"strings"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"streamcore/src/auth"
	"streamcore/src/config"
)

// testSecret returns the JWT secret loaded by the running config, falling back
// to a default so the test file is self-documenting when run via make.
func testSecret() string {
	if s := config.Config.JwtSecret; s != "" {
		return s
	}
	return os.Getenv("JWT_SECRET")
}

// generateTestToken creates a signed token with the same method as auth.GenerateJWT.
func generateTestToken(userID, username string, expiry time.Time) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["userID"] = userID
	claims["username"] = username
	claims["exp"] = expiry.Unix()
	return token.SignedString([]byte(testSecret()))
}

// ---------------------------------------------------------------------------
// TestGenerateAndValidateJWT — happy path round-trip
// ---------------------------------------------------------------------------

func TestGenerateAndValidateJWT(t *testing.T) {
	tokenString, err := auth.GenerateJWT("42", "alice")
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}
	if tokenString == "" {
		t.Fatal("GenerateJWT returned empty token string")
	}
	if err := auth.ValidateJWTToken(tokenString); err != nil {
		t.Errorf("ValidateJWTToken on fresh token: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestParseJWTClaims — userID and userName extracted correctly
// ---------------------------------------------------------------------------

func TestParseJWTClaims(t *testing.T) {
	wantID := "99"
	wantName := "bob"

	tokenString, err := auth.GenerateJWT(wantID, wantName)
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	gotID, gotName := auth.ParseJWTToken(tokenString)
	if gotID != wantID {
		t.Errorf("userID: got %q, want %q", gotID, wantID)
	}
	if gotName != wantName {
		t.Errorf("userName: got %q, want %q", gotName, wantName)
	}
}

// ---------------------------------------------------------------------------
// TestValidateJWT_Tampered — flip a character in the signature
// ---------------------------------------------------------------------------

func TestValidateJWT_Tampered(t *testing.T) {
	tokenString, err := auth.GenerateJWT("1", "eve")
	if err != nil {
		t.Fatalf("GenerateJWT: %v", err)
	}

	// Tamper: flip the last character of the base64url signature segment.
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3-part JWT, got %d parts", len(parts))
	}
	sig := []byte(parts[2])
	sig[len(sig)-1] ^= 0x01 // flip one bit
	tampered := strings.Join([]string{parts[0], parts[1], string(sig)}, ".")

	if err := auth.ValidateJWTToken(tampered); err == nil {
		t.Error("ValidateJWTToken should reject a tampered token but returned nil error")
	}
}

// ---------------------------------------------------------------------------
// TestValidateJWT_WrongSecret — token signed with a different key
// ---------------------------------------------------------------------------

func TestValidateJWT_WrongSecret(t *testing.T) {
	// Sign directly with a different secret, bypassing auth.GenerateJWT.
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["userID"] = "1"
	claims["username"] = "mallory"
	claims["exp"] = time.Now().Add(time.Hour).Unix()

	wrongSecret := "totally-different-secret-key"
	tokenString, err := token.SignedString([]byte(wrongSecret))
	if err != nil {
		t.Fatalf("sign with wrong secret: %v", err)
	}

	if err := auth.ValidateJWTToken(tokenString); err == nil {
		t.Error("ValidateJWTToken should reject a token signed with the wrong secret")
	}
}

// ---------------------------------------------------------------------------
// TestValidateJWT_Expired — token whose exp claim is in the past
// ---------------------------------------------------------------------------

func TestValidateJWT_Expired(t *testing.T) {
	tokenString, err := generateTestToken("2", "olduser", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}

	if err := auth.ValidateJWTToken(tokenString); err == nil {
		t.Error("ValidateJWTToken should reject an expired token but returned nil error")
	}
}

// ---------------------------------------------------------------------------
// TestValidateJWT_Empty — empty string must be rejected cleanly
// ---------------------------------------------------------------------------

func TestValidateJWT_Empty(t *testing.T) {
	if err := auth.ValidateJWTToken(""); err == nil {
		t.Error("ValidateJWTToken should reject an empty token string")
	}
}
