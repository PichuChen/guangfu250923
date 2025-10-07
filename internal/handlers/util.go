package handlers

import (
	"crypto/rand"
	"math/big"
	"strconv"
)

// parsePositiveInt parses a query parameter into an int with bounds and default.
// If invalid or out of range it falls back to defaultValue.
func parsePositiveInt(raw string, defaultValue, min, max int) int {
	if raw == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// GeneratePin returns a numeric PIN of given length using crypto/rand.
func GeneratePin(length int) string {
	if length <= 0 {
		length = 6
	}
	const digits = "123456789"
	res := make([]byte, length)
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			// fallback: deterministic but unlikely; just use '1'
			res[i] = '1'
			continue
		}
		res[i] = digits[n.Int64()]
	}
	return string(res)
}

// isValidPin6 validates that s is exactly 6 digits.
func isValidPin6(s *string) bool {
	if s == nil || len(*s) != 6 {
		return false
	}
	for i := 0; i < 6; i++ {
		if (*s)[i] < '1' || (*s)[i] > '9' {
			return false
		}
	}
	return true
}
