package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"duckchat/internal/room"
)

// T006: Contract test POST /api/rooms.create
func TestRoomsCreateContract(t *testing.T) {
	// Setup test server with real dependencies
	server := setupRoomTestServer(t)
	defer server.cleanup()

	// Create a test user and get auth token first
	testToken := createTestUserAndToken(t, server, "room-creator-user")

	tests := []struct {
		name           string
		request        interface{}
		authToken      string
		expectedStatus int
		validateResult func(t *testing.T, response map[string]interface{})
		validateError  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "valid room creation request",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.create",
				"params": map[string]interface{}{
					"name": "Test Room",
				},
				"id": "req-001",
			},
			authToken:      testToken,
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
				roomID, ok := result["room_id"].(string)
				if !ok || roomID == "" {
					t.Error("Missing or invalid room_id")
				}

				if result["name"] != "Test Room" {
					t.Errorf("Expected name 'Test Room', got %v", result["name"])
				}

				// created_at should be present and a valid timestamp
				createdAt, ok := result["created_at"].(string)
				if !ok {
					t.Error("Missing created_at field")
				} else {
					// Validate ISO 8601 format
					if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
						t.Errorf("created_at should be in RFC3339 format, got: %s", createdAt)
					}
				}

				// Additional fields that might be present
				if participantCount, exists := result["participant_count"]; exists {
					if count, ok := participantCount.(float64); !ok || count < 1 {
						t.Errorf("participant_count should be >= 1, got %v", participantCount)
					}
				}
			},
		},
		{
			name: "missing authorization token",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.create",
				"params": map[string]interface{}{
					"name": "Test Room",
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
				"method":  "rooms.create",
				"params": map[string]interface{}{
					"name": "Test Room",
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
			name: "missing room name",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.create",
				"params":  map[string]interface{}{}, // Empty params
				"id":      "req-004",
			},
			authToken:      testToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-004")
			},
		},
		{
			name: "empty room name",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.create",
				"params": map[string]interface{}{
					"name": "", // Empty name
				},
				"id": "req-005",
			},
			authToken:      testToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-005")
			},
		},
		{
			name: "room name too long",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.create",
				"params": map[string]interface{}{
					"name": generateLongString(101), // Exceeds 100 char limit
				},
				"id": "req-006",
			},
			authToken:      testToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-006")
			},
		},
		{
			name: "invalid method name",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "room.create", // Wrong method name (singular)
				"params": map[string]interface{}{
					"name": "Test Room",
				},
				"id": "req-007",
			},
			authToken:      testToken,
			expectedStatus: http.StatusNotFound,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32601, "Method not found", "req-007")
			},
		},
		{
			name: "maximum length room name",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.create",
				"params": map[string]interface{}{
					"name": generateLongString(100), // Exactly 100 chars
				},
				"id": "req-008",
			},
			authToken:      testToken,
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, response map[string]interface{}) {
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				expectedName := generateLongString(100)
				if result["name"] != expectedName {
					t.Errorf("Expected name to be preserved exactly")
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
			req := httptest.NewRequest("POST", "/api/rooms", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			if tt.authToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.authToken)
			}

			w := httptest.NewRecorder()

			// Apply auth middleware then room handler
			handler := server.middleware.RequireAuth(server.roomHandler.HandleRoom)
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

// Extended test server for room operations
type RoomTestServer struct {
	*TestServer
	roomService *room.Service
	roomHandler *room.Handler
}

func setupRoomTestServer(t *testing.T) *RoomTestServer {
	baseServer := setupTestServer(t)

	// Initialize room-specific components
	roomRepo := room.NewRepository(baseServer.redisClient)
	roomService := room.NewService(roomRepo)
	roomHandler := room.NewHandler(roomService)

	return &RoomTestServer{
		TestServer:  baseServer,
		roomService: roomService,
		roomHandler: roomHandler,
	}
}

// Helper function to create a test user and return auth token
func createTestUserAndToken(t *testing.T, server *RoomTestServer, userID string) string {
	ctx := context.Background()

	// Create user with specific ID that matches the JWT token
	_, err := server.userService.CreateUserWithID(ctx, userID, "Test User")
	if err != nil {
		t.Logf("User creation might have failed (could be duplicate): %v", err)
	}

	// Generate auth token
	tokenPair, err := server.authService.GenerateToken(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	return tokenPair.AccessToken
}