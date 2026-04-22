package httputil

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

var usernameRegexp = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if utf8.RuneCountInString(email) > 255 {
		return fmt.Errorf("email must not exceed 255 characters")
	}

	addr, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email format")
	}
	if addr.Address != email {
		return fmt.Errorf("invalid email format")
	}

	atIdx := strings.LastIndex(email, "@")
	if atIdx < 0 {
		return fmt.Errorf("invalid email format")
	}
	local := email[:atIdx]
	domain := email[atIdx+1:]

	if utf8.RuneCountInString(local) > 64 {
		return fmt.Errorf("email local part must not exceed 64 characters")
	}
	if strings.HasPrefix(domain, "[") {
		return fmt.Errorf("IP address literals are not allowed in email")
	}

	if err := validateEmailDomain(domain); err != nil {
		return err
	}
	return nil
}

func validateEmailDomain(domain string) error {
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("invalid email domain")
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return fmt.Errorf("email domain must have a TLD")
	}

	tld := labels[len(labels)-1]
	if utf8.RuneCountInString(tld) < 2 {
		return fmt.Errorf("email TLD must be at least 2 characters")
	}

	for _, label := range labels {
		if label == "" {
			return fmt.Errorf("invalid email domain")
		}
		if utf8.RuneCountInString(label) > 63 {
			return fmt.Errorf("email domain label must not exceed 63 characters")
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("email domain label must not start or end with a hyphen")
		}
	}
	return nil
}

func ValidatePassword(password string) error {
	if utf8.RuneCountInString(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	return nil
}

func ValidateUsername(username string) error {
	if utf8.RuneCountInString(username) < 3 || utf8.RuneCountInString(username) > 50 {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}
	if !usernameRegexp.MatchString(username) {
		return fmt.Errorf("username must contain only letters, digits and underscores")
	}
	return nil
}
