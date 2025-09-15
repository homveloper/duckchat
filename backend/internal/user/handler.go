package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"duckchat/internal/jsonrpc"
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


// HandleUser handles user-related JSON-RPC requests
func (h *Handler) HandleUser(w http.ResponseWriter, r *http.Request) {
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
	case "users.create":
		h.handleCreateUser(w, r.Context(), req)
	case "users.get":
		h.handleGetUser(w, r.Context(), req)
	case "users.updateUsername":
		h.handleUpdateUsername(w, r.Context(), req)
	case "users.info":
		h.handleGetUserInfo(w, r.Context(), req)
	default:
		jsonrpc.WriteError(w, jsonrpc.MethodNotFound, "Method not found", nil, req.ID)
	}
}

// handleCreateUser processes user creation requests
func (h *Handler) handleCreateUser(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse parameters
	var createReq CreateUserRequest
	if err := h.parseParams(req.Params, &createReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate request
	if err := ValidateCreateUserRequest(&createReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Validation failed", err.Error(), req.ID)
		return
	}

	// Process user creation
	user, err := h.service.CreateUser(ctx, createReq.Username)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "User creation failed", err.Error(), req.ID)
		return
	}

	// Convert to response format
	userInfo := &UserInfo{
		UserID:   user.ID.String(),
		Username: user.Username.String(),
	}

	jsonrpc.WriteResponse(w, userInfo, req.ID)
}

// handleGetUser processes user retrieval requests
func (h *Handler) handleGetUser(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse parameters
	var getUserReq GetUserRequest
	if err := h.parseParams(req.Params, &getUserReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Convert string ID to UserID
	userID, err := ParseUserID(getUserReq.UserID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Invalid user ID", err.Error(), req.ID)
		return
	}

	// Get user
	user, err := h.service.GetUser(ctx, userID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to get user", err.Error(), req.ID)
		return
	}

	// Convert to response format
	userInfo := &UserInfo{
		UserID:   user.ID.String(),
		Username: user.Username.String(),
	}

	jsonrpc.WriteResponse(w, userInfo, req.ID)
}

// handleUpdateUsername processes username update requests
func (h *Handler) handleUpdateUsername(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Get user ID from context (set by auth middleware)
	userIDString, ok := ctx.Value("user_id").(string)
	if !ok {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Authentication required", nil, req.ID)
		return
	}

	userID, err := ParseUserID(userIDString)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Invalid user ID in token", err.Error(), req.ID)
		return
	}

	// Parse parameters
	var updateReq UpdateUsernameRequest
	if err := h.parseParams(req.Params, &updateReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate request
	if err := ValidateUpdateUsernameRequest(&updateReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Validation failed", err.Error(), req.ID)
		return
	}

	// Update username
	user, err := h.service.UpdateUsername(ctx, userID, updateReq.NewUsername)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Username update failed", err.Error(), req.ID)
		return
	}

	// Convert to response format with updated_at timestamp
	updateResponse := h.service.ToUserUpdateResponse(user)

	jsonrpc.WriteResponse(w, updateResponse, req.ID)
}

// handleGetUserInfo processes user info requests
func (h *Handler) handleGetUserInfo(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Get user ID from context (set by auth middleware)
	userIDString, ok := ctx.Value("user_id").(string)
	if !ok {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Authentication required", nil, req.ID)
		return
	}

	userID, err := ParseUserID(userIDString)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Invalid user ID in token", err.Error(), req.ID)
		return
	}

	// Get user info
	userInfo, err := h.service.GetUserInfo(ctx, userID)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.InternalError, "Failed to get user info", err.Error(), req.ID)
		return
	}

	jsonrpc.WriteResponse(w, userInfo, req.ID)
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