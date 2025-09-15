package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// UserSession represents an active user session
type UserSession struct {
	ID           string    `json:"id" redis:"id"`
	UserID       string    `json:"user_id" redis:"user_id"`
	Username     string    `json:"username" redis:"username"`
	ActiveRooms  []string  `json:"active_rooms" redis:"active_rooms"`
	ConnectedAt  time.Time `json:"connected_at" redis:"connected_at"`
	LastSeenAt   time.Time `json:"last_seen_at" redis:"last_seen_at"`
	ConnectionID string    `json:"connection_id,omitempty" redis:"connection_id"` // For SSE connections
	IsActive     bool      `json:"is_active" redis:"is_active"`
}

// NewUserSession creates a new UserSession
func NewUserSession(userID, username string) (*UserSession, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id cannot be empty")
	}

	if _, err := uuid.Parse(userID); err != nil {
		return nil, fmt.Errorf("user_id must be a valid UUID: %w", err)
	}

	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	now := time.Now().UTC()
	return &UserSession{
		ID:          uuid.New().String(),
		UserID:      userID,
		Username:    username,
		ActiveRooms: make([]string, 0),
		ConnectedAt: now,
		LastSeenAt:  now,
		IsActive:    true,
	}, nil
}

// Validate validates the UserSession struct
func (us *UserSession) Validate() error {
	if us.ID == "" {
		return fmt.Errorf("session id cannot be empty")
	}

	if _, err := uuid.Parse(us.ID); err != nil {
		return fmt.Errorf("session id must be a valid UUID: %w", err)
	}

	if us.UserID == "" {
		return fmt.Errorf("user_id cannot be empty")
	}

	if _, err := uuid.Parse(us.UserID); err != nil {
		return fmt.Errorf("user_id must be a valid UUID: %w", err)
	}

	if us.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if us.ConnectedAt.IsZero() {
		return fmt.Errorf("connected_at cannot be zero")
	}

	if us.LastSeenAt.IsZero() {
		return fmt.Errorf("last_seen_at cannot be zero")
	}

	if us.LastSeenAt.Before(us.ConnectedAt) {
		return fmt.Errorf("last_seen_at cannot be before connected_at")
	}

	return nil
}

// RedisKey returns the Redis key for this session
func (us *UserSession) RedisKey() string {
	return fmt.Sprintf("session:%s", us.ID)
}

// UpdateLastSeen updates the last seen timestamp
func (us *UserSession) UpdateLastSeen() {
	us.LastSeenAt = time.Now().UTC()
}

// AddRoom adds a room to the active rooms list
func (us *UserSession) AddRoom(roomID string) error {
	if _, err := uuid.Parse(roomID); err != nil {
		return fmt.Errorf("room_id must be a valid UUID: %w", err)
	}

	// Check if room is already in the list
	for _, existingRoomID := range us.ActiveRooms {
		if existingRoomID == roomID {
			return nil // Already exists, no need to add
		}
	}

	us.ActiveRooms = append(us.ActiveRooms, roomID)
	us.UpdateLastSeen()
	return nil
}

// RemoveRoom removes a room from the active rooms list
func (us *UserSession) RemoveRoom(roomID string) {
	for i, existingRoomID := range us.ActiveRooms {
		if existingRoomID == roomID {
			us.ActiveRooms = append(us.ActiveRooms[:i], us.ActiveRooms[i+1:]...)
			break
		}
	}
	us.UpdateLastSeen()
}

// IsInRoom checks if the user is in a specific room
func (us *UserSession) IsInRoom(roomID string) bool {
	for _, existingRoomID := range us.ActiveRooms {
		if existingRoomID == roomID {
			return true
		}
	}
	return false
}

// SetConnectionID sets the SSE connection ID
func (us *UserSession) SetConnectionID(connectionID string) {
	us.ConnectionID = connectionID
	us.UpdateLastSeen()
}

// ClearConnection clears the SSE connection
func (us *UserSession) ClearConnection() {
	us.ConnectionID = ""
	us.UpdateLastSeen()
}

// Disconnect marks the session as inactive
func (us *UserSession) Disconnect() {
	us.IsActive = false
	us.ConnectionID = ""
	us.UpdateLastSeen()
}

// SSEConnection represents an SSE connection for a room
type SSEConnection struct {
	SessionID    string    `json:"session_id"`
	UserID       string    `json:"user_id"`
	Username     string    `json:"username"`
	RoomID       string    `json:"room_id"`
	ConnectedAt  time.Time `json:"connected_at"`
	ConnectionID string    `json:"connection_id"`
}

// NewSSEConnection creates a new SSE connection record
func NewSSEConnection(sessionID, userID, username, roomID string) (*SSEConnection, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id cannot be empty")
	}

	if userID == "" {
		return nil, fmt.Errorf("user_id cannot be empty")
	}

	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}

	if roomID == "" {
		return nil, fmt.Errorf("room_id cannot be empty")
	}

	return &SSEConnection{
		SessionID:    sessionID,
		UserID:       userID,
		Username:     username,
		RoomID:       roomID,
		ConnectedAt:  time.Now().UTC(),
		ConnectionID: uuid.New().String(),
	}, nil
}

// RedisKey returns the Redis Set key for SSE connections in this room
func (sse *SSEConnection) RedisKey() string {
	return fmt.Sprintf("sse:room:%s", sse.RoomID)
}

// ConnectionData returns the connection data for Redis Set storage
func (sse *SSEConnection) ConnectionData() map[string]interface{} {
	return map[string]interface{}{
		"session_id":    sse.SessionID,
		"user_id":       sse.UserID,
		"username":      sse.Username,
		"connected_at":  sse.ConnectedAt,
		"connection_id": sse.ConnectionID,
	}
}