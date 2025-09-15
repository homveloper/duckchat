package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"duckchat/internal/jsonrpc"
)

// Handler handles authentication JSON-RPC requests
type Handler struct {
	service *Service
}

// NewHandler creates a new auth handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}


// HandleAuth handles authentication-related JSON-RPC requests
func (h *Handler) HandleAuth(w http.ResponseWriter, r *http.Request) {
	// Set headers
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
	case "auth.login":
		h.handleLogin(w, r.Context(), req)
	case "auth.login.guest":
		h.handleGuestLogin(w, r.Context(), req)
	case "auth.login.google":
		h.handleGoogleLogin(w, r.Context(), req)
	case "auth.login.apple":
		h.handleAppleLogin(w, r.Context(), req)
	case "auth.link":
		h.handleLinkAuth(w, r.Context(), req)
	case "auth.logout":
		h.handleLogout(w, r.Context(), req)
	case "auth.refresh":
		h.handleRefresh(w, r.Context(), req)
	case "auth.validate":
		h.handleValidate(w, r.Context(), req)
	default:
		jsonrpc.WriteError(w, jsonrpc.MethodNotFound, "Method not found", nil, req.ID)
	}
}

// handleLogin processes generic login requests (delegates to specific providers)
func (h *Handler) handleLogin(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse parameters
	var loginReq LoginRequest
	if err := h.parseParams(req.Params, &loginReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Validate request
	if err := ValidateLoginRequest(&loginReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Validation failed", err.Error(), req.ID)
		return
	}

	// Process login based on provider
	authResponse, err := h.service.LoginWithProvider(ctx, &loginReq)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Login failed", err.Error(), req.ID)
		return
	}

	// Send response
	jsonrpc.WriteResponse(w, authResponse, req.ID)
}

// handleGuestLogin processes guest login requests
func (h *Handler) handleGuestLogin(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse credentials for guest login
	var creds GuestLoginCredentials
	if err := h.parseParams(req.Params, &creds); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Create login request
	loginReq := &LoginRequest{
		Provider:    ProviderGuest,
		Credentials: creds,
	}

	// Validate request
	if err := ValidateLoginRequest(loginReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Validation failed", err.Error(), req.ID)
		return
	}

	// Process login
	authResponse, err := h.service.LoginWithProvider(ctx, loginReq)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Guest login failed", err.Error(), req.ID)
		return
	}

	// Send response
	jsonrpc.WriteResponse(w, authResponse, req.ID)
}

// handleGoogleLogin processes Google OAuth login requests
func (h *Handler) handleGoogleLogin(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse credentials for Google login
	var creds GoogleLoginCredentials
	if err := h.parseParams(req.Params, &creds); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Create login request
	loginReq := &LoginRequest{
		Provider:    ProviderGoogle,
		Credentials: creds,
	}

	// Validate request
	if err := ValidateLoginRequest(loginReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Validation failed", err.Error(), req.ID)
		return
	}

	// Process login
	authResponse, err := h.service.LoginWithProvider(ctx, loginReq)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Google login failed", err.Error(), req.ID)
		return
	}

	// Send response
	jsonrpc.WriteResponse(w, authResponse, req.ID)
}

// handleAppleLogin processes Apple OAuth login requests
func (h *Handler) handleAppleLogin(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse credentials for Apple login
	var creds AppleLoginCredentials
	if err := h.parseParams(req.Params, &creds); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Create login request
	loginReq := &LoginRequest{
		Provider:    ProviderApple,
		Credentials: creds,
	}

	// Validate request
	if err := ValidateLoginRequest(loginReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.ValidationFailed, "Validation failed", err.Error(), req.ID)
		return
	}

	// Process login
	authResponse, err := h.service.LoginWithProvider(ctx, loginReq)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Apple login failed", err.Error(), req.ID)
		return
	}

	// Send response
	jsonrpc.WriteResponse(w, authResponse, req.ID)
}

// handleLinkAuth processes requests to link additional auth providers to existing account
func (h *Handler) handleLinkAuth(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Extract token from request headers to get current user
	token := h.extractToken(ctx)
	if token == "" {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Missing authorization token", nil, req.ID)
		return
	}

	// Validate token to get user ID
	claims, err := h.service.ValidateToken(ctx, token)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Invalid token", err.Error(), req.ID)
		return
	}

	// Parse link request parameters
	var linkReq LinkAuthRequest
	if err := h.parseParams(req.Params, &linkReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Process link request
	err = h.service.LinkAuthProvider(ctx, claims.UserID, &linkReq)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Failed to link auth provider", err.Error(), req.ID)
		return
	}

	// Send response
	jsonrpc.WriteResponse(w, map[string]interface{}{
		"status":   "success",
		"user_id":  claims.UserID,
		"provider": linkReq.Provider,
	}, req.ID)
}

// handleLogout processes logout requests
func (h *Handler) handleLogout(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Extract token from request headers
	token := h.extractToken(ctx)
	if token == "" {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Missing authorization token", nil, req.ID)
		return
	}

	// Validate token to get user ID
	claims, err := h.service.ValidateToken(ctx, token)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Invalid token", err.Error(), req.ID)
		return
	}

	// Process logout
	err = h.service.LogoutUser(ctx, claims.UserID, token)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Logout failed", err.Error(), req.ID)
		return
	}

	// Send response
	jsonrpc.WriteResponse(w, map[string]string{"status": "success"}, req.ID)
}

// handleRefresh processes token refresh requests
func (h *Handler) handleRefresh(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Parse parameters
	var refreshReq RefreshRequest
	if err := h.parseParams(req.Params, &refreshReq); err != nil {
		jsonrpc.WriteError(w, jsonrpc.InvalidParams, "Invalid params", err.Error(), req.ID)
		return
	}

	// Process refresh
	tokenPair, err := h.service.RefreshToken(ctx, refreshReq.UserID, refreshReq.OldToken)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Token refresh failed", err.Error(), req.ID)
		return
	}

	// Send response
	authResponse := &AuthResponse{
		UserID:      refreshReq.UserID,
		AccessToken: tokenPair.AccessToken,
		TokenType:   tokenPair.TokenType,
		ExpiresIn:   tokenPair.ExpiresIn,
	}
	jsonrpc.WriteResponse(w, authResponse, req.ID)
}

// handleValidate processes token validation requests
func (h *Handler) handleValidate(w http.ResponseWriter, ctx context.Context, req jsonrpc.Request) {
	// Extract token from request headers
	token := h.extractToken(ctx)
	if token == "" {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Missing authorization token", nil, req.ID)
		return
	}

	// Validate token
	claims, err := h.service.ValidateToken(ctx, token)
	if err != nil {
		jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Token validation failed", err.Error(), req.ID)
		return
	}

	// Send response
	result := map[string]interface{}{
		"valid":      true,
		"user_id":    claims.UserID,
		"expires_at": claims.ExpiresAt.Unix(),
		"issued_at":  claims.IssuedAt.Unix(),
	}
	jsonrpc.WriteResponse(w, result, req.ID)
}

// parseParams parses JSON-RPC parameters into a struct
func (h *Handler) parseParams(params interface{}, target interface{}) error {
	if params == nil {
		return nil
	}

	// Convert to JSON bytes and unmarshal into target
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

// extractToken extracts Bearer token from request context or headers
func (h *Handler) extractToken(ctx context.Context) string {
	// Try to get from context first (set by middleware)
	if token, ok := ctx.Value("token").(string); ok {
		return token
	}

	// This is a fallback - in practice, middleware should set this
	return ""
}


// AuthMiddleware validates JWT tokens from Authorization header for protected endpoints
func (h *Handler) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Missing authorization header", nil, nil)
			return
		}

		// Parse Bearer token
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Invalid authorization format", nil, nil)
			return
		}

		token := authHeader[len(bearerPrefix):]
		if token == "" {
			jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Missing token", nil, nil)
			return
		}

		// Validate token
		claims, err := h.service.ValidateToken(r.Context(), token)
		if err != nil {
			jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Invalid token", err.Error(), nil)
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "token", token)
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "claims", claims)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// CookieAuthMiddleware validates JWT tokens from cookies for SSE and other cookie-based requests
func (h *Handler) CookieAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from cookie
		cookie, err := r.Cookie("auth_token")
		if err != nil || cookie.Value == "" {
			jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Missing auth token cookie", nil, nil)
			return
		}

		token := cookie.Value

		// Validate token
		claims, err := h.service.ValidateToken(r.Context(), token)
		if err != nil {
			jsonrpc.WriteError(w, jsonrpc.AuthenticationRequired, "Invalid token", err.Error(), nil)
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), "token", token)
		ctx = context.WithValue(ctx, "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "claims", claims)

		// Call next handler with updated context
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}