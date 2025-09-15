package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"duckchat/internal/models"
)

// ParticipantService handles business logic for room participants using Redis
type ParticipantService struct {
	client *redis.Client
}

// NewParticipantService creates a new ParticipantService
func NewParticipantService(client *redis.Client) *ParticipantService {
	return &ParticipantService{
		client: client,
	}
}

// JoinRoomRequest represents the request to join a room
type JoinRoomRequest struct {
	RoomID   string `json:"room_id" validate:"required,uuid"`
	UserID   string `json:"user_id" validate:"required,uuid"`
	Username string `json:"username" validate:"required"`
}

// JoinRoomResponse represents the response after joining a room
type JoinRoomResponse struct {
	Participant      *models.RoomParticipant `json:"participant"`
	ParticipantCount int                     `json:"participant_count"`
	AlreadyMember    bool                    `json:"already_member"`
}

// JoinRoom adds a user as a participant to a room
func (s *ParticipantService) JoinRoom(ctx context.Context, req *JoinRoomRequest) (*JoinRoomResponse, error) {
	// Validate request
	if err := s.validateJoinRoomRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if user is already a participant
	isParticipant, err := s.IsUserInRoom(ctx, req.RoomID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing participation: %w", err)
	}

	if isParticipant {
		// Get existing participant info
		participant, err := s.GetParticipant(ctx, req.RoomID, req.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing participant: %w", err)
		}

		count, err := s.GetParticipantCount(ctx, req.RoomID)
		if err != nil {
			return nil, fmt.Errorf("failed to get participant count: %w", err)
		}

		return &JoinRoomResponse{
			Participant:      participant,
			ParticipantCount: count,
			AlreadyMember:    true,
		}, nil
	}

	// Create new participant
	participant, err := models.NewRoomParticipant(req.RoomID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to create participant model: %w", err)
	}

	// Get Redis key for room participants
	participantsKey := fmt.Sprintf("room:%s:participants", req.RoomID)

	// Create participant data
	participantData := map[string]interface{}{
		"user_id":   req.UserID,
		"username":  req.Username,
		"joined_at": participant.JoinedAt.Format(time.RFC3339),
		"is_active": participant.IsActive,
	}

	// Add participant to room using Redis Hash
	participantJSON, err := json.Marshal(participantData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal participant data: %w", err)
	}

	// Store participant data
	err = s.client.HSet(ctx, participantsKey, req.UserID, string(participantJSON)).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to add participant to Redis: %w", err)
	}

	// Update participant count
	count, err := s.updateParticipantCount(ctx, req.RoomID)
	if err != nil {
		return nil, fmt.Errorf("failed to update participant count: %w", err)
	}

	return &JoinRoomResponse{
		Participant:      participant,
		ParticipantCount: count,
		AlreadyMember:    false,
	}, nil
}

// LeaveRoomRequest represents the request to leave a room
type LeaveRoomRequest struct {
	RoomID string `json:"room_id" validate:"required,uuid"`
	UserID string `json:"user_id" validate:"required,uuid"`
}

// LeaveRoomResponse represents the response after leaving a room
type LeaveRoomResponse struct {
	Success          bool `json:"success"`
	ParticipantCount int  `json:"participant_count"`
}

// LeaveRoom removes a user from a room's participants
func (s *ParticipantService) LeaveRoom(ctx context.Context, req *LeaveRoomRequest) (*LeaveRoomResponse, error) {
	// Validate request
	if err := s.validateLeaveRoomRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if user is a participant
	isParticipant, err := s.IsUserInRoom(ctx, req.RoomID, req.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to check participation: %w", err)
	}

	if !isParticipant {
		return nil, fmt.Errorf("user is not a participant of this room")
	}

	// Remove participant from room
	participantsKey := fmt.Sprintf("room:%s:participants", req.RoomID)
	err = s.client.HDel(ctx, participantsKey, req.UserID).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to remove participant from Redis: %w", err)
	}

	// Update participant count
	count, err := s.updateParticipantCount(ctx, req.RoomID)
	if err != nil {
		return nil, fmt.Errorf("failed to update participant count: %w", err)
	}

	return &LeaveRoomResponse{
		Success:          true,
		ParticipantCount: count,
	}, nil
}

// GetParticipantsRequest represents the request to get room participants
type GetParticipantsRequest struct {
	RoomID string `json:"room_id" validate:"required,uuid"`
	Limit  int    `json:"limit" validate:"min=1,max=100"`
	Offset int    `json:"offset" validate:"min=0"`
}

// GetParticipantsResponse represents the response with participant list
type GetParticipantsResponse struct {
	Participants []*models.ParticipantInfo `json:"participants"`
	TotalCount   int                       `json:"total_count"`
}

// GetParticipants retrieves all participants of a room
func (s *ParticipantService) GetParticipants(ctx context.Context, req *GetParticipantsRequest) (*GetParticipantsResponse, error) {
	// Validate request
	if err := s.validateGetParticipantsRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get all participants from Redis Hash
	participantsKey := fmt.Sprintf("room:%s:participants", req.RoomID)
	result := s.client.HGetAll(ctx, participantsKey)

	participantData, err := result.Result()
	if err != nil {
		if err == redis.Nil {
			return &GetParticipantsResponse{
				Participants: []*models.ParticipantInfo{},
				TotalCount:   0,
			}, nil
		}
		return nil, fmt.Errorf("failed to get participants from Redis: %w", err)
	}

	participants := make([]*models.ParticipantInfo, 0)
	for userID, dataJSON := range participantData {
		var participantInfo models.ParticipantInfo
		if err := json.Unmarshal([]byte(dataJSON), &participantInfo); err != nil {
			continue // Skip invalid participant data
		}

		// Ensure UserID is set correctly
		participantInfo.UserID = userID
		participants = append(participants, &participantInfo)
	}

	// Apply pagination
	totalCount := len(participants)
	if req.Offset >= totalCount {
		return &GetParticipantsResponse{
			Participants: []*models.ParticipantInfo{},
			TotalCount:   totalCount,
		}, nil
	}

	end := req.Offset + req.Limit
	if end > totalCount {
		end = totalCount
	}

	paginatedParticipants := participants[req.Offset:end]

	return &GetParticipantsResponse{
		Participants: paginatedParticipants,
		TotalCount:   totalCount,
	}, nil
}

// GetParticipant retrieves a specific participant from a room
func (s *ParticipantService) GetParticipant(ctx context.Context, roomID, userID string) (*models.RoomParticipant, error) {
	if roomID == "" || userID == "" {
		return nil, fmt.Errorf("roomID and userID are required")
	}

	participantsKey := fmt.Sprintf("room:%s:participants", roomID)
	result := s.client.HGet(ctx, participantsKey, userID)

	dataJSON, err := result.Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("participant not found")
		}
		return nil, fmt.Errorf("failed to get participant: %w", err)
	}

	var participantData map[string]interface{}
	if err := json.Unmarshal([]byte(dataJSON), &participantData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal participant data: %w", err)
	}

	// Create RoomParticipant from the data
	participant := &models.RoomParticipant{
		RoomID:   roomID,
		UserID:   userID,
		IsActive: true, // Default to active
	}

	if joinedAtStr, ok := participantData["joined_at"].(string); ok {
		if joinedAt, err := time.Parse(time.RFC3339, joinedAtStr); err == nil {
			participant.JoinedAt = joinedAt
		}
	}

	if isActive, ok := participantData["is_active"].(bool); ok {
		participant.IsActive = isActive
	}

	return participant, nil
}

// IsUserInRoom checks if a user is a participant in a room
func (s *ParticipantService) IsUserInRoom(ctx context.Context, roomID, userID string) (bool, error) {
	if roomID == "" || userID == "" {
		return false, fmt.Errorf("roomID and userID are required")
	}

	participantsKey := fmt.Sprintf("room:%s:participants", roomID)
	exists := s.client.HExists(ctx, participantsKey, userID)

	result, err := exists.Result()
	if err != nil {
		return false, fmt.Errorf("failed to check participant existence: %w", err)
	}

	return result, nil
}

// GetParticipantCount returns the number of participants in a room
func (s *ParticipantService) GetParticipantCount(ctx context.Context, roomID string) (int, error) {
	if roomID == "" {
		return 0, fmt.Errorf("roomID is required")
	}

	participantsKey := fmt.Sprintf("room:%s:participants", roomID)
	result := s.client.HLen(ctx, participantsKey)

	count, err := result.Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get participant count: %w", err)
	}

	return int(count), nil
}

// GetUserRoomsRequest represents the request to get rooms a user participates in
type GetUserRoomsRequest struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

// GetUserRoomsResponse represents the response with user's rooms
type GetUserRoomsResponse struct {
	RoomIDs []string `json:"room_ids"`
}

// GetUserRooms retrieves all rooms a user is participating in
// Note: This is a more expensive operation as it requires scanning multiple keys
func (s *ParticipantService) GetUserRooms(ctx context.Context, req *GetUserRoomsRequest) (*GetUserRoomsResponse, error) {
	if err := s.validateGetUserRoomsRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Scan for room participant keys
	pattern := "room:*:participants"
	var cursor uint64
	var roomIDs []string

	for {
		result := s.client.Scan(ctx, cursor, pattern, 100)
		keys, newCursor, err := result.Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan for participant keys: %w", err)
		}

		// Check each participant key for the user
		for _, key := range keys {
			exists := s.client.HExists(ctx, key, req.UserID)
			if isParticipant, err := exists.Result(); err == nil && isParticipant {
				// Extract room ID from key (format: room:{room_id}:participants)
				parts := strings.Split(key, ":")
				if len(parts) == 3 {
					roomIDs = append(roomIDs, parts[1])
				}
			}
		}

		cursor = newCursor
		if cursor == 0 {
			break
		}
	}

	return &GetUserRoomsResponse{
		RoomIDs: roomIDs,
	}, nil
}

// updateParticipantCount updates the participant count in the room metadata
func (s *ParticipantService) updateParticipantCount(ctx context.Context, roomID string) (int, error) {
	count, err := s.GetParticipantCount(ctx, roomID)
	if err != nil {
		return 0, err
	}

	// Update the room's participant count in both the room document and room list
	roomKey := fmt.Sprintf("room:%s", roomID)
	roomListKey := "rooms:list"

	// Update room document
	err = s.client.Do(ctx, "JSON.SET", roomKey, "$.participant_count", count).Err()
	if err != nil {
		return count, fmt.Errorf("failed to update room participant count: %w", err)
	}

	// Update room list
	err = s.client.Do(ctx, "JSON.SET", roomListKey, fmt.Sprintf("$.%s.participant_count", roomID), count).Err()
	if err != nil {
		return count, fmt.Errorf("failed to update room list participant count: %w", err)
	}

	return count, nil
}

// Validation methods

func (s *ParticipantService) validateJoinRoomRequest(req *JoinRoomRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.RoomID == "" {
		return fmt.Errorf("room_id is required")
	}

	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	if req.Username == "" {
		return fmt.Errorf("username is required")
	}

	return nil
}

func (s *ParticipantService) validateLeaveRoomRequest(req *LeaveRoomRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.RoomID == "" {
		return fmt.Errorf("room_id is required")
	}

	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	return nil
}

func (s *ParticipantService) validateGetParticipantsRequest(req *GetParticipantsRequest) error {
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

	if req.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}

	return nil
}

func (s *ParticipantService) validateGetUserRoomsRequest(req *GetUserRoomsRequest) error {
	if req == nil {
		return fmt.Errorf("request cannot be nil")
	}

	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}

	return nil
}