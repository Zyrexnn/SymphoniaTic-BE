package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const orderCodeCharset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

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

// GenerateCryptoAlphanumericCode generates a cryptographically secure uppercase alphanumeric code without ambiguous characters (0/O, 1/I).
func GenerateCryptoAlphanumericCode(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("panjang kode harus lebih besar dari nol")
	}
	charsetLen := big.NewInt(int64(len(orderCodeCharset)))
	out := make([]byte, length)
	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("gagal menghasilkan karakter acak kriptografis: %w", err)
		}
		out[i] = orderCodeCharset[idx.Int64()]
	}
	return string(out), nil
}
