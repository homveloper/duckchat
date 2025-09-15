package room

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"duckchat/internal/jsonrpc"
	"duckchat/internal/services"
)

// Handler handles room-related JSON-RPC requests
type Handler struct {
	service            *Service
	roomService        *services.RoomService
	participantService *services.ParticipantService
	sessionService     *services.SessionService
}

// NewHandler creates a new room handler
func NewHandler(service *Service, roomService *services.RoomService, participantService *services.ParticipantService, sessionService *services.SessionService) *Handler {
	return &Handler{
		service:            service,
		roomService:        roomService,
		participantService: participantService,
		sessionService:     sessionService,
	}
}


// HandleRoom handles room-related JSON-RPC requests
func (h *Handler) HandleRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse request
	var req jsonrpc.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonrpc.WriteError(w, jsonrpc.ParseError, "Parse error", nil, nil)
		return
	}

	// Validate JSON-RPC version
	if req.JSONRpc != "2.0" {
		jsonrpc.WriteError(w, jsonrpc.InvalidRequest, "Invalid Request", nil, req.ID)
		return
	}

	// Route method
	switch req.Method {
	case "rooms.list":
		h.handleListRooms(w, r.Context(), req)
	case "rooms.create":
		h.handleCreateRoom(w, r.Context(), req)
	case "rooms.join":
		h.handleJoinRoom(w, r.Context(), req)
	case "rooms.leave":
		h.handleLeaveRoom(w, r.Context(), req)
	case "rooms.info":
		h.handleGetRoomInfo(w, r.Context(), req)
	case "rooms.participants":
		h.handleGetParticipants(w, r.Context(), req)
	default:
		jsonrpc.WriteError(w, jsonrpc.MethodNotFound, "Method not found", nil, req.ID)
	}
}

// handleListRooms processes room listing requests
func (h *Handler) handleListRooms(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse parameters (optional)
	type ListRoomsParams struct {
		Filters *struct {
			IsActive *bool `json:"is_active,omitempty"`
		} `json:"filters,omitempty"`
		Limit  *int `json:"limit,omitempty"`
		Offset *int `json:"offset,omitempty"`
	}

	var params ListRoomsParams
	if req.Params != nil {
		if err := h.parseParams(req.Params, &params); err != nil {
			jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
			return
		}
	}

	// Set defaults
	limit := 50
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = *params.Limit
	}

	offset := 0
	if params.Offset != nil && *params.Offset >= 0 {
		offset = *params.Offset
	}

	var isActive *bool
	if params.Filters != nil {
		isActive = params.Filters.IsActive
	}

	// Create service request
	serviceReq := &services.ListRoomsRequest{
		IsActive: isActive,
		Limit:    limit,
		Offset:   offset,
	}

	// Call service
	resp, err := h.roomService.ListRooms(ctx, serviceReq)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to retrieve rooms", err.Error(), req.ID)
		return
	}

	result := map[string]interface{}{
		"rooms":       resp.Rooms,
		"total_count": resp.TotalCount,
	}

	jsonrpc.WriteResponse(w, result, req.ID)
}

// handleCreateRoom processes room creation requests
func (h *Handler) handleCreateRoom(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Authentication required", nil, req.ID)
		return
	}

	// Parse parameters
	var createReq CreateRoomRequest
	if err := h.parseParams(req.Params, &createReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate request
	if err := ValidateCreateRoomRequest(&createReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Validation failed", err.Error(), req.ID)
		return
	}

	// Create room
	room, err := h.service.CreateRoom(ctx, createReq.Name, userID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Room creation failed", err.Error(), req.ID)
		return
	}

	// Convert to response format using service layer
	roomInfo := h.service.ToRoomInfo(room, []string{userID})

	jsonrpc.WriteResponse(w, roomInfo, req.ID)
}

// handleJoinRoom processes room join requests
func (h *Handler) handleJoinRoom(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Authentication required", nil, req.ID)
		return
	}

	// Parse parameters
	var joinReq JoinRoomRequest
	if err := h.parseParams(req.Params, &joinReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate request
	if err := ValidateJoinRoomRequest(&joinReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Validation failed", err.Error(), req.ID)
		return
	}

	// Convert string ID to RoomID
	roomID, err := ParseRoomID(joinReq.RoomID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Invalid room ID", err.Error(), req.ID)
		return
	}

	// Join room
	_, err = h.service.JoinRoom(ctx, roomID, userID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to join room", err.Error(), req.ID)
		return
	}

	// Get room info with participants
	roomInfo, err := h.service.GetRoomInfo(ctx, roomID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to get room info", err.Error(), req.ID)
		return
	}

	jsonrpc.WriteResponse(w, roomInfo, req.ID)
}

// handleLeaveRoom processes room leave requests
func (h *Handler) handleLeaveRoom(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Authentication required", nil, req.ID)
		return
	}

	// Parse parameters
	var leaveReq LeaveRoomRequest
	if err := h.parseParams(req.Params, &leaveReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate room ID
	if leaveReq.RoomID == "" {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Room ID is required", nil, req.ID)
		return
	}

	// Convert string ID to RoomID
	roomID, err := ParseRoomID(leaveReq.RoomID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Invalid room ID", err.Error(), req.ID)
		return
	}

	// Leave room
	err = h.service.LeaveRoom(ctx, roomID, userID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to leave room", err.Error(), req.ID)
		return
	}

	jsonrpc.WriteResponse(w, map[string]string{"status": "success"}, req.ID)
}

// handleGetRoomInfo processes room info requests
func (h *Handler) handleGetRoomInfo(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse parameters
	var infoReq RoomInfoRequest
	if err := h.parseParams(req.Params, &infoReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate room ID
	if infoReq.RoomID == "" {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Room ID is required", nil, req.ID)
		return
	}

	// Convert string ID to RoomID
	roomID, err := ParseRoomID(infoReq.RoomID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Invalid room ID", err.Error(), req.ID)
		return
	}

	// Get room info
	roomInfo, err := h.service.GetRoomInfo(ctx, roomID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to get room info", err.Error(), req.ID)
		return
	}

	jsonrpc.WriteResponse(w, roomInfo, req.ID)
}

// handleGetParticipants processes room participants requests
func (h *Handler) handleGetParticipants(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse parameters
	var participantsReq ParticipantsRequest
	if err := h.parseParams(req.Params, &participantsReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate room ID
	if participantsReq.RoomID == "" {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Room ID is required", nil, req.ID)
		return
	}

	// Convert string ID to RoomID
	roomID, err := ParseRoomID(participantsReq.RoomID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Invalid room ID", err.Error(), req.ID)
		return
	}

	// Get participants
	participants, err := h.service.GetRoomParticipants(ctx, roomID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to get participants", err.Error(), req.ID)
		return
	}

	result := map[string]interface{}{
		"room_id":      participantsReq.RoomID,
		"participants": participants,
	}

	jsonrpc.WriteResponse(w, result, req.ID)
}

// parseParams parses JSON-RPC parameters into a struct
func (h *Handler) parseParams(params interface{}, target interface{}) error {
	if params == nil {
		return nil
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to marshal params: %w", err)
	}

	err = json.Unmarshal(paramsJSON, target)
	if err != nil {
		return fmt.Errorf("failed to unmarshal params: %w", err)
	}

	return nil
}


// Request types
type LeaveRoomRequest struct {
	RoomID string `json:"room_id"`
}

type RoomInfoRequest struct {
	RoomID string `json:"room_id"`
}

type ParticipantsRequest struct {
	RoomID string `json:"room_id"`
}