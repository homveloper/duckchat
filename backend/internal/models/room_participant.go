package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RoomParticipant represents a user's membership in a chat room
type RoomParticipant struct {
	ID       string    `json:"id" redis:"id"`
	RoomID   string    `json:"room_id" redis:"room_id"`
	UserID   string    `json:"user_id" redis:"user_id"`
	JoinedAt time.Time `json:"joined_at" redis:"joined_at"`
	LeftAt   *time.Time `json:"left_at,omitempty" redis:"left_at"`
	IsActive bool      `json:"is_active" redis:"is_active"`
}

// NewRoomParticipant creates a new RoomParticipant
func NewRoomParticipant(roomID, userID string) (*RoomParticipant, error) {
	if roomID == "" {
		return nil, fmt.Errorf("room_id cannot be empty")
	}

	if _, err := uuid.Parse(roomID); err != nil {
		return nil, fmt.Errorf("room_id must be a valid UUID: %w", err)
	}

	if userID == "" {
		return nil, fmt.Errorf("user_id cannot be empty")
	}

	if _, err := uuid.Parse(userID); err != nil {
		return nil, fmt.Errorf("user_id must be a valid UUID: %w", err)
	}

	return &RoomParticipant{
		ID:       uuid.New().String(),
		RoomID:   roomID,
		UserID:   userID,
		JoinedAt: time.Now().UTC(),
		IsActive: true,
	}, nil
}

// Validate validates the RoomParticipant struct
func (rp *RoomParticipant) Validate() error {
	if rp.ID == "" {
		return fmt.Errorf("participant id cannot be empty")
	}

	if _, err := uuid.Parse(rp.ID); err != nil {
		return fmt.Errorf("participant id must be a valid UUID: %w", err)
	}

	if rp.RoomID == "" {
		return fmt.Errorf("room_id cannot be empty")
	}

	if _, err := uuid.Parse(rp.RoomID); err != nil {
		return fmt.Errorf("room_id must be a valid UUID: %w", err)
	}

	if rp.UserID == "" {
		return fmt.Errorf("user_id cannot be empty")
	}

	if _, err := uuid.Parse(rp.UserID); err != nil {
		return fmt.Errorf("user_id must be a valid UUID: %w", err)
	}

	if rp.JoinedAt.IsZero() {
		return fmt.Errorf("joined_at cannot be zero")
	}

	if rp.JoinedAt.After(time.Now().Add(time.Minute)) {
		return fmt.Errorf("joined_at cannot be in the future")
	}

	if rp.LeftAt != nil && rp.LeftAt.Before(rp.JoinedAt) {
		return fmt.Errorf("left_at cannot be before joined_at")
	}

	return nil
}

// Leave marks the participant as having left the room
func (rp *RoomParticipant) Leave() {
	now := time.Now().UTC()
	rp.LeftAt = &now
	rp.IsActive = false
}

// Rejoin marks the participant as active again (for rejoining)
func (rp *RoomParticipant) Rejoin() {
	rp.LeftAt = nil
	rp.IsActive = true
}

// RedisKey returns the Redis key for this participant record
func (rp *RoomParticipant) RedisKey() string {
	return fmt.Sprintf("participant:%s", rp.ID)
}

// ParticipantInfo represents participant information for display
type ParticipantInfo struct {
	UserID   string    `json:"user_id"`
	Username string    `json:"username,omitempty"` // Filled from user service
	JoinedAt time.Time `json:"joined_at"`
	IsActive bool      `json:"is_active"`
}

// ToParticipantInfo converts RoomParticipant to ParticipantInfo
func (rp *RoomParticipant) ToParticipantInfo() *ParticipantInfo {
	return &ParticipantInfo{
		UserID:   rp.UserID,
		JoinedAt: rp.JoinedAt,
		IsActive: rp.IsActive,
	}
}

// ParticipantJoinEvent represents a participant join event
type ParticipantJoinEvent struct {
	Event     string    `json:"event"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
}

// ToJoinEvent converts RoomParticipant to ParticipantJoinEvent
func (rp *RoomParticipant) ToJoinEvent(username string) *ParticipantJoinEvent {
	return &ParticipantJoinEvent{
		Event:     "UserJoinedRoom",
		RoomID:    rp.RoomID,
		UserID:    rp.UserID,
		Username:  username,
		Timestamp: rp.JoinedAt,
	}
}

// ParticipantLeaveEvent represents a participant leave event
type ParticipantLeaveEvent struct {
	Event     string    `json:"event"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	Timestamp time.Time `json:"timestamp"`
}

// ToLeaveEvent converts RoomParticipant to ParticipantLeaveEvent
func (rp *RoomParticipant) ToLeaveEvent(username string) *ParticipantLeaveEvent {
	if rp.LeftAt == nil {
		return nil
	}

	return &ParticipantLeaveEvent{
		Event:     "UserLeftRoom",
		RoomID:    rp.RoomID,
		UserID:    rp.UserID,
		Username:  username,
		Timestamp: *rp.LeftAt,
	}
}