package message

import (
	"context"
	"fmt"
	"time"

	"duckchat/internal/room"
)

// Service handles message business logic and coordinates repository operations
type Service struct {
	repo        Repository
	roomService *room.Service
}

// NewService creates a new message service
func NewService(repo Repository, roomService *room.Service) *Service {
	return &Service{
		repo:        repo,
		roomService: roomService,
	}
}

// SendMessage creates and sends a new message
func (s *Service) SendMessage(ctx context.Context, roomID, userID, content string) (*Message, error) {
	// Create message entity with validation
	message, err := NewTextMessage(roomID, userID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Save to repository
	err = s.repo.Save(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	return message, nil
}

// SendSystemMessage creates and sends a system message
func (s *Service) SendSystemMessage(ctx context.Context, roomID, userID, content string) (*Message, error) {
	// Create system message entity
	message, err := NewSystemMessage(roomID, userID, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create system message: %w", err)
	}

	// Save to repository
	err = s.repo.Save(ctx, message)
	if err != nil {
		return nil, fmt.Errorf("failed to save system message: %w", err)
	}

	return message, nil
}

// GetMessage retrieves a message by ID
func (s *Service) GetMessage(ctx context.Context, messageID MessageID) (*Message, error) {
	message, err := s.repo.FindByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return message, nil
}

// GetRoomHistory retrieves message history for a room with pagination
func (s *Service) GetRoomHistory(ctx context.Context, roomID string, limit int, before *time.Time) (*MessageHistory, error) {
	if limit <= 0 || limit > 100 {
		limit = 50 // Default limit
	}

	history, err := s.repo.GetRoomHistory(ctx, roomID, limit, before)
	if err != nil {
		return nil, fmt.Errorf("failed to get room history: %w", err)
	}

	return history, nil
}

// GetRecentMessages gets the most recent messages for a room
func (s *Service) GetRecentMessages(ctx context.Context, roomID string, limit int) ([]Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50 // Default limit
	}

	messages, err := s.repo.GetRecentMessages(ctx, roomID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}

	return messages, nil
}

// GetMessagesByUser gets messages sent by a specific user in a room
func (s *Service) GetMessagesByUser(ctx context.Context, roomID, userID string, limit int) ([]Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50 // Default limit
	}

	// Note: This requires a method in the repository
	// For now, we'll get recent messages and filter
	allMessages, err := s.repo.GetRecentMessages(ctx, roomID, limit*2)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	var userMessages []Message
	for _, message := range allMessages {
		if message.UserID == userID {
			userMessages = append(userMessages, message)
			if len(userMessages) >= limit {
				break
			}
		}
	}

	return userMessages, nil
}

// ValidateMessageRequest validates a message send request
func (s *Service) ValidateMessageRequest(ctx context.Context, roomID, userID, content string) error {
	if roomID == "" {
		return fmt.Errorf("room ID is required")
	}
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}
	if content == "" {
		return fmt.Errorf("message content is required")
	}
	if len(content) > 1000 {
		return fmt.Errorf("message content too long (max 1000 characters)")
	}

	// Validate that room exists
	parsedRoomID, err := room.ParseRoomID(roomID)
	if err != nil {
		return fmt.Errorf("invalid room ID format: %w", err)
	}

	_, err = s.roomService.GetRoom(ctx, parsedRoomID)
	if err != nil {
		return fmt.Errorf("room does not exist")
	}

	return nil
}

// ValidateRoomExists checks if a room exists (helper for room validation)
func (s *Service) ValidateRoomExists(ctx context.Context, roomID string) error {
	parsedRoomID, err := room.ParseRoomID(roomID)
	if err != nil {
		return fmt.Errorf("invalid room ID format: %w", err)
	}

	_, err = s.roomService.GetRoom(ctx, parsedRoomID)
	if err != nil {
		return fmt.Errorf("room does not exist")
	}

	return nil
}

// MessageResponse represents a message for API responses
type MessageResponse struct {
	MessageID   string `json:"message_id"`
	RoomID      string `json:"room_id"`
	UserID      string `json:"user_id"`
	Content     string `json:"content"`
	Timestamp   string `json:"timestamp"`
	MessageType string `json:"message_type"`
	IsRecent    bool   `json:"is_recent"`
}

// ToMessageResponse converts a Message to MessageResponse
func (s *Service) ToMessageResponse(message *Message) *MessageResponse {
	return &MessageResponse{
		MessageID:   message.ID.String(),
		RoomID:      message.RoomID,
		UserID:      message.UserID,
		Content:     message.Content,
		Timestamp:   message.Timestamp.Format("2006-01-02T15:04:05Z07:00"),
		MessageType: string(message.MessageType),
		IsRecent:    message.IsRecent(),
	}
}

// SendMessageRequest represents a message send request
type SendMessageRequest struct {
	RoomID  string `json:"room_id"`
	Content string `json:"content"`
}

// GetHistoryRequest represents a message history request
type GetHistoryRequest struct {
	RoomID string     `json:"room_id"`
	Limit  int        `json:"limit,omitempty"`
	Before *time.Time `json:"before,omitempty"`
}

// MessageHistoryResponse represents message history for API responses
type MessageHistoryResponse struct {
	Messages []MessageResponse `json:"messages"`
	HasMore  bool              `json:"has_more"`
}

// ToMessageHistoryResponse converts MessageHistory to MessageHistoryResponse
func (s *Service) ToMessageHistoryResponse(history *MessageHistory) *MessageHistoryResponse {
	responses := make([]MessageResponse, len(history.Messages))
	for i, message := range history.Messages {
		responses[i] = *s.ToMessageResponse(&message)
	}

	return &MessageHistoryResponse{
		Messages: responses,
		HasMore:  history.HasMore,
	}
}

// CreateUserJoinMessage creates a system message for user joining
func (s *Service) CreateUserJoinMessage(ctx context.Context, roomID, userID, username string) (*Message, error) {
	content := fmt.Sprintf("%s joined the room", username)
	return s.SendSystemMessage(ctx, roomID, userID, content)
}

// CreateUserLeaveMessage creates a system message for user leaving
func (s *Service) CreateUserLeaveMessage(ctx context.Context, roomID, userID, username string) (*Message, error) {
	content := fmt.Sprintf("%s left the room", username)
	return s.SendSystemMessage(ctx, roomID, userID, content)
}

// CreateUsernameChangeMessage creates a system message for username change
func (s *Service) CreateUsernameChangeMessage(ctx context.Context, roomID, userID, oldUsername, newUsername string) (*Message, error) {
	content := fmt.Sprintf("%s changed their name to %s", oldUsername, newUsername)
	return s.SendSystemMessage(ctx, roomID, userID, content)
}

// GetRoomMessageCount gets the total number of messages in a room
func (s *Service) GetRoomMessageCount(ctx context.Context, roomID string) (int64, error) {
	// This would require adding the method to the repository interface
	// For now, we'll return 0 as a placeholder
	return 0, nil
}