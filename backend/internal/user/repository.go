package user

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Repository interface for user data operations
type Repository interface {
	Save(ctx context.Context, user *User) error
	FindByID(ctx context.Context, userID UserID) (*User, error)
	UpdateUsername(ctx context.Context, userID UserID, username Username) error

	SaveSession(ctx context.Context, session *UserSession) error
	GetSession(ctx context.Context, userID UserID) (*UserSession, error)
	DeleteSession(ctx context.Context, userID UserID) error
}

// redisRepository implements Repository using Redis JSON
type redisRepository struct {
	client *redis.Client
}

// NewRepository creates a new Redis-based user repository
func NewRepository(client *redis.Client) Repository {
	return &redisRepository{client: client}
}

// Save stores a user in Redis as JSON document
func (r *redisRepository) Save(ctx context.Context, user *User) error {
	key := fmt.Sprintf("user:%s", user.ID.String())

	userJSON, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	err = r.client.JSONSet(ctx, key, "$", string(userJSON)).Err()
	if err != nil {
		return fmt.Errorf("failed to save user to Redis: %w", err)
	}

	return nil
}

// FindByID retrieves a user by ID from Redis
func (r *redisRepository) FindByID(ctx context.Context, userID UserID) (*User, error) {
	key := fmt.Sprintf("user:%s", userID.String())

	result, err := r.client.JSONGet(ctx, key, "$").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("user not found: %s", userID.String())
		}
		return nil, fmt.Errorf("failed to get user from Redis: %w", err)
	}

	var users []User
	err = json.Unmarshal([]byte(result), &users)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user not found: %s", userID.String())
	}

	return &users[0], nil
}

// UpdateUsername updates only the username and updated_at fields
func (r *redisRepository) UpdateUsername(ctx context.Context, userID UserID, username Username) error {
	key := fmt.Sprintf("user:%s", userID.String())

	// Update username
	err := r.client.JSONSet(ctx, key, "$.username", fmt.Sprintf(`"%s"`, username.String())).Err()
	if err != nil {
		return fmt.Errorf("failed to update username: %w", err)
	}

	// Update timestamp
	now := time.Now()
	err = r.client.JSONSet(ctx, key, "$.updated_at", fmt.Sprintf(`"%s"`, now.Format(time.RFC3339))).Err()
	if err != nil {
		return fmt.Errorf("failed to update timestamp: %w", err)
	}

	return nil
}

// SaveSession stores a user session in Redis
func (r *redisRepository) SaveSession(ctx context.Context, session *UserSession) error {
	key := fmt.Sprintf("session:%s", session.UserID.String())

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	err = r.client.JSONSet(ctx, key, "$", string(sessionJSON)).Err()
	if err != nil {
		return fmt.Errorf("failed to save session to Redis: %w", err)
	}

	return nil
}

// GetSession retrieves a user session from Redis
func (r *redisRepository) GetSession(ctx context.Context, userID UserID) (*UserSession, error) {
	key := fmt.Sprintf("session:%s", userID.String())

	result, err := r.client.JSONGet(ctx, key, "$").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found: %s", userID.String())
		}
		return nil, fmt.Errorf("failed to get session from Redis: %w", err)
	}

	var sessions []UserSession
	err = json.Unmarshal([]byte(result), &sessions)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	if len(sessions) == 0 {
		return nil, fmt.Errorf("session not found: %s", userID.String())
	}

	return &sessions[0], nil
}

// DeleteSession removes a user session from Redis
func (r *redisRepository) DeleteSession(ctx context.Context, userID UserID) error {
	key := fmt.Sprintf("session:%s", userID.String())

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// UpdateSessionActivity updates the last seen time for a session
func (r *redisRepository) UpdateSessionActivity(ctx context.Context, userID UserID) error {
	key := fmt.Sprintf("session:%s", userID.String())

	now := time.Now()
	err := r.client.JSONSet(ctx, key, "$.last_seen", fmt.Sprintf(`"%s"`, now.Format(time.RFC3339))).Err()
	if err != nil {
		return fmt.Errorf("failed to update session activity: %w", err)
	}

	return nil
}

// AddRoomToSession adds a room to user's active rooms
func (r *redisRepository) AddRoomToSession(ctx context.Context, userID UserID, roomID string) error {
	key := fmt.Sprintf("session:%s", userID.String())

	err := r.client.JSONArrAppend(ctx, key, "$.active_rooms", fmt.Sprintf(`"%s"`, roomID)).Err()
	if err != nil {
		return fmt.Errorf("failed to add room to session: %w", err)
	}

	return r.UpdateSessionActivity(ctx, userID)
}

// RemoveRoomFromSession removes a room from user's active rooms
func (r *redisRepository) RemoveRoomFromSession(ctx context.Context, userID UserID, roomID string) error {
	key := fmt.Sprintf("session:%s", userID.String())

	// Get current active rooms
	result, err := r.client.JSONGet(ctx, key, "$.active_rooms").Result()
	if err != nil {
		return fmt.Errorf("failed to get active rooms: %w", err)
	}

	var activeRoomsArray [][]string
	err = json.Unmarshal([]byte(result), &activeRoomsArray)
	if err != nil {
		return fmt.Errorf("failed to unmarshal active rooms: %w", err)
	}

	if len(activeRoomsArray) == 0 {
		return nil // No active rooms
	}

	activeRooms := activeRoomsArray[0]

	// Remove the room
	var updatedRooms []string
	for _, room := range activeRooms {
		if room != roomID {
			updatedRooms = append(updatedRooms, room)
		}
	}

	// Update the active rooms array
	updatedJSON, err := json.Marshal(updatedRooms)
	if err != nil {
		return fmt.Errorf("failed to marshal updated rooms: %w", err)
	}

	err = r.client.JSONSet(ctx, key, "$.active_rooms", string(updatedJSON)).Err()
	if err != nil {
		return fmt.Errorf("failed to update active rooms: %w", err)
	}

	return r.UpdateSessionActivity(ctx, userID)
}