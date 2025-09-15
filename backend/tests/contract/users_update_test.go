package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"duckchat/internal/user"
)

// T010: Contract test POST /api/users.updateUsername
func TestUsersUpdateUsernameContract(t *testing.T) {
	// Setup test server with real dependencies
	server := setupUserTestServer(t)
	defer server.cleanup()

	// Create test user and token
	userToken := createTestUserAndToken(t, server.RoomTestServer, "username-updater")

	tests := []struct {
		name           string
		request        interface{}
		authToken      string
		expectedStatus int
		validateResult func(t *testing.T, response map[string]interface{})
		validateError  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "valid username update request",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params": map[string]interface{}{
					"new_username": "UpdatedUsername",
				},
				"id": "req-001",
			},
			authToken:      userToken,
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
				userID, ok := result["user_id"].(string)
				if !ok || userID == "" {
					t.Error("Missing or invalid user_id")
				}

				if result["username"] != "UpdatedUsername" {
					t.Errorf("Expected username 'UpdatedUsername', got %v", result["username"])
				}

				// updated_at should be present and valid timestamp
				updatedAt, ok := result["updated_at"].(string)
				if !ok {
					t.Error("Missing updated_at field")
				} else {
					// Validate ISO 8601 format
					if _, err := time.Parse(time.RFC3339, updatedAt); err != nil {
						t.Errorf("updated_at should be in RFC3339 format, got: %s", updatedAt)
					}
				}
			},
		},
		{
			name: "missing authorization token",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params": map[string]interface{}{
					"new_username": "UpdatedUsername",
				},
				"id": "req-002",
			},
			authToken:      "", // No token
			expectedStatus: http.StatusUnauthorized,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32001, "Authentication failed", "req-002")
			},
		},
		{
			name: "invalid authorization token",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params": map[string]interface{}{
					"new_username": "UpdatedUsername",
				},
				"id": "req-003",
			},
			authToken:      "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32001, "Authentication failed", "req-003")
			},
		},
		{
			name: "missing new_username parameter",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params":  map[string]interface{}{}, // Empty params
				"id":      "req-004",
			},
			authToken:      userToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-004")
			},
		},
		{
			name: "empty new_username",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params": map[string]interface{}{
					"new_username": "", // Empty username
				},
				"id": "req-005",
			},
			authToken:      userToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-005")
			},
		},
		{
			name: "new_username too long",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params": map[string]interface{}{
					"new_username": generateLongString(51), // Exceeds 50 char limit
				},
				"id": "req-006",
			},
			authToken:      userToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-006")
			},
		},
		{
			name: "maximum length username",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params": map[string]interface{}{
					"new_username": generateLongString(50), // Exactly 50 chars
				},
				"id": "req-007",
			},
			authToken:      userToken,
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, response map[string]interface{}) {
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				expectedUsername := generateLongString(50)
				if result["username"] != expectedUsername {
					t.Error("Expected long username to be preserved exactly")
				}
			},
		},
		{
			name: "invalid method name",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "user.updateUsername", // Wrong method name (singular)
				"params": map[string]interface{}{
					"new_username": "UpdatedUsername",
				},
				"id": "req-008",
			},
			authToken:      userToken,
			expectedStatus: http.StatusNotFound,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32601, "Method not found", "req-008")
			},
		},
		{
			name: "username with special characters",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params": map[string]interface{}{
					"new_username": "User_Name-123",
				},
				"id": "req-009",
			},
			authToken:      userToken,
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, response map[string]interface{}) {
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				if result["username"] != "User_Name-123" {
					t.Errorf("Expected username with special chars 'User_Name-123', got %v", result["username"])
				}
			},
		},
		{
			name: "unicode username",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params": map[string]interface{}{
					"new_username": "사용자이름123",
				},
				"id": "req-010",
			},
			authToken:      userToken,
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, response map[string]interface{}) {
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				if result["username"] != "사용자이름123" {
					t.Errorf("Expected unicode username '사용자이름123', got %v", result["username"])
				}
			},
		},
		{
			name: "single character username",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params": map[string]interface{}{
					"new_username": "A", // Minimum length (1 char)
				},
				"id": "req-011",
			},
			authToken:      userToken,
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, response map[string]interface{}) {
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				if result["username"] != "A" {
					t.Errorf("Expected single char username 'A', got %v", result["username"])
				}
			},
		},
		{
			name: "update to same username",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "users.updateUsername",
				"params": map[string]interface{}{
					"new_username": "UpdatedUsername", // Same as previous test
				},
				"id": "req-012",
			},
			authToken:      userToken,
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, response map[string]interface{}) {
				// Should succeed even if username is the same
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				if result["username"] != "UpdatedUsername" {
					t.Errorf("Expected same username 'UpdatedUsername', got %v", result["username"])
				}
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

			// Make HTTP request with authentication
			req := httptest.NewRequest("POST", "/api/users", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			if tt.authToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.authToken)
			}

			w := httptest.NewRecorder()

			// Apply auth middleware then user handler
			handler := server.middleware.RequireAuth(server.userHandler.HandleUser)
			handler(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Response body: %s", tt.expectedStatus, w.Code, w.Body.String())
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

// Extended test server for user operations
type UserTestServer struct {
	*RoomTestServer
	userHandler *user.Handler
}

func setupUserTestServer(t *testing.T) *UserTestServer {
	roomServer := setupRoomTestServer(t)

	// Initialize user handler (service already exists in base server)
	userHandler := user.NewHandler(roomServer.userService)

	return &UserTestServer{
		RoomTestServer: roomServer,
		userHandler:    userHandler,
	}
}