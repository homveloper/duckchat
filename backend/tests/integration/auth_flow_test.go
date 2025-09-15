package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"duckchat/internal/auth"
	"duckchat/internal/middleware"
	"duckchat/internal/user"
)

// T012: Integration test - Complete user authentication flow
func TestCompleteAuthenticationFlow(t *testing.T) {
	// Setup integration test server
	server := setupIntegrationTestServer(t)
	defer server.cleanup()

	t.Run("complete authentication lifecycle", func(t *testing.T) {
		testUserID := "integration-auth-user"

		// Step 1: Login (authenticate user)
		loginResponse := performLogin(t, server, testUserID)

		// Validate login response structure
		if loginResponse.AccessToken == "" {
			t.Error("Login should return access token")
		}
		if loginResponse.UserID != testUserID {
			t.Errorf("Expected user_id '%s', got '%s'", testUserID, loginResponse.UserID)
		}
		if loginResponse.TokenType != "Bearer" {
			t.Errorf("Expected token_type 'Bearer', got '%s'", loginResponse.TokenType)
		}
		if loginResponse.ExpiresIn <= 0 {
			t.Error("expires_in should be positive")
		}

		// Step 2: Validate token works for authenticated requests
		validateTokenAccess(t, server, loginResponse.AccessToken, testUserID)

		// Step 3: Test token refresh (if implemented)
		// refreshedToken := refreshToken(t, server, testUserID, loginResponse.AccessToken)
		// validateTokenAccess(t, server, refreshedToken, testUserID)

		// Step 4: Test logout
		performLogout(t, server, loginResponse.AccessToken)

		// Step 5: Verify token is invalidated after logout
		validateTokenInvalidated(t, server, loginResponse.AccessToken)
	})

	t.Run("multiple concurrent user sessions", func(t *testing.T) {
		// Test multiple users can authenticate simultaneously
		users := []string{"concurrent-user-1", "concurrent-user-2", "concurrent-user-3"}
		tokens := make(map[string]string)

		// All users login concurrently
		for _, userID := range users {
			loginResponse := performLogin(t, server, userID)
			tokens[userID] = loginResponse.AccessToken
		}

		// Verify all tokens work independently
		for userID, token := range tokens {
			validateTokenAccess(t, server, token, userID)
		}

		// Logout one user, others should remain valid
		performLogout(t, server, tokens[users[0]])
		validateTokenInvalidated(t, server, tokens[users[0]])

		// Other users should still be valid
		for i := 1; i < len(users); i++ {
			validateTokenAccess(t, server, tokens[users[i]], users[i])
		}
	})

	t.Run("token expiration handling", func(t *testing.T) {
		// This test would require controlling time or using short-lived tokens
		// For now, we test the structure without actual expiration
		testUserID := "expiry-test-user"
		loginResponse := performLogin(t, server, testUserID)

		// Verify token validation response includes expiration info
		validateResponse := performTokenValidation(t, server, loginResponse.AccessToken)

		if validateResponse.Valid != true {
			t.Error("Token should be valid immediately after creation")
		}
		if validateResponse.UserID != testUserID {
			t.Errorf("Expected user_id '%s', got '%s'", testUserID, validateResponse.UserID)
		}
		if validateResponse.ExpiresAt <= time.Now().Unix() {
			t.Error("Token expiration should be in the future")
		}
		if validateResponse.IssuedAt > time.Now().Unix() {
			t.Error("Token issue time should be in the past or present")
		}
	})

	t.Run("invalid authentication attempts", func(t *testing.T) {
		// Test various invalid authentication scenarios

		// Invalid token format
		testInvalidToken(t, server, "invalid-token-format")

		// Empty token
		testInvalidToken(t, server, "")

		// Malformed JWT
		testInvalidToken(t, server, "header.payload")

		// Token from different secret/issuer
		testInvalidToken(t, server, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c")
	})

	t.Run("authentication edge cases", func(t *testing.T) {
		// Test edge cases and boundary conditions

		// Very long user ID (at limit)
		longUserID := generateLongString(50)
		loginResponse := performLogin(t, server, longUserID)
		validateTokenAccess(t, server, loginResponse.AccessToken, longUserID)

		// Single character user ID
		shortUserID := "a"
		loginResponse2 := performLogin(t, server, shortUserID)
		validateTokenAccess(t, server, loginResponse2.AccessToken, shortUserID)

		// User ID with special characters
		specialUserID := "user-123_test.example"
		loginResponse3 := performLogin(t, server, specialUserID)
		validateTokenAccess(t, server, loginResponse3.AccessToken, specialUserID)
	})
}

// Helper structures for test responses
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	UserID      string `json:"user_id"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type TokenValidationResponse struct {
	Valid     bool   `json:"valid"`
	UserID    string `json:"user_id"`
	ExpiresAt int64  `json:"expires_at"`
	IssuedAt  int64  `json:"issued_at"`
}

// Test server for integration tests
type IntegrationTestServer struct {
	redisClient    *redis.Client
	authService    *auth.Service
	userService    *user.Service
	authHandler    *auth.Handler
	authMiddleware *middleware.AuthMiddleware
	cleanup        func()
}

func setupIntegrationTestServer(t *testing.T) *IntegrationTestServer {
	// Create Redis test client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       2, // Different DB for integration tests
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available for integration tests: %v", err)
	}

	// Clean test data
	redisClient.FlushDB(context.Background())

	// Initialize components
	authRepo := auth.NewRepository(redisClient)
	userRepo := user.NewRepository(redisClient)

	authService := auth.NewService(authRepo, "integration-test-jwt-secret")
	userService := user.NewService(userRepo)

	authHandler := auth.NewHandler(authService)
	authMiddleware := middleware.NewAuthMiddleware(authService)

	return &IntegrationTestServer{
		redisClient:    redisClient,
		authService:    authService,
		userService:    userService,
		authHandler:    authHandler,
		authMiddleware: authMiddleware,
		cleanup: func() {
			redisClient.FlushDB(context.Background())
			redisClient.Close()
		},
	}
}

// Helper functions for integration test steps

func performLogin(t *testing.T, server *IntegrationTestServer, userID string) LoginResponse {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "auth.login",
		"params": map[string]interface{}{
			"user_id": userID,
		},
		"id": "login-test",
	})

	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.authHandler.HandleAuth(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Login failed: status %d, body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	result := response["result"].(map[string]interface{})
	return LoginResponse{
		AccessToken: result["access_token"].(string),
		UserID:      result["user_id"].(string),
		TokenType:   result["token_type"].(string),
		ExpiresIn:   int64(result["expires_in"].(float64)),
	}
}

func validateTokenAccess(t *testing.T, server *IntegrationTestServer, token string, expectedUserID string) {
	// Test that token works for authenticated endpoint
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "auth.validate",
		"params":  map[string]interface{}{},
		"id":      "validate-test",
	})

	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.authHandler.HandleAuth)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Token validation failed: status %d, body: %s", w.Code, w.Body.String())
		return
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	result := response["result"].(map[string]interface{})
	if result["user_id"] != expectedUserID {
		t.Errorf("Expected user_id '%s', got '%s'", expectedUserID, result["user_id"])
	}
	if result["valid"] != true {
		t.Error("Token should be valid")
	}
}

func performLogout(t *testing.T, server *IntegrationTestServer, token string) {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "auth.logout",
		"params":  map[string]interface{}{},
		"id":      "logout-test",
	})

	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.authHandler.HandleAuth)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Logout failed: status %d, body: %s", w.Code, w.Body.String())
	}
}

func validateTokenInvalidated(t *testing.T, server *IntegrationTestServer, token string) {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "auth.validate",
		"params":  map[string]interface{}{},
		"id":      "validate-after-logout",
	})

	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.authHandler.HandleAuth)
	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Token should be invalid after logout, got status %d", w.Code)
	}
}

func performTokenValidation(t *testing.T, server *IntegrationTestServer, token string) TokenValidationResponse {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "auth.validate",
		"params":  map[string]interface{}{},
		"id":      "detailed-validate",
	})

	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.authHandler.HandleAuth)
	handler(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	result := response["result"].(map[string]interface{})
	return TokenValidationResponse{
		Valid:     result["valid"].(bool),
		UserID:    result["user_id"].(string),
		ExpiresAt: int64(result["expires_at"].(float64)),
		IssuedAt:  int64(result["issued_at"].(float64)),
	}
}

func testInvalidToken(t *testing.T, server *IntegrationTestServer, invalidToken string) {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "auth.validate",
		"params":  map[string]interface{}{},
		"id":      "invalid-token-test",
	})

	req := httptest.NewRequest("POST", "/api/auth", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	if invalidToken != "" {
		req.Header.Set("Authorization", "Bearer "+invalidToken)
	}

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.authHandler.HandleAuth)
	handler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Invalid token should return 401, got %d", w.Code)
	}
}

func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}