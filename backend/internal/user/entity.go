package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// User represents a user profile with changeable username
type User struct {
	ID        UserID    `json:"user_id"`
	Username  Username  `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserID is a value object for user identification
type UserID string

// Username is a value object with validation rules
type Username struct {
	value string
}

// Domain errors
var (
	ErrInvalidUsername = errors.New("username must be 1-50 characters, alphanumeric and spaces only")
	ErrEmptyUserID     = errors.New("user ID cannot be empty")
)

// NewUserID creates a new UserID
func NewUserID() UserID {
	return UserID(uuid.New().String())
}

// ParseUserID creates UserID from string
func ParseUserID(id string) (UserID, error) {
	if id == "" {
		return UserID(""), ErrEmptyUserID
	}
	return UserID(id), nil
}

// String returns string representation of UserID
func (u UserID) String() string {
	return string(u)
}

// NewUsername creates a Username with validation
func NewUsername(name string) (Username, error) {
	if len(name) < 1 || len(name) > 50 {
		return Username{}, ErrInvalidUsername
	}
	return Username{value: name}, nil
}

// String returns string representation of Username
func (u Username) String() string {
	return u.value
}

// IsEmpty checks if username is empty
func (u Username) IsEmpty() bool {
	return u.value == ""
}

// NewUser creates a new User with validation
func NewUser(username string) (*User, error) {
	userID := NewUserID()
	validUsername, err := NewUsername(username)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		ID:        userID,
		Username:  validUsername,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ChangeUsername updates the username and timestamp
func (u *User) ChangeUsername(newUsername string) error {
	validUsername, err := NewUsername(newUsername)
	if err != nil {
		return err
	}

	u.Username = validUsername
	u.UpdatedAt = time.Now()
	return nil
}

// UserSession represents user's connection and active rooms
type UserSession struct {
	UserID       UserID    `json:"user_id"`
	ActiveRooms  []string  `json:"active_rooms"`
	LastSeen     time.Time `json:"last_seen"`
	JWTClaims    JWTClaims `json:"jwt_claims"`
	ConnectionID string    `json:"connection_id"`
	Metadata     Metadata  `json:"metadata"`
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	Subject   string `json:"sub"`
	ExpiresAt int64  `json:"exp"`
}

// Metadata represents session metadata
type Metadata struct {
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`
}

// NewUserSession creates a new user session
func NewUserSession(userID UserID, connectionID string) *UserSession {
	return &UserSession{
		UserID:       userID,
		ActiveRooms:  make([]string, 0),
		LastSeen:     time.Now(),
		ConnectionID: connectionID,
		Metadata:     Metadata{},
	}
}

// AddRoom adds a room to active rooms
func (s *UserSession) AddRoom(roomID string) {
	for _, room := range s.ActiveRooms {
		if room == roomID {
			return // Already in room
		}
	}
	s.ActiveRooms = append(s.ActiveRooms, roomID)
	s.UpdateLastSeen()
}

// RemoveRoom removes a room from active rooms
func (s *UserSession) RemoveRoom(roomID string) {
	for i, room := range s.ActiveRooms {
		if room == roomID {
			s.ActiveRooms = append(s.ActiveRooms[:i], s.ActiveRooms[i+1:]...)
			break
		}
	}
	s.UpdateLastSeen()
}

// UpdateLastSeen updates the last seen timestamp
func (s *UserSession) UpdateLastSeen() {
	s.LastSeen = time.Now()
}

// IsInRoom checks if user is in a specific room
func (s *UserSession) IsInRoom(roomID string) bool {
	for _, room := range s.ActiveRooms {
		if room == roomID {
			return true
		}
	}
	return false
}