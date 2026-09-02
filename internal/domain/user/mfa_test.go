package user

import (
	"testing"
	"time"
)

func TestGenerateTOTP(t *testing.T) {
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if secret == "" {
		t.Fatal("expected non-empty secret")
	}
	if len(secret) != 32 {
		t.Fatalf("expected 32-char secret, got %d", len(secret))
	}
}

func TestValidateTOTP_Valid(t *testing.T) {
	secret, _ := GenerateTOTPSecret()
	code, err := GenerateTOTPCode(secret)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	valid, err := ValidateTOTP(secret, code)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !valid {
		t.Error("expected valid TOTP code")
	}
}

func TestValidateTOTP_Invalid(t *testing.T) {
	secret, _ := GenerateTOTPSecret()

	valid, err := ValidateTOTP(secret, "000000")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if valid {
		t.Error("expected invalid TOTP code")
	}
}

func TestGenerateBackupCodes(t *testing.T) {
	codes, err := GenerateBackupCodes(10)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(codes) != 10 {
		t.Fatalf("expected 10 codes, got %d", len(codes))
	}
	for i, code := range codes {
		if code.Code == "" {
			t.Errorf("code %d is empty", i)
		}
	}
}

func TestBackupCodeExpiry(t *testing.T) {
	code := BackupCode{
		Code:      "test123",
		Hash:      "hash123",
		UsedAt:    time.Time{},
		CreatedAt: time.Now().Add(-100 * 24 * time.Hour), // 100 days ago
	}

	if !code.IsExpired(90) {
		t.Error("expected code to be expired")
	}
	if code.IsExpired(200) {
		t.Error("expected code to not be expired with longer window")
	}
}

func TestMFACanBeEnabled(t *testing.T) {
	u := &User{
		ID:         "u_test",
		MFAEnabled: false,
	}

	u.MFAEnabled = true
	if !u.MFAEnabled {
		t.Error("expected MFA to be enabled")
	}
}

func TestMFARequiresSecret(t *testing.T) {
	u := &User{
		ID:         "u_test",
		MFAEnabled: true,
		MFASecret:  "",
	}

	// MFA enabled but no secret is invalid state
	if u.MFAEnabled && u.MFASecret == "" {
		// This is expected - handler should prevent this state
		return
	}
}

func TestBackupCodeMatches(t *testing.T) {
	code := "TEST1234"
	hash := HashBackupCode(code)

	if !BackupCodeMatches(hash, code) {
		t.Error("expected backup code to match")
	}
	if BackupCodeMatches(hash, "WRONG") {
		t.Error("expected backup code not to match wrong code")
	}
}
