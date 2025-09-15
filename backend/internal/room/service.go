package room

import (
	"context"
	"fmt"
)

// Service handles room business logic and coordinates repository operations
type Service struct {
	repo Repository
}

// NewService creates a new room service
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// CreateRoom creates a new chat room
func (s *Service) CreateRoom(ctx context.Context, name, createdBy string) (*ChatRoom, error) {
	// Create room entity with validation
	room, err := NewChatRoom(name, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}

	// Save to repository
	err = s.repo.Save(ctx, room)
	if err != nil {
		return nil, fmt.Errorf("failed to save room: %w", err)
	}

	// Add creator as participant
	err = s.repo.AddParticipant(ctx, room.ID, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to add creator as participant: %w", err)
	}

	return room, nil
}

// GetRoom retrieves a room by ID
func (s *Service) GetRoom(ctx context.Context, roomID RoomID) (*ChatRoom, error) {
	room, err := s.repo.FindByID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	return room, nil
}

// JoinRoom adds a user to a room
func (s *Service) JoinRoom(ctx context.Context, roomID RoomID, userID string) (*ChatRoom, error) {
	// Get room
	room, err := s.repo.FindByID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	// Check if user can join
	if !room.CanJoin() {
		return nil, fmt.Errorf("room is full")
	}

	// Check if user is already a participant
	isParticipant, err := s.repo.IsParticipant(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check participant status: %w", err)
	}

	if isParticipant {
		return room, nil // Already joined
	}

	// Add participant to room
	err = s.repo.AddParticipant(ctx, roomID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to add participant: %w", err)
	}

	// Get updated room
	updatedRoom, err := s.repo.FindByID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated room: %w", err)
	}

	return updatedRoom, nil
}

// LeaveRoom removes a user from a room
func (s *Service) LeaveRoom(ctx context.Context, roomID RoomID, userID string) error {
	// Check if user is a participant
	isParticipant, err := s.repo.IsParticipant(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("failed to check participant status: %w", err)
	}

	if !isParticipant {
		return nil // Not in room
	}

	// Remove participant from room
	err = s.repo.RemoveParticipant(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}

	return nil
}

// GetRoomParticipants returns all participants in a room
func (s *Service) GetRoomParticipants(ctx context.Context, roomID RoomID) ([]string, error) {
	participants, err := s.repo.GetParticipants(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	return participants, nil
}

// IsUserInRoom checks if a user is in a specific room
func (s *Service) IsUserInRoom(ctx context.Context, roomID RoomID, userID string) (bool, error) {
	isParticipant, err := s.repo.IsParticipant(ctx, roomID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check participant status: %w", err)
	}

	return isParticipant, nil
}

// UpdateRoomActivity updates the last activity timestamp for a room
func (s *Service) UpdateRoomActivity(ctx context.Context, roomID RoomID) error {
	err := s.repo.UpdateActivity(ctx, roomID)
	if err != nil {
		return fmt.Errorf("failed to update room activity: %w", err)
	}

	return nil
}

// GetRoomInfo returns room information for API responses
func (s *Service) GetRoomInfo(ctx context.Context, roomID RoomID) (*RoomInfo, error) {
	room, err := s.repo.FindByID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	participants, err := s.repo.GetParticipants(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get participants: %w", err)
	}

	return &RoomInfo{
		RoomID:           room.ID.String(),
		Name:             room.Name,
		ParticipantCount: room.ParticipantCount,
		Participants:     participants,
		CreatedBy:        room.CreatedBy,
		CreatedAt:        room.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		LastActivity:     room.LastActivity.Format("2006-01-02T15:04:05Z07:00"),
		State:            room.GetState(),
		CanJoin:          room.CanJoin(),
	}, nil
}

// ValidateRoomAccess ensures a user can access a room
func (s *Service) ValidateRoomAccess(ctx context.Context, roomID RoomID, userID string) error {
	// Check if room exists
	_, err := s.repo.FindByID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("room not found: %w", err)
	}

	// Check if user is a participant
	isParticipant, err := s.repo.IsParticipant(ctx, roomID, userID)
	if err != nil {
		return fmt.Errorf("failed to check participant status: %w", err)
	}

	if !isParticipant {
		return fmt.Errorf("user is not a participant in this room")
	}

	return nil
}

// RoomInfo represents room information for API responses
type RoomInfo struct {
	RoomID           string    `json:"room_id"`
	Name             string    `json:"name"`
	ParticipantCount int       `json:"participant_count"`
	Participants     []string  `json:"participants"`
	CreatedBy        string    `json:"created_by"`
	CreatedAt        string    `json:"created_at"` // Will be formatted as RFC3339
	LastActivity     string    `json:"last_activity"`
	State            RoomState `json:"state"`
	CanJoin          bool      `json:"can_join"`
}

// CreateRoomRequest represents a room creation request
type CreateRoomRequest struct {
	Name string `json:"name"`
}

// JoinRoomRequest represents a room join request
type JoinRoomRequest struct {
	RoomID string `json:"room_id"`
}

// ValidateCreateRoomRequest validates a room creation request
func ValidateCreateRoomRequest(req *CreateRoomRequest) error {
	if req.Name == "" {
		return fmt.Errorf("room name is required")
	}
	if len(req.Name) < 1 || len(req.Name) > 100 {
		return fmt.Errorf("room name must be 1-100 characters")
	}
	return nil
}

// ValidateJoinRoomRequest validates a room join request
func ValidateJoinRoomRequest(req *JoinRoomRequest) error {
	if req.RoomID == "" {
		return fmt.Errorf("room ID is required")
	}
	return nil
}

// ToRoomInfo converts a ChatRoom entity to RoomInfo response
func (s *Service) ToRoomInfo(room *ChatRoom, participants []string) *RoomInfo {
	return &RoomInfo{
		RoomID:           room.ID.String(),
		Name:             room.Name,
		ParticipantCount: room.ParticipantCount,
		Participants:     participants,
		CreatedBy:        room.CreatedBy,
		CreatedAt:        room.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		LastActivity:     room.LastActivity.Format("2006-01-02T15:04:05Z07:00"),
		State:            room.GetState(),
		CanJoin:          room.CanJoin(),
	}
}