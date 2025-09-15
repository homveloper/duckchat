package contract

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

// T005: Contract test POST /api/auth.login
func TestAuthLoginContract(t *testing.T) {
	// Setup test server with real dependencies
	server := setupTestServer(t)
	defer server.cleanup()

	tests := []struct {
		name           string
		request        interface{}
		expectedStatus int
		validateResult func(t *testing.T, response map[string]interface{})
		validateError  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "valid login request",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "auth.login",
				"params": map[string]interface{}{
					"user_id": "test-user-123",
				},
				"id": "req-001",
			},
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, response map[string]interface{}) {
				// Validate JSON-RPC 2.0 response structure
				if response["jsonrpc"] != "2.0" {
					t.Errorf("Expected jsonrpc '2.0', got %v", response["jsonrpc"])
				}
				if response["id"] != "req-001" {
					t.Errorf("Expected id 'req-001', got %v", response["id"])
				}

				// Validate result structure per contract
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				// Check required fields from api.json
				if result["access_token"] == "" {
					t.Error("Missing or empty access_token")
				}
				if result["user_id"] != "test-user-123" {
					t.Errorf("Expected user_id 'test-user-123', got %v", result["user_id"])
				}
				if result["token_type"] != "Bearer" {
					t.Errorf("Expected token_type 'Bearer', got %v", result["token_type"])
				}

				// Validate token format (JWT should have 3 parts)
				token, ok := result["access_token"].(string)
				if !ok {
					t.Fatal("access_token should be string")
				}

				// Basic JWT structure validation (header.payload.signature)
				parts := bytes.Split([]byte(token), []byte("."))
				if len(parts) != 3 {
					t.Errorf("JWT should have 3 parts, got %d", len(parts))
				}
			},
		},
		{
			name: "missing jsonrpc version",
			request: map[string]interface{}{
				"method": "auth.login",
				"params": map[string]interface{}{
					"user_id": "test-user-123",
				},
				"id": "req-002",
			},
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32600, "Invalid Request", "req-002")
			},
		},
		{
			name: "invalid method name",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "auth.signin", // Wrong method name
				"params": map[string]interface{}{
					"user_id": "test-user-123",
				},
				"id": "req-003",
			},
			expectedStatus: http.StatusNotFound,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32601, "Method not found", "req-003")
			},
		},
		{
			name: "missing required params",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "auth.login",
				"params":  map[string]interface{}{}, // Empty params
				"id":      "req-004",
			},
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-004")
			},
		},
		{
			name: "invalid user_id format",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "auth.login",
				"params": map[string]interface{}{
					"user_id": "", // Empty user_id
				},
				"id": "req-005",
			},
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-005")
			},
		},
		{
			name: "user_id too long",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "auth.login",
				"params": map[string]interface{}{
					"user_id": generateLongString(51), // Exceeds 50 char limit
				},
				"id": "req-006",
			},
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-006")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request
			requestBody, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Make HTTP request
			req := httptest.NewRequest("POST", "/api/auth", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.authHandler.HandleAuth(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check content type
			expectedContentType := "application/json"
			if contentType := w.Header().Get("Content-Type"); contentType != expectedContentType {
				t.Errorf("Expected Content-Type %s, got %s", expectedContentType, contentType)
			}

			// Parse response
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response JSON: %v", err)
			}

			// Validate response structure
			if tt.validateResult != nil {
				if response["error"] != nil {
					t.Errorf("Expected success but got error: %v", response["error"])
				}
				tt.validateResult(t, response)
			}

			if tt.validateError != nil {
				if response["result"] != nil {
					t.Errorf("Expected error but got result: %v", response["result"])
				}
				tt.validateError(t, response)
			}
		})
	}
}

// Helper function to validate JSON-RPC error response
func validateJSONRPCError(t *testing.T, response map[string]interface{}, expectedCode int, expectedMessage string, expectedID string) {
	if response["jsonrpc"] != "2.0" {
		t.Errorf("Expected jsonrpc '2.0', got %v", response["jsonrpc"])
	}
	// For middleware errors (like auth failures), ID might be null
	if expectedID != "" && response["id"] != expectedID && response["id"] != nil {
		t.Errorf("Expected id '%s', got %v", expectedID, response["id"])
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected error to be an object")
	}

	if int(errorObj["code"].(float64)) != expectedCode {
		t.Errorf("Expected error code %d, got %v", expectedCode, errorObj["code"])
	}
	if errorObj["message"] != expectedMessage {
		t.Errorf("Expected error message '%s', got %v", expectedMessage, errorObj["message"])
	}
}

// Helper function to generate long string for testing
func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}

// Test server setup with real dependencies
type TestServer struct {
	redisClient *redis.Client
	authService *auth.Service
	userService *user.Service
	authHandler *auth.Handler
	middleware  *middleware.AuthMiddleware
	cleanup     func()
}

func setupTestServer(t *testing.T) *TestServer {
	// Create Redis test client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use different DB for testing
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available for contract tests: %v", err)
	}

	// Clean test data
	redisClient.FlushDB(context.Background())

	// Initialize repositories
	authRepo := auth.NewRepository(redisClient)
	userRepo := user.NewRepository(redisClient)

	// Initialize services
	authService := auth.NewService(authRepo, "test-jwt-secret-key")
	userService := user.NewService(userRepo)

	// Initialize handlers
	authHandler := auth.NewHandler(authService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(authService)

	return &TestServer{
		redisClient: redisClient,
		authService: authService,
		userService: userService,
		authHandler: authHandler,
		middleware:  authMiddleware,
		cleanup: func() {
			redisClient.FlushDB(context.Background())
			redisClient.Close()
		},
	}
}

func TestMain(m *testing.M) {
	// This will run before all tests
	// You can add global setup here if needed
	m.Run()
}