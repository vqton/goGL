package user

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// GenerateTOTPSecret generates a random 20-byte secret and returns it as a
// base32-encoded string (32 characters) suitable for authenticator apps.
func GenerateTOTPSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret), nil
}

// GenerateTOTPCode generates a 6-digit TOTP code for the given secret.
func GenerateTOTPCode(secret string) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}

	counter := time.Now().Unix() / 30
	buf := make([]byte, 8)
	buf[0] = byte(counter >> 56)
	buf[1] = byte(counter >> 48)
	buf[2] = byte(counter >> 40)
	buf[3] = byte(counter >> 32)
	buf[4] = byte(counter >> 24)
	buf[5] = byte(counter >> 16)
	buf[6] = byte(counter >> 8)
	buf[7] = byte(counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	code := int32((int32(sum[offset])&0x7f)<<24 |
		int32(sum[offset+1])<<16 |
		int32(sum[offset+2])<<8 |
		int32(sum[offset+3]))
	code = code % 1000000

	return fmt.Sprintf("%06d", code), nil
}

// ValidateTOTP validates a TOTP code against the secret.
// It checks the current time window and one window before/after for clock skew.
func ValidateTOTP(secret, code string) (bool, error) {
	for _, offset := range []int64{-1, 0, 1} {
		expected, err := generateTOTPForCounter(secret, time.Now().Unix()/30+offset)
		if err != nil {
			return false, err
		}
		if code == expected {
			return true, nil
		}
	}
	return false, nil
}

func generateTOTPForCounter(secret string, counter int64) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}

	buf := make([]byte, 8)
	buf[0] = byte(counter >> 56)
	buf[1] = byte(counter >> 48)
	buf[2] = byte(counter >> 40)
	buf[3] = byte(counter >> 32)
	buf[4] = byte(counter >> 24)
	buf[5] = byte(counter >> 16)
	buf[6] = byte(counter >> 8)
	buf[7] = byte(counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	totp := int32((int32(sum[offset])&0x7f)<<24 |
		int32(sum[offset+1])<<16 |
		int32(sum[offset+2])<<8 |
		int32(sum[offset+3]))
	totp = totp % 1000000

	return fmt.Sprintf("%06d", totp), nil
}

// BackupCode represents a one-time backup code for MFA recovery.
type BackupCode struct {
	Code      string    `json:"code"`
	Hash      string    `json:"hash"`
	UsedAt    time.Time `json:"used_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired checks if the backup code is older than the expiry days.
func (b *BackupCode) IsExpired(expiryDays int) bool {
	if b.CreatedAt.IsZero() {
		return false
	}
	return time.Since(b.CreatedAt) > time.Duration(expiryDays)*24*time.Hour
}

// GenerateBackupCodes generates n random backup codes.
func GenerateBackupCodes(n int) ([]BackupCode, error) {
	codes := make([]BackupCode, n)
	for i := 0; i < n; i++ {
		b := make([]byte, 6)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		code := strings.ToUpper(hex.EncodeToString(b))[:8]
		codes[i] = BackupCode{
			Code:      code,
			Hash:      HashBackupCode(code),
			CreatedAt: time.Now(),
		}
	}
	return codes, nil
}

// HashBackupCode creates a SHA-256 hash of a backup code.
func HashBackupCode(code string) string {
	h := sha256.Sum256([]byte(strings.ToUpper(code)))
	return hex.EncodeToString(h[:])
}

// BackupCodeMatches checks if a plaintext code matches a stored hash.
func BackupCodeMatches(hash, code string) bool {
	return hash == HashBackupCode(code)
}
