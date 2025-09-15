package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"duckchat/internal/models"
)

// SessionService handles business logic for user sessions using Redis
type SessionService struct {
	client *redis.Client
}

// NewSessionService creates a new SessionService
func NewSessionService(client *redis.Client) *SessionService {
	return &SessionService{
		client: client,
	}
}

// CreateSessionRequest represents the request to create a new session
type CreateSessionRequest struct {
	UserID   string `json:"user_id" validate:"required,uuid"`
	Username string `json:"username" validate:"required"`
}

// CreateSessionResponse represents the response after creating a session
type CreateSessionResponse struct {
	Session *models.UserSession `json:"session"`
}

// CreateSession creates a new user session
func (s *SessionService) CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error) {
	// Validate request
	if err := s.validateCreateSessionRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create new session model
	session, err := models.NewUserSession(req.UserID, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to create session model: %w", err)
	}

	// Store session in Redis with expiration (24 hours default)
	sessionKey := session.RedisKey()
	sessionData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	// Set session with 24 hour expiration
	err = s.client.Set(ctx, sessionKey, string(sessionData), 24*time.Hour).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to store session: %w", err)
	}

	return &CreateSessionResponse{
		Session: session,
	}, nil
}

// GetSessionRequest represents the request to get a session
type GetSessionRequest struct {
	SessionID string `json:"session_id" validate:"required,uuid"`
}

// GetSessionResponse represents the response with session data
type GetSessionResponse struct {
	Session *models.UserSession `json:"session"`
}

// GetSession retrieves a session by ID
func (s *SessionService) GetSession(ctx context.Context, req *GetSessionRequest) (*GetSessionResponse, error) {
	// Validate request
	if err := s.validateGetSessionRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	sessionKey := fmt.Sprintf("session:%s", req.SessionID)
	result := s.client.Get(ctx, sessionKey)

	sessionData, err := result.Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	var session models.UserSession
	if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &GetSessionResponse{
		Session: &session,
	}, nil
}

// UpdateSessionRequest represents the request to update a session
type UpdateSessionRequest struct {
	SessionID    string   `json:"session_id" validate:"required,uuid"`
	ActiveRooms  []string `json:"active_rooms,omitempty"`
	ConnectionID *string  `json:"connection_id,omitempty"`
}

// UpdateSessionResponse represents the response after updating a session
type UpdateSessionResponse struct {
	Session *models.UserSession `json:"session"`
}

// UpdateSession updates an existing session
func (s *SessionService) UpdateSession(ctx context.Context, req *UpdateSessionRequest) (*UpdateSessionResponse, error) {
	// Validate request
	if err := s.validateUpdateSessionRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get existing session
	getReq := &GetSessionRequest{SessionID: req.SessionID}
	getResp, err := s.GetSession(ctx, getReq)
	if err != nil {
		return nil, err
	}

	session := getResp.Session

	// Update fields
	updated := false
	if req.ActiveRooms != nil {
		session.ActiveRooms = req.ActiveRooms
		updated = true
	}

	if req.ConnectionID != nil {
		session.SetConnectionID(*req.ConnectionID)
		updated = true
	}

	if updated {
		// Update last seen timestamp
		session.UpdateLastSeen()

		// Save updated session
		sessionKey := session.RedisKey()
		sessionData, err := json.Marshal(session)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal session: %w", err)
		}

		// Extend expiration when updating
		err = s.client.Set(ctx, sessionKey, string(sessionData), 24*time.Hour).Err()
		if err != nil {
			return nil, fmt.Errorf("failed to update session: %w", err)
		}
	}

	return &UpdateSessionResponse{
		Session: session,
	}, nil
}

// AddRoomToSessionRequest represents the request to add a room to session
type AddRoomToSessionRequest struct {
	SessionID string `json:"session_id" validate:"required,uuid"`
	RoomID    string `json:"room_id" validate:"required,uuid"`
}

// AddRoomToSession adds a room to a session's active rooms
func (s *SessionService) AddRoomToSession(ctx context.Context, req *AddRoomToSessionRequest) (*UpdateSessionResponse, error) {
	if err := s.validateAddRoomToSessionRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get existing session
	getReq := &GetSessionRequest{SessionID: req.SessionID}
	getResp, err := s.GetSession(ctx, getReq)
	if err != nil {
		return nil, err
	}

	session := getResp.Session

	// Add room to session
	if err := session.AddRoom(req.RoomID); err != nil {
		return nil, fmt.Errorf("failed to add room to session: %w", err)
	}

	// Save updated session
	sessionKey := session.RedisKey()
	sessionData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	err = s.client.Set(ctx, sessionKey, string(sessionData), 24*time.Hour).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &UpdateSessionResponse{
		Session: session,
	}, nil
}

// RemoveRoomFromSessionRequest represents the request to remove a room from session
type RemoveRoomFromSessionRequest struct {
	SessionID string `json:"session_id" validate:"required,uuid"`
	RoomID    string `json:"room_id" validate:"required,uuid"`
}

// RemoveRoomFromSession removes a room from a session's active rooms
func (s *SessionService) RemoveRoomFromSession(ctx context.Context, req *RemoveRoomFromSessionRequest) (*UpdateSessionResponse, error) {
	if err := s.validateRemoveRoomFromSessionRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get existing session
	getReq := &GetSessionRequest{SessionID: req.SessionID}
	getResp, err := s.GetSession(ctx, getReq)
	if err != nil {
		return nil, err
	}

	session := getResp.Session

	// Remove room from session
	session.RemoveRoom(req.RoomID)

	// Save updated session
	sessionKey := session.RedisKey()
	sessionData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal session: %w", err)
	}

	err = s.client.Set(ctx, sessionKey, string(sessionData), 24*time.Hour).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	return &UpdateSessionResponse{
		Session: session,
	}, nil
}

// DeleteSessionRequest represents the request to delete a session
type DeleteSessionRequest struct {
	SessionID string `json:"session_id" validate:"required,uuid"`
}

// DeleteSessionResponse represents the response after deleting a session
type DeleteSessionResponse struct {
	Success bool `json:"success"`
}

// DeleteSession deletes a session
func (s *SessionService) DeleteSession(ctx context.Context, req *DeleteSessionRequest) (*DeleteSessionResponse, error) {
	if err := s.validateDeleteSessionRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	sessionKey := fmt.Sprintf("session:%s", req.SessionID)
	err := s.client.Del(ctx, sessionKey).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to delete session: %w", err)
	}

	return &DeleteSessionResponse{
		Success: true,
	}, nil
}

// ExtendSessionRequest represents the request to extend session expiration
type ExtendSessionRequest struct {
	SessionID string        `json:"session_id" validate:"required,uuid"`
	Duration  time.Duration `json:"duration,omitempty"` // Optional, defaults to 24 hours
}

// ExtendSessionResponse represents the response after extending session
type ExtendSessionResponse struct {
	Success   bool      `json:"success"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ExtendSession extends the expiration time of a session
func (s *SessionService) ExtendSession(ctx context.Context, req *ExtendSessionRequest) (*ExtendSessionResponse, error) {
	if err := s.validateExtendSessionRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	duration := req.Duration
	if duration == 0 {
		duration = 24 * time.Hour // Default to 24 hours
	}

	sessionKey := fmt.Sprintf("session:%s", req.SessionID)

	// Check if session exists
	exists := s.client.Exists(ctx, sessionKey)
	if count, err := exists.Result(); err != nil || count == 0 {
		return nil, fmt.Errorf("session not found or expired")
	}

	// Extend expiration
	err := s.client.Expire(ctx, sessionKey, duration).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to extend session: %w", err)
	}

	expiresAt := time.Now().Add(duration)

	return &ExtendSessionResponse{
		Success:   true,
		ExpiresAt: expiresAt,
	}, nil
}

// GetActiveSessionsRequest represents the request to get active sessions
type GetActiveSessionsRequest struct {
	UserID string `json:"user_id,omitempty"` // Optional filter by user
	Limit  int    `json:"limit" validate:"min=1,max=100"`
}

// GetActiveSessionsResponse represents the response with active sessions
type GetActiveSessionsResponse struct {
	Sessions []*models.UserSession `json:"sessions"`
	Count    int                   `json:"count"`
}

// GetActiveSessions retrieves active sessions, optionally filtered by user
func (s *SessionService) GetActiveSessions(ctx context.Context, req *GetActiveSessionsRequest) (*GetActiveSessionsResponse, error) {
	if err := s.validateGetActiveSessionsRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Scan for session keys
	pattern := "session:*"
	var cursor uint64
	var sessions []*models.UserSession

	for {
		result := s.client.Scan(ctx, cursor, pattern, int64(req.Limit))
		keys, newCursor, err := result.Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan for session keys: %w", err)
		}

		// Get session data for each key
		for _, key := range keys {
			sessionResult := s.client.Get(ctx, key)
			sessionData, err := sessionResult.Result()
			if err != nil {
				continue // Skip expired or invalid sessions
			}

			var session models.UserSession
			if err := json.Unmarshal([]byte(sessionData), &session); err != nil {
				continue // Skip invalid session data
			}

			// Apply user filter if specified
			if req.UserID != "" && session.UserID != req.UserID {
				continue
			}

			sessions = append(sessions, &session)

			// Limit results
			if len(sessions) >= req.Limit {
				break
			}
		}

		cursor = newCursor
		if cursor == 0 || len(sessions) >= req.Limit {
			break
		}
	}

	return &GetActiveSessionsResponse{
		Sessions: sessions,
		Count:    len(sessions),
	}, nil
}

// Validation methods

func (s *SessionService) validateCreateSessionRequest(req *CreateSessionRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	if req.Username == "" {
		return fmt.Errorf("username is required")
	}

	return nil
}

func (s *SessionService) validateGetSessionRequest(req *GetSessionRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	return nil
}

func (s *SessionService) validateUpdateSessionRequest(req *UpdateSessionRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	return nil
}

func (s *SessionService) validateAddRoomToSessionRequest(req *AddRoomToSessionRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	if req.RoomID == "" {
		return fmt.Errorf("room_id is required")
	}

	return nil
}

func (s *SessionService) validateRemoveRoomFromSessionRequest(req *RemoveRoomFromSessionRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	if req.RoomID == "" {
		return fmt.Errorf("room_id is required")
	}

	return nil
}

func (s *SessionService) validateDeleteSessionRequest(req *DeleteSessionRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	return nil
}

func (s *SessionService) validateExtendSessionRequest(req *ExtendSessionRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}

	if req.Duration < 0 {
		return fmt.Errorf("duration cannot be negative")
	}

	// Maximum 7 days
	if req.Duration > 7*24*time.Hour {
		return fmt.Errorf("duration cannot exceed 7 days")
	}

	return nil
}

func (s *SessionService) validateGetActiveSessionsRequest(req *GetActiveSessionsRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0")
	}

	if req.Limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}

	return nil
}