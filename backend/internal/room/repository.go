package room

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Repository interface for room data operations
type Repository interface {
	Save(ctx context.Context, room *ChatRoom) error
	FindByID(ctx context.Context, roomID RoomID) (*ChatRoom, error)
	UpdateActivity(ctx context.Context, roomID RoomID) error
	UpdateParticipantCount(ctx context.Context, roomID RoomID, count int) error

	AddParticipant(ctx context.Context, roomID RoomID, userID string) error
	RemoveParticipant(ctx context.Context, roomID RoomID, userID string) error
	GetParticipants(ctx context.Context, roomID RoomID) ([]string, error)
	IsParticipant(ctx context.Context, roomID RoomID, userID string) (bool, error)
}

// redisRepository implements Repository using Redis JSON and Sets
type redisRepository struct {
	client *redis.Client
}

// NewRepository creates a new Redis-based room repository
func NewRepository(client *redis.Client) Repository {
	return &redisRepository{client: client}
}

// Save stores a room in Redis as JSON document
func (r *redisRepository) Save(ctx context.Context, room *ChatRoom) error {
	key := fmt.Sprintf("room:%s", room.ID.String())

	roomJSON, err := json.Marshal(room)
	if err != nil {
		return fmt.Errorf("failed to marshal room: %w", err)
	}

	err = r.client.JSONSet(ctx, key, "$", string(roomJSON)).Err()
	if err != nil {
		return fmt.Errorf("failed to save room to Redis: %w", err)
	}

	return nil
}

// FindByID retrieves a room by ID from Redis
func (r *redisRepository) FindByID(ctx context.Context, roomID RoomID) (*ChatRoom, error) {
	key := fmt.Sprintf("room:%s", roomID.String())

	result, err := r.client.JSONGet(ctx, key, "$").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("room not found: %s", roomID.String())
		}
		return nil, fmt.Errorf("failed to get room from Redis: %w", err)
	}

	var rooms []ChatRoom
	err = json.Unmarshal([]byte(result), &rooms)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal room: %w", err)
	}

	if len(rooms) == 0 {
		return nil, fmt.Errorf("room not found: %s", roomID.String())
	}

	return &rooms[0], nil
}

// UpdateActivity updates the last activity timestamp for a room
func (r *redisRepository) UpdateActivity(ctx context.Context, roomID RoomID) error {
	key := fmt.Sprintf("room:%s", roomID.String())

	now := time.Now()
	err := r.client.JSONSet(ctx, key, "$.last_activity", fmt.Sprintf(`"%s"`, now.Format(time.RFC3339))).Err()
	if err != nil {
		return fmt.Errorf("failed to update room activity: %w", err)
	}

	return nil
}

// UpdateParticipantCount updates the participant count for a room
func (r *redisRepository) UpdateParticipantCount(ctx context.Context, roomID RoomID, count int) error {
	key := fmt.Sprintf("room:%s", roomID.String())

	err := r.client.JSONSet(ctx, key, "$.participant_count", fmt.Sprintf("%d", count)).Err()
	if err != nil {
		return fmt.Errorf("failed to update participant count: %w", err)
	}

	return r.UpdateActivity(ctx, roomID)
}

// AddParticipant adds a user to the room's participant set
func (r *redisRepository) AddParticipant(ctx context.Context, roomID RoomID, userID string) error {
	participantsKey := fmt.Sprintf("room_participants:%s", roomID.String())

	// Add to participants set
	err := r.client.SAdd(ctx, participantsKey, userID).Err()
	if err != nil {
		return fmt.Errorf("failed to add participant to set: %w", err)
	}

	// Get current participant count
	count, err := r.client.SCard(ctx, participantsKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get participant count: %w", err)
	}

	// Update room participant count
	return r.UpdateParticipantCount(ctx, roomID, int(count))
}

// RemoveParticipant removes a user from the room's participant set
func (r *redisRepository) RemoveParticipant(ctx context.Context, roomID RoomID, userID string) error {
	participantsKey := fmt.Sprintf("room_participants:%s", roomID.String())

	// Remove from participants set
	err := r.client.SRem(ctx, participantsKey, userID).Err()
	if err != nil {
		return fmt.Errorf("failed to remove participant from set: %w", err)
	}

	// Get current participant count
	count, err := r.client.SCard(ctx, participantsKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get participant count: %w", err)
	}

	// Update room participant count
	return r.UpdateParticipantCount(ctx, roomID, int(count))
}

// GetParticipants retrieves all participants in a room
func (r *redisRepository) GetParticipants(ctx context.Context, roomID RoomID) ([]string, error) {
	participantsKey := fmt.Sprintf("room_participants:%s", roomID.String())

	participants, err := r.client.SMembers(ctx, participantsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	return participants, nil
}

// IsParticipant checks if a user is a participant in a room
func (r *redisRepository) IsParticipant(ctx context.Context, roomID RoomID, userID string) (bool, error) {
	participantsKey := fmt.Sprintf("room_participants:%s", roomID.String())

	exists, err := r.client.SIsMember(ctx, participantsKey, userID).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check participant membership: %w", err)
	}

	return exists, nil
}

// GetRoomStats returns room statistics
func (r *redisRepository) GetRoomStats(ctx context.Context, roomID RoomID) (map[string]interface{}, error) {
	participantsKey := fmt.Sprintf("room_participants:%s", roomID.String())

	// Get participant count
	participantCount, err := r.client.SCard(ctx, participantsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get participant count: %w", err)
	}

	// Get participants list
	participants, err := r.GetParticipants(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants list: %w", err)
	}

	return map[string]interface{}{
		"participant_count": participantCount,
		"participants":      participants,
	}, nil
}