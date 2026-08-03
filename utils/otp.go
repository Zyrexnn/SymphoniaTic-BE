package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateCryptoOTP generates a cryptographically secure 6-digit numeric OTP string.
func GenerateCryptoOTP() (string, error) {
	maxVal := big.NewInt(900000)
	n, err := rand.Int(rand.Reader, maxVal)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure random number: %w", err)
	}

	code := n.Int64() + 100000
	return fmt.Sprintf("%06d", code), nil
}
