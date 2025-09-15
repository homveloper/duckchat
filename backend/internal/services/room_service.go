package services

import (
	"context"
	"fmt"
	"time"

	"duckchat/internal/models"
	"duckchat/internal/repository"
)

// RoomService handles business logic for chat rooms
type RoomService struct {
	roomRepo *repository.ChatRoomRepository
}

// NewRoomService creates a new RoomService
func NewRoomService(roomRepo *repository.ChatRoomRepository) *RoomService {
	return &RoomService{
		roomRepo: roomRepo,
	}
}

// CreateRoomRequest represents the request to create a room
type CreateRoomRequest struct {
	Title       string  `json:"title" validate:"required,min=3,max=100"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
	CreatedBy   string  `json:"created_by" validate:"required"`
}

// CreateRoomResponse represents the response after creating a room
type CreateRoomResponse struct {
	Room            *models.ChatRoom `json:"room"`
	ParticipantAdded bool            `json:"participant_added"`
}

// CreateRoom creates a new chat room and adds the creator as a participant
func (s *RoomService) CreateRoom(ctx context.Context, req *CreateRoomRequest) (*CreateRoomResponse, error) {
	// Validate request
	if err := s.validateCreateRoomRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create room model
	room, err := models.NewChatRoom(req.Title, req.CreatedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create room model: %w", err)
	}

	// Save room to repository
	if err := s.roomRepo.Create(ctx, room); err != nil {
		return nil, fmt.Errorf("failed to save room: %w", err)
	}

	return &CreateRoomResponse{
		Room:            room,
		ParticipantAdded: true, // Creator is automatically added as participant
	}, nil
}

// ListRoomsRequest represents the request to list rooms
type ListRoomsRequest struct {
	IsActive *bool `json:"is_active,omitempty"`
	Limit    int   `json:"limit" validate:"min=1,max=100"`
	Offset   int   `json:"offset" validate:"min=0"`
}

// ListRoomsResponse represents the response with room list
type ListRoomsResponse struct {
	Rooms      []*models.ChatRoomSummary `json:"rooms"`
	TotalCount int                       `json:"total_count"`
	HasMore    bool                      `json:"has_more"`
}

// ListRooms retrieves a list of chat rooms with filtering and pagination
func (s *RoomService) ListRooms(ctx context.Context, req *ListRoomsRequest) (*ListRoomsResponse, error) {
	// Validate request
	if err := s.validateListRoomsRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get rooms from repository
	rooms, err := s.roomRepo.List(ctx, req.IsActive, req.Limit, req.Offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list rooms: %w", err)
	}

	// Get total count
	totalCount, err := s.roomRepo.GetTotalCount(ctx, req.IsActive)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Check if there are more results
	hasMore := req.Offset+len(rooms) < totalCount

	return &ListRoomsResponse{
		Rooms:      rooms,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

// GetRoomRequest represents the request to get a specific room
type GetRoomRequest struct {
	RoomID string `json:"room_id" validate:"required,uuid"`
}

// GetRoomResponse represents the response with room details
type GetRoomResponse struct {
	Room *models.ChatRoom `json:"room"`
}

// GetRoom retrieves a specific room by ID
func (s *RoomService) GetRoom(ctx context.Context, req *GetRoomRequest) (*GetRoomResponse, error) {
	// Validate request
	if err := s.validateGetRoomRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get room from repository
	room, err := s.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	return &GetRoomResponse{
		Room: room,
	}, nil
}

// UpdateRoomRequest represents the request to update a room
type UpdateRoomRequest struct {
	RoomID      string  `json:"room_id" validate:"required,uuid"`
	Title       *string `json:"title,omitempty" validate:"omitempty,min=3,max=100"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
	UpdatedBy   string  `json:"updated_by" validate:"required"`
}

// UpdateRoomResponse represents the response after updating a room
type UpdateRoomResponse struct {
	Room *models.ChatRoom `json:"room"`
}

// UpdateRoom updates an existing chat room
func (s *RoomService) UpdateRoom(ctx context.Context, req *UpdateRoomRequest) (*UpdateRoomResponse, error) {
	// Validate request
	if err := s.validateUpdateRoomRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get existing room
	room, err := s.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	// Update fields if provided
	updated := false
	if req.Title != nil && *req.Title != room.Title {
		room.Title = *req.Title
		updated = true
	}

	if updated {
		// Save updated room
		if err := s.roomRepo.Update(ctx, room); err != nil {
			return nil, fmt.Errorf("failed to update room: %w", err)
		}
	}

	return &UpdateRoomResponse{
		Room: room,
	}, nil
}

// DeactivateRoomRequest represents the request to deactivate a room
type DeactivateRoomRequest struct {
	RoomID        string `json:"room_id" validate:"required,uuid"`
	DeactivatedBy string `json:"deactivated_by" validate:"required"`
}

// DeactivateRoomResponse represents the response after deactivating a room
type DeactivateRoomResponse struct {
	Room *models.ChatRoom `json:"room"`
}

// DeactivateRoom soft-deletes a room by marking it as inactive
func (s *RoomService) DeactivateRoom(ctx context.Context, req *DeactivateRoomRequest) (*DeactivateRoomResponse, error) {
	// Validate request
	if err := s.validateDeactivateRoomRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get existing room
	room, err := s.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	// Deactivate room
	if err := s.roomRepo.Delete(ctx, req.RoomID); err != nil {
		return nil, fmt.Errorf("failed to deactivate room: %w", err)
	}

	// Get updated room
	room, err = s.roomRepo.GetByID(ctx, req.RoomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated room: %w", err)
	}

	return &DeactivateRoomResponse{
		Room: room,
	}, nil
}

// UpdateLastMessage updates the last message timestamp for a room
func (s *RoomService) UpdateLastMessage(ctx context.Context, roomID string, timestamp time.Time) error {
	if roomID == "" {
		return fmt.Errorf("room_id cannot be empty")
	}

	return s.roomRepo.UpdateLastMessage(ctx, roomID, timestamp)
}

// UpdateParticipantCount updates the participant count for a room
func (s *RoomService) UpdateParticipantCount(ctx context.Context, roomID string, count int) error {
	if roomID == "" {
		return fmt.Errorf("room_id cannot be empty")
	}

	if count < 0 {
		return fmt.Errorf("participant count cannot be negative")
	}

	return s.roomRepo.UpdateParticipantCount(ctx, roomID, count)
}

// Validation methods

func (s *RoomService) validateCreateRoomRequest(req *CreateRoomRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.Title == "" {
		return fmt.Errorf("title is required")
	}

	if len(req.Title) < 3 {
		return fmt.Errorf("title must be at least 3 characters")
	}

	if len(req.Title) > 100 {
		return fmt.Errorf("title cannot exceed 100 characters")
	}

	if req.CreatedBy == "" {
		return fmt.Errorf("created_by is required")
	}

	if req.Description != nil && len(*req.Description) > 500 {
		return fmt.Errorf("description cannot exceed 500 characters")
	}

	return nil
}

func (s *RoomService) validateListRoomsRequest(req *ListRoomsRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.Limit <= 0 {
		return fmt.Errorf("limit must be greater than 0")
	}

	if req.Limit > 100 {
		return fmt.Errorf("limit cannot exceed 100")
	}

	if req.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}

	return nil
}

func (s *RoomService) validateGetRoomRequest(req *GetRoomRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.RoomID == "" {
		return fmt.Errorf("room_id is required")
	}

	return nil
}

func (s *RoomService) validateUpdateRoomRequest(req *UpdateRoomRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.RoomID == "" {
		return fmt.Errorf("room_id is required")
	}

	if req.UpdatedBy == "" {
		return fmt.Errorf("updated_by is required")
	}

	if req.Title != nil {
		if len(*req.Title) < 3 {
			return fmt.Errorf("title must be at least 3 characters")
		}
		if len(*req.Title) > 100 {
			return fmt.Errorf("title cannot exceed 100 characters")
		}
	}

	if req.Description != nil && len(*req.Description) > 500 {
		return fmt.Errorf("description cannot exceed 500 characters")
	}

	return nil
}

func (s *RoomService) validateDeactivateRoomRequest(req *DeactivateRoomRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.RoomID == "" {
		return fmt.Errorf("room_id is required")
	}

	if req.DeactivatedBy == "" {
		return fmt.Errorf("deactivated_by is required")
	}

	return nil
}