package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"duckchat/internal/models"
)

// ChatRoomRepository handles Redis operations for chat rooms
type ChatRoomRepository struct {
	client *redis.Client
}

// NewChatRoomRepository creates a new ChatRoomRepository
func NewChatRoomRepository(client *redis.Client) *ChatRoomRepository {
	return &ChatRoomRepository{
		client: client,
	}
}

// Create creates a new chat room in Redis
func (r *ChatRoomRepository) Create(ctx context.Context, room *models.ChatRoom) error {
	if err := room.Validate(); err != nil {
		return fmt.Errorf("invalid room: %w", err)
	}

	// Marshal room to JSON
	roomData, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("failed to marshal room: %w", err)
	}

	// Store room data using RedisJSON
	key := room.RedisKey()
	result := r.client.Do(ctx, "JSON.SET", key, "$", string(roomData))
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to create room in Redis: %w", err)
	}

	// Add room to global room list for easy lookup
	roomListKey := "rooms:list"
	roomSummary := map[string]interface{}{
		"id":                room.ID,
		"title":             room.Title,
		"created_at":        room.CreatedAt.Format(time.RFC3339),
		"participant_count": room.ParticipantCount,
		"is_active":         room.IsActive,
	}

	if room.LastMessageAt != nil {
		roomSummary["last_message_at"] = room.LastMessageAt.Format(time.RFC3339)
	}

	summaryData, err := json.Marshal(roomSummary)
	if err != nil {
		return fmt.Errorf("failed to marshal room summary: %w", err)
	}

	// Store room summary in the list
	result = r.client.Do(ctx, "JSON.SET", roomListKey, fmt.Sprintf("$.%s", room.ID), summaryData)
	if err := result.Err(); err != nil {
		// If the rooms list doesn't exist, create it
		if err.Error() == "ERR Path '$.' does not exist" || err.Error() == "ERR new objects must be created at the root" {
			emptyList := "{}"
			r.client.Do(ctx, "JSON.SET", roomListKey, "$", emptyList)
			result = r.client.Do(ctx, "JSON.SET", roomListKey, fmt.Sprintf("$.%s", room.ID), summaryData)
			if err := result.Err(); err != nil {
				return fmt.Errorf("failed to add room to list: %w", err)
			}
		} else {
			return fmt.Errorf("failed to add room to list: %w", err)
		}
	}

	return nil
}

// GetByID retrieves a chat room by ID
func (r *ChatRoomRepository) GetByID(ctx context.Context, roomID string) (*models.ChatRoom, error) {
	key := fmt.Sprintf("room:%s", roomID)

	result := r.client.Do(ctx, "JSON.GET", key, "$")
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("room not found: %s", roomID)
		}
		return nil, fmt.Errorf("failed to get room from Redis: %w", err)
	}

	// Parse the JSON array response (RedisJSON returns an array)
	var jsonArray []json.RawMessage
	if err := json.Unmarshal([]byte(result.Val().(string)), &jsonArray); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON array response: %w", err)
	}

	if len(jsonArray) == 0 {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	// Parse the room data from the first element
	var room models.ChatRoom
	if err := json.Unmarshal(jsonArray[0], &room); err != nil {
		return nil, fmt.Errorf("failed to unmarshal room data: %w", err)
	}

	return &room, nil
}

// List retrieves all active chat rooms with optional filtering
func (r *ChatRoomRepository) List(ctx context.Context, isActive *bool, limit int, offset int) ([]*models.ChatRoomSummary, error) {
	roomListKey := "rooms:list"

	result := r.client.Do(ctx, "JSON.GET", roomListKey, "$")
	if err := result.Err(); err != nil {
		if err == redis.Nil {
			return []*models.ChatRoomSummary{}, nil // Return empty list if no rooms exist
		}
		return nil, fmt.Errorf("failed to get room list from Redis: %w", err)
	}

	resultString := result.Val().(string)

	// Parse the JSON array response (RedisJSON with $ returns an array)
	var jsonArray []json.RawMessage
	if err := json.Unmarshal([]byte(resultString), &jsonArray); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON array response: %w", err)
	}

	if len(jsonArray) == 0 {
		return []*models.ChatRoomSummary{}, nil
	}

	// Parse the rooms data from the first element
	var roomsMap map[string]interface{}
	if err := json.Unmarshal(jsonArray[0], &roomsMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rooms data: %w", err)
	}

	rooms := make([]*models.ChatRoomSummary, 0)
	for _, roomData := range roomsMap {
		roomBytes, err := json.Marshal(roomData)
		if err != nil {
			continue
		}

		var room models.ChatRoomSummary
		if err := json.Unmarshal(roomBytes, &room); err != nil {
			continue
		}

		// Apply active filter if specified
		if isActive != nil && room.IsActive != *isActive {
			continue
		}

		rooms = append(rooms, &room)
	}

	// Apply pagination
	if offset >= len(rooms) {
		return []*models.ChatRoomSummary{}, nil
	}

	end := offset + limit
	if end > len(rooms) {
		end = len(rooms)
	}

	return rooms[offset:end], nil
}

// Update updates a chat room in Redis
func (r *ChatRoomRepository) Update(ctx context.Context, room *models.ChatRoom) error {
	if err := room.Validate(); err != nil {
		return fmt.Errorf("invalid room: %w", err)
	}

	// Check if room exists first
	key := room.RedisKey()
	exists := r.client.Exists(ctx, key)
	if exists.Err() != nil {
		return fmt.Errorf("failed to check room existence: %w", exists.Err())
	}
	if exists.Val() == 0 {
		return fmt.Errorf("room not found: %s", room.ID)
	}

	// Marshal room to JSON
	roomData, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("failed to marshal room: %w", err)
	}

	// Update room data
	result := r.client.Do(ctx, "JSON.SET", key, "$", string(roomData))
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to update room in Redis: %w", err)
	}

	// Update room in the global list as well
	roomListKey := "rooms:list"
	roomSummary := room.ToSummary()
	summaryData, err := json.Marshal(roomSummary)
	if err != nil {
		return fmt.Errorf("failed to marshal room summary: %w", err)
	}

	result = r.client.Do(ctx, "JSON.SET", roomListKey, fmt.Sprintf("$.%s", room.ID), string(summaryData))
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to update room in list: %w", err)
	}

	return nil
}

// Delete soft-deletes a chat room by marking it as inactive
func (r *ChatRoomRepository) Delete(ctx context.Context, roomID string) error {
	room, err := r.GetByID(ctx, roomID)
	if err != nil {
		return err
	}

	room.IsActive = false
	return r.Update(ctx, room)
}

// GetTotalCount returns the total number of active rooms
func (r *ChatRoomRepository) GetTotalCount(ctx context.Context, isActive *bool) (int, error) {
	rooms, err := r.List(ctx, isActive, 1000, 0) // Get all rooms to count them
	if err != nil {
		return 0, err
	}
	return len(rooms), nil
}

// UpdateParticipantCount updates the participant count for a room
func (r *ChatRoomRepository) UpdateParticipantCount(ctx context.Context, roomID string, count int) error {
	key := fmt.Sprintf("room:%s", roomID)

	result := r.client.Do(ctx, "JSON.SET", key, "$.participant_count", count)
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to update participant count: %w", err)
	}

	// Also update in the room list
	roomListKey := "rooms:list"
	result = r.client.Do(ctx, "JSON.SET", roomListKey, fmt.Sprintf("$.%s.participant_count", roomID), count)
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to update participant count in list: %w", err)
	}

	return nil
}

// UpdateLastMessage updates the last message timestamp for a room
func (r *ChatRoomRepository) UpdateLastMessage(ctx context.Context, roomID string, timestamp time.Time) error {
	key := fmt.Sprintf("room:%s", roomID)
	timeStr := timestamp.Format(time.RFC3339)

	result := r.client.Do(ctx, "JSON.SET", key, "$.last_message_at", fmt.Sprintf(`"%s"`, timeStr))
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to update last message timestamp: %w", err)
	}

	// Also update in the room list
	roomListKey := "rooms:list"
	result = r.client.Do(ctx, "JSON.SET", roomListKey, fmt.Sprintf("$.%s.last_message_at", roomID), fmt.Sprintf(`"%s"`, timeStr))
	if err := result.Err(); err != nil {
		return fmt.Errorf("failed to update last message timestamp in list: %w", err)
	}

	return nil
}