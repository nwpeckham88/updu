package auth

import (
	"errors"
	"unicode"
	"unicode/utf8"

	"github.com/updu/updu/internal/config"
)

type passwordRequirements struct {
	minLength      int
	requireUpper   bool
	requireLower   bool
	requireDigit   bool
	requireSpecial bool
}

func ValidatePassword(password, policy string) error {
	requirements := passwordRequirementsFor(policy)
	if utf8.RuneCountInString(password) < requirements.minLength {
		return errors.New(PasswordPolicyHint(policy))
	}

	if !requirements.requireUpper && !requirements.requireLower && !requirements.requireDigit && !requirements.requireSpecial {
		return nil
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		if unicode.IsUpper(r) {
			hasUpper = true
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			hasSpecial = true
		}
	}

	if requirements.requireUpper && !hasUpper {
		return errors.New(PasswordPolicyHint(policy))
	}
	if requirements.requireLower && !hasLower {
		return errors.New(PasswordPolicyHint(policy))
	}
	if requirements.requireDigit && !hasDigit {
		return errors.New(PasswordPolicyHint(policy))
	}
	if requirements.requireSpecial && !hasSpecial {
		return errors.New(PasswordPolicyHint(policy))
	}

	return nil
}

func PasswordPolicyHint(policy string) string {
	switch config.NormalizePasswordPolicy(policy) {
	case config.PasswordPolicyStrong:
		return "password must be at least 10 characters and include uppercase, lowercase, and a number"
	case config.PasswordPolicyVerySecure:
		return "password must be at least 12 characters and include uppercase, lowercase, a number, and a special character"
	case config.PasswordPolicyOff, config.PasswordPolicyDefault:
		fallthrough
	default:
		return "password must be at least 8 characters"
	}
}

func passwordRequirementsFor(policy string) passwordRequirements {
	switch config.NormalizePasswordPolicy(policy) {
	case config.PasswordPolicyStrong:
		return passwordRequirements{
			minLength:    10,
			requireUpper: true,
			requireLower: true,
			requireDigit: true,
		}
	case config.PasswordPolicyVerySecure:
		return passwordRequirements{
			minLength:      12,
			requireUpper:   true,
			requireLower:   true,
			requireDigit:   true,
			requireSpecial: true,
		}
	case config.PasswordPolicyOff, config.PasswordPolicyDefault:
		fallthrough
	default:
		return passwordRequirements{minLength: 8}
	}
}
