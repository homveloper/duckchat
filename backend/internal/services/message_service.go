package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"duckchat/internal/models"
)

// MessageService handles business logic for chat messages using Redis Streams
type MessageService struct {
	client *redis.Client
}

// NewMessageService creates a new MessageService
func NewMessageService(client *redis.Client) *MessageService {
	return &MessageService{
		client: client,
	}
}

// SendMessageRequest represents the request to send a message
type SendMessageRequest struct {
	RoomID  string `json:"room_id" validate:"required,uuid"`
	UserID  string `json:"user_id" validate:"required,uuid"`
	Content string `json:"content" validate:"required,min=1,max=1000"`
}

// SendMessageResponse represents the response after sending a message
type SendMessageResponse struct {
	StreamID  string                `json:"stream_id"`  // Redis Stream ID (timestamp-sequence)
	Message   *models.MessageWithUser `json:"message"`
	Timestamp time.Time             `json:"timestamp"`
}

// SendMessage sends a message to a room's Redis Stream
func (s *MessageService) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	// Validate request
	if err := s.validateSendMessageRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create message model for validation
	message, err := models.NewMessage(req.RoomID, req.UserID, req.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to create message model: %w", err)
	}

	// Get Redis Stream key for the room
	streamKey := fmt.Sprintf("room:%s:messages", req.RoomID)

	// Prepare message data for Redis Stream
	messageData := map[string]interface{}{
		"user_id":   req.UserID,
		"content":   req.Content,
		"timestamp": message.CreatedAt.Unix(),
	}

	// Add message to Redis Stream
	result := s.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: messageData,
	})

	streamID, err := result.Result()
	if err != nil {
		return nil, fmt.Errorf("failed to add message to stream: %w", err)
	}

	// Create response with stream message
	messageWithUser := message.ToMessageWithUser("")

	return &SendMessageResponse{
		StreamID:  streamID,
		Message:   messageWithUser,
		Timestamp: message.CreatedAt,
	}, nil
}

// GetMessagesRequest represents the request to get message history
type GetMessagesRequest struct {
	RoomID        string  `json:"room_id" validate:"required,uuid"`
	Limit         int     `json:"limit" validate:"min=1,max=100"`
	LastMessageID *string `json:"last_message_id,omitempty"` // For pagination
	Direction     string  `json:"direction" validate:"oneof=newer older"` // "newer" or "older"
}

// GetMessagesResponse represents the response with message history
type GetMessagesResponse struct {
	Messages       []*models.StreamMessage `json:"messages"`
	HasMore        bool                    `json:"has_more"`
	LastMessageID  string                  `json:"last_message_id,omitempty"`
	FirstMessageID string                  `json:"first_message_id,omitempty"`
}

// GetMessages retrieves message history from Redis Stream with cursor-based pagination
func (s *MessageService) GetMessages(ctx context.Context, req *GetMessagesRequest) (*GetMessagesResponse, error) {
	// Validate request
	if err := s.validateGetMessagesRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get Redis Stream key for the room
	streamKey := fmt.Sprintf("room:%s:messages", req.RoomID)

	var messages []*models.StreamMessage
	var lastMessageID, firstMessageID string
	var hasMore bool
	var err error

	if req.Direction == "newer" || req.Direction == "" {
		// Get newer messages (default direction)
		messages, lastMessageID, firstMessageID, hasMore, err = s.getNewerMessages(ctx, streamKey, req)
	} else {
		// Get older messages (for scrolling back in history)
		messages, lastMessageID, firstMessageID, hasMore, err = s.getOlderMessages(ctx, streamKey, req)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}

	return &GetMessagesResponse{
		Messages:       messages,
		HasMore:        hasMore,
		LastMessageID:  lastMessageID,
		FirstMessageID: firstMessageID,
	}, nil
}

// getNewerMessages gets messages newer than the specified cursor
func (s *MessageService) getNewerMessages(ctx context.Context, streamKey string, req *GetMessagesRequest) ([]*models.StreamMessage, string, string, bool, error) {
	// Determine start position
	start := "-" // From beginning if no cursor
	if req.LastMessageID != nil && *req.LastMessageID != "" {
		// Get messages after the cursor (exclusive)
		start = fmt.Sprintf("(%s", *req.LastMessageID) // "(" makes it exclusive
	}

	// Read messages from stream
	result := s.client.XRead(ctx, &redis.XReadArgs{
		Streams: []string{streamKey, start},
		Count:   int64(req.Limit + 1), // Get one extra to check if there are more
		Block:   0, // Don't block
	})

	streams, err := result.Result()
	if err != nil {
		if err == redis.Nil {
			return []*models.StreamMessage{}, "", "", false, nil
		}
		return nil, "", "", false, fmt.Errorf("failed to read from stream: %w", err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return []*models.StreamMessage{}, "", "", false, nil
	}

	streamMessages := streams[0].Messages
	messages := make([]*models.StreamMessage, 0)

	// Process messages (take only the requested limit)
	limit := req.Limit
	if len(streamMessages) > limit {
		streamMessages = streamMessages[:limit]
	}

	var lastMessageID, firstMessageID string
	for i, msg := range streamMessages {
		streamMessage, err := s.parseStreamMessage(msg)
		if err != nil {
			continue // Skip invalid messages
		}

		messages = append(messages, streamMessage)

		if i == 0 {
			firstMessageID = msg.ID
		}
		lastMessageID = msg.ID
	}

	// Check if there are more messages
	hasMore := len(streams[0].Messages) > req.Limit

	return messages, lastMessageID, firstMessageID, hasMore, nil
}

// getOlderMessages gets messages older than the specified cursor
func (s *MessageService) getOlderMessages(ctx context.Context, streamKey string, req *GetMessagesRequest) ([]*models.StreamMessage, string, string, bool, error) {
	// Determine range for older messages
	end := "+" // To end if no cursor
	if req.LastMessageID != nil && *req.LastMessageID != "" {
		// Get messages before the cursor (exclusive)
		end = fmt.Sprintf("(%s", *req.LastMessageID)
	}

	// Read messages in reverse order using XREVRANGE
	result := s.client.XRevRange(ctx, streamKey, end, "-")

	allMessages, err := result.Result()
	if err != nil {
		if err == redis.Nil {
			return []*models.StreamMessage{}, "", "", false, nil
		}
		return nil, "", "", false, fmt.Errorf("failed to read from stream: %w", err)
	}

	if len(allMessages) == 0 {
		return []*models.StreamMessage{}, "", "", false, nil
	}

	// Take only the requested limit
	limit := req.Limit
	messages := make([]*models.StreamMessage, 0)

	count := 0
	var lastMessageID, firstMessageID string

	for i, msg := range allMessages {
		if count >= limit {
			break
		}

		streamMessage, err := s.parseStreamMessage(msg)
		if err != nil {
			continue // Skip invalid messages
		}

		messages = append(messages, streamMessage)
		count++

		if i == 0 {
			firstMessageID = msg.ID
		}
		lastMessageID = msg.ID
	}

	// Check if there are more messages
	hasMore := len(allMessages) > req.Limit

	return messages, lastMessageID, firstMessageID, hasMore, nil
}

// parseStreamMessage converts Redis Stream message to StreamMessage model
func (s *MessageService) parseStreamMessage(msg redis.XMessage) (*models.StreamMessage, error) {
	streamMessage := &models.StreamMessage{
		StreamID: msg.ID,
	}

	// Parse fields
	if userID, ok := msg.Values["user_id"].(string); ok {
		streamMessage.UserID = userID
	}

	if username, ok := msg.Values["username"].(string); ok {
		streamMessage.Username = username
	}

	if content, ok := msg.Values["content"].(string); ok {
		streamMessage.Content = content
	}

	if timestampStr, ok := msg.Values["timestamp"].(string); ok {
		if timestamp, err := strconv.ParseInt(timestampStr, 10, 64); err == nil {
			streamMessage.SentAt = time.Unix(timestamp, 0)
		}
	}

	return streamMessage, nil
}

// GetStreamInfoRequest represents the request to get stream information
type GetStreamInfoRequest struct {
	RoomID string `json:"room_id" validate:"required,uuid"`
}

// GetStreamInfoResponse represents stream information
type GetStreamInfoResponse struct {
	RoomID       string `json:"room_id"`
	StreamKey    string `json:"stream_key"`
	MessageCount int64  `json:"message_count"`
	FirstID      string `json:"first_id,omitempty"`
	LastID       string `json:"last_id,omitempty"`
}

// GetStreamInfo retrieves information about a room's message stream
func (s *MessageService) GetStreamInfo(ctx context.Context, req *GetStreamInfoRequest) (*GetStreamInfoResponse, error) {
	if err := s.validateGetStreamInfoRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	streamKey := fmt.Sprintf("room:%s:messages", req.RoomID)

	// Get stream info
	result := s.client.XInfoStream(ctx, streamKey)
	info, err := result.Result()
	if err != nil {
		if err == redis.Nil {
			// Stream doesn't exist yet
			return &GetStreamInfoResponse{
				RoomID:       req.RoomID,
				StreamKey:    streamKey,
				MessageCount: 0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	return &GetStreamInfoResponse{
		RoomID:       req.RoomID,
		StreamKey:    streamKey,
		MessageCount: info.Length,
		FirstID:      info.FirstEntry.ID,
		LastID:       info.LastEntry.ID,
	}, nil
}

// Validation methods

func (s *MessageService) validateSendMessageRequest(req *SendMessageRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.RoomID == "" {
		return fmt.Errorf("room_id is required")
	}

	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}


	if req.Content == "" {
		return fmt.Errorf("content is required")
	}

	if len(req.Content) > 1000 {
		return fmt.Errorf("content cannot exceed 1000 characters")
	}

	return nil
}

func (s *MessageService) validateGetMessagesRequest(req *GetMessagesRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.RoomID == "" {
		return fmt.Errorf("room_id is required")
	}

	if req.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0")
	}

	if req.Limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}

	if req.Direction != "" && req.Direction != "newer" && req.Direction != "older" {
		return fmt.Errorf("direction must be 'newer' or 'older'")
	}

	return nil
}

func (s *MessageService) validateGetStreamInfoRequest(req *GetStreamInfoRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.RoomID == "" {
		return fmt.Errorf("room_id is required")
	}

	return nil
}