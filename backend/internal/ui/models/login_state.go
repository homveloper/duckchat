package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// LoginFormState manages login form presentation and validation state
type LoginFormState struct {
	Username         string            `json:"username"`
	Password         string            `json:"password,omitempty"` // Omit in JSON for security
	IsSubmitting     bool              `json:"isSubmitting"`
	ValidationErrors map[string]string `json:"validationErrors"`
	GeneralError     string            `json:"generalError"`
	CSRFToken        string            `json:"csrfToken"`
	RedirectURL      string            `json:"redirectUrl,omitempty"`
	IsGuestMode      bool              `json:"isGuestMode"`
}

// NewLoginFormState creates a new login form state
func NewLoginFormState(csrfToken string) *LoginFormState {
	return &LoginFormState{
		Username:         "",
		Password:         "",
		IsSubmitting:     false,
		ValidationErrors: make(map[string]string),
		GeneralError:     "",
		CSRFToken:        csrfToken,
		RedirectURL:      "/rooms",
		IsGuestMode:      true, // Default to guest mode
	}
}

// LoginFormStateWithError creates a login form state with an error
func LoginFormStateWithError(csrfToken, generalError string) *LoginFormState {
	state := NewLoginFormState(csrfToken)
	state.GeneralError = generalError
	return state
}

// Validation constants
const (
	MinUsernameLength = 3
	MaxUsernameLength = 50
	MinPasswordLength = 6
	MaxPasswordLength = 128 // Reasonable max for password input
)

var (
	// Username validation regex - alphanumeric + underscore only
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

	// Common validation errors
	ErrUsernameRequired    = errors.New("username is required")
	ErrUsernameTooShort    = fmt.Errorf("username must be at least %d characters", MinUsernameLength)
	ErrUsernameTooLong     = fmt.Errorf("username must be no more than %d characters", MaxUsernameLength)
	ErrUsernameInvalidChar = errors.New("username can only contain letters, numbers, and underscores")
	ErrPasswordRequired    = errors.New("password is required")
	ErrPasswordTooShort    = fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	ErrPasswordTooLong     = fmt.Errorf("password must be no more than %d characters", MaxPasswordLength)
	ErrInvalidCredentials  = errors.New("invalid username or password")
	ErrCSRFTokenInvalid    = errors.New("invalid CSRF token")
	ErrTooManyAttempts     = errors.New("too many login attempts, please try again later")
)

// ValidateInput validates the login form input
func (s *LoginFormState) ValidateInput() error {
	// Clear previous validation errors
	s.ValidationErrors = make(map[string]string)

	hasErrors := false

	// Validate username/nickname
	if s.IsGuestMode {
		if err := s.validateNickname(); err != nil {
			s.ValidationErrors["nickname"] = err.Error()
			hasErrors = true
		}
	} else {
		if err := s.validateUsername(); err != nil {
			s.ValidationErrors["username"] = err.Error()
			hasErrors = true
		}
	}

	// Validate password (only for non-guest mode)
	if !s.IsGuestMode {
		if err := s.validatePassword(); err != nil {
			s.ValidationErrors["password"] = err.Error()
			hasErrors = true
		}
	}

	// Validate CSRF token
	if err := s.validateCSRFToken(); err != nil {
		s.ValidationErrors["csrf_token"] = err.Error()
		hasErrors = true
	}

	if hasErrors {
		return errors.New("validation failed")
	}

	return nil
}

// validateUsername validates the username field
func (s *LoginFormState) validateUsername() error {
	username := strings.TrimSpace(s.Username)

	if username == "" {
		return ErrUsernameRequired
	}

	if len(username) < MinUsernameLength {
		return ErrUsernameTooShort
	}

	if len(username) > MaxUsernameLength {
		return ErrUsernameTooLong
	}

	if !usernameRegex.MatchString(username) {
		return ErrUsernameInvalidChar
	}

	return nil
}

// validateNickname validates the nickname field for guest mode
func (s *LoginFormState) validateNickname() error {
	nickname := strings.TrimSpace(s.Username)

	if nickname == "" {
		return errors.New("nickname is required")
	}

	if len(nickname) < 2 {
		return errors.New("nickname must be at least 2 characters")
	}

	if len(nickname) > 30 {
		return errors.New("nickname must be no more than 30 characters")
	}

	// For nicknames, allow more flexible character set including Korean, spaces, etc.
	// Just check for basic safety (no control characters)
	for _, r := range nickname {
		if r < 32 || r == 127 { // Control characters
			return errors.New("nickname contains invalid characters")
		}
	}

	return nil
}

// validatePassword validates the password field
func (s *LoginFormState) validatePassword() error {
	if s.Password == "" {
		return ErrPasswordRequired
	}

	if len(s.Password) < MinPasswordLength {
		return ErrPasswordTooShort
	}

	if len(s.Password) > MaxPasswordLength {
		return ErrPasswordTooLong
	}

	return nil
}

// validateCSRFToken validates the CSRF token
func (s *LoginFormState) validateCSRFToken() error {
	if s.CSRFToken == "" {
		return ErrCSRFTokenRequired
	}

	// In real implementation, this would validate against stored token
	// For now, just check it's not empty and has reasonable length
	if len(s.CSRFToken) < 16 {
		return ErrCSRFTokenInvalid
	}

	return nil
}

// SetSubmitting sets the submission state
func (s *LoginFormState) SetSubmitting(submitting bool) {
	s.IsSubmitting = submitting
}

// SetGeneralError sets a general error message
func (s *LoginFormState) SetGeneralError(err error) {
	if err != nil {
		s.GeneralError = err.Error()
	} else {
		s.GeneralError = ""
	}
}

// SetFieldError sets a field-specific error
func (s *LoginFormState) SetFieldError(field, message string) {
	if s.ValidationErrors == nil {
		s.ValidationErrors = make(map[string]string)
	}
	s.ValidationErrors[field] = message
}

// ClearErrors clears all errors
func (s *LoginFormState) ClearErrors() {
	s.ValidationErrors = make(map[string]string)
	s.GeneralError = ""
}

// HasErrors returns true if there are any errors
func (s *LoginFormState) HasErrors() bool {
	return s.GeneralError != "" || len(s.ValidationErrors) > 0
}

// HasFieldError returns true if a specific field has an error
func (s *LoginFormState) HasFieldError(field string) bool {
	_, exists := s.ValidationErrors[field]
	return exists
}

// GetFieldError returns the error for a specific field
func (s *LoginFormState) GetFieldError(field string) string {
	return s.ValidationErrors[field]
}

// SanitizeInput sanitizes input to prevent XSS attacks
func (s *LoginFormState) SanitizeInput() {
	// Trim whitespace and basic sanitization
	s.Username = strings.TrimSpace(s.Username)
	// Note: Password should not be trimmed as spaces might be intentional

	// Remove any HTML tags from username (basic XSS prevention)
	s.Username = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(s.Username, "")
}

// ToTemplateData converts the state to data suitable for template rendering
func (s *LoginFormState) ToTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"Username":         s.Username,
		"IsSubmitting":     s.IsSubmitting,
		"ValidationErrors": s.ValidationErrors,
		"GeneralError":     s.GeneralError,
		"CSRFToken":        s.CSRFToken,
		"RedirectURL":      s.RedirectURL,
		"HasErrors":        s.HasErrors(),
	}
}

// LoginAttemptTracker tracks login attempts for rate limiting
type LoginAttemptTracker struct {
	attempts    map[string][]time.Time // IP -> attempts timestamps
	maxAttempts int
	timeWindow  time.Duration
}

// NewLoginAttemptTracker creates a new attempt tracker
func NewLoginAttemptTracker() *LoginAttemptTracker {
	return &LoginAttemptTracker{
		attempts:    make(map[string][]time.Time),
		maxAttempts: 5,               // Max 5 attempts
		timeWindow:  5 * time.Minute, // Within 5 minutes
	}
}

// IsRateLimited checks if an IP is rate limited
func (t *LoginAttemptTracker) IsRateLimited(ip string) bool {
	now := time.Now()
	attempts := t.attempts[ip]

	// Clean old attempts outside time window
	validAttempts := make([]time.Time, 0)
	for _, attempt := range attempts {
		if now.Sub(attempt) < t.timeWindow {
			validAttempts = append(validAttempts, attempt)
		}
	}
	t.attempts[ip] = validAttempts

	return len(validAttempts) >= t.maxAttempts
}

// RecordAttempt records a failed login attempt
func (t *LoginAttemptTracker) RecordAttempt(ip string) {
	now := time.Now()
	t.attempts[ip] = append(t.attempts[ip], now)
}

// GetRetryAfter returns how long to wait before next attempt
func (t *LoginAttemptTracker) GetRetryAfter(ip string) time.Duration {
	attempts := t.attempts[ip]
	if len(attempts) == 0 {
		return 0
	}

	// Find oldest attempt within time window
	now := time.Now()
	oldestAttempt := attempts[0]
	for _, attempt := range attempts {
		if now.Sub(attempt) < t.timeWindow && attempt.Before(oldestAttempt) {
			oldestAttempt = attempt
		}
	}

	elapsed := now.Sub(oldestAttempt)
	if elapsed >= t.timeWindow {
		return 0
	}

	return t.timeWindow - elapsed
}

// ClearAttempts clears attempts for an IP (e.g., after successful login)
func (t *LoginAttemptTracker) ClearAttempts(ip string) {
	delete(t.attempts, ip)
}
