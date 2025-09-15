package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"duckchat/internal/message"
)

// T008: Contract test POST /api/messages.send
func TestMessagesSendContract(t *testing.T) {
	// Setup test server with real dependencies
	server := setupMessageTestServer(t)
	defer server.cleanup()

	// Create test user and room
	userToken := createTestUserAndToken(t, server.RoomTestServer, "message-sender")
	testRoomID := createTestRoom(t, server.RoomTestServer, userToken, "Test Message Room")

	tests := []struct {
		name           string
		request        interface{}
		authToken      string
		expectedStatus int
		validateResult func(t *testing.T, response map[string]interface{})
		validateError  func(t *testing.T, response map[string]interface{})
	}{
		{
			name: "valid message send request",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.send",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"content": "Hello, World!",
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
				messageID, ok := result["message_id"].(string)
				if !ok || messageID == "" {
					t.Error("Missing or invalid message_id")
				}

				// timestamp should be present and valid
				timestamp, ok := result["timestamp"].(string)
				if !ok {
					t.Error("Missing timestamp field")
				} else {
					// Validate ISO 8601 format
					if _, err := time.Parse(time.RFC3339, timestamp); err != nil {
						t.Errorf("timestamp should be in RFC3339 format, got: %s", timestamp)
					}
				}

				// Additional fields that might be present in implementation
				if content, exists := result["content"]; exists {
					if content != "Hello, World!" {
						t.Errorf("Expected content 'Hello, World!', got %v", content)
					}
				}

				if roomID, exists := result["room_id"]; exists {
					if roomID != testRoomID {
						t.Errorf("Expected room_id '%s', got %v", testRoomID, roomID)
					}
				}
			},
		},
		{
			name: "missing authorization token",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.send",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"content": "Hello, World!",
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
				"method":  "messages.send",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"content": "Hello, World!",
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
			name: "missing room_id parameter",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.send",
				"params": map[string]interface{}{
					"content": "Hello, World!",
				},
				"id": "req-004",
			},
			authToken:      userToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-004")
			},
		},
		{
			name: "missing content parameter",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.send",
				"params": map[string]interface{}{
					"room_id": testRoomID,
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
			name: "empty content",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.send",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"content": "", // Empty content
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
			name: "content too long",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.send",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"content": generateLongString(1001), // Exceeds 1000 char limit
				},
				"id": "req-007",
			},
			authToken:      userToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32002, "Validation failed", "req-007")
			},
		},
		{
			name: "non-existent room_id",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.send",
				"params": map[string]interface{}{
					"room_id": "non-existent-room",
					"content": "Hello, World!",
				},
				"id": "req-008",
			},
			authToken:      userToken,
			expectedStatus: http.StatusBadRequest,
			validateError: func(t *testing.T, response map[string]interface{}) {
				// Should return message error for non-existent room
				validateJSONRPCError(t, response, -32005, "Failed to send message", "req-008")
			},
		},
		{
			name: "maximum length content",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.send",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"content": generateLongString(1000), // Exactly 1000 chars
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

				// Should succeed with maximum length content
				messageID, ok := result["message_id"].(string)
				if !ok || messageID == "" {
					t.Error("Missing or invalid message_id for max length content")
				}
			},
		},
		{
			name: "invalid method name",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "message.send", // Wrong method name (singular)
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"content": "Hello, World!",
				},
				"id": "req-010",
			},
			authToken:      userToken,
			expectedStatus: http.StatusNotFound,
			validateError: func(t *testing.T, response map[string]interface{}) {
				validateJSONRPCError(t, response, -32601, "Method not found", "req-010")
			},
		},
		{
			name: "unicode content support",
			request: map[string]interface{}{
				"jsonrpc": "2.0",
				"method":  "messages.send",
				"params": map[string]interface{}{
					"room_id": testRoomID,
					"content": "Hello 👋 World! 안녕하세요 🌍",
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

				// Should handle unicode content properly
				messageID, ok := result["message_id"].(string)
				if !ok || messageID == "" {
					t.Error("Missing or invalid message_id for unicode content")
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

// Extended test server for message operations
type MessageTestServer struct {
	*RoomTestServer
	messageService *message.Service
	messageHandler *message.Handler
}

func setupMessageTestServer(t *testing.T) *MessageTestServer {
	roomServer := setupRoomTestServer(t)

	// Initialize message-specific components
	messageRepo := message.NewRepository(roomServer.redisClient)
	messageService := message.NewService(messageRepo, roomServer.roomService)
	messageHandler := message.NewHandler(messageService)

	return &MessageTestServer{
		RoomTestServer: roomServer,
		messageService: messageService,
		messageHandler: messageHandler,
	}
}