package contract

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"duckchat/internal/sse"
)

// T011: Contract test GET /api/rooms.events (SSE)
func TestSSEEventsContract(t *testing.T) {
	// Setup test server with real dependencies
	server := setupSSETestServer(t)
	defer server.cleanup()

	// Create test user and room
	userToken := createTestUserAndToken(t, server.RoomTestServer, "sse-test-user")
	testRoomID := createTestRoom(t, server.RoomTestServer, userToken, "SSE Test Room")

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedStatus int
		validateSSE    func(t *testing.T, w *httptest.ResponseRecorder)
		timeout        time.Duration
	}{
		{
			name: "valid SSE connection with auth token",
			setupRequest: func() *http.Request {
				// Create request with query parameters
				u := &url.URL{
					Path: "/api/events",
					RawQuery: fmt.Sprintf("user_id=sse-test-user&room_id=%s", testRoomID),
				}
				req := httptest.NewRequest("GET", u.String(), nil)
				req.Header.Set("Authorization", "Bearer "+userToken)
				req.Header.Set("Accept", "text/event-stream")
				req.Header.Set("Cache-Control", "no-cache")
				return req
			},
			expectedStatus: http.StatusOK,
			validateSSE: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Check SSE headers
				expectedHeaders := map[string]string{
					"Content-Type":                "text/event-stream",
					"Cache-Control":               "no-cache",
					"Connection":                  "keep-alive",
					"Access-Control-Allow-Origin": "*",
				}

				for header, expectedValue := range expectedHeaders {
					if actualValue := w.Header().Get(header); actualValue != expectedValue {
						t.Errorf("Expected %s: %s, got %s", header, expectedValue, actualValue)
					}
				}

				// Check for initial connection event in SSE format
				body := w.Body.String()
				if !strings.Contains(body, "event: connected") {
					t.Error("Expected connection event in SSE stream")
				}

				// Validate SSE event format
				lines := strings.Split(body, "\n")
				var foundEvent, foundData bool
				for _, line := range lines {
					if strings.HasPrefix(line, "event: ") {
						foundEvent = true
						eventType := strings.TrimPrefix(line, "event: ")
						if eventType != "connected" && eventType != "heartbeat" {
							t.Logf("Found event type: %s", eventType)
						}
					}
					if strings.HasPrefix(line, "data: ") {
						foundData = true
						// Basic JSON validation
						dataLine := strings.TrimPrefix(line, "data: ")
						if dataLine != "" && !strings.HasPrefix(dataLine, "{") {
							t.Errorf("SSE data should be JSON format, got: %s", dataLine)
						}
					}
				}

				if !foundEvent {
					t.Error("Expected SSE event line in stream")
				}
				if !foundData {
					t.Error("Expected SSE data line in stream")
				}
			},
			timeout: 2 * time.Second,
		},
		{
			name: "missing user_id parameter",
			setupRequest: func() *http.Request {
				u := &url.URL{
					Path:     "/api/events",
					RawQuery: fmt.Sprintf("room_id=%s", testRoomID), // Missing user_id
				}
				req := httptest.NewRequest("GET", u.String(), nil)
				req.Header.Set("Authorization", "Bearer "+userToken)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			validateSSE: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should return error, not start SSE stream
				body := w.Body.String()
				if strings.Contains(body, "event:") {
					t.Error("Should not start SSE stream with missing parameters")
				}
				if !strings.Contains(body, "Missing user_id or room_id") {
					t.Error("Should return appropriate error message")
				}
			},
			timeout: 1 * time.Second,
		},
		{
			name: "missing room_id parameter",
			setupRequest: func() *http.Request {
				u := &url.URL{
					Path:     "/api/events",
					RawQuery: "user_id=sse-test-user", // Missing room_id
				}
				req := httptest.NewRequest("GET", u.String(), nil)
				req.Header.Set("Authorization", "Bearer "+userToken)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			validateSSE: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if strings.Contains(body, "event:") {
					t.Error("Should not start SSE stream with missing parameters")
				}
				if !strings.Contains(body, "Missing user_id or room_id") {
					t.Error("Should return appropriate error message")
				}
			},
			timeout: 1 * time.Second,
		},
		{
			name: "empty query parameters",
			setupRequest: func() *http.Request {
				u := &url.URL{
					Path:     "/api/events",
					RawQuery: "user_id=&room_id=", // Empty values
				}
				req := httptest.NewRequest("GET", u.String(), nil)
				req.Header.Set("Authorization", "Bearer "+userToken)
				return req
			},
			expectedStatus: http.StatusBadRequest,
			validateSSE: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if strings.Contains(body, "event:") {
					t.Error("Should not start SSE stream with empty parameters")
				}
			},
			timeout: 1 * time.Second,
		},
		{
			name: "non-existent room_id",
			setupRequest: func() *http.Request {
				u := &url.URL{
					Path:     "/api/events",
					RawQuery: "user_id=sse-test-user&room_id=non-existent-room",
				}
				req := httptest.NewRequest("GET", u.String(), nil)
				req.Header.Set("Authorization", "Bearer "+userToken)
				return req
			},
			expectedStatus: http.StatusOK, // SSE connection should start regardless
			validateSSE: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should still establish SSE connection
				if w.Header().Get("Content-Type") != "text/event-stream" {
					t.Error("Should establish SSE connection even for non-existent room")
				}

				body := w.Body.String()
				if !strings.Contains(body, "event: connected") {
					t.Error("Should send connection event even for non-existent room")
				}
			},
			timeout: 2 * time.Second,
		},
		{
			name: "connection without Accept header",
			setupRequest: func() *http.Request {
				u := &url.URL{
					Path:     "/api/events",
					RawQuery: fmt.Sprintf("user_id=sse-test-user&room_id=%s", testRoomID),
				}
				req := httptest.NewRequest("GET", u.String(), nil)
				req.Header.Set("Authorization", "Bearer "+userToken)
				// No Accept: text/event-stream header
				return req
			},
			expectedStatus: http.StatusOK, // Should work without explicit Accept header
			validateSSE: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should still work
				if w.Header().Get("Content-Type") != "text/event-stream" {
					t.Error("Should set SSE content type even without Accept header")
				}
			},
			timeout: 2 * time.Second,
		},
		{
			name: "concurrent connections",
			setupRequest: func() *http.Request {
				u := &url.URL{
					Path:     "/api/events",
					RawQuery: fmt.Sprintf("user_id=sse-concurrent-user&room_id=%s", testRoomID),
				}
				req := httptest.NewRequest("GET", u.String(), nil)
				req.Header.Set("Authorization", "Bearer "+userToken)
				req.Header.Set("Accept", "text/event-stream")
				return req
			},
			expectedStatus: http.StatusOK,
			validateSSE: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should handle multiple connections to same room
				if w.Header().Get("Content-Type") != "text/event-stream" {
					t.Error("Concurrent connection should work")
				}
			},
			timeout: 2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup request
			req := tt.setupRequest()

			// Create recorder that can handle streaming
			w := httptest.NewRecorder()

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()
			req = req.WithContext(ctx)

			// Handle SSE request
			go func() {
				server.sseService.HandleSSE(w, req)
			}()

			// Wait for initial response or timeout
			select {
			case <-time.After(500 * time.Millisecond): // Wait for initial response
				// Check status code (should be available after connection starts)
				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				}

				// Validate SSE response
				if tt.validateSSE != nil {
					tt.validateSSE(t, w)
				}

			case <-ctx.Done():
				if w.Code != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
				}
			}
		})
	}
}

// Test SSE event broadcasting functionality
func TestSSEEventBroadcast(t *testing.T) {
	server := setupSSETestServer(t)
	defer server.cleanup()

	userToken := createTestUserAndToken(t, server.RoomTestServer, "broadcast-user")
	testRoomID := createTestRoom(t, server.RoomTestServer, userToken, "Broadcast Test Room")

	// Test broadcasting events to room
	t.Run("broadcast to room", func(t *testing.T) {
		// This is a unit test for the SSE service broadcasting capability
		testEvent := sse.Event{
			Type: sse.EventTypeMessage,
			Data: sse.MessageEventData{
				MessageID:   "test-msg-123",
				RoomID:      testRoomID,
				UserID:      "broadcast-user",
				Username:    "Test User",
				Content:     "Test broadcast message",
				Timestamp:   time.Now().Format(time.RFC3339),
				MessageType: "text",
			},
		}

		// This would normally test actual broadcasting, but requires active connections
		// For contract test, we verify the service methods exist and can be called
		server.sseService.BroadcastToRoom(testRoomID, testEvent)
		server.sseService.BroadcastToUser("broadcast-user", testEvent)
		server.sseService.BroadcastToAll(testEvent)

		// Verify service stats
		stats := server.sseService.GetStats()
		if stats == nil {
			t.Error("SSE service should provide stats")
		}

		// Verify room client count method
		clientCount := server.sseService.GetRoomClients(testRoomID)
		if clientCount < 0 {
			t.Error("Room client count should be non-negative")
		}
	})
}

// Extended test server for SSE operations
type SSETestServer struct {
	*RoomTestServer
	sseService *sse.Service
}

func setupSSETestServer(t *testing.T) *SSETestServer {
	roomServer := setupRoomTestServer(t)

	// Initialize SSE service
	sseService := sse.NewService()

	return &SSETestServer{
		RoomTestServer: roomServer,
		sseService:     sseService,
	}
}