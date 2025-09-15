package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// T007: Contract test POST /api/rooms.join
func TestRoomsJoinContract(t *testing.T) {
	// Setup test server with real dependencies
	server := setupRoomTestServer(t)
	defer server.cleanup()

	// Create test users and tokens
	creatorToken := createTestUserAndToken(t, server, "room-creator")
	joinerToken := createTestUserAndToken(t, server, "room-joiner")

	// Create a test room first
	testRoomID := createTestRoom(t, server, creatorToken, "Test Room for Joining")

	tests := []struct {
		name           string
		request        interface{}
		authToken      string
		expectedStatus int
		validateResult func(t *testing.T, response map[string]interface{})
		validateError  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "valid room join request",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.join",
				"params": map[string]interface{}{
					"room_id": testRoomID,
				},
				"id": "req-001",
			},
			authToken:      joinerToken,
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
				if result["room_id"] != testRoomID {
					t.Errorf("Expected room_id '%s', got %v", testRoomID, result["room_id"])
				}

				if result["name"] != "Test Room for Joining" {
					t.Errorf("Expected name 'Test Room for Joining', got %v", result["name"])
				}

				// participant_count should be at least 2 (creator + joiner)
				participantCount, ok := result["participant_count"].(float64)
				if !ok {
					t.Error("participant_count should be a number")
				} else if participantCount < 2 {
					t.Errorf("Expected participant_count >= 2, got %v", participantCount)
				}

				// Additional fields that might be present
				if participants, exists := result["participants"]; exists {
					if participantList, ok := participants.([]interface{}); ok {
						if len(participantList) < 2 {
							t.Errorf("Expected at least 2 participants, got %d", len(participantList))
						}
					}
				}
			},
		},
		{
			name: "join room that user is already in",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.join",
				"params": map[string]interface{}{
					"room_id": testRoomID,
				},
				"id": "req-002",
			},
			authToken:      creatorToken, // Creator tries to join their own room
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, response map[string]interface{}) {
				// Should succeed (idempotent operation)
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				if result["room_id"] != testRoomID {
					t.Errorf("Expected room_id '%s', got %v", testRoomID, result["room_id"])
				}
			},
		},
		{
			name: "missing authorization token",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.join",
				"params": map[string]interface{}{
					"room_id": testRoomID,
				},
				"id": "req-003",
			},
			authToken:      "", // No token
			expectedStatus: http.StatusUnauthorized,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32001, "Authentication failed", "req-003")
			},
		},
		{
			name: "invalid authorization token",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.join",
				"params": map[string]interface{}{
					"room_id": testRoomID,
				},
				"id": "req-004",
			},
			authToken:      "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32001, "Authentication failed", "req-004")
			},
		},
		{
			name: "missing room_id parameter",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.join",
				"params":  map[string]interface{}{}, // Empty params
				"id":      "req-005",
			},
			authToken:      joinerToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-005")
			},
		},
		{
			name: "empty room_id parameter",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.join",
				"params": map[string]interface{}{
					"room_id": "", // Empty room_id
				},
				"id": "req-006",
			},
			authToken:      joinerToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-006")
			},
		},
		{
			name: "non-existent room_id",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.join",
				"params": map[string]interface{}{
					"room_id": "non-existent-room-id",
				},
				"id": "req-007",
			},
			authToken:      joinerToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				// Should return room error for non-existent room
				validateJSONRPCError(t, response, -32004, "Failed to join room", "req-007")
			},
		},
		{
			name: "invalid method name",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "room.join", // Wrong method name (singular)
				"params": map[string]interface{}{
					"room_id": testRoomID,
				},
				"id": "req-008",
			},
			authToken:      joinerToken,
			expectedStatus: http.StatusNotFound,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32601, "Method not found", "req-008")
			},
		},
		{
			name: "malformed room_id",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "rooms.join",
				"params": map[string]interface{}{
					"room_id": 12345, // Number instead of string
				},
				"id": "req-009",
			},
			authToken:      joinerToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				// Should fail parameter validation
				validateJSONRPCError(t, response, -32602, "Invalid params", "req-009")
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

// Helper function to create a test room and return room ID
func createTestRoom(t *testing.T, server *RoomTestServer, creatorToken string, roomName string) string {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "rooms.create",
		"params": map[string]interface{}{
			"name": roomName,
		},
		"id": "create-test-room",
	})

	req := httptest.NewRequest("POST", "/api/rooms", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+creatorToken)

	w := httptest.NewRecorder()
	handler := server.middleware.RequireAuth(server.roomHandler.HandleRoom)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to create test room: status %d, body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse room creation response: %v", err)
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("Failed to get result from room creation response")
	}

	roomID, ok := result["room_id"].(string)
	if !ok {
		t.Fatalf("Failed to get room_id from room creation response")
	}

	return roomID
}