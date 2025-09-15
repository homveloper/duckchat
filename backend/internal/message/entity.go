package message

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Message represents a chat message
type Message struct {
	ID          MessageID   `json:"id"`
	RoomID      string      `json:"room_id"`
	UserID      string      `json:"user_id"`
	Content     string      `json:"content"`
	Timestamp   time.Time   `json:"timestamp"`
	MessageType MessageType `json:"message_type"`
	Metadata    Metadata    `json:"metadata"`
}

// MessageID is a value object for message identification
type MessageID string

// MessageType represents the type of message
type MessageType string

const (
	TypeText   MessageType = "text"
	TypeSystem MessageType = "system"
)

// Metadata represents message metadata
type Metadata struct {
	Client string `json:"client"`
	Edited bool   `json:"edited"`
}

// Domain errors
var (
	ErrInvalidMessageContent = errors.New("message content must be 1-1000 characters for text messages")
	ErrEmptyMessageID       = errors.New("message ID cannot be empty")
	ErrEmptyRoomID          = errors.New("room ID cannot be empty")
	ErrEmptyUserID          = errors.New("user ID cannot be empty")
	ErrInvalidMessageType   = errors.New("invalid message type")
	ErrFutureTimestamp      = errors.New("message timestamp cannot be in the future")
)

// NewMessageID creates a new MessageID using UUID v7 for ordering
func NewMessageID() MessageID {
	uuidV7, _ := uuid.NewV7()
	return MessageID(uuidV7.String())
}

// ParseMessageID creates MessageID from string
func ParseMessageID(id string) (MessageID, error) {
	if id == "" {
		return MessageID(""), ErrEmptyMessageID
	}
	return MessageID(id), nil
}

// String returns string representation of MessageID
func (m MessageID) String() string {
	return string(m)
}

// IsValid checks if MessageType is valid
func (mt MessageType) IsValid() bool {
	return mt == TypeText || mt == TypeSystem
}

// NewMessage creates a new Message with validation
func NewMessage(roomID, userID, content string, messageType MessageType) (*Message, error) {
	if roomID == "" {
		return nil, ErrEmptyRoomID
	}
	if userID == "" {
		return nil, ErrEmptyUserID
	}
	if !messageType.IsValid() {
		return nil, ErrInvalidMessageType
	}

	// Validate content based on message type
	if messageType == TypeText {
		if len(content) < 1 || len(content) > 1000 {
			return nil, ErrInvalidMessageContent
		}
	}

	messageID := NewMessageID()
	timestamp := time.Now()

	return &Message{
		ID:          messageID,
		RoomID:      roomID,
		UserID:      userID,
		Content:     content,
		Timestamp:   timestamp,
		MessageType: messageType,
		Metadata: Metadata{
			Client: "web",
			Edited: false,
		},
	}, nil
}

// NewTextMessage creates a new text message
func NewTextMessage(roomID, userID, content string) (*Message, error) {
	return NewMessage(roomID, userID, content, TypeText)
}

// NewSystemMessage creates a new system message
func NewSystemMessage(roomID, userID, content string) (*Message, error) {
	return NewMessage(roomID, userID, content, TypeSystem)
}

// IsText checks if message is a text message
func (m *Message) IsText() bool {
	return m.MessageType == TypeText
}

// IsSystem checks if message is a system message
func (m *Message) IsSystem() bool {
	return m.MessageType == TypeSystem
}

// Age returns the age of the message
func (m *Message) Age() time.Duration {
	return time.Since(m.Timestamp)
}

// IsRecent checks if message was sent within the last minute
func (m *Message) IsRecent() bool {
	return m.Age() < time.Minute
}

// Validate performs additional validation on the message
func (m *Message) Validate() error {
	if m.RoomID == "" {
		return ErrEmptyRoomID
	}
	if m.UserID == "" {
		return ErrEmptyUserID
	}
	if !m.MessageType.IsValid() {
		return ErrInvalidMessageType
	}
	if m.Timestamp.After(time.Now()) {
		return ErrFutureTimestamp
	}
	if m.MessageType == TypeText && (len(m.Content) < 1 || len(m.Content) > 1000) {
		return ErrInvalidMessageContent
	}
	return nil
}

// MessageHistory represents a collection of messages with pagination
type MessageHistory struct {
	Messages []Message `json:"messages"`
	HasMore  bool      `json:"has_more"`
	NextID   string    `json:"next_id,omitempty"`
}

// NewMessageHistory creates a new MessageHistory
func NewMessageHistory(messages []Message, hasMore bool) *MessageHistory {
	return &MessageHistory{
		Messages: messages,
		HasMore:  hasMore,
	}
}

// Add adds a message to the history
func (mh *MessageHistory) Add(message Message) {
	mh.Messages = append(mh.Messages, message)
}

// Count returns the number of messages
func (mh *MessageHistory) Count() int {
	return len(mh.Messages)
}

// IsEmpty checks if history is empty
func (mh *MessageHistory) IsEmpty() bool {
	return len(mh.Messages) == 0
}