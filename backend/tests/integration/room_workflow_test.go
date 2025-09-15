package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"duckchat/internal/room"
	"duckchat/internal/user"
)

// T013: Integration test - Room creation and joining workflow
func TestCompleteRoomWorkflow(t *testing.T) {
	server := setupRoomIntegrationServer(t)
	defer server.cleanup()

	t.Run("complete room lifecycle with multiple users", func(t *testing.T) {
		// Create multiple test users
		creatorToken := performUserLogin(t, server, "room-creator")
		joiner1Token := performUserLogin(t, server, "room-joiner-1")
		joiner2Token := performUserLogin(t, server, "room-joiner-2")
		joiner3Token := performUserLogin(t, server, "room-joiner-3")

		// Step 1: Creator creates a room
		roomResponse := createRoom(t, server, creatorToken, "Integration Test Room")
		roomID := roomResponse.RoomID

		if roomResponse.Name != "Integration Test Room" {
			t.Errorf("Expected room name 'Integration Test Room', got '%s'", roomResponse.Name)
		}
		if roomResponse.ParticipantCount != 1 {
			t.Errorf("Expected 1 initial participant (creator), got %d", roomResponse.ParticipantCount)
		}

		// Step 2: Multiple users join the room
		joinResponse1 := joinRoom(t, server, joiner1Token, roomID)
		validateJoinResponse(t, joinResponse1, roomID, "Integration Test Room", 2)

		joinResponse2 := joinRoom(t, server, joiner2Token, roomID)
		validateJoinResponse(t, joinResponse2, roomID, "Integration Test Room", 3)

		joinResponse3 := joinRoom(t, server, joiner3Token, roomID)
		validateJoinResponse(t, joinResponse3, roomID, "Integration Test Room", 4)

		// Step 3: Get room info and verify all participants
		roomInfo := getRoomInfo(t, server, creatorToken, roomID)
		if roomInfo.ParticipantCount != 4 {
			t.Errorf("Expected 4 participants total, got %d", roomInfo.ParticipantCount)
		}
		if len(roomInfo.Participants) != 4 {
			t.Errorf("Expected 4 participants in list, got %d", len(roomInfo.Participants))
		}

		// Verify specific participants
		expectedParticipants := map[string]bool{
			"room-creator":   false,
			"room-joiner-1":  false,
			"room-joiner-2":  false,
			"room-joiner-3":  false,
		}
		for _, participant := range roomInfo.Participants {
			if _, exists := expectedParticipants[participant]; exists {
				expectedParticipants[participant] = true
			}
		}
		for userID, found := range expectedParticipants {
			if !found {
				t.Errorf("Expected participant '%s' not found in room", userID)
			}
		}

		// Step 4: Users leave the room one by one
		leaveRoom(t, server, joiner1Token, roomID)

		// Verify participant count decreases
		roomInfo = getRoomInfo(t, server, creatorToken, roomID)
		if roomInfo.ParticipantCount != 3 {
			t.Errorf("Expected 3 participants after one left, got %d", roomInfo.ParticipantCount)
		}

		leaveRoom(t, server, joiner2Token, roomID)
		roomInfo = getRoomInfo(t, server, creatorToken, roomID)
		if roomInfo.ParticipantCount != 2 {
			t.Errorf("Expected 2 participants after two left, got %d", roomInfo.ParticipantCount)
		}

		// Step 5: Test idempotent operations
		// User tries to join room they're already in
		joinResponse := joinRoom(t, server, joiner3Token, roomID)
		if joinResponse.ParticipantCount != 2 {
			t.Error("Joining room user is already in should be idempotent")
		}

		// User tries to leave room they're not in
		leaveRoom(t, server, joiner1Token, roomID) // Already left, should not error
	})

	t.Run("concurrent room operations", func(t *testing.T) {
		// Test concurrent users joining the same room
		creatorToken := performUserLogin(t, server, "concurrent-creator")
		roomResponse := createRoom(t, server, creatorToken, "Concurrent Test Room")
		roomID := roomResponse.RoomID

		// Create multiple joiners concurrently
		numJoiners := 5
		joinerTokens := make([]string, numJoiners)
		for i := 0; i < numJoiners; i++ {
			joinerTokens[i] = performUserLogin(t, server, fmt.Sprintf("concurrent-joiner-%d", i))
		}

		// All joiners attempt to join simultaneously
		type joinResult struct {
			success bool
			count   int
		}
		results := make(chan joinResult, numJoiners)

		for i := 0; i < numJoiners; i++ {
			go func(token string) {
				response := joinRoom(t, server, token, roomID)
				results <- joinResult{
					success: response.RoomID == roomID,
					count:   response.ParticipantCount,
				}
			}(joinerTokens[i])
		}

		// Collect all results
		successCount := 0
		for i := 0; i < numJoiners; i++ {
			result := <-results
			if result.success {
				successCount++
			}
		}

		if successCount != numJoiners {
			t.Errorf("Expected all %d joiners to succeed, got %d", numJoiners, successCount)
		}

		// Verify final room state
		finalRoomInfo := getRoomInfo(t, server, creatorToken, roomID)
		expectedTotal := numJoiners + 1 // joiners + creator
		if finalRoomInfo.ParticipantCount != expectedTotal {
			t.Errorf("Expected %d total participants, got %d", expectedTotal, finalRoomInfo.ParticipantCount)
		}
	})

	t.Run("room capacity and limits", func(t *testing.T) {
		// Test room behavior at capacity limits
		creatorToken := performUserLogin(t, server, "capacity-creator")
		roomResponse := createRoom(t, server, creatorToken, "Capacity Test Room")
		roomID := roomResponse.RoomID

		// Join users up to a reasonable limit (test the limit enforcement)
		maxUsers := 10 // Assuming reasonable limit for testing
		joinerTokens := make([]string, maxUsers)

		for i := 0; i < maxUsers; i++ {
			joinerTokens[i] = performUserLogin(t, server, fmt.Sprintf("capacity-user-%d", i))
			joinRoom(t, server, joinerTokens[i], roomID)
		}

		// Verify room can handle the capacity
		roomInfo := getRoomInfo(t, server, creatorToken, roomID)
		expectedCount := maxUsers + 1 // Users + creator
		if roomInfo.ParticipantCount != expectedCount {
			t.Errorf("Expected %d participants, got %d", expectedCount, roomInfo.ParticipantCount)
		}

		// Verify room state is still active
		if !roomInfo.CanJoin {
			t.Log("Room has reached capacity and cannot accept more users")
		}
	})

	t.Run("room persistence and state", func(t *testing.T) {
		// Test that room state persists across operations
		creatorToken := performUserLogin(t, server, "persist-creator")
		roomResponse := createRoom(t, server, creatorToken, "Persistence Test Room")
		roomID := roomResponse.RoomID

		// Record initial room creation time
		initialInfo := getRoomInfo(t, server, creatorToken, roomID)
		initialCreatedAt := initialInfo.CreatedAt

		// Wait a bit and perform operations
		time.Sleep(100 * time.Millisecond)

		joinerToken := performUserLogin(t, server, "persist-joiner")
		joinRoom(t, server, joinerToken, roomID)

		// Verify room info is consistent
		updatedInfo := getRoomInfo(t, server, creatorToken, roomID)

		if updatedInfo.RoomID != roomID {
			t.Error("Room ID should remain consistent")
		}
		if updatedInfo.Name != "Persistence Test Room" {
			t.Error("Room name should remain consistent")
		}
		if updatedInfo.CreatedBy != "persist-creator" {
			t.Error("Room creator should remain consistent")
		}
		if updatedInfo.CreatedAt != initialCreatedAt {
			t.Error("Room creation time should remain consistent")
		}

		// Last activity should be more recent
		if updatedInfo.LastActivity <= initialInfo.LastActivity {
			t.Error("Last activity should be updated after room operations")
		}

		// Participant count should be updated
		if updatedInfo.ParticipantCount != 2 {
			t.Errorf("Expected 2 participants, got %d", updatedInfo.ParticipantCount)
		}
	})

	t.Run("invalid room operations", func(t *testing.T) {
		userToken := performUserLogin(t, server, "invalid-ops-user")

		// Try to join non-existent room
		testInvalidRoomJoin(t, server, userToken, "non-existent-room-id")

		// Try to get info for non-existent room
		testInvalidRoomInfo(t, server, userToken, "non-existent-room-id")

		// Try to leave room user was never in
		leaveRoom(t, server, userToken, "non-existent-room-id") // Should not error

		// Create room with various invalid inputs tested in contract tests
		// These should be handled gracefully
	})
}

// Helper structures and functions for room workflow tests

type RoomResponse struct {
	RoomID           string `json:"room_id"`
	Name             string `json:"name"`
	CreatedAt        string `json:"created_at"`
	ParticipantCount int    `json:"participant_count"`
}

type JoinResponse struct {
	RoomID           string `json:"room_id"`
	Name             string `json:"name"`
	ParticipantCount int    `json:"participant_count"`
}

type RoomInfoResponse struct {
	RoomID           string   `json:"room_id"`
	Name             string   `json:"name"`
	ParticipantCount int      `json:"participant_count"`
	Participants     []string `json:"participants"`
	CreatedBy        string   `json:"created_by"`
	CreatedAt        string   `json:"created_at"`
	LastActivity     string   `json:"last_activity"`
	CanJoin          bool     `json:"can_join"`
}

// Extended test server for room integration tests
type RoomIntegrationServer struct {
	*IntegrationTestServer
	userHandler *user.Handler
	roomHandler *room.Handler
	roomService *room.Service
}

func setupRoomIntegrationServer(t *testing.T) *RoomIntegrationServer {
	baseServer := setupIntegrationTestServer(t)

	// Initialize room components
	roomRepo := room.NewRepository(baseServer.redisClient)
	roomService := room.NewService(roomRepo)
	roomHandler := room.NewHandler(roomService)

	userHandler := user.NewHandler(baseServer.userService)

	return &RoomIntegrationServer{
		IntegrationTestServer: baseServer,
		userHandler:           userHandler,
		roomHandler:           roomHandler,
		roomService:           roomService,
	}
}

func performUserLogin(t *testing.T, server *RoomIntegrationServer, userID string) string {
	loginResponse := performLogin(t, server.IntegrationTestServer, userID)
	return loginResponse.AccessToken
}

func createRoom(t *testing.T, server *RoomIntegrationServer, token string, roomName string) RoomResponse {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "rooms.create",
		"params": map[string]interface{}{
			"name": roomName,
		},
		"id": "create-room-test",
	})

	req := httptest.NewRequest("POST", "/api/rooms", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.roomHandler.HandleRoom)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Room creation failed: status %d, body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	result := response["result"].(map[string]interface{})
	return RoomResponse{
		RoomID:           result["room_id"].(string),
		Name:             result["name"].(string),
		CreatedAt:        result["created_at"].(string),
		ParticipantCount: int(result["participant_count"].(float64)),
	}
}

func joinRoom(t *testing.T, server *RoomIntegrationServer, token string, roomID string) JoinResponse {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "rooms.join",
		"params": map[string]interface{}{
			"room_id": roomID,
		},
		"id": "join-room-test",
	})

	req := httptest.NewRequest("POST", "/api/rooms", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.roomHandler.HandleRoom)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Room join failed: status %d, body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	result := response["result"].(map[string]interface{})
	return JoinResponse{
		RoomID:           result["room_id"].(string),
		Name:             result["name"].(string),
		ParticipantCount: int(result["participant_count"].(float64)),
	}
}

func leaveRoom(t *testing.T, server *RoomIntegrationServer, token string, roomID string) {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "rooms.leave",
		"params": map[string]interface{}{
			"room_id": roomID,
		},
		"id": "leave-room-test",
	})

	req := httptest.NewRequest("POST", "/api/rooms", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.roomHandler.HandleRoom)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Logf("Room leave result: status %d, body: %s", w.Code, w.Body.String())
	}
}

func getRoomInfo(t *testing.T, server *RoomIntegrationServer, token string, roomID string) RoomInfoResponse {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "rooms.info",
		"params": map[string]interface{}{
			"room_id": roomID,
		},
		"id": "room-info-test",
	})

	req := httptest.NewRequest("POST", "/api/rooms", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.roomHandler.HandleRoom)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Room info failed: status %d, body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	result := response["result"].(map[string]interface{})

	participants := make([]string, 0)
	if participantList, ok := result["participants"].([]interface{}); ok {
		for _, p := range participantList {
			if participant, ok := p.(string); ok {
				participants = append(participants, participant)
			}
		}
	}

	return RoomInfoResponse{
		RoomID:           result["room_id"].(string),
		Name:             result["name"].(string),
		ParticipantCount: int(result["participant_count"].(float64)),
		Participants:     participants,
		CreatedBy:        result["created_by"].(string),
		CreatedAt:        result["created_at"].(string),
		LastActivity:     result["last_activity"].(string),
		CanJoin:          result["can_join"].(bool),
	}
}

func validateJoinResponse(t *testing.T, response JoinResponse, expectedRoomID string, expectedName string, expectedCount int) {
	if response.RoomID != expectedRoomID {
		t.Errorf("Expected room_id '%s', got '%s'", expectedRoomID, response.RoomID)
	}
	if response.Name != expectedName {
		t.Errorf("Expected room name '%s', got '%s'", expectedName, response.Name)
	}
	if response.ParticipantCount != expectedCount {
		t.Errorf("Expected participant count %d, got %d", expectedCount, response.ParticipantCount)
	}
}

func testInvalidRoomJoin(t *testing.T, server *RoomIntegrationServer, token string, roomID string) {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "rooms.join",
		"params": map[string]interface{}{
			"room_id": roomID,
		},
		"id": "invalid-join-test",
	})

	req := httptest.NewRequest("POST", "/api/rooms", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.roomHandler.HandleRoom)
	handler(w, req)

	if w.Code == http.StatusOK {
		t.Error("Joining non-existent room should fail")
	}
}

func testInvalidRoomInfo(t *testing.T, server *RoomIntegrationServer, token string, roomID string) {
	requestBody, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "rooms.info",
		"params": map[string]interface{}{
			"room_id": roomID,
		},
		"id": "invalid-info-test",
	})

	req := httptest.NewRequest("POST", "/api/rooms", bytes.NewBuffer(requestBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	handler := server.authMiddleware.RequireAuth(server.roomHandler.HandleRoom)
	handler(w, req)

	if w.Code == http.StatusOK {
		t.Error("Getting info for non-existent room should fail")
	}
}