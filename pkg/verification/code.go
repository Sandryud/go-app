package verification

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// GenerateNumericCode генерирует криптографически стойкий числовой код заданной длины.
func GenerateNumericCode(length int) (string, error) {
	const digits = "0123456789"

	if length <= 0 {
		return "", fmt.Errorf("length must be positive")
	}

	code := make([]byte, length)
	max := big.NewInt(int64(len(digits)))

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("failed to generate random digit: %w", err)
		}
		code[i] = digits[n.Int64()]
	}

	return string(code), nil
}
