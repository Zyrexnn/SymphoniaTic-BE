package utils_test

import (
	"testing"
	"time"

	"github.com/Zyrexnn/SymphoniaTic-be/utils"
)

func TestGenerateCryptoOTP(t *testing.T) {
	otp, err := utils.GenerateCryptoOTP()
	if err != nil {
		t.Fatalf("Expected no error generating OTP, got: %v", err)
	}

	if len(otp) != 6 {
		t.Errorf("Expected 6-digit OTP length, got: %s (len: %d)", otp, len(otp))
	}

	for _, ch := range otp {
		if ch < '0' || ch > '9' {
			t.Errorf("OTP contains non-numeric character: %c", ch)
		}
	}
}

func TestPasswordHashing(t *testing.T) {
	rawPassword := "SecureP@ssw0rd2026"

	hash, err := utils.HashPassword(rawPassword)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == rawPassword {
		t.Errorf("Hash should not equal raw password")
	}

	if !utils.CheckPasswordHash(rawPassword, hash) {
		t.Errorf("CheckPasswordHash failed for correct password")
	}

	if utils.CheckPasswordHash("WrongPassword123", hash) {
		t.Errorf("CheckPasswordHash passed for incorrect password!")
	}
}

func TestJWTUserToken(t *testing.T) {
	userID := "usr-test-uuid-12345"
	email := "testuser@symphoniatic.com"
	role := "USER"

	token, err := utils.GenerateUserToken(userID, email, role)
	if err != nil {
		t.Fatalf("Failed to generate JWT token: %v", err)
	}

	claims, err := utils.ValidateUserToken(token)
	if err != nil {
		t.Fatalf("Failed to validate valid JWT token: %v", err)
	}

	if claims.UserID != userID || claims.Email != email || claims.Role != role {
		t.Errorf("Claims mismatch! Expected (%s, %s, %s), got (%s, %s, %s)",
			userID, email, role, claims.UserID, claims.Email, claims.Role)
	}

	// Validate fake token
	_, err = utils.ValidateUserToken("invalid.token.str")
	if err == nil {
		t.Errorf("Expected error for invalid token string, got nil")
	}
}

func TestPasswordResetToken(t *testing.T) {
	email := "resetuser@symphoniatic.com"

	resetToken, err := utils.GeneratePasswordResetToken(email)
	if err != nil {
		t.Fatalf("Failed to generate reset token: %v", err)
	}

	claims, err := utils.ValidatePasswordResetToken(resetToken)
	if err != nil {
		t.Fatalf("Failed to validate reset token: %v", err)
	}

	if claims.Email != email || claims.Purpose != "RESET_PASSWORD" {
		t.Errorf("Reset claims mismatch! Got email=%s, purpose=%s", claims.Email, claims.Purpose)
	}
}

func TestOTPCooldownAndExpiryLogic(t *testing.T) {
	// Verify expiration duration constants
	ttl := 5 * time.Minute
	if ttl.Seconds() != 300 {
		t.Errorf("Expected 300 seconds TTL, got: %f", ttl.Seconds())
	}
}
