package models

import (
	"time"
	"fmt"
	"strings"
	"github.com/google/uuid"
)

// ChatRoom represents a chat room entity stored in Redis
type ChatRoom struct {
	ID               string    `json:"id" redis:"id"`
	Title            string    `json:"title" redis:"title"`
	CreatedAt        time.Time `json:"created_at" redis:"created_at"`
	CreatedBy        string    `json:"created_by" redis:"created_by"`
	IsActive         bool      `json:"is_active" redis:"is_active"`
	ParticipantCount int       `json:"participant_count,omitempty" redis:"participant_count"`
	LastMessageAt    *time.Time `json:"last_message_at,omitempty" redis:"last_message_at"`
}

// NewChatRoom creates a new ChatRoom with validation
func NewChatRoom(title, createdBy string) (*ChatRoom, error) {
	if err := validateRoomTitle(title); err != nil {
		return nil, err
	}

	if createdBy == "" {
		return nil, fmt.Errorf("created_by cannot be empty")
	}

	return &ChatRoom{
		ID:               uuid.New().String(),
		Title:            strings.TrimSpace(title),
		CreatedAt:        time.Now().UTC(),
		CreatedBy:        createdBy,
		IsActive:         true,
		ParticipantCount: 1, // Creator is automatically a participant
	}, nil
}

// Validate validates the ChatRoom struct
func (r *ChatRoom) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("room id cannot be empty")
	}

	if _, err := uuid.Parse(r.ID); err != nil {
		return fmt.Errorf("room id must be a valid UUID: %w", err)
	}

	if err := validateRoomTitle(r.Title); err != nil {
		return err
	}

	if r.CreatedBy == "" {
		return fmt.Errorf("created_by cannot be empty")
	}

	if r.CreatedAt.IsZero() {
		return fmt.Errorf("created_at cannot be zero")
	}

	if r.CreatedAt.After(time.Now().Add(time.Minute)) {
		return fmt.Errorf("created_at cannot be in the future")
	}

	if r.ParticipantCount < 0 {
		return fmt.Errorf("participant_count cannot be negative")
	}

	return nil
}

// RedisKey returns the Redis key for this room
func (r *ChatRoom) RedisKey() string {
	return fmt.Sprintf("room:%s", r.ID)
}

// ParticipantsRedisKey returns the Redis key for room participants
func (r *ChatRoom) ParticipantsRedisKey() string {
	return fmt.Sprintf("room:%s:participants", r.ID)
}

// MessagesRedisStreamKey returns the Redis Stream key for room messages
func (r *ChatRoom) MessagesRedisStreamKey() string {
	return fmt.Sprintf("room:%s:messages", r.ID)
}

// SSEConnectionsRedisKey returns the Redis Set key for SSE connections
func (r *ChatRoom) SSEConnectionsRedisKey() string {
	return fmt.Sprintf("sse:room:%s", r.ID)
}

// UpdateLastMessage updates the last message timestamp
func (r *ChatRoom) UpdateLastMessage() {
	now := time.Now().UTC()
	r.LastMessageAt = &now
}

// validateRoomTitle validates room title according to business rules
func validateRoomTitle(title string) error {
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return fmt.Errorf("room title cannot be empty")
	}

	if len(trimmed) < 3 {
		return fmt.Errorf("room title must be at least 3 characters")
	}

	if len(trimmed) > 100 {
		return fmt.Errorf("room title cannot exceed 100 characters")
	}

	return nil
}

// ChatRoomSummary represents a lightweight room view for listings
type ChatRoomSummary struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	CreatedAt        time.Time  `json:"created_at"`
	ParticipantCount int        `json:"participant_count"`
	LastMessageAt    *time.Time `json:"last_message_at,omitempty"`
	IsActive         bool       `json:"is_active"`
}

// ToSummary converts a ChatRoom to ChatRoomSummary
func (r *ChatRoom) ToSummary() *ChatRoomSummary {
	return &ChatRoomSummary{
		ID:               r.ID,
		Title:            r.Title,
		CreatedAt:        r.CreatedAt,
		ParticipantCount: r.ParticipantCount,
		LastMessageAt:    r.LastMessageAt,
		IsActive:         r.IsActive,
	}
}