package shop

import (
	"crypto/rand"
	"fmt"
)

// Helper function to generate a redeem code
func generateRedeemCode() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const codeLength = 16
	b := make([]byte, codeLength)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	// Format: XXXX-XXXX-XXXX-XXXX
	return fmt.Sprintf("%s-%s-%s-%s", string(b[0:4]), string(b[4:8]), string(b[8:12]), string(b[12:16]))
}
