package api

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// ValidatePassword validates a password according to security requirements:
// - Minimum 8 characters
// - At least one uppercase letter
// - At least one lowercase letter
// - At least one number
// - At least one special character/symbol
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	var errors []string

	if !hasUpper {
		errors = append(errors, "at least one uppercase letter")
	}
	if !hasLower {
		errors = append(errors, "at least one lowercase letter")
	}
	if !hasNumber {
		errors = append(errors, "at least one number")
	}
	if !hasSpecial {
		errors = append(errors, "at least one special character")
	}

	if len(errors) > 0 {
		return fmt.Errorf("password must contain: %s", strings.Join(errors, ", "))
	}

	return nil
}

// IsValidEmail validates email format using regex
func IsValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}
