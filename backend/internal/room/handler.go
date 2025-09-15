package room

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Handler handles room-related JSON-RPC requests
type Handler struct {
	service *Service
}

// NewHandler creates a new room handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Error codes
const (
	ParseError      = -32700
	InvalidRequest  = -32600
	MethodNotFound  = -32601
	InvalidParams   = -32602
	InternalError   = -32603
	AuthError       = -32001
	ValidationError = -32002
	RoomError       = -32004
)

// HandleRoom handles room-related JSON-RPC requests
func (h *Handler) HandleRoom(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse request
	var req JSONRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, ParseError, "Parse error", nil, nil)
		return
	}

	// Validate JSON-RPC version
	if req.JSONRPC != "2.0" {
		h.sendError(w, InvalidRequest, "Invalid Request", nil, req.ID)
		return
	}

	// Route method
	switch req.Method {
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
		h.sendError(w, MethodNotFound, "Method not found", nil, req.ID)
	}
}

// handleCreateRoom processes room creation requests
func (h *Handler) handleCreateRoom(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.sendError(w, AuthError, "Authentication required", nil, req.ID)
		return
	}

	// Parse parameters
	var createReq CreateRoomRequest
	if err := h.parseParams(req.Params, &createReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate request
	if err := ValidateCreateRoomRequest(&createReq); err != nil {
		h.sendError(w, ValidationError, "Validation failed", err.Error(), req.ID)
		return
	}

	// Create room
	room, err := h.service.CreateRoom(ctx, createReq.Name, userID)
	if err != nil {
		h.sendError(w, RoomError, "Room creation failed", err.Error(), req.ID)
		return
	}

	// Convert to response format using service layer
	roomInfo := h.service.ToRoomInfo(room, []string{userID})

	h.sendResult(w, roomInfo, req.ID)
}

// handleJoinRoom processes room join requests
func (h *Handler) handleJoinRoom(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.sendError(w, AuthError, "Authentication required", nil, req.ID)
		return
	}

	// Parse parameters
	var joinReq JoinRoomRequest
	if err := h.parseParams(req.Params, &joinReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate request
	if err := ValidateJoinRoomRequest(&joinReq); err != nil {
		h.sendError(w, ValidationError, "Validation failed", err.Error(), req.ID)
		return
	}

	// Convert string ID to RoomID
	roomID, err := ParseRoomID(joinReq.RoomID)
	if err != nil {
		h.sendError(w, ValidationError, "Invalid room ID", err.Error(), req.ID)
		return
	}

	// Join room
	_, err = h.service.JoinRoom(ctx, roomID, userID)
	if err != nil {
		h.sendError(w, RoomError, "Failed to join room", err.Error(), req.ID)
		return
	}

	// Get room info with participants
	roomInfo, err := h.service.GetRoomInfo(ctx, roomID)
	if err != nil {
		h.sendError(w, RoomError, "Failed to get room info", err.Error(), req.ID)
		return
	}

	h.sendResult(w, roomInfo, req.ID)
}

// handleLeaveRoom processes room leave requests
func (h *Handler) handleLeaveRoom(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.sendError(w, AuthError, "Authentication required", nil, req.ID)
		return
	}

	// Parse parameters
	var leaveReq LeaveRoomRequest
	if err := h.parseParams(req.Params, &leaveReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate room ID
	if leaveReq.RoomID == "" {
		h.sendError(w, ValidationError, "Room ID is required", nil, req.ID)
		return
	}

	// Convert string ID to RoomID
	roomID, err := ParseRoomID(leaveReq.RoomID)
	if err != nil {
		h.sendError(w, ValidationError, "Invalid room ID", err.Error(), req.ID)
		return
	}

	// Leave room
	err = h.service.LeaveRoom(ctx, roomID, userID)
	if err != nil {
		h.sendError(w, RoomError, "Failed to leave room", err.Error(), req.ID)
		return
	}

	h.sendResult(w, map[string]string{"status": "success"}, req.ID)
}

// handleGetRoomInfo processes room info requests
func (h *Handler) handleGetRoomInfo(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Parse parameters
	var infoReq RoomInfoRequest
	if err := h.parseParams(req.Params, &infoReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate room ID
	if infoReq.RoomID == "" {
		h.sendError(w, ValidationError, "Room ID is required", nil, req.ID)
		return
	}

	// Convert string ID to RoomID
	roomID, err := ParseRoomID(infoReq.RoomID)
	if err != nil {
		h.sendError(w, ValidationError, "Invalid room ID", err.Error(), req.ID)
		return
	}

	// Get room info
	roomInfo, err := h.service.GetRoomInfo(ctx, roomID)
	if err != nil {
		h.sendError(w, RoomError, "Failed to get room info", err.Error(), req.ID)
		return
	}

	h.sendResult(w, roomInfo, req.ID)
}

// handleGetParticipants processes room participants requests
func (h *Handler) handleGetParticipants(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Parse parameters
	var participantsReq ParticipantsRequest
	if err := h.parseParams(req.Params, &participantsReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate room ID
	if participantsReq.RoomID == "" {
		h.sendError(w, ValidationError, "Room ID is required", nil, req.ID)
		return
	}

	// Convert string ID to RoomID
	roomID, err := ParseRoomID(participantsReq.RoomID)
	if err != nil {
		h.sendError(w, ValidationError, "Invalid room ID", err.Error(), req.ID)
		return
	}

	// Get participants
	participants, err := h.service.GetRoomParticipants(ctx, roomID)
	if err != nil {
		h.sendError(w, RoomError, "Failed to get participants", err.Error(), req.ID)
		return
	}

	result := map[string]interface{}{
		"room_id":      participantsReq.RoomID,
		"participants": participants,
	}

	h.sendResult(w, result, req.ID)
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

// sendResult sends a successful JSON-RPC response
func (h *Handler) sendResult(w http.ResponseWriter, result interface{}, id interface{}) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// sendError sends a JSON-RPC error response
func (h *Handler) sendError(w http.ResponseWriter, code int, message string, data interface{}, id interface{}) {
	response := JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}

	var httpStatus int
	switch code {
	case ParseError, InvalidRequest, InvalidParams, ValidationError:
		httpStatus = http.StatusBadRequest
	case MethodNotFound:
		httpStatus = http.StatusNotFound
	case AuthError:
		httpStatus = http.StatusUnauthorized
	case RoomError:
		httpStatus = http.StatusBadRequest
	default:
		httpStatus = http.StatusInternalServerError
	}

	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(response)
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