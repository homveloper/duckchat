package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Handler handles user-related JSON-RPC requests
type Handler struct {
	service *Service
}

// NewHandler creates a new user handler
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
	UserError       = -32003
)

// HandleUser handles user-related JSON-RPC requests
func (h *Handler) HandleUser(w http.ResponseWriter, r *http.Request) {
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
	case "users.create":
		h.handleCreateUser(w, r.Context(), req)
	case "users.get":
		h.handleGetUser(w, r.Context(), req)
	case "users.updateUsername":
		h.handleUpdateUsername(w, r.Context(), req)
	case "users.info":
		h.handleGetUserInfo(w, r.Context(), req)
	default:
		h.sendError(w, MethodNotFound, "Method not found", nil, req.ID)
	}
}

// handleCreateUser processes user creation requests
func (h *Handler) handleCreateUser(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Parse parameters
	var createReq CreateUserRequest
	if err := h.parseParams(req.Params, &createReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate request
	if err := ValidateCreateUserRequest(&createReq); err != nil {
		h.sendError(w, ValidationError, "Validation failed", err.Error(), req.ID)
		return
	}

	// Process user creation
	user, err := h.service.CreateUser(ctx, createReq.Username)
	if err != nil {
		h.sendError(w, UserError, "User creation failed", err.Error(), req.ID)
		return
	}

	// Convert to response format
	userInfo := &UserInfo{
		UserID:   user.ID.String(),
		Username: user.Username.String(),
	}

	h.sendResult(w, userInfo, req.ID)
}

// handleGetUser processes user retrieval requests
func (h *Handler) handleGetUser(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Parse parameters
	var getUserReq GetUserRequest
	if err := h.parseParams(req.Params, &getUserReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Convert string ID to UserID
	userID, err := ParseUserID(getUserReq.UserID)
	if err != nil {
		h.sendError(w, ValidationError, "Invalid user ID", err.Error(), req.ID)
		return
	}

	// Get user
	user, err := h.service.GetUser(ctx, userID)
	if err != nil {
		h.sendError(w, UserError, "Failed to get user", err.Error(), req.ID)
		return
	}

	// Convert to response format
	userInfo := &UserInfo{
		UserID:   user.ID.String(),
		Username: user.Username.String(),
	}

	h.sendResult(w, userInfo, req.ID)
}

// handleUpdateUsername processes username update requests
func (h *Handler) handleUpdateUsername(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Get user ID from context (set by auth middleware)
	userIDString, ok := ctx.Value("user_id").(string)
	if !ok {
		h.sendError(w, AuthError, "Authentication required", nil, req.ID)
		return
	}

	userID, err := ParseUserID(userIDString)
	if err != nil {
		h.sendError(w, AuthError, "Invalid user ID in token", err.Error(), req.ID)
		return
	}

	// Parse parameters
	var updateReq UpdateUsernameRequest
	if err := h.parseParams(req.Params, &updateReq); err != nil {
		h.sendError(w, InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate request
	if err := ValidateUpdateUsernameRequest(&updateReq); err != nil {
		h.sendError(w, ValidationError, "Validation failed", err.Error(), req.ID)
		return
	}

	// Update username
	user, err := h.service.UpdateUsername(ctx, userID, updateReq.NewUsername)
	if err != nil {
		h.sendError(w, UserError, "Username update failed", err.Error(), req.ID)
		return
	}

	// Convert to response format with updated_at timestamp
	updateResponse := h.service.ToUserUpdateResponse(user)

	h.sendResult(w, updateResponse, req.ID)
}

// handleGetUserInfo processes user info requests
func (h *Handler) handleGetUserInfo(w http.ResponseWriter, ctx context.Context, req JSONRPCRequest) {
	// Get user ID from context (set by auth middleware)
	userIDString, ok := ctx.Value("user_id").(string)
	if !ok {
		h.sendError(w, AuthError, "Authentication required", nil, req.ID)
		return
	}

	userID, err := ParseUserID(userIDString)
	if err != nil {
		h.sendError(w, AuthError, "Invalid user ID in token", err.Error(), req.ID)
		return
	}

	// Get user info
	userInfo, err := h.service.GetUserInfo(ctx, userID)
	if err != nil {
		h.sendError(w, UserError, "Failed to get user info", err.Error(), req.ID)
		return
	}

	h.sendResult(w, userInfo, req.ID)
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
	case UserError:
		httpStatus = http.StatusBadRequest
	default:
		httpStatus = http.StatusInternalServerError
	}

	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(response)
}

// Request/Response types
type CreateUserRequest struct {
	Username string `json:"username"`
}

type GetUserRequest struct {
	UserID string `json:"user_id"`
}

type UpdateUsernameRequest struct {
	NewUsername string `json:"new_username"`
}

// Validation functions
func ValidateCreateUserRequest(req *CreateUserRequest) error {
	if req.Username == "" {
		return fmt.Errorf("username is required")
	}
	if len(req.Username) < 1 || len(req.Username) > 50 {
		return fmt.Errorf("username must be 1-50 characters")
	}
	return nil
}

func ValidateUpdateUsernameRequest(req *UpdateUsernameRequest) error {
	if req.NewUsername == "" {
		return fmt.Errorf("new username is required")
	}
	if len(req.NewUsername) < 1 || len(req.NewUsername) > 50 {
		return fmt.Errorf("new username must be 1-50 characters")
	}
	return nil
}