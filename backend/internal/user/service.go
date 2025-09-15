package user

import (
	"context"
	"fmt"
	"time"
)

// Service handles user business logic and coordinates repository operations
type Service struct {
	repo Repository
}

// NewService creates a new user service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// CreateUser creates a new user with validation
func (s *Service) CreateUser(ctx context.Context, username string) (*User, error) {
	// Create user entity with validation
	user, err := NewUser(username)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Save to repository
	err = s.repo.Save(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return user, nil
}

// CreateUserWithID creates a user with a specific ID (primarily for testing)
func (s *Service) CreateUserWithID(ctx context.Context, userID string, username string) (*User, error) {
	// Parse the user ID
	parsedUserID, err := ParseUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	// Create username with validation
	validUsername, err := NewUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid username: %w", err)
	}

	// Create user entity manually with specific ID
	user := &User{
		ID:        parsedUserID,
		Username:  validUsername,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save to repository
	err = s.repo.Save(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return user, nil
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, userID UserID) (*User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// UpdateUsername updates a user's username
func (s *Service) UpdateUsername(ctx context.Context, userID UserID, newUsername string) (*User, error) {
	// Get current user
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Update username using domain logic
	err = user.ChangeUsername(newUsername)
	if err != nil {
		return nil, fmt.Errorf("failed to change username: %w", err)
	}

	// Save updated user
	err = s.repo.Save(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to save updated user: %w", err)
	}

	return user, nil
}

// CreateSession creates a new user session
func (s *Service) CreateSession(ctx context.Context, userID UserID, connectionID string) (*UserSession, error) {
	// Verify user exists
	_, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Create session
	session := NewUserSession(userID, connectionID)

	// Save session
	err = s.repo.SaveSession(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// GetSession retrieves a user session
func (s *Service) GetSession(ctx context.Context, userID UserID) (*UserSession, error) {
	session, err := s.repo.GetSession(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// JoinRoom adds a room to user's active rooms
func (s *Service) JoinRoom(ctx context.Context, userID UserID, roomID string) error {
	// Get current session
	session, err := s.repo.GetSession(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Add room using domain logic
	session.AddRoom(roomID)

	// Save updated session
	err = s.repo.SaveSession(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// LeaveRoom removes a room from user's active rooms
func (s *Service) LeaveRoom(ctx context.Context, userID UserID, roomID string) error {
	// Get current session
	session, err := s.repo.GetSession(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Remove room using domain logic
	session.RemoveRoom(roomID)

	// Save updated session
	err = s.repo.SaveSession(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// IsUserInRoom checks if user is in a specific room
func (s *Service) IsUserInRoom(ctx context.Context, userID UserID, roomID string) (bool, error) {
	session, err := s.repo.GetSession(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get session: %w", err)
	}

	return session.IsInRoom(roomID), nil
}

// UpdateActivity updates user's last seen timestamp
func (s *Service) UpdateActivity(ctx context.Context, userID UserID) error {
	session, err := s.repo.GetSession(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.UpdateLastSeen()

	err = s.repo.SaveSession(ctx, session)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// DeleteSession removes a user session
func (s *Service) DeleteSession(ctx context.Context, userID UserID) error {
	err := s.repo.DeleteSession(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// GetUserActiveRooms returns user's active rooms
func (s *Service) GetUserActiveRooms(ctx context.Context, userID UserID) ([]string, error) {
	session, err := s.repo.GetSession(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session.ActiveRooms, nil
}

// UserInfo represents user information for API responses
type UserInfo struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// UserUpdateResponse represents the response for username update operations
type UserUpdateResponse struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	UpdatedAt string `json:"updated_at"` // RFC3339 format
}

// GetUserInfo returns user information for API responses
func (s *Service) GetUserInfo(ctx context.Context, userID UserID) (*UserInfo, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &UserInfo{
		UserID:   user.ID.String(),
		Username: user.Username.String(),
	}, nil
}

// ToUserUpdateResponse converts a User entity to UserUpdateResponse
func (s *Service) ToUserUpdateResponse(user *User) *UserUpdateResponse {
	return &UserUpdateResponse{
		UserID:    user.ID.String(),
		Username:  user.Username.String(),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
}