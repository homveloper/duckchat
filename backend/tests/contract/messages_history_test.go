package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// T009: Contract test POST /api/messages.history
func TestMessagesHistoryContract(t *testing.T) {
	// Setup test server with real dependencies
	server := setupMessageTestServer(t)
	defer server.cleanup()

	// Create test user and room
	userToken := createTestUserAndToken(t, server.RoomTestServer, "history-user")
	testRoomID := createTestRoom(t, server.RoomTestServer, userToken, "Test History Room")

	// Send some test messages to have history
	testMessages := []string{
		"First message",
		"Second message",
		"Third message with unicode 🚀",
		"Fourth message",
		"Fifth message",
	}

	messageIDs := make([]string, 0, len(testMessages))
	for _, content := range testMessages {
		msgID := sendTestMessage(t, server, userToken, testRoomID, content)
		messageIDs = append(messageIDs, msgID)
		time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	}

	tests := []struct {
		name           string
		request        interface{}
		authToken      string
		expectedStatus int
		validateResult func(t *testing.T, response map[string]interface{})
		validateError  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "valid history request with default limit",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.history",
				"params": map[string]interface{}{
					"room_id": testRoomID,
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
				messages, ok := result["messages"].([]interface{})
				if !ok {
					t.Fatal("Expected messages to be an array")
				}

				if len(messages) != 5 {
					t.Errorf("Expected 5 messages, got %d", len(messages))
				}

				// Validate message structure
				for i, msgInterface := range messages {
					msg, ok := msgInterface.(map[string]interface{})
					if !ok {
						t.Fatalf("Message %d should be an object", i)
					}

					// Check required fields per contract
					if msg["message_id"] == "" {
						t.Errorf("Message %d missing message_id", i)
					}
					if msg["user_id"] == "" {
						t.Errorf("Message %d missing user_id", i)
					}
					if msg["username"] == "" {
						t.Errorf("Message %d missing username", i)
					}
					if msg["content"] == "" {
						t.Errorf("Message %d missing content", i)
					}

					// timestamp should be valid RFC3339
					if timestamp, ok := msg["timestamp"].(string); ok {
						if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
							t.Errorf("Message %d timestamp should be RFC3339 format: %s", i, timestamp)
						}
					} else {
						t.Errorf("Message %d missing or invalid timestamp", i)
					}

					// message_type should be present and valid
					if msgType, ok := msg["message_type"].(string); ok {
						if msgType != "text" && msgType != "system" {
							t.Errorf("Message %d invalid message_type: %s", i, msgType)
						}
					} else {
						t.Errorf("Message %d missing message_type", i)
					}
				}

				// has_more should be present and boolean
				if hasMore, ok := result["has_more"].(bool); ok {
					// For 5 messages with default limit, should be false
					if hasMore {
						t.Error("Expected has_more to be false with 5 messages")
					}
				} else {
					t.Error("Missing or invalid has_more field")
				}
			},
		},
		{
			name: "history request with custom limit",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.history",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"limit":   3,
				},
				"id": "req-002",
			},
			authToken:      userToken,
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, response map[string]interface{}) {
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				messages, ok := result["messages"].([]interface{})
				if !ok {
					t.Fatal("Expected messages to be an array")
				}

				if len(messages) != 3 {
					t.Errorf("Expected 3 messages with limit=3, got %d", len(messages))
				}

				// has_more should be true since we have 5 messages but limit is 3
				if hasMore, ok := result["has_more"].(bool); ok {
					if !hasMore {
						t.Error("Expected has_more to be true with limit < total messages")
					}
				} else {
					t.Error("Missing or invalid has_more field")
				}
			},
		},
		{
			name: "history request with before timestamp",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.history",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"limit":   10,
					"before":  time.Now().Format(time.RFC3339), // Current time
				},
				"id": "req-003",
			},
			authToken:      userToken,
			expectedStatus: http.StatusOK,
			validateResult: func(t *testing.T, response map[string]interface{}) {
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				messages, ok := result["messages"].([]interface{})
				if !ok {
					t.Fatal("Expected messages to be an array")
				}

				// Should get all messages before current time
				if len(messages) != 5 {
					t.Errorf("Expected 5 messages before current time, got %d", len(messages))
				}
			},
		},
		{
			name: "missing authorization token",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.history",
				"params": map[string]interface{}{
					"room_id": testRoomID,
				},
				"id": "req-004",
			},
			authToken:      "", // No token
			expectedStatus: http.StatusUnauthorized,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32001, "Authentication failed", "req-004")
			},
		},
		{
			name: "invalid authorization token",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.history",
				"params": map[string]interface{}{
					"room_id": testRoomID,
				},
				"id": "req-005",
			},
			authToken:      "invalid-token",
			expectedStatus: http.StatusUnauthorized,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32001, "Authentication failed", "req-005")
			},
		},
		{
			name: "missing room_id parameter",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.history",
				"params":  map[string]interface{}{}, // Empty params
				"id":      "req-006",
			},
			authToken:      userToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-006")
			},
		},
		{
			name: "non-existent room_id",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.history",
				"params": map[string]interface{}{
					"room_id": "non-existent-room",
				},
				"id": "req-007",
			},
			authToken:      userToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32005, "Failed to get message history", "req-007")
			},
		},
		{
			name: "limit exceeding maximum",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.history",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"limit":   101, // Exceeds max of 100
				},
				"id": "req-008",
			},
			authToken:      userToken,
			expectedStatus: http.StatusOK, // Should be clamped to 100, not error
			validateResult: func(t *testing.T, response map[string]interface{}) {
				// Should succeed but limit should be clamped
				result, ok := response["result"].(map[string]interface{})
				if !ok {
					t.Fatal("Expected result to be an object")
				}

				messages, ok := result["messages"].([]interface{})
				if !ok {
					t.Fatal("Expected messages to be an array")
				}

				// Should return available messages (5) since limit was clamped
				if len(messages) != 5 {
					t.Errorf("Expected 5 messages (all available), got %d", len(messages))
				}
			},
		},
		{
			name: "invalid before timestamp format",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.history",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"before":  "invalid-timestamp",
				},
				"id": "req-009",
			},
			authToken:      userToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32602, "Invalid params", "req-009")
			},
		},
		{
			name: "empty room (no messages)",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.history",
				"params": map[string]interface{}{
					"room_id": createTestRoom(t, server.RoomTestServer, userToken, "Empty Room"),
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

				messages, ok := result["messages"].([]interface{})
				if !ok {
					t.Fatal("Expected messages to be an array")
				}

				if len(messages) != 0 {
					t.Errorf("Expected 0 messages in empty room, got %d", len(messages))
				}

				if hasMore, ok := result["has_more"].(bool); ok {
					if hasMore {
						t.Error("Expected has_more to be false for empty room")
					}
				} else {
					t.Error("Missing or invalid has_more field")
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
			req := httptest.NewRequest("POST", "/api/messages", bytes.NewBuffer(requestBody))
			req.Header.Set("Content-Type", "application/json")

			if tt.authToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.authToken)
			}

			w := httptest.NewRecorder()

			// Apply auth middleware then message handler
			handler := server.middleware.RequireAuth(server.messageHandler.HandleMessage)
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

// Helper function to send a test message and return message ID
func sendTestMessage(t *testing.T, server *MessageTestServer, userToken string, roomID string, content string) string {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "messages.send",
		"params": map[string]interface{}{
			"room_id": roomID,
			"content": content,
		},
		"id": "send-test-message",
	})

	req := httptest.NewRequest("POST", "/api/messages", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+userToken)

	w := httptest.NewRecorder()
	handler := server.middleware.RequireAuth(server.messageHandler.HandleMessage)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to send test message: status %d, body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse message send response: %v", err)
	}

	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("Failed to get result from message send response")
	}

	messageID, ok := result["message_id"].(string)
	if !ok {
		t.Fatalf("Failed to get message_id from message send response")
	}

	return messageID
}