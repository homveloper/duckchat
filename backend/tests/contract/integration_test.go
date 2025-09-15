package contract

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"duckchat/internal/config"
	"duckchat/internal/handlers"
	"duckchat/internal/jsonrpc"
	"duckchat/internal/repository"
	"duckchat/internal/services"
)

// setupMiddlewareWithHandlers is a helper function that sets up middleware with real handlers
// This can be used by other contract tests to test against real implementations
func setupMiddlewareWithHandlers(t *testing.T) *jsonrpc.Middleware {
	// Setup Redis
	redisClient, err := config.GetRedisClient()
	if err != nil {
		t.Skipf("Skipping test - Redis not available: %v", err)
	}

	// Setup repository
	roomRepo := repository.NewChatRoomRepository(redisClient)

	// Setup services
	roomService := services.NewRoomService(roomRepo)
	messageService := services.NewMessageService(redisClient)
	participantService := services.NewParticipantService(redisClient)
	sessionService := services.NewSessionService(redisClient)

	// Setup handlers with services
	chatHandlers := handlers.NewChatHandlers(roomService, messageService, participantService, sessionService)

	// Setup JSON-RPC middleware
	middleware := jsonrpc.NewMiddleware()
	chatHandlers.RegisterMethods(middleware)

	return middleware
}

// TestJSONRPCIntegration tests the JSON-RPC methods with real implementations
func TestJSONRPCIntegration(t *testing.T) {
	// Setup Redis
	redisClient, err := config.GetRedisClient()
	if err != nil {
		t.Skipf("Skipping integration test - Redis not available: %v", err)
	}
	defer redisClient.Close()

	// Setup repository
	roomRepo := repository.NewChatRoomRepository(redisClient)

	// Setup services
	roomService := services.NewRoomService(roomRepo)
	messageService := services.NewMessageService(redisClient)
	participantService := services.NewParticipantService(redisClient)
	sessionService := services.NewSessionService(redisClient)

	// Setup handlers with services
	chatHandlers := handlers.NewChatHandlers(roomService, messageService, participantService, sessionService)

	// Setup JSON-RPC middleware
	middleware := jsonrpc.NewMiddleware()
	chatHandlers.RegisterMethods(middleware)

	t.Run("rooms.list should return empty list initially", func(t *testing.T) {
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "rooms.list",
			"params":  map[string]interface{}{},
			"id":      1,
		}

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("POST", "/api/rooms.list", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["jsonrpc"] != "2.0" {
			t.Errorf("Expected jsonrpc 2.0, got %v", response["jsonrpc"])
		}

		if response["error"] != nil {
			t.Errorf("Expected no error, got %v", response["error"])
		}

		result, ok := response["result"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected result to be an object")
		}

		rooms, ok := result["rooms"].([]interface{})
		if !ok {
			t.Fatal("Expected rooms to be an array")
		}

		if len(rooms) != 0 {
			t.Errorf("Expected empty rooms array, got %d rooms", len(rooms))
		}

		totalCount, ok := result["total_count"].(float64)
		if !ok || totalCount != 0 {
			t.Errorf("Expected total_count to be 0, got %v", result["total_count"])
		}
	})

	t.Run("rooms.create should create a room successfully", func(t *testing.T) {
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "rooms.create",
			"params": map[string]interface{}{
				"title": "Integration Test Room",
			},
			"id": 2,
		}

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("POST", "/api/rooms.create", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["jsonrpc"] != "2.0" {
			t.Errorf("Expected jsonrpc 2.0, got %v", response["jsonrpc"])
		}

		if response["error"] != nil {
			t.Errorf("Expected no error, got %v", response["error"])
		}

		result, ok := response["result"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected result to be an object")
		}

		if result["title"] != "Integration Test Room" {
			t.Errorf("Expected title 'Integration Test Room', got %v", result["title"])
		}

		if result["is_active"] != true {
			t.Error("Expected room to be active")
		}

		if result["participant_count"] != float64(1) {
			t.Errorf("Expected participant_count 1, got %v", result["participant_count"])
		}

		// Verify room ID is a valid UUID format
		roomID, ok := result["id"].(string)
		if !ok || len(roomID) != 36 {
			t.Errorf("Expected valid UUID room ID, got %v", result["id"])
		}
	})

	t.Run("rooms.create should fail with empty title", func(t *testing.T) {
		request := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "rooms.create",
			"params": map[string]interface{}{
				"title": "",
			},
			"id": 3,
		}

		requestBody, _ := json.Marshal(request)
		req := httptest.NewRequest("POST", "/api/rooms.create", bytes.NewBuffer(requestBody))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["error"] == nil {
			t.Error("Expected error response for empty title")
		}

		errorObj, ok := response["error"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected error to be an object")
		}

		if code, ok := errorObj["code"].(float64); ok {
			if int(code) != jsonrpc.ValidationFailed {
				t.Errorf("Expected validation error code %d, got %d", jsonrpc.ValidationFailed, int(code))
			}
		} else {
			t.Error("Expected error code to be a number")
		}
	})
}