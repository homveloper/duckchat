package models

import (
	"errors"
	"fmt"
	"time"
)

// ChatPageState represents the complete chat interface state
type ChatPageState struct {
	CurrentUser  *UserContext        `json:"currentUser"`
	CurrentRoom  *ChatRoomContext    `json:"currentRoom"`
	Messages     []*MessageViewModel `json:"messages"`
	OnlineUsers  []*UserContext      `json:"onlineUsers"`
	IsConnected  bool                `json:"isConnected"`
	UnreadCount  int                 `json:"unreadCount"`
	LastSeen     *time.Time          `json:"lastSeen,omitempty"`
	IsLoading    bool                `json:"isLoading"`
	LoadingError string              `json:"loadingError,omitempty"`
	ConnectionID string              `json:"connectionId,omitempty"` // SSE connection ID
}

// UserContext represents authenticated user information
type UserContext struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	AvatarURL    *string    `json:"avatarUrl,omitempty"`
	IsOnline     bool       `json:"isOnline"`
	LastActivity *time.Time `json:"lastActivity,omitempty"`
	IsTyping     bool       `json:"isTyping"`
}

// ChatRoomContext represents the current chat room
type ChatRoomContext struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Description      *string    `json:"description,omitempty"`
	IsPrivate        bool       `json:"isPrivate"`
	ParticipantCount int        `json:"participantCount"`
	CreatedAt        time.Time  `json:"createdAt"`
	LastActivity     *time.Time `json:"lastActivity,omitempty"`
	UserRole         string     `json:"userRole"` // "owner", "member"
}

// MessageViewModel represents a chat message for display
type MessageViewModel struct {
	ID            string     `json:"id"`
	Content       string     `json:"content"`
	Username      string     `json:"username"`
	UserID        string     `json:"userId"`
	Timestamp     time.Time  `json:"timestamp"`
	IsCurrentUser bool       `json:"isCurrentUser"`
	AvatarURL     *string    `json:"avatarUrl,omitempty"`
	MessageType   string     `json:"messageType"` // "text", "system", "notification"
	IsEdited      bool       `json:"isEdited"`
	EditedAt      *time.Time `json:"editedAt,omitempty"`
}

// MessageType constants
const (
	MessageTypeText         = "text"
	MessageTypeSystem       = "system"
	MessageTypeNotification = "notification"
	MessageTypeUserJoined   = "user_joined"
	MessageTypeUserLeft     = "user_left"
)

// UserRole constants
const (
	UserRoleOwner  = "owner"
	UserRoleMember = "member"
)

var (
	ErrAccessDenied     = errors.New("access denied to this room")
	ErrUserNotInRoom    = errors.New("user is not a member of this room")
	ErrConnectionFailed = errors.New("failed to connect to chat room")
)

// NewChatPageState creates a new chat page state
func NewChatPageState(user *UserContext, room *ChatRoomContext) *ChatPageState {
	return &ChatPageState{
		CurrentUser:  user,
		CurrentRoom:  room,
		Messages:     make([]*MessageViewModel, 0),
		OnlineUsers:  make([]*UserContext, 0),
		IsConnected:  false,
		UnreadCount:  0,
		IsLoading:    true,
		LoadingError: "",
		ConnectionID: "",
	}
}

// AddMessage adds a new message to the chat state
func (s *ChatPageState) AddMessage(message *MessageViewModel) {
	if s.CurrentUser != nil {
		message.IsCurrentUser = message.UserID == s.CurrentUser.ID
	}

	s.Messages = append(s.Messages, message)

	// Update unread count if not current user and we have a last seen timestamp
	if !message.IsCurrentUser && s.LastSeen != nil && message.Timestamp.After(*s.LastSeen) {
		s.UnreadCount++
	}

	// Update room's last activity
	if s.CurrentRoom != nil {
		now := time.Now()
		s.CurrentRoom.LastActivity = &now
	}
}

// AddSystemMessage adds a system message to the chat
func (s *ChatPageState) AddSystemMessage(content string) {
	message := &MessageViewModel{
		ID:            fmt.Sprintf("system_%d", time.Now().UnixNano()),
		Content:       content,
		Username:      "System",
		UserID:        "system",
		Timestamp:     time.Now(),
		MessageType:   MessageTypeSystem,
		IsCurrentUser: false,
	}

	s.Messages = append(s.Messages, message)
}

// AddUserJoinedMessage adds a user joined notification
func (s *ChatPageState) AddUserJoinedMessage(user *UserContext) {
	content := fmt.Sprintf("%s joined the room", user.Username)
	message := &MessageViewModel{
		ID:            fmt.Sprintf("join_%s_%d", user.ID, time.Now().UnixNano()),
		Content:       content,
		Username:      user.Username,
		UserID:        user.ID,
		Timestamp:     time.Now(),
		MessageType:   MessageTypeUserJoined,
		IsCurrentUser: s.CurrentUser != nil && user.ID == s.CurrentUser.ID,
	}

	s.Messages = append(s.Messages, message)
}

// AddUserLeftMessage adds a user left notification
func (s *ChatPageState) AddUserLeftMessage(user *UserContext) {
	content := fmt.Sprintf("%s left the room", user.Username)
	message := &MessageViewModel{
		ID:            fmt.Sprintf("left_%s_%d", user.ID, time.Now().UnixNano()),
		Content:       content,
		Username:      user.Username,
		UserID:        user.ID,
		Timestamp:     time.Now(),
		MessageType:   MessageTypeUserLeft,
		IsCurrentUser: s.CurrentUser != nil && user.ID == s.CurrentUser.ID,
	}

	s.Messages = append(s.Messages, message)
}

// UpdateOnlineUser updates or adds a user to the online users list
func (s *ChatPageState) UpdateOnlineUser(user *UserContext) {
	// Find existing user and update
	for i, existingUser := range s.OnlineUsers {
		if existingUser.ID == user.ID {
			s.OnlineUsers[i] = user
			return
		}
	}

	// Add new user if not found
	s.OnlineUsers = append(s.OnlineUsers, user)
}

// RemoveOnlineUser removes a user from the online users list
func (s *ChatPageState) RemoveOnlineUser(userID string) {
	for i, user := range s.OnlineUsers {
		if user.ID == userID {
			s.OnlineUsers = append(s.OnlineUsers[:i], s.OnlineUsers[i+1:]...)
			return
		}
	}
}

// SetUserTyping sets the typing status for a user
func (s *ChatPageState) SetUserTyping(userID string, isTyping bool) {
	for _, user := range s.OnlineUsers {
		if user.ID == userID {
			user.IsTyping = isTyping
			return
		}
	}
}

// GetTypingUsers returns a list of users currently typing
func (s *ChatPageState) GetTypingUsers() []*UserContext {
	typingUsers := make([]*UserContext, 0)
	for _, user := range s.OnlineUsers {
		if user.IsTyping && (s.CurrentUser == nil || user.ID != s.CurrentUser.ID) {
			typingUsers = append(typingUsers, user)
		}
	}
	return typingUsers
}

// SetConnected sets the connection status
func (s *ChatPageState) SetConnected(connected bool) {
	s.IsConnected = connected
	if connected {
		s.LoadingError = ""
	}
}

// SetConnectionError sets a connection error
func (s *ChatPageState) SetConnectionError(err error) {
	s.IsConnected = false
	s.IsLoading = false
	if err != nil {
		s.LoadingError = err.Error()
	}
}

// SetLoading sets the loading state
func (s *ChatPageState) SetLoading(loading bool) {
	s.IsLoading = loading
	if !loading && s.LoadingError != "" {
		s.LoadingError = ""
	}
}

// MarkAllAsRead marks all messages as read and resets unread count
func (s *ChatPageState) MarkAllAsRead() {
	s.UnreadCount = 0
	now := time.Now()
	s.LastSeen = &now
}

// GetRecentMessages returns the most recent N messages
func (s *ChatPageState) GetRecentMessages(limit int) []*MessageViewModel {
	if len(s.Messages) <= limit {
		return s.Messages
	}
	return s.Messages[len(s.Messages)-limit:]
}

// HasPermission checks if the current user has permission for an action
func (s *ChatPageState) HasPermission(action string) bool {
	if s.CurrentUser == nil || s.CurrentRoom == nil {
		return false
	}

	switch action {
	case "send_message":
		return true // All room members can send messages
	case "edit_room":
		return s.CurrentRoom.UserRole == UserRoleOwner
	case "delete_room":
		return s.CurrentRoom.UserRole == UserRoleOwner
	case "kick_user":
		return s.CurrentRoom.UserRole == UserRoleOwner
	default:
		return false
	}
}

// ValidateAccess validates that the current user can access this room
func (s *ChatPageState) ValidateAccess() error {
	if s.CurrentUser == nil {
		return errors.New("user not authenticated")
	}

	if s.CurrentRoom == nil {
		return ErrRoomNotFound
	}

	// For private rooms, user must be explicitly allowed
	// This would typically check against a database
	if s.CurrentRoom.IsPrivate {
		// In real implementation, check membership in database
		return nil // Placeholder - assume access is valid
	}

	return nil
}

// ToTemplateData converts the state to data suitable for template rendering
func (s *ChatPageState) ToTemplateData() map[string]interface{} {
	data := map[string]interface{}{
		"CurrentUser":    s.CurrentUser,
		"CurrentRoom":    s.CurrentRoom,
		"Messages":       s.Messages,
		"OnlineUsers":    s.OnlineUsers,
		"IsConnected":    s.IsConnected,
		"UnreadCount":    s.UnreadCount,
		"IsLoading":      s.IsLoading,
		"LoadingError":   s.LoadingError,
		"TypingUsers":    s.GetTypingUsers(),
		"HasPermissions": make(map[string]bool),
	}

	// Add permission checks
	permissions := []string{"send_message", "edit_room", "delete_room", "kick_user"}
	permissionMap := make(map[string]bool)
	for _, permission := range permissions {
		permissionMap[permission] = s.HasPermission(permission)
	}
	data["HasPermissions"] = permissionMap

	return data
}

// GetConnectionStatus returns human-readable connection status
func (s *ChatPageState) GetConnectionStatus() string {
	if s.IsLoading {
		return "Connecting..."
	}
	if !s.IsConnected {
		if s.LoadingError != "" {
			return "Connection failed"
		}
		return "Disconnected"
	}
	return "Connected"
}

// GetParticipantSummary returns a summary of room participants
func (s *ChatPageState) GetParticipantSummary() string {
	onlineCount := len(s.OnlineUsers)
	totalCount := s.CurrentRoom.ParticipantCount

	if onlineCount == totalCount {
		return fmt.Sprintf("%d online", onlineCount)
	}
	return fmt.Sprintf("%d of %d online", onlineCount, totalCount)
}

// GetFormattedTypingStatus returns formatted typing status message
func (s *ChatPageState) GetFormattedTypingStatus() string {
	typingUsers := s.GetTypingUsers()

	switch len(typingUsers) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%s is typing...", typingUsers[0].Username)
	case 2:
		return fmt.Sprintf("%s and %s are typing...", typingUsers[0].Username, typingUsers[1].Username)
	default:
		return fmt.Sprintf("%s and %d others are typing...", typingUsers[0].Username, len(typingUsers)-1)
	}
}
