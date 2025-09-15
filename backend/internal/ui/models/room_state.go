package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// RoomCreationState handles new chat room creation interface state
type RoomCreationState struct {
	RoomName         string            `json:"roomName"`
	Description      string            `json:"description"`
	IsPrivate        bool              `json:"isPrivate"`
	IsCreating       bool              `json:"isCreating"`
	ValidationErrors map[string]string `json:"validationErrors"`
	GeneralError     string            `json:"generalError"`
	CSRFToken        string            `json:"csrfToken"`
	CreatedRoomID    string            `json:"createdRoomId,omitempty"`
}

// RoomEditState handles existing room editing interface
type RoomEditState struct {
	RoomID           string            `json:"roomId"`
	OriginalName     string            `json:"originalName"`
	RoomName         string            `json:"roomName"`
	Description      string            `json:"description"`
	IsPrivate        bool              `json:"isPrivate"`
	IsUpdating       bool              `json:"isUpdating"`
	ValidationErrors map[string]string `json:"validationErrors"`
	GeneralError     string            `json:"generalError"`
	CSRFToken        string            `json:"csrfToken"`
	HasChanges       bool              `json:"hasChanges"`
}

// Room validation constants
const (
	MinRoomNameLength     = 3
	MaxRoomNameLength     = 30
	MaxRoomDescLength     = 200
	MaxRoomsPerUser       = 10  // Maximum rooms a user can create
	RoomNameCooldownMins  = 5   // Minutes before room name can be reused
)

var (
	// Room name validation regex - alphanumeric, spaces, hyphens, and underscores
	roomNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-_]+$`)

	// Room validation errors
	ErrRoomNameRequired     = errors.New("room name is required")
	ErrRoomNameTooShort     = fmt.Errorf("room name must be at least %d characters", MinRoomNameLength)
	ErrRoomNameTooLong      = fmt.Errorf("room name must be no more than %d characters", MaxRoomNameLength)
	ErrRoomNameInvalidChar  = errors.New("room name can only contain letters, numbers, spaces, hyphens, and underscores")
	ErrRoomNameTaken        = errors.New("room name is already taken")
	ErrRoomNameReserved     = errors.New("room name is reserved")
	ErrRoomDescTooLong      = fmt.Errorf("description must be no more than %d characters", MaxRoomDescLength)
	ErrTooManyRooms         = fmt.Errorf("maximum %d rooms per user allowed", MaxRoomsPerUser)
	ErrCSRFTokenRequired    = errors.New("CSRF token is required")
	ErrRoomNotFound         = errors.New("room not found")
	ErrUnauthorizedEdit     = errors.New("you don't have permission to edit this room")
	ErrNoChangesDetected    = errors.New("no changes detected")
)

// Reserved room names that cannot be used
var reservedRoomNames = map[string]bool{
	"system":      true,
	"admin":       true,
	"moderator":   true,
	"general":     true,
	"main":        true,
	"lobby":       true,
	"welcome":     true,
	"help":        true,
	"support":     true,
	"announcements": true,
	"news":        true,
	"updates":     true,
}

// NewRoomCreationState creates a new room creation state
func NewRoomCreationState(csrfToken string) *RoomCreationState {
	return &RoomCreationState{
		RoomName:         "",
		Description:      "",
		IsPrivate:        false,
		IsCreating:       false,
		ValidationErrors: make(map[string]string),
		GeneralError:     "",
		CSRFToken:        csrfToken,
		CreatedRoomID:    "",
	}
}

// NewRoomEditState creates a new room edit state
func NewRoomEditState(roomID, roomName, description, csrfToken string, isPrivate bool) *RoomEditState {
	return &RoomEditState{
		RoomID:           roomID,
		OriginalName:     roomName,
		RoomName:         roomName,
		Description:      description,
		IsPrivate:        isPrivate,
		IsUpdating:       false,
		ValidationErrors: make(map[string]string),
		GeneralError:     "",
		CSRFToken:        csrfToken,
		HasChanges:       false,
	}
}

// ValidateInput validates the room creation input
func (s *RoomCreationState) ValidateInput() error {
	// Clear previous validation errors
	s.ValidationErrors = make(map[string]string)

	hasErrors := false

	// Validate room name
	if err := s.validateRoomName(); err != nil {
		s.ValidationErrors["room_name"] = err.Error()
		hasErrors = true
	}

	// Validate description
	if err := s.validateDescription(); err != nil {
		s.ValidationErrors["description"] = err.Error()
		hasErrors = true
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

// ValidateInput validates the room edit input
func (s *RoomEditState) ValidateInput() error {
	// Clear previous validation errors
	s.ValidationErrors = make(map[string]string)

	hasErrors := false

	// Check if there are any changes
	if !s.hasAnyChanges() {
		s.ValidationErrors["general"] = ErrNoChangesDetected.Error()
		hasErrors = true
	}

	// Validate room name
	if err := s.validateRoomName(); err != nil {
		s.ValidationErrors["room_name"] = err.Error()
		hasErrors = true
	}

	// Validate description
	if err := s.validateDescription(); err != nil {
		s.ValidationErrors["description"] = err.Error()
		hasErrors = true
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

// validateRoomName validates the room name field
func (s *RoomCreationState) validateRoomName() error {
	return validateRoomNameCommon(s.RoomName)
}

// validateRoomName validates the room name field for edit state
func (s *RoomEditState) validateRoomName() error {
	return validateRoomNameCommon(s.RoomName)
}

// validateRoomNameCommon performs common room name validation
func validateRoomNameCommon(roomName string) error {
	name := strings.TrimSpace(roomName)

	if name == "" {
		return ErrRoomNameRequired
	}

	if len(name) < MinRoomNameLength {
		return ErrRoomNameTooShort
	}

	if len(name) > MaxRoomNameLength {
		return ErrRoomNameTooLong
	}

	if !roomNameRegex.MatchString(name) {
		return ErrRoomNameInvalidChar
	}

	// Check for reserved names
	nameLower := strings.ToLower(name)
	if reservedRoomNames[nameLower] {
		return ErrRoomNameReserved
	}

	// Check for inappropriate content (basic check)
	if containsInappropriateContent(name) {
		return errors.New("room name contains inappropriate content")
	}

	return nil
}

// validateDescription validates the description field
func (s *RoomCreationState) validateDescription() error {
	return validateDescriptionCommon(s.Description)
}

// validateDescription validates the description field for edit state
func (s *RoomEditState) validateDescription() error {
	return validateDescriptionCommon(s.Description)
}

// validateDescriptionCommon performs common description validation
func validateDescriptionCommon(description string) error {
	// Description is optional, but if provided, must meet length requirements
	if len(description) > MaxRoomDescLength {
		return ErrRoomDescTooLong
	}

	// Check for inappropriate content if description is provided
	if description != "" && containsInappropriateContent(description) {
		return errors.New("description contains inappropriate content")
	}

	return nil
}

// validateCSRFToken validates the CSRF token
func (s *RoomCreationState) validateCSRFToken() error {
	return validateCSRFTokenCommon(s.CSRFToken)
}

// validateCSRFToken validates the CSRF token for edit state
func (s *RoomEditState) validateCSRFToken() error {
	return validateCSRFTokenCommon(s.CSRFToken)
}

// validateCSRFTokenCommon performs common CSRF token validation
func validateCSRFTokenCommon(token string) error {
	if token == "" {
		return ErrCSRFTokenRequired
	}

	// Basic validation - in real implementation, validate against stored token
	if len(token) < 16 {
		return ErrCSRFTokenRequired
	}

	return nil
}

// containsInappropriateContent checks for basic inappropriate content
func containsInappropriateContent(content string) bool {
	// This is a simple implementation. In production, you'd use a more
	// sophisticated content filtering system
	inappropriateWords := []string{
		"spam", "test123", "admin123", // Add more as needed
	}

	contentLower := strings.ToLower(content)
	for _, word := range inappropriateWords {
		if strings.Contains(contentLower, word) {
			return true
		}
	}

	return false
}

// hasAnyChanges checks if any fields have changed
func (s *RoomEditState) hasAnyChanges() bool {
	return s.RoomName != s.OriginalName ||
		   s.Description != s.Description || // Compare with original if you store it
		   s.HasChanges // Or use a manual flag
}

// SetCreating sets the creation state
func (s *RoomCreationState) SetCreating(creating bool) {
	s.IsCreating = creating
}

// SetUpdating sets the updating state
func (s *RoomEditState) SetUpdating(updating bool) {
	s.IsUpdating = updating
}

// SetGeneralError sets a general error message
func (s *RoomCreationState) SetGeneralError(err error) {
	if err != nil {
		s.GeneralError = err.Error()
	} else {
		s.GeneralError = ""
	}
}

// SetGeneralError sets a general error message for edit state
func (s *RoomEditState) SetGeneralError(err error) {
	if err != nil {
		s.GeneralError = err.Error()
	} else {
		s.GeneralError = ""
	}
}

// SetFieldError sets a field-specific error
func (s *RoomCreationState) SetFieldError(field, message string) {
	if s.ValidationErrors == nil {
		s.ValidationErrors = make(map[string]string)
	}
	s.ValidationErrors[field] = message
}

// SetFieldError sets a field-specific error for edit state
func (s *RoomEditState) SetFieldError(field, message string) {
	if s.ValidationErrors == nil {
		s.ValidationErrors = make(map[string]string)
	}
	s.ValidationErrors[field] = message
}

// ClearErrors clears all errors
func (s *RoomCreationState) ClearErrors() {
	s.ValidationErrors = make(map[string]string)
	s.GeneralError = ""
}

// ClearErrors clears all errors for edit state
func (s *RoomEditState) ClearErrors() {
	s.ValidationErrors = make(map[string]string)
	s.GeneralError = ""
}

// HasErrors returns true if there are any errors
func (s *RoomCreationState) HasErrors() bool {
	return s.GeneralError != "" || len(s.ValidationErrors) > 0
}

// HasErrors returns true if there are any errors for edit state
func (s *RoomEditState) HasErrors() bool {
	return s.GeneralError != "" || len(s.ValidationErrors) > 0
}

// HasFieldError returns true if a specific field has an error
func (s *RoomCreationState) HasFieldError(field string) bool {
	_, exists := s.ValidationErrors[field]
	return exists
}

// HasFieldError returns true if a specific field has an error for edit state
func (s *RoomEditState) HasFieldError(field string) bool {
	_, exists := s.ValidationErrors[field]
	return exists
}

// GetFieldError returns the error for a specific field
func (s *RoomCreationState) GetFieldError(field string) string {
	return s.ValidationErrors[field]
}

// GetFieldError returns the error for a specific field for edit state
func (s *RoomEditState) GetFieldError(field string) string {
	return s.ValidationErrors[field]
}

// SanitizeInput sanitizes input to prevent XSS attacks
func (s *RoomCreationState) SanitizeInput() {
	// Trim whitespace
	s.RoomName = strings.TrimSpace(s.RoomName)
	s.Description = strings.TrimSpace(s.Description)

	// Remove HTML tags for basic XSS prevention
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	s.RoomName = htmlTagRegex.ReplaceAllString(s.RoomName, "")
	s.Description = htmlTagRegex.ReplaceAllString(s.Description, "")

	// Normalize multiple spaces to single spaces in room name
	spaceRegex := regexp.MustCompile(`\s+`)
	s.RoomName = spaceRegex.ReplaceAllString(s.RoomName, " ")
}

// SanitizeInput sanitizes input for edit state
func (s *RoomEditState) SanitizeInput() {
	// Trim whitespace
	s.RoomName = strings.TrimSpace(s.RoomName)
	s.Description = strings.TrimSpace(s.Description)

	// Remove HTML tags for basic XSS prevention
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	s.RoomName = htmlTagRegex.ReplaceAllString(s.RoomName, "")
	s.Description = htmlTagRegex.ReplaceAllString(s.Description, "")

	// Normalize multiple spaces to single spaces in room name
	spaceRegex := regexp.MustCompile(`\s+`)
	s.RoomName = spaceRegex.ReplaceAllString(s.RoomName, " ")

	// Update changes flag
	s.updateChangesFlag()
}

// updateChangesFlag updates the HasChanges flag
func (s *RoomEditState) updateChangesFlag() {
	s.HasChanges = s.RoomName != s.OriginalName
}

// SetRoomCreated sets the created room ID after successful creation
func (s *RoomCreationState) SetRoomCreated(roomID string) {
	s.CreatedRoomID = roomID
	s.IsCreating = false
	s.ClearErrors()
}

// Reset resets the form state after successful creation
func (s *RoomCreationState) Reset(newCSRFToken string) {
	s.RoomName = ""
	s.Description = ""
	s.IsPrivate = false
	s.IsCreating = false
	s.ValidationErrors = make(map[string]string)
	s.GeneralError = ""
	s.CSRFToken = newCSRFToken
	s.CreatedRoomID = ""
}

// GetRoomNameSuggestions provides alternative room names if current is taken
func (s *RoomCreationState) GetRoomNameSuggestions() []string {
	baseName := strings.TrimSpace(s.RoomName)
	if baseName == "" {
		return []string{}
	}

	suggestions := make([]string, 0, 3)

	// Add numbered variations
	for i := 1; i <= 3; i++ {
		suggestion := fmt.Sprintf("%s-%d", baseName, i)
		if len(suggestion) <= MaxRoomNameLength {
			suggestions = append(suggestions, suggestion)
		}
	}

	// Add timestamp variation if name is short enough
	if len(baseName) <= MaxRoomNameLength-8 {
		timestamp := time.Now().Format("01-02")
		suggestions = append(suggestions, fmt.Sprintf("%s-%s", baseName, timestamp))
	}

	return suggestions
}

// ToTemplateData converts the state to data suitable for template rendering
func (s *RoomCreationState) ToTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"RoomName":         s.RoomName,
		"Description":      s.Description,
		"IsPrivate":        s.IsPrivate,
		"IsCreating":       s.IsCreating,
		"ValidationErrors": s.ValidationErrors,
		"GeneralError":     s.GeneralError,
		"CSRFToken":        s.CSRFToken,
		"HasErrors":        s.HasErrors(),
		"CreatedRoomID":    s.CreatedRoomID,
		"MaxNameLength":    MaxRoomNameLength,
		"MaxDescLength":    MaxRoomDescLength,
		"NameSuggestions":  s.GetRoomNameSuggestions(),
	}
}

// ToTemplateData converts the edit state to data suitable for template rendering
func (s *RoomEditState) ToTemplateData() map[string]interface{} {
	return map[string]interface{}{
		"RoomID":           s.RoomID,
		"OriginalName":     s.OriginalName,
		"RoomName":         s.RoomName,
		"Description":      s.Description,
		"IsPrivate":        s.IsPrivate,
		"IsUpdating":       s.IsUpdating,
		"ValidationErrors": s.ValidationErrors,
		"GeneralError":     s.GeneralError,
		"CSRFToken":        s.CSRFToken,
		"HasErrors":        s.HasErrors(),
		"HasChanges":       s.HasChanges,
		"MaxNameLength":    MaxRoomNameLength,
		"MaxDescLength":    MaxRoomDescLength,
		"CanSave":          s.HasChanges && !s.HasErrors() && !s.IsUpdating,
	}
}

// RoomQuotaTracker tracks room creation quotas per user
type RoomQuotaTracker struct {
	userRooms   map[string]int        // UserID -> room count
	lastCreated map[string]time.Time  // UserID -> last creation time
}

// NewRoomQuotaTracker creates a new room quota tracker
func NewRoomQuotaTracker() *RoomQuotaTracker {
	return &RoomQuotaTracker{
		userRooms:   make(map[string]int),
		lastCreated: make(map[string]time.Time),
	}
}

// CanCreateRoom checks if user can create another room
func (r *RoomQuotaTracker) CanCreateRoom(userID string) bool {
	currentCount := r.userRooms[userID]
	return currentCount < MaxRoomsPerUser
}

// GetRemainingQuota returns remaining room quota for user
func (r *RoomQuotaTracker) GetRemainingQuota(userID string) int {
	currentCount := r.userRooms[userID]
	remaining := MaxRoomsPerUser - currentCount
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RecordRoomCreation records a new room creation
func (r *RoomQuotaTracker) RecordRoomCreation(userID string) {
	r.userRooms[userID]++
	r.lastCreated[userID] = time.Now()
}

// RecordRoomDeletion records a room deletion
func (r *RoomQuotaTracker) RecordRoomDeletion(userID string) {
	if r.userRooms[userID] > 0 {
		r.userRooms[userID]--
	}
}

// GetLastCreationTime returns last room creation time for user
func (r *RoomQuotaTracker) GetLastCreationTime(userID string) *time.Time {
	if lastTime, exists := r.lastCreated[userID]; exists {
		return &lastTime
	}
	return nil
}

// RoomNameRegistry tracks used room names and cooldowns
type RoomNameRegistry struct {
	usedNames    map[string]time.Time // name -> last used time
	cooldownMins int
}

// NewRoomNameRegistry creates a new room name registry
func NewRoomNameRegistry() *RoomNameRegistry {
	return &RoomNameRegistry{
		usedNames:    make(map[string]time.Time),
		cooldownMins: RoomNameCooldownMins,
	}
}

// IsNameAvailable checks if a room name is available
func (r *RoomNameRegistry) IsNameAvailable(roomName string) bool {
	nameLower := strings.ToLower(strings.TrimSpace(roomName))

	// Check if name is reserved
	if reservedRoomNames[nameLower] {
		return false
	}

	// Check cooldown period
	if lastUsed, exists := r.usedNames[nameLower]; exists {
		cooldownExpiry := lastUsed.Add(time.Duration(r.cooldownMins) * time.Minute)
		return time.Now().After(cooldownExpiry)
	}

	return true
}

// ReserveName reserves a room name
func (r *RoomNameRegistry) ReserveName(roomName string) {
	nameLower := strings.ToLower(strings.TrimSpace(roomName))
	r.usedNames[nameLower] = time.Now()
}

// ReleaseName releases a room name (when room is deleted)
func (r *RoomNameRegistry) ReleaseName(roomName string) {
	nameLower := strings.ToLower(strings.TrimSpace(roomName))
	delete(r.usedNames, nameLower)
}

// GetCooldownRemaining returns remaining cooldown time for a name
func (r *RoomNameRegistry) GetCooldownRemaining(roomName string) time.Duration {
	nameLower := strings.ToLower(strings.TrimSpace(roomName))

	if lastUsed, exists := r.usedNames[nameLower]; exists {
		cooldownExpiry := lastUsed.Add(time.Duration(r.cooldownMins) * time.Minute)
		remaining := time.Until(cooldownExpiry)
		if remaining > 0 {
			return remaining
		}
	}

	return 0
}