// backend/internal/validation/validation_test.go

package validation

import (
	"testing"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  error
	}{
		{"Valid username", "john_doe123", nil},
		{"Too short", "jo", ErrUsernameTooShort},
		{"Too long", "thisusernameiswaytooooooooooooooooooooooooooooooooooooooooolong", ErrUsernameTooLong},
		{"Empty", "", ErrUsernameInvalid},
		{"Invalid characters", "user@name", ErrUsernameFormat},
		{"Valid with hyphen", "john-doe", nil},
		{"Valid with dot", "john.doe", nil},
		{"Valid with underscore", "john_doe", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUsername(tt.username)
			if err != tt.wantErr {
				t.Errorf("ValidateUsername() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{"Valid email", "test@example.com", nil},
		{"Empty email", "", ErrEmailInvalid},
		{"No @ symbol", "testexample.com", ErrEmailInvalid},
		{"No domain", "test@", ErrEmailInvalid},
		{"No TLD", "test@example", ErrEmailInvalid},
		{"Valid with plus", "test+tag@example.com", nil},
		{"Valid with dot", "test.name@example.com", nil},
		{"Valid with subdomain", "test@sub.example.com", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if err != tt.wantErr {
				t.Errorf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		username string
		wantErr  error
	}{
		{"Valid password", "Test1234!", "", nil},
		{"Too short", "Test12!", "", ErrPasswordTooShort},
		{"No uppercase", "test1234!", "", ErrPasswordNoUppercase},
		{"No lowercase", "TEST1234!", "", ErrPasswordNoLowercase},
		{"No number", "Testabcd!", "", ErrPasswordNoNumber},
		{"No special char", "Test1234", "", ErrPasswordNoSpecial},
		{"Common password", "Password123!", "", ErrPasswordCommonWord},
		{"Contains username", "TestUser123!", "user", ErrPasswordContainsUser},
		{"Complex valid", "C0mpl3x!P@ssw0rd", "", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, tt.username)
			if err != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLoginRequest(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		wantErr  bool
	}{
		{"Valid credentials", "validuser", "password", false},
		{"Empty username", "", "password", true},
		{"Empty password", "validuser", "", true},
		{"Both empty", "", "", true},
		{"Short username", "us", "password", true},
		{"Invalid username chars", "user@name", "password", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLoginRequest(tt.username, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLoginRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRegistrationRequest(t *testing.T) {
	tests := []struct {
		name        string
		username    string
		email       string
		password    string
		displayName string
		wantErr     bool
	}{
		{
			"Valid registration",
			"validuser",
			"valid@example.com",
			"Valid123!",
			"Valid User",
			false,
		},
		{
			"Invalid username",
			"u",
			"valid@example.com",
			"Valid123!",
			"Valid User",
			true,
		},
		{
			"Invalid email",
			"validuser",
			"invalid-email",
			"Valid123!",
			"Valid User",
			true,
		},
		{
			"Weak password",
			"validuser",
			"valid@example.com",
			"weak",
			"Valid User",
			true,
		},
		{
			"Empty display name",
			"validuser",
			"valid@example.com",
			"Valid123!",
			"",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistrationRequest(tt.username, tt.email, tt.password, tt.displayName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRegistrationRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
