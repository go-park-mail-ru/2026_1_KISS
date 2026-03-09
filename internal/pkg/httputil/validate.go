package httputil

import (
	"fmt"
	"regexp"
)

var (
	emailRegexp    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

func ValidateEmail(email string) error {
	if !emailRegexp.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}

func ValidateUsername(username string) error {
	if len(username) < 3 || len(username) > 50 {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}
	if !usernameRegexp.MatchString(username) {
		return fmt.Errorf("username must contain only letters, digits and underscores")
	}
	return nil
}
