package twofa

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		name     string
		phone    string
		expected string
	}{
		{
			name:     "standard US phone",
			phone:    "+12345678901",
			expected: "+1234567****",
		},
		{
			name:     "short phone (4 chars)",
			phone:    "1234",
			expected: "****",
		},
		{
			name:     "very short phone (3 chars)",
			phone:    "123",
			expected: "****",
		},
		{
			name:     "empty phone",
			phone:    "",
			expected: "****",
		},
		{
			name:     "exactly 5 chars",
			phone:    "12345",
			expected: "1****",
		},
		{
			name:     "international format",
			phone:    "+993612345678",
			expected: "+99361234****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskPhone(tt.phone)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDeviceName(t *testing.T) {
	tests := []struct {
		name      string
		userAgent string
		expected  string
	}{
		{
			name:      "iPhone",
			userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 15_0 like Mac OS X)",
			expected:  "iPhone",
		},
		{
			name:      "Android",
			userAgent: "Mozilla/5.0 (Linux; Android 12; Pixel 6)",
			expected:  "Android Device",
		},
		{
			name:      "Chrome on Mac",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120.0.0.0",
			expected:  "Chrome on Mac",
		},
		{
			name:      "Chrome on Windows",
			userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/120.0.0.0",
			expected:  "Chrome on Windows",
		},
		{
			name:      "Chrome on Linux",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0.0.0",
			expected:  "Chrome Browser",
		},
		{
			name:      "Safari",
			userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Safari/605.1.15",
			expected:  "Safari Browser",
		},
		{
			name:      "Firefox",
			userAgent: "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/120.0",
			expected:  "Firefox Browser",
		},
		{
			name:      "unknown user agent",
			userAgent: "some-custom-agent/1.0",
			expected:  "Unknown Device",
		},
		{
			name:      "empty user agent",
			userAgent: "",
			expected:  "Unknown Device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDeviceName(tt.userAgent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateOTP_Length(t *testing.T) {
	svc := &Service{}

	for i := 0; i < 50; i++ {
		otp := svc.generateOTP()
		assert.Len(t, otp, OTPLength, "OTP should be %d digits long", OTPLength)
	}
}

func TestGenerateOTP_OnlyDigits(t *testing.T) {
	svc := &Service{}

	for i := 0; i < 50; i++ {
		otp := svc.generateOTP()
		for _, ch := range otp {
			assert.True(t, ch >= '0' && ch <= '9',
				"OTP should only contain digits, got '%c'", ch)
		}
	}
}

func TestGenerateOTP_Randomness(t *testing.T) {
	svc := &Service{}

	otps := make(map[string]bool)
	for i := 0; i < 100; i++ {
		otp := svc.generateOTP()
		otps[otp] = true
	}

	// With 100 random 6-digit OTPs, we should have a significant number of unique values
	assert.Greater(t, len(otps), 50, "expected more than 50 unique OTPs from 100 generations")
}

func TestGenerateSecureToken_Length(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name   string
		length int
	}{
		{"16 bytes", 16},
		{"32 bytes", 32},
		{"64 bytes", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := svc.generateSecureToken(tt.length)
			assert.NotEmpty(t, token)
			// Base32 encoding of N bytes produces ceil(N*8/5) characters
			assert.Greater(t, len(token), tt.length, "encoded token should be longer than raw bytes")
		})
	}
}

func TestGenerateSecureToken_Uniqueness(t *testing.T) {
	svc := &Service{}

	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token := svc.generateSecureToken(32)
		tokens[token] = true
	}

	assert.Equal(t, 100, len(tokens), "all tokens should be unique")
}

func TestGenerateBackupCode_Format(t *testing.T) {
	svc := &Service{}

	for i := 0; i < 50; i++ {
		code := svc.generateBackupCode()

		// Should be in format XXXX-XXXX (9 characters with dash)
		assert.Len(t, code, BackupCodeLength+1, "backup code should be %d chars + 1 dash", BackupCodeLength)
		assert.Equal(t, '-', rune(code[4]), "backup code should have dash at position 4")

		// Check allowed characters (no confusing chars like I, O, 0, 1)
		allowed := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
		for j, ch := range code {
			if j == 4 {
				continue // Skip the dash
			}
			assert.Contains(t, allowed, string(ch),
				"character '%c' at position %d is not in allowed set", ch, j)
		}
	}
}

func TestGenerateBackupCode_Uniqueness(t *testing.T) {
	svc := &Service{}

	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code := svc.generateBackupCode()
		codes[code] = true
	}

	assert.Equal(t, 100, len(codes), "all backup codes should be unique")
}

func TestGenerateBackupCodes_Count(t *testing.T) {
	svc := &Service{}

	codes, hashes, err := svc.generateBackupCodes()

	assert.NoError(t, err)
	assert.Len(t, codes, BackupCodesCount)
	assert.Len(t, hashes, BackupCodesCount)
}

func TestGenerateBackupCodes_HashesAreBcrypt(t *testing.T) {
	svc := &Service{}

	_, hashes, err := svc.generateBackupCodes()

	assert.NoError(t, err)
	for _, hash := range hashes {
		// bcrypt hashes start with "$2a$" or "$2b$"
		assert.True(t, len(hash) > 0, "hash should not be empty")
		assert.True(t, hash[0] == '$', "bcrypt hash should start with $")
	}
}

func TestGenerateBackupCodes_CodesAndHashesDiffer(t *testing.T) {
	svc := &Service{}

	codes, hashes, err := svc.generateBackupCodes()

	assert.NoError(t, err)
	for i := range codes {
		assert.NotEqual(t, codes[i], hashes[i], "code and hash should not be the same")
	}
}

func TestConstants(t *testing.T) {
	assert.Equal(t, 6, OTPLength)
	assert.Equal(t, 10, OTPExpiryMinutes)
	assert.Equal(t, 5, OTPMaxAttempts)
	assert.Equal(t, 5, PendingLoginExpiryMin)
	assert.Equal(t, 30, TrustedDeviceDays)
	assert.Equal(t, 10, BackupCodesCount)
	assert.Equal(t, 8, BackupCodeLength)
	assert.Equal(t, 32, TOTPSecretLength)
}

func TestParseDeviceName_CaseInsensitive(t *testing.T) {
	// Ensure parsing is case-insensitive
	assert.Equal(t, "iPhone", parseDeviceName("IPHONE DEVICE"))
	assert.Equal(t, "Android Device", parseDeviceName("ANDROID 12"))
	assert.Equal(t, "Chrome Browser", parseDeviceName("CHROME/120"))
	assert.Equal(t, "Safari Browser", parseDeviceName("SAFARI/17"))
	assert.Equal(t, "Firefox Browser", parseDeviceName("FIREFOX/120"))
}
