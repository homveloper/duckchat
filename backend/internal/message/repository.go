package message

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Repository interface for message data operations
type Repository interface {
	Save(ctx context.Context, message *Message) error
	FindByID(ctx context.Context, messageID MessageID) (*Message, error)
	GetRoomHistory(ctx context.Context, roomID string, limit int, before *time.Time) (*MessageHistory, error)
	GetRecentMessages(ctx context.Context, roomID string, limit int) ([]Message, error)
}

// redisRepository implements Repository using Redis JSON and Streams
type redisRepository struct {
	client *redis.Client
}

// NewRepository creates a new Redis-based message repository
func NewRepository(client *redis.Client) Repository {
	return &redisRepository{client: client}
}

// Save stores a message in Redis JSON and adds to stream for ordering
func (r *redisRepository) Save(ctx context.Context, message *Message) error {
	// Validate message
	if err := message.Validate(); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	// Store message as JSON document
	messageKey := fmt.Sprintf("message:%s", message.ID.String())
	messageJSON, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	err = r.client.JSONSet(ctx, messageKey, "$", string(messageJSON)).Err()
	if err != nil {
		return fmt.Errorf("failed to save message to Redis: %w", err)
	}

	// Add message ID to room stream for ordering
	streamKey := fmt.Sprintf("messages:%s", message.RoomID)
	_, err = r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"message_id": message.ID.String(),
			"timestamp":  message.Timestamp.Unix(),
		},
	}).Result()

	if err != nil {
		return fmt.Errorf("failed to add message to stream: %w", err)
	}

	return nil
}

// FindByID retrieves a message by ID from Redis
func (r *redisRepository) FindByID(ctx context.Context, messageID MessageID) (*Message, error) {
	key := fmt.Sprintf("message:%s", messageID.String())

	result, err := r.client.JSONGet(ctx, key, "$").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("message not found: %s", messageID.String())
		}
		return nil, fmt.Errorf("failed to get message from Redis: %w", err)
	}

	var messages []Message
	err = json.Unmarshal([]byte(result), &messages)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	if len(messages) == 0 {
		return nil, fmt.Errorf("message not found: %s", messageID.String())
	}

	return &messages[0], nil
}

// GetRoomHistory retrieves message history for a room with pagination
func (r *redisRepository) GetRoomHistory(ctx context.Context, roomID string, limit int, before *time.Time) (*MessageHistory, error) {
	streamKey := fmt.Sprintf("messages:%s", roomID)

	// Determine the start point for pagination
	start := "-"
	if before != nil {
		// Convert timestamp to stream ID format
		start = fmt.Sprintf("%d-0", before.Unix()*1000)
	}

	// Get message IDs from stream
	streams, err := r.client.XRevRangeN(ctx, streamKey, "+", start, int64(limit+1)).Result()
	if err != nil {
		if err == redis.Nil {
			return NewMessageHistory([]Message{}, false), nil
		}
		return nil, fmt.Errorf("failed to get message stream: %w", err)
	}

	// Check if there are more messages
	hasMore := len(streams) > limit
	if hasMore {
		streams = streams[:limit] // Remove the extra message
	}

	// Get actual messages from JSON documents
	messages := make([]Message, 0, len(streams))
	for _, stream := range streams {
		if messageIDVal, exists := stream.Values["message_id"]; exists {
			messageID := messageIDVal.(string)

			message, err := r.FindByID(ctx, MessageID(messageID))
			if err != nil {
				// Log error but continue with other messages
				continue
			}

			messages = append(messages, *message)
		}
	}

	return NewMessageHistory(messages, hasMore), nil
}

// GetRecentMessages gets the most recent messages for a room
func (r *redisRepository) GetRecentMessages(ctx context.Context, roomID string, limit int) ([]Message, error) {
	streamKey := fmt.Sprintf("messages:%s", roomID)

	// Get recent message IDs from stream
	streams, err := r.client.XRevRangeN(ctx, streamKey, "+", "-", int64(limit)).Result()
	if err != nil {
		if err == redis.Nil {
			return []Message{}, nil
		}
		return nil, fmt.Errorf("failed to get recent messages stream: %w", err)
	}

	// Get actual messages from JSON documents
	messages := make([]Message, 0, len(streams))
	for i := len(streams) - 1; i >= 0; i-- { // Reverse to get chronological order
		stream := streams[i]
		if messageIDVal, exists := stream.Values["message_id"]; exists {
			messageID := messageIDVal.(string)

			message, err := r.FindByID(ctx, MessageID(messageID))
			if err != nil {
				// Log error but continue with other messages
				continue
			}

			messages = append(messages, *message)
		}
	}

	return messages, nil
}

// GetMessagesByUser gets messages sent by a specific user in a room
func (r *redisRepository) GetMessagesByUser(ctx context.Context, roomID, userID string, limit int) ([]Message, error) {
	// This is a simplified version - in production, you might want to maintain separate indexes
	messages, err := r.GetRecentMessages(ctx, roomID, limit*2) // Get more to filter
	if err != nil {
		return nil, err
	}

	var userMessages []Message
	for _, message := range messages {
		if message.UserID == userID {
			userMessages = append(userMessages, message)
			if len(userMessages) >= limit {
				break
			}
		}
	}

	return userMessages, nil
}

// DeleteMessage removes a message (for admin operations)
func (r *redisRepository) DeleteMessage(ctx context.Context, messageID MessageID) error {
	key := fmt.Sprintf("message:%s", messageID.String())

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	// Note: This doesn't remove from stream for simplicity in MVP
	// In production, you might want to mark as deleted instead
	return nil
}

// GetRoomMessageCount gets the total number of messages in a room
func (r *redisRepository) GetRoomMessageCount(ctx context.Context, roomID string) (int64, error) {
	streamKey := fmt.Sprintf("messages:%s", roomID)

	count, err := r.client.XLen(ctx, streamKey).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get message count: %w", err)
	}

	return count, nil
}
