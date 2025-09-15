package message

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"duckchat/internal/jsonrpc"
	"duckchat/internal/services"
)

// Handler handles message-related JSON-RPC requests
type Handler struct {
	service            *Service
	messageService     *services.MessageService
	participantService *services.ParticipantService
}

// NewHandler creates a new message handler
func NewHandler(service *Service, messageService *services.MessageService, participantService *services.ParticipantService) *Handler {
	return &Handler{
		service:            service,
		messageService:     messageService,
		participantService: participantService,
	}
}


// HandleMessage handles message-related JSON-RPC requests
func (h *Handler) HandleMessage(w http.ResponseWriter, r *http.Request) {
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
	case "messages.send":
		h.handleSendMessage(w, r.Context(), req)
	case "messages.history":
		h.handleGetHistory(w, r.Context(), req)
	case "messages.recent":
		h.handleGetRecent(w, r.Context(), req)
	case "messages.get":
		h.handleGetMessage(w, r.Context(), req)
	default:
		jsonrpc.WriteError(w, jsonrpc.MethodNotFound, "Method not found", nil, req.ID)
	}
}

// handleSendMessage processes message send requests
func (h *Handler) handleSendMessage(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Authentication required", nil, req.ID)
		return
	}

	// Parse parameters
	type SendMessageParams struct {
		RoomID  string `json:"room_id"`
		Content string `json:"content"`
	}

	var params SendMessageParams
	if err := h.parseParams(req.Params, &params); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate required fields
	if params.RoomID == "" {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Room ID is required", nil, req.ID)
		return
	}

	if params.Content == "" {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Message content is required", nil, req.ID)
		return
	}

	// Check if user is participant of the room
	isParticipant, err := h.participantService.IsUserInRoom(ctx, params.RoomID, userID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to check room participation", err.Error(), req.ID)
		return
	}

	if !isParticipant {
		jsonrpc.WriteError(w, jsonrpc.PermissionDenied, "User is not a participant of this room", nil, req.ID)
		return
	}

	// Create service request
	serviceReq := &services.SendMessageRequest{
		RoomID:  params.RoomID,
		UserID:  userID,
		Content: params.Content,
	}

	// Call service
	_, err = h.messageService.SendMessage(ctx, serviceReq)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to send message", err.Error(), req.ID)
		return
	}

	// CQRS pattern: Command returns only success status
	// The actual message will be received via SSE events
	result := map[string]interface{}{
		"success": true,
	}

	jsonrpc.WriteResponse(w, result, req.ID)
}

// handleGetHistory processes message history requests
func (h *Handler) handleGetHistory(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Authentication required", nil, req.ID)
		return
	}

	// Parse parameters
	type GetHistoryParams struct {
		RoomID        string  `json:"room_id"`
		Limit         *int    `json:"limit,omitempty"`
		LastMessageID *string `json:"last_message_id,omitempty"`
	}

	var params GetHistoryParams
	if err := h.parseParams(req.Params, &params); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate room ID
	if params.RoomID == "" {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Validation failed", "Room ID is required", req.ID)
		return
	}

	// Check if user is participant of the room
	isParticipant, err := h.participantService.IsUserInRoom(ctx, params.RoomID, userID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to check room participation", err.Error(), req.ID)
		return
	}

	if !isParticipant {
		jsonrpc.WriteError(w, jsonrpc.PermissionDenied, "User is not a participant of this room", nil, req.ID)
		return
	}

	// Set defaults
	limit := 50
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = *params.Limit
	}

	// Create service request
	serviceReq := &services.GetMessagesRequest{
		RoomID:        params.RoomID,
		Limit:         limit,
		LastMessageID: params.LastMessageID,
		Direction:     "newer", // Default to newer messages
	}

	// Call service
	resp, err := h.messageService.GetMessages(ctx, serviceReq)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to get message history", err.Error(), req.ID)
		return
	}

	result := map[string]interface{}{
		"messages":         resp.Messages,
		"room_id":          params.RoomID,
		"has_more":         resp.HasMore,
		"last_message_id":  resp.LastMessageID,
		"first_message_id": resp.FirstMessageID,
	}

	jsonrpc.WriteResponse(w, result, req.ID)
}

// handleGetRecent processes recent messages requests
func (h *Handler) handleGetRecent(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse parameters
	var recentReq GetRecentRequest
	if err := h.parseParams(req.Params, &recentReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate room ID
	if recentReq.RoomID == "" {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Room ID is required", nil, req.ID)
		return
	}

	// Get recent messages
	messages, err := h.service.GetRecentMessages(ctx, recentReq.RoomID, recentReq.Limit)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to get recent messages", err.Error(), req.ID)
		return
	}

	// Convert to response format
	messageResponses := make([]MessageResponse, len(messages))
	for i, msg := range messages {
		messageResponses[i] = *h.service.ToMessageResponse(&msg)
	}

	result := map[string]interface{}{
		"room_id":  recentReq.RoomID,
		"messages": messageResponses,
	}

	jsonrpc.WriteResponse(w, result, req.ID)
}

// handleGetMessage processes single message retrieval requests
func (h *Handler) handleGetMessage(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse parameters
	var getReq GetMessageRequest
	if err := h.parseParams(req.Params, &getReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate message ID
	if getReq.MessageID == "" {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Message ID is required", nil, req.ID)
		return
	}

	// Convert string ID to MessageID
	messageID, err := ParseMessageID(getReq.MessageID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Invalid message ID", err.Error(), req.ID)
		return
	}

	// Get message
	message, err := h.service.GetMessage(ctx, messageID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to get message", err.Error(), req.ID)
		return
	}

	// Convert to response format
	messageResponse := h.service.ToMessageResponse(message)
	jsonrpc.WriteResponse(w, messageResponse, req.ID)
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
type GetRecentRequest struct {
	RoomID string `json:"room_id"`
	Limit  int    `json:"limit,omitempty"`
}

type GetMessageRequest struct {
	MessageID string `json:"message_id"`
}

