package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Message represents a chat message entity
type Message struct {
	ID        string     `json:"id" redis:"id"`
	RoomID    string     `json:"room_id" redis:"room_id"`
	UserID    string     `json:"user_id" redis:"user_id"`
	Content   string     `json:"content" redis:"content"`
	CreatedAt time.Time  `json:"created_at" redis:"created_at"`
	EditedAt  *time.Time `json:"edited_at,omitempty" redis:"edited_at"`
}

// NewMessage creates a new Message with validation
func NewMessage(roomID, userID, content string) (*Message, error) {
	if err := validateMessageContent(content); err != nil {
		return nil, err
	}

	if roomID == "" {
		return nil, fmt.Errorf("room_id cannot be empty")
	}

	if userID == "" {
		return nil, fmt.Errorf("user_id cannot be empty")
	}

	return &Message{
		ID:        uuid.New().String(),
		RoomID:    roomID,
		UserID:    userID,
		Content:   strings.TrimSpace(content),
		CreatedAt: time.Now().UTC(),
	}, nil
}

// Validate validates the Message struct
func (m *Message) Validate() error {
	if m.ID == "" {
		return fmt.Errorf("message id cannot be empty")
	}

	if _, err := uuid.Parse(m.ID); err != nil {
		return fmt.Errorf("message id must be a valid UUID: %w", err)
	}

	if m.RoomID == "" {
		return fmt.Errorf("room_id cannot be empty")
	}

	if _, err := uuid.Parse(m.RoomID); err != nil {
		return fmt.Errorf("room_id must be a valid UUID: %w", err)
	}

	if m.UserID == "" {
		return fmt.Errorf("user_id cannot be empty")
	}

	if _, err := uuid.Parse(m.UserID); err != nil {
		return fmt.Errorf("user_id must be a valid UUID: %w", err)
	}

	if err := validateMessageContent(m.Content); err != nil {
		return err
	}

	if m.CreatedAt.IsZero() {
		return fmt.Errorf("created_at cannot be zero")
	}

	if m.CreatedAt.After(time.Now().Add(time.Minute)) {
		return fmt.Errorf("created_at cannot be in the future")
	}

	return nil
}

// Edit updates the message content and sets edited timestamp
func (m *Message) Edit(newContent string) error {
	if err := validateMessageContent(newContent); err != nil {
		return err
	}

	m.Content = strings.TrimSpace(newContent)
	now := time.Now().UTC()
	m.EditedAt = &now

	return nil
}

// validateMessageContent validates message content according to business rules
func validateMessageContent(content string) error {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return fmt.Errorf("message content cannot be empty")
	}

	if len(trimmed) > 1000 {
		return fmt.Errorf("message content cannot exceed 1000 characters")
	}

	return nil
}

// MessageWithUser represents a message with user information for display
type MessageWithUser struct {
	ID        string     `json:"id"`
	RoomID    string     `json:"room_id"`
	UserID    string     `json:"user_id"`
	Username  string     `json:"username"`
	Content   string     `json:"content"`
	CreatedAt time.Time  `json:"created_at"`
	EditedAt  *time.Time `json:"edited_at,omitempty"`
}

// ToMessageWithUser converts a Message to MessageWithUser
func (m *Message) ToMessageWithUser(username string) *MessageWithUser {
	return &MessageWithUser{
		ID:        m.ID,
		RoomID:    m.RoomID,
		UserID:    m.UserID,
		Username:  username,
		Content:   m.Content,
		CreatedAt: m.CreatedAt,
		EditedAt:  m.EditedAt,
	}
}

// StreamMessage represents a message as stored in Redis Streams
type StreamMessage struct {
	StreamID string    `json:"stream_id"` // Redis Stream entry ID (timestamp-sequence)
	UserID   string    `json:"user_id"`
	Username string    `json:"username"`
	Content  string    `json:"content"`
	SentAt   time.Time `json:"sent_at"`
}

// ToStreamMessage converts a Message to StreamMessage for Redis Stream storage
func (m *Message) ToStreamMessage(username string) *StreamMessage {
	return &StreamMessage{
		UserID:   m.UserID,
		Username: username,
		Content:  m.Content,
		SentAt:   m.CreatedAt,
	}
}

// MessageEvent represents a message event for the event stream
type MessageEvent struct {
	Event     string    `json:"event"`
	RoomID    string    `json:"room_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	MessageID string    `json:"message_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// ToMessageEvent converts a Message to MessageEvent
func (m *Message) ToMessageEvent(username string) *MessageEvent {
	return &MessageEvent{
		Event:     "MessageSent",
		RoomID:    m.RoomID,
		UserID:    m.UserID,
		Username:  username,
		MessageID: m.ID,
		Content:   m.Content,
		Timestamp: m.CreatedAt,
	}
}