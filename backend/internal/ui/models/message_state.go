package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// MessageInputState manages message composition interface state
type MessageInputState struct {
	Content            string     `json:"content"`
	IsSending          bool       `json:"isSending"`
	CharacterCount     int        `json:"characterCount"`
	ValidationError    string     `json:"validationError"`
	CSRFToken          string     `json:"csrfToken"`
	RoomID             string     `json:"roomId"`
	IsConnected        bool       `json:"isConnected"`
	IsTyping           bool       `json:"isTyping"`
	LastTypingTime     *time.Time `json:"lastTypingTime,omitempty"`
	RateLimitRemaining int        `json:"rateLimitRemaining"`
}

// TypingIndicatorState manages typing indicator display
type TypingIndicatorState struct {
	TypingUsers   []string  `json:"typingUsers"`
	LastUpdated   time.Time `json:"lastUpdated"`
	IsVisible     bool      `json:"isVisible"`
	FormattedText string    `json:"formattedText"`
}

// Message input validation constants
const (
	MaxMessageLength     = 1000
	MinMessageLength     = 1
	MaxMessagesPerMinute = 10
	TypingTimeoutSeconds = 3
	TypingThrottleMs     = 1000 // Throttle typing events to 1 per second
)

var (
	// Message validation errors
	ErrMessageRequired  = errors.New("message content is required")
	ErrMessageTooLong   = fmt.Errorf("message must be no more than %d characters", MaxMessageLength)
	ErrMessageEmpty     = errors.New("message cannot be empty or contain only whitespace")
	ErrMessageRateLimit = fmt.Errorf("rate limit exceeded: maximum %d messages per minute", MaxMessagesPerMinute)
	ErrNotConnected     = errors.New("cannot send message: not connected to chat room")
	ErrInvalidRoom      = errors.New("invalid room ID")

	// Message content validation regex
	messageContentRegex = regexp.MustCompile(`^[\s\S]*\S[\s\S]*$`) // Must contain at least one non-whitespace character
)

// NewMessageInputState creates a new message input state
func NewMessageInputState(roomID, csrfToken string) *MessageInputState {
	return &MessageInputState{
		Content:            "",
		IsSending:          false,
		CharacterCount:     0,
		ValidationError:    "",
		CSRFToken:          csrfToken,
		RoomID:             roomID,
		IsConnected:        false,
		IsTyping:           false,
		LastTypingTime:     nil,
		RateLimitRemaining: MaxMessagesPerMinute,
	}
}

// NewTypingIndicatorState creates a new typing indicator state
func NewTypingIndicatorState() *TypingIndicatorState {
	return &TypingIndicatorState{
		TypingUsers:   make([]string, 0),
		LastUpdated:   time.Now(),
		IsVisible:     false,
		FormattedText: "",
	}
}

// ValidateInput validates the message input
func (s *MessageInputState) ValidateInput() error {
	s.ValidationError = ""

	// Check connection status
	if !s.IsConnected {
		s.ValidationError = ErrNotConnected.Error()
		return ErrNotConnected
	}

	// Validate CSRF token
	if err := s.validateCSRFToken(); err != nil {
		s.ValidationError = err.Error()
		return err
	}

	// Validate room ID
	if err := s.validateRoomID(); err != nil {
		s.ValidationError = err.Error()
		return err
	}

	// Validate message content
	if err := s.validateContent(); err != nil {
		s.ValidationError = err.Error()
		return err
	}

	// Check rate limiting
	if err := s.validateRateLimit(); err != nil {
		s.ValidationError = err.Error()
		return err
	}

	return nil
}

// validateContent validates the message content
func (s *MessageInputState) validateContent() error {
	content := strings.TrimSpace(s.Content)

	if content == "" {
		return ErrMessageRequired
	}

	if len(content) < MinMessageLength {
		return ErrMessageEmpty
	}

	if len(s.Content) > MaxMessageLength {
		return ErrMessageTooLong
	}

	// Check if message contains at least one non-whitespace character
	if !messageContentRegex.MatchString(s.Content) {
		return ErrMessageEmpty
	}

	return nil
}

// validateCSRFToken validates the CSRF token
func (s *MessageInputState) validateCSRFToken() error {
	if s.CSRFToken == "" {
		return ErrCSRFTokenRequired
	}

	// Basic validation - in real implementation, validate against stored token
	if len(s.CSRFToken) < 16 {
		return ErrCSRFTokenRequired
	}

	return nil
}

// validateRoomID validates the room ID
func (s *MessageInputState) validateRoomID() error {
	if s.RoomID == "" {
		return ErrInvalidRoom
	}

	// Basic validation - room ID should be non-empty and reasonable length
	if len(s.RoomID) < 1 || len(s.RoomID) > 255 {
		return ErrInvalidRoom
	}

	return nil
}

// validateRateLimit checks if rate limit is exceeded
func (s *MessageInputState) validateRateLimit() error {
	if s.RateLimitRemaining <= 0 {
		return ErrMessageRateLimit
	}
	return nil
}

// UpdateContent updates the message content and character count
func (s *MessageInputState) UpdateContent(content string) {
	s.Content = content
	s.CharacterCount = len(content)
	s.ValidationError = ""

	// Update typing status if content changes
	s.updateTypingStatus()
}

// updateTypingStatus updates typing indicator based on content changes
func (s *MessageInputState) updateTypingStatus() {
	now := time.Now()
	hasContent := strings.TrimSpace(s.Content) != ""

	if hasContent && s.IsConnected {
		// Start typing or update typing time
		if !s.IsTyping || (s.LastTypingTime != nil && now.Sub(*s.LastTypingTime) > time.Duration(TypingThrottleMs)*time.Millisecond) {
			s.IsTyping = true
			s.LastTypingTime = &now
		}
	} else {
		// Stop typing if no content or disconnected
		s.IsTyping = false
		s.LastTypingTime = nil
	}
}

// SetSending sets the sending state
func (s *MessageInputState) SetSending(sending bool) {
	s.IsSending = sending
	if sending {
		s.ValidationError = ""
		s.IsTyping = false // Stop typing when sending
		s.LastTypingTime = nil
	}
}

// SetConnected sets the connection status
func (s *MessageInputState) SetConnected(connected bool) {
	s.IsConnected = connected
	if !connected {
		s.IsTyping = false
		s.LastTypingTime = nil
		s.ValidationError = ErrNotConnected.Error()
	} else {
		s.ValidationError = ""
	}
}

// SetValidationError sets a validation error message
func (s *MessageInputState) SetValidationError(err error) {
	if err != nil {
		s.ValidationError = err.Error()
	} else {
		s.ValidationError = ""
	}
}

// ClearContent clears the message content after sending
func (s *MessageInputState) ClearContent() {
	s.Content = ""
	s.CharacterCount = 0
	s.ValidationError = ""
	s.IsTyping = false
	s.LastTypingTime = nil
}

// DecrementRateLimit decrements the rate limit counter
func (s *MessageInputState) DecrementRateLimit() {
	if s.RateLimitRemaining > 0 {
		s.RateLimitRemaining--
	}
}

// ResetRateLimit resets the rate limit counter (called periodically)
func (s *MessageInputState) ResetRateLimit() {
	s.RateLimitRemaining = MaxMessagesPerMinute
}

// IsTypingExpired checks if typing indicator should expire
func (s *MessageInputState) IsTypingExpired() bool {
	if !s.IsTyping || s.LastTypingTime == nil {
		return false
	}
	return time.Since(*s.LastTypingTime) > time.Duration(TypingTimeoutSeconds)*time.Second
}

// ExpireTyping expires the typing indicator if needed
func (s *MessageInputState) ExpireTyping() {
	if s.IsTypingExpired() {
		s.IsTyping = false
		s.LastTypingTime = nil
	}
}

// GetRemainingCharacters returns remaining characters before limit
func (s *MessageInputState) GetRemainingCharacters() int {
	return MaxMessageLength - s.CharacterCount
}

// IsAtCharacterLimit returns true if at character limit
func (s *MessageInputState) IsAtCharacterLimit() bool {
	return s.CharacterCount >= MaxMessageLength
}

// CanSend returns true if message can be sent
func (s *MessageInputState) CanSend() bool {
	return !s.IsSending &&
		s.IsConnected &&
		s.ValidationError == "" &&
		strings.TrimSpace(s.Content) != "" &&
		s.CharacterCount <= MaxMessageLength &&
		s.RateLimitRemaining > 0
}

// HasError returns true if there is a validation error
func (s *MessageInputState) HasError() bool {
	return s.ValidationError != ""
}

// SanitizeContent sanitizes message content
func (s *MessageInputState) SanitizeContent() {
	// Remove any HTML tags to prevent XSS
	s.Content = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(s.Content, "")

	// Normalize whitespace but preserve intentional line breaks
	s.Content = regexp.MustCompile(`[ \t]+`).ReplaceAllString(s.Content, " ")

	s.CharacterCount = len(s.Content)
}

// ToTemplateData converts the state to data suitable for template rendering
func (s *MessageInputState) ToTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"Content":             s.Content,
		"IsSending":           s.IsSending,
		"CharacterCount":      s.CharacterCount,
		"MaxCharacterCount":   MaxMessageLength,
		"RemainingCharacters": s.GetRemainingCharacters(),
		"IsAtLimit":           s.IsAtCharacterLimit(),
		"ValidationError":     s.ValidationError,
		"CSRFToken":           s.CSRFToken,
		"RoomID":              s.RoomID,
		"IsConnected":         s.IsConnected,
		"IsTyping":            s.IsTyping,
		"CanSend":             s.CanSend(),
		"RateLimitRemaining":  s.RateLimitRemaining,
		"HasError":            s.ValidationError != "",
	}
}

// UpdateTypingUsers updates the typing indicator with new users
func (t *TypingIndicatorState) UpdateTypingUsers(typingUsers []string) {
	t.TypingUsers = make([]string, len(typingUsers))
	copy(t.TypingUsers, typingUsers)
	t.LastUpdated = time.Now()
	t.updateFormattedText()
	t.IsVisible = len(t.TypingUsers) > 0
}

// AddTypingUser adds a user to the typing list
func (t *TypingIndicatorState) AddTypingUser(username string) {
	// Check if user is already in the list
	for _, user := range t.TypingUsers {
		if user == username {
			return
		}
	}

	t.TypingUsers = append(t.TypingUsers, username)
	t.LastUpdated = time.Now()
	t.updateFormattedText()
	t.IsVisible = true
}

// RemoveTypingUser removes a user from the typing list
func (t *TypingIndicatorState) RemoveTypingUser(username string) {
	for i, user := range t.TypingUsers {
		if user == username {
			t.TypingUsers = append(t.TypingUsers[:i], t.TypingUsers[i+1:]...)
			break
		}
	}

	t.LastUpdated = time.Now()
	t.updateFormattedText()
	t.IsVisible = len(t.TypingUsers) > 0
}

// ClearTypingUsers clears all typing users
func (t *TypingIndicatorState) ClearTypingUsers() {
	t.TypingUsers = make([]string, 0)
	t.LastUpdated = time.Now()
	t.FormattedText = ""
	t.IsVisible = false
}

// updateFormattedText updates the formatted typing indicator text
func (t *TypingIndicatorState) updateFormattedText() {
	switch len(t.TypingUsers) {
	case 0:
		t.FormattedText = ""
	case 1:
		t.FormattedText = fmt.Sprintf("%s is typing...", t.TypingUsers[0])
	case 2:
		t.FormattedText = fmt.Sprintf("%s and %s are typing...", t.TypingUsers[0], t.TypingUsers[1])
	case 3:
		t.FormattedText = fmt.Sprintf("%s, %s, and %s are typing...", t.TypingUsers[0], t.TypingUsers[1], t.TypingUsers[2])
	default:
		t.FormattedText = fmt.Sprintf("%s, %s, and %d others are typing...", t.TypingUsers[0], t.TypingUsers[1], len(t.TypingUsers)-2)
	}
}

// IsStale checks if typing indicator is stale and should be hidden
func (t *TypingIndicatorState) IsStale() bool {
	return time.Since(t.LastUpdated) > time.Duration(TypingTimeoutSeconds+1)*time.Second
}

// ToTemplateData converts the typing indicator state to template data
func (t *TypingIndicatorState) ToTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"TypingUsers":   t.TypingUsers,
		"IsVisible":     t.IsVisible && !t.IsStale(),
		"FormattedText": t.FormattedText,
		"Count":         len(t.TypingUsers),
	}
}

// MessageRateLimiter tracks message rate limiting per user
type MessageRateLimiter struct {
	attempts    map[string][]time.Time // UserID -> message timestamps
	maxMessages int
	timeWindow  time.Duration
}

// NewMessageRateLimiter creates a new rate limiter
func NewMessageRateLimiter() *MessageRateLimiter {
	return &MessageRateLimiter{
		attempts:    make(map[string][]time.Time),
		maxMessages: MaxMessagesPerMinute,
		timeWindow:  time.Minute,
	}
}

// IsRateLimited checks if a user is rate limited
func (r *MessageRateLimiter) IsRateLimited(userID string) bool {
	now := time.Now()
	attempts := r.attempts[userID]

	// Clean old attempts outside time window
	validAttempts := make([]time.Time, 0)
	for _, attempt := range attempts {
		if now.Sub(attempt) < r.timeWindow {
			validAttempts = append(validAttempts, attempt)
		}
	}
	r.attempts[userID] = validAttempts

	return len(validAttempts) >= r.maxMessages
}

// RecordMessage records a message attempt
func (r *MessageRateLimiter) RecordMessage(userID string) {
	now := time.Now()
	r.attempts[userID] = append(r.attempts[userID], now)
}

// GetRemainingMessages returns remaining messages in current window
func (r *MessageRateLimiter) GetRemainingMessages(userID string) int {
	now := time.Now()
	attempts := r.attempts[userID]

	validAttempts := 0
	for _, attempt := range attempts {
		if now.Sub(attempt) < r.timeWindow {
			validAttempts++
		}
	}

	remaining := r.maxMessages - validAttempts
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GetRetryAfter returns how long to wait before next message
func (r *MessageRateLimiter) GetRetryAfter(userID string) time.Duration {
	attempts := r.attempts[userID]
	if len(attempts) < r.maxMessages {
		return 0
	}

	// Find oldest attempt within time window
	now := time.Now()
	oldestAttempt := attempts[0]
	for _, attempt := range attempts {
		if now.Sub(attempt) < r.timeWindow && attempt.Before(oldestAttempt) {
			oldestAttempt = attempt
		}
	}

	elapsed := now.Sub(oldestAttempt)
	if elapsed >= r.timeWindow {
		return 0
	}

	return r.timeWindow - elapsed
}
