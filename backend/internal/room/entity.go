package room

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// ChatRoom represents a conversation space
type ChatRoom struct {
	ID               RoomID    `json:"id"`
	Name             string    `json:"name"`
	CreatedAt        time.Time `json:"created_at"`
	CreatedBy        string    `json:"created_by"`
	ParticipantCount int       `json:"participant_count"`
	LastActivity     time.Time `json:"last_activity"`
	Metadata         Metadata  `json:"metadata"`
}

// RoomID is a value object for room identification
type RoomID string

// Metadata represents room metadata
type Metadata struct {
	MaxParticipants int    `json:"max_participants"`
	CreatedVia      string `json:"created_via"`
}

// Domain errors
var (
	ErrInvalidRoomName         = errors.New("room name must be 1-100 characters")
	ErrEmptyRoomID            = errors.New("room ID cannot be empty")
	ErrNegativeParticipants   = errors.New("participant count cannot be negative")
	ErrMaxParticipantsReached = errors.New("maximum participants reached")
)

// Constants
const (
	DefaultMaxParticipants = 50
)

// NewRoomID creates a new RoomID
func NewRoomID() RoomID {
	return RoomID(uuid.New().String())
}

// ParseRoomID creates RoomID from string
func ParseRoomID(id string) (RoomID, error) {
	if id == "" {
		return RoomID(""), ErrEmptyRoomID
	}
	return RoomID(id), nil
}

// String returns string representation of RoomID
func (r RoomID) String() string {
	return string(r)
}

// NewChatRoom creates a new ChatRoom with validation
func NewChatRoom(name, createdBy string) (*ChatRoom, error) {
	if len(name) < 1 || len(name) > 100 {
		return nil, ErrInvalidRoomName
	}

	roomID := NewRoomID()
	now := time.Now()

	return &ChatRoom{
		ID:               roomID,
		Name:             name,
		CreatedAt:        now,
		CreatedBy:        createdBy,
		ParticipantCount: 1, // Creator is the first participant
		LastActivity:     now,
		Metadata: Metadata{
			MaxParticipants: DefaultMaxParticipants,
			CreatedVia:      "web",
		},
	}, nil
}

// AddParticipant increments participant count with validation
func (r *ChatRoom) AddParticipant() error {
	if r.ParticipantCount >= r.Metadata.MaxParticipants {
		return ErrMaxParticipantsReached
	}

	r.ParticipantCount++
	r.UpdateActivity()
	return nil
}

// RemoveParticipant decrements participant count
func (r *ChatRoom) RemoveParticipant() error {
	if r.ParticipantCount <= 0 {
		return ErrNegativeParticipants
	}

	r.ParticipantCount--
	r.UpdateActivity()
	return nil
}

// UpdateActivity updates the last activity timestamp
func (r *ChatRoom) UpdateActivity() {
	r.LastActivity = time.Now()
}

// IsActive checks if room has recent activity (within 1 hour)
func (r *ChatRoom) IsActive() bool {
	return time.Since(r.LastActivity) < time.Hour
}

// CanJoin checks if a user can join the room
func (r *ChatRoom) CanJoin() bool {
	return r.ParticipantCount < r.Metadata.MaxParticipants
}

// RoomState represents the current state of a room
type RoomState int

const (
	StateCreated RoomState = iota
	StateActive
	StateInactive
)

// GetState returns the current state of the room
func (r *ChatRoom) GetState() RoomState {
	if r.ParticipantCount == 0 {
		if time.Since(r.CreatedAt) < time.Minute {
			return StateCreated
		}
		return StateInactive
	}
	return StateActive
}

// String returns the string representation of RoomState
func (s RoomState) String() string {
	switch s {
	case StateCreated:
		return "created"
	case StateActive:
		return "active"
	case StateInactive:
		return "inactive"
	default:
		return "unknown"
	}
}

// MarshalJSON implements the json.Marshaler interface
func (s RoomState) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}