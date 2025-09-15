package middleware

import (
	"context"
	"net/http"
	"strings"

	"duckchat/internal/auth"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	authService *auth.Service
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authService *auth.Service) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

// RequireAuth is middleware that requires valid authentication
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			m.sendAuthError(w, "Missing authorization header")
			return
		}

		// Parse Bearer token
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			m.sendAuthError(w, "Invalid authorization format")
			return
		}

		token := authHeader[len(bearerPrefix):]
		if token == "" {
			m.sendAuthError(w, "Missing token")
			return
		}

		// Validate token
		claims, err := m.authService.ValidateToken(r.Context(), token)
		if err != nil {
			m.sendAuthError(w, "Invalid token: "+err.Error())
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

// OptionalAuth is middleware that optionally validates authentication
func (m *AuthMiddleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			// No auth header, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Parse Bearer token
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			// Invalid format, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		token := authHeader[len(bearerPrefix):]
		if token == "" {
			// Empty token, continue without authentication
			next.ServeHTTP(w, r)
			return
		}

		// Validate token
		claims, err := m.authService.ValidateToken(r.Context(), token)
		if err != nil {
			// Invalid token, continue without authentication
			next.ServeHTTP(w, r)
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

// CORS middleware for handling cross-origin requests
func (m *AuthMiddleware) CORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// Logging middleware
func (m *AuthMiddleware) Logging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Simple logging - in production, use proper logger
		userID := "anonymous"
		if uid, ok := r.Context().Value("user_id").(string); ok {
			userID = uid
		}

		_ = userID

		// Log request
		// fmt.Printf("[%s] %s %s - User: %s\n", time.Now().Format("2006-01-02 15:04:05"), r.Method, r.URL.Path, userID)

		next.ServeHTTP(w, r)
	}
}

// sendAuthError sends a JSON-RPC authentication error
func (m *AuthMiddleware) sendAuthError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)

	errorResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    -32001,
			"message": "Authentication failed",
			"data":    message,
		},
		"id": nil,
	}

	_ = errorResponse

	// Simple JSON encoding - in production, use proper error handling
	w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32001,"message":"Authentication failed","data":"` + message + `"},"id":null}`))
}

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value("user_id").(string)
	return userID, ok
}

// GetTokenFromContext extracts token from request context
func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value("token").(string)
	return token, ok
}

// GetClaimsFromContext extracts token claims from request context
func GetClaimsFromContext(ctx context.Context) (*auth.TokenClaims, bool) {
	claims, ok := ctx.Value("claims").(*auth.TokenClaims)
	return claims, ok
}
