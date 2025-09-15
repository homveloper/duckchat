package message

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Handler handles message-related JSON-RPC requests
type Handler struct {
	service *Service
}

// NewHandler creates a new message handler
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
	MessageError    = -32005
)

// HandleMessage handles message-related JSON-RPC requests
func (h *Handler) HandleMessage(w http.ResponseWriter, r *http.Request) {
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
	case "messages.send":
		h.handleSendMessage(w, r.Context(), req)
	case "messages.history":
		h.handleGetHistory(w, r.Context(), req)
	case "messages.recent":
		h.handleGetRecent(w, r.Context(), req)
	case "messages.get":
		h.handleGetMessage(w, r.Context(), req)
	default:
		h.sendError(w, MethodNotFound, "Method not found", nil, req.ID)
	}
}

// handleSendMessage processes message send requests
func (h *Handler) handleSendMessage(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Get user ID from context
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		h.sendError(w, AuthError, "Authentication required", nil, req.ID)
		return
	}

	// Parse parameters
	var sendReq SendMessageRequest
	if err := h.parseParams(req.Params, &sendReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate request
	if err := h.service.ValidateMessageRequest(ctx, sendReq.RoomID, userID, sendReq.Content); err != nil {
		// Check if error is about non-existent room (business logic error)
		if err.Error() == "room does not exist" {
			h.sendError(w, MessageError, "Failed to send message", err.Error(), req.ID)
		} else {
			h.sendError(w, ValidationError, "Validation failed", err.Error(), req.ID)
		}
		return
	}

	// Send message
	message, err := h.service.SendMessage(ctx, sendReq.RoomID, userID, sendReq.Content)
	if err != nil {
		h.sendError(w, MessageError, "Failed to send message", err.Error(), req.ID)
		return
	}

	// Convert to response format (username resolution would happen here)
	messageResponse := h.service.ToMessageResponse(message, "user") // TODO: Resolve username
	h.sendResult(w, messageResponse, req.ID)
}

// handleGetHistory processes message history requests
func (h *Handler) handleGetHistory(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Parse parameters
	var historyReq GetHistoryRequest
	if err := h.parseParams(req.Params, &historyReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate room ID
	if historyReq.RoomID == "" {
		h.sendError(w, ValidationError, "Validation failed", "Room ID is required", req.ID)
		return
	}

	// Validate room exists using the same validation logic as message send
	if err := h.service.ValidateRoomExists(ctx, historyReq.RoomID); err != nil {
		h.sendError(w, MessageError, "Failed to get message history", err.Error(), req.ID)
		return
	}

	// Get message history
	history, err := h.service.GetRoomHistory(ctx, historyReq.RoomID, historyReq.Limit, historyReq.Before)
	if err != nil {
		h.sendError(w, MessageError, "Failed to get message history", err.Error(), req.ID)
		return
	}

	// Convert to response format with username resolution
	usernameResolver := func(userID string) string {
		// TODO: Implement actual username resolution via user service
		return "user"
	}

	historyResponse := h.service.ToMessageHistoryResponse(history, usernameResolver)
	h.sendResult(w, historyResponse, req.ID)
}

// handleGetRecent processes recent messages requests
func (h *Handler) handleGetRecent(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Parse parameters
	var recentReq GetRecentRequest
	if err := h.parseParams(req.Params, &recentReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate room ID
	if recentReq.RoomID == "" {
		h.sendError(w, ValidationError, "Room ID is required", nil, req.ID)
		return
	}

	// Get recent messages
	messages, err := h.service.GetRecentMessages(ctx, recentReq.RoomID, recentReq.Limit)
	if err != nil {
		h.sendError(w, MessageError, "Failed to get recent messages", err.Error(), req.ID)
		return
	}

	// Convert to response format
	messageResponses := make([]MessageResponse, len(messages))
	for i, msg := range messages {
		messageResponses[i] = *h.service.ToMessageResponse(&msg, "user") // TODO: Resolve username
	}

	result := map[string]interface{}{
		"room_id":  recentReq.RoomID,
		"messages": messageResponses,
	}

	h.sendResult(w, result, req.ID)
}

// handleGetMessage processes single message retrieval requests
func (h *Handler) handleGetMessage(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Parse parameters
	var getReq GetMessageRequest
	if err := h.parseParams(req.Params, &getReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate message ID
	if getReq.MessageID == "" {
		h.sendError(w, ValidationError, "Message ID is required", nil, req.ID)
		return
	}

	// Convert string ID to MessageID
	messageID, err := ParseMessageID(getReq.MessageID)
	if err != nil {
		h.sendError(w, ValidationError, "Invalid message ID", err.Error(), req.ID)
		return
	}

	// Get message
	message, err := h.service.GetMessage(ctx, messageID)
	if err != nil {
		h.sendError(w, MessageError, "Failed to get message", err.Error(), req.ID)
		return
	}

	// Convert to response format
	messageResponse := h.service.ToMessageResponse(message, "user") // TODO: Resolve username
	h.sendResult(w, messageResponse, req.ID)
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
	case MessageError:
		httpStatus = http.StatusBadRequest
	default:
		httpStatus = http.StatusInternalServerError
	}

	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(response)
}

// Request types
type GetRecentRequest struct {
	RoomID string `json:"room_id"`
	Limit  int    `json:"limit,omitempty"`
}

type GetMessageRequest struct {
	MessageID string `json:"message_id"`
}

