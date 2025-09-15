package contract

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestEventsSSEEndpoint tests the GET /events/{roomId} SSE endpoint contract
func TestEventsSSEEndpoint(t *testing.T) {
	// This test MUST fail initially as the endpoint doesn't exist yet
	// Following TDD - RED phase

	tests := []struct {
		name            string
		path            string
		authCookie      *http.Cookie
		headers         map[string]string
		expectedStatus  int
		expectedHeaders map[string]string
		checkResponse   func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Authenticated user connects to SSE stream",
			path: "/events/room-123",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			headers: map[string]string{
				"Accept": "text/event-stream",
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Content-Type":                "text/event-stream",
				"Cache-Control":               "no-cache",
				"Connection":                  "keep-alive",
				"Access-Control-Allow-Origin": "*",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				// Should establish SSE connection
				if !strings.Contains(body, "data:") {
					t.Error("SSE stream should contain data events")
				}
			},
		},
		{
			name: "SSE stream with Last-Event-ID header for resumption",
			path: "/events/room-123",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			headers: map[string]string{
				"Accept":        "text/event-stream",
				"Last-Event-ID": "msg-456",
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Content-Type": "text/event-stream",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should resume from specified event ID
				body := w.Body.String()
				if !strings.Contains(body, "id:") {
					t.Error("SSE stream should include event IDs for resumption")
				}
			},
		},
		{
			name:           "Unauthenticated user gets unauthorized",
			path:           "/events/room-123",
			authCookie:     nil,
			expectedStatus: http.StatusUnauthorized, // 401
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should not establish SSE connection
			},
		},
		{
			name: "Invalid auth token gets unauthorized",
			path: "/events/room-123",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "invalid_jwt_token",
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should not establish SSE connection
			},
		},
		{
			name: "Non-existent room returns not found",
			path: "/events/nonexistent-room",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			headers: map[string]string{
				"Accept": "text/event-stream",
			},
			expectedStatus: http.StatusNotFound, // 404
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should not establish SSE connection
			},
		},
		{
			name: "Access denied to private room",
			path: "/events/private-room-789",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "non_member_jwt_token",
			},
			headers: map[string]string{
				"Accept": "text/event-stream",
			},
			expectedStatus: http.StatusForbidden, // 403
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should not establish SSE connection
			},
		},
		{
			name: "Invalid room ID format returns bad request",
			path: "/events/invalid@room#id",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			headers: map[string]string{
				"Accept": "text/event-stream",
			},
			expectedStatus: http.StatusBadRequest, // 400
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should not establish SSE connection
			},
		},
		{
			name: "Non-SSE Accept header still works",
			path: "/events/room-123",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			headers: map[string]string{
				"Accept": "text/html,application/xhtml+xml",
			},
			expectedStatus: http.StatusOK, // Should still provide SSE
			expectedHeaders: map[string]string{
				"Content-Type": "text/event-stream",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should establish SSE connection regardless of Accept header
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Add auth cookie if provided
			if tt.authCookie != nil {
				req.AddCookie(tt.authCookie)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// This will fail until we implement the SSE handler
			// TODO: Replace with actual handler once implemented
			mockSSEHandler := func(w http.ResponseWriter, r *http.Request) {
				// This mock will be replaced with real implementation
				t.Errorf("SSE handler not implemented yet - this test should fail (TDD RED phase)")
				w.WriteHeader(http.StatusNotFound)
			}

			// Call handler
			mockSSEHandler(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check headers
			for key, expectedValue := range tt.expectedHeaders {
				actualValue := w.Header().Get(key)
				if actualValue != expectedValue {
					t.Errorf("Expected header %s: %s, got: %s", key, expectedValue, actualValue)
				}
			}

			// Check response
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestEventsSSEMessageFormat tests SSE message format compliance
func TestEventsSSEMessageFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/events/room-123", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("SSE handler not implemented yet - message format test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	body := w.Body.String()

	// SSE format requirements
	sseFormatChecks := []string{
		"event:", // Event type
		"id:",    // Event ID for resumption
		"data:",  // Event data
		"\n\n",   // Double newline to separate events
	}

	for _, check := range sseFormatChecks {
		if !strings.Contains(body, check) {
			t.Errorf("SSE stream missing format element: %s", check)
		}
	}

	// Should contain proper SSE event types
	expectedEvents := []string{
		"event: message",
		"event: user-joined",
		"event: user-left",
		"event: typing",
	}

	foundEventType := false
	for _, eventType := range expectedEvents {
		if strings.Contains(body, eventType) {
			foundEventType = true
			break
		}
	}
	if !foundEventType {
		t.Error("SSE stream should include proper event types")
	}
}

// TestEventsSSEKeepAlive tests SSE keep-alive functionality
func TestEventsSSEKeepAlive(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/events/room-123", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})

	// Create context with timeout for testing
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("SSE handler not implemented yet - keep-alive test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	body := w.Body.String()

	// Should send keep-alive comments or heartbeat
	keepAliveIndicators := []string{
		": keep-alive",
		": heartbeat",
		"event: ping",
		"event: heartbeat",
	}

	foundKeepAlive := false
	for _, indicator := range keepAliveIndicators {
		if strings.Contains(body, indicator) {
			foundKeepAlive = true
			break
		}
	}
	if !foundKeepAlive {
		t.Error("SSE stream should include keep-alive mechanism")
	}
}

// TestEventsSSESecurity tests security requirements for SSE
func TestEventsSSESecurity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/events/room-123", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("SSE handler not implemented yet - security test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	// Security headers check for SSE
	securityHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Cache-Control":          "no-cache",
	}

	for header, expectedValue := range securityHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Missing security header %s: expected %s, got %s", header, expectedValue, actualValue)
		}
	}
}

// TestEventsSSEPerformance tests SSE performance requirements
func TestEventsSSEPerformance(t *testing.T) {
	// Performance requirement: < 100ms to establish SSE connection
	req := httptest.NewRequest(http.MethodGet, "/events/room-123", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("SSE handler not implemented yet - performance test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	// Measure connection establishment time
	start := time.Now()
	mockHandler(w, req)
	duration := time.Since(start)

	// Check performance requirement
	if duration > 100*time.Millisecond {
		t.Errorf("SSE connection took %v, expected < 100ms", duration)
	}
}

// TestEventsSSERoomValidation tests room ID validation for SSE
func TestEventsSSERoomValidation(t *testing.T) {
	invalidRoomTests := []struct {
		name   string
		roomId string
		status int
	}{
		{
			name:   "Empty room ID",
			roomId: "",
			status: http.StatusBadRequest,
		},
		{
			name:   "Room ID with special characters",
			roomId: "room@#$%",
			status: http.StatusBadRequest,
		},
		{
			name:   "Room ID with spaces",
			roomId: "room with spaces",
			status: http.StatusBadRequest,
		},
		{
			name:   "Room ID with path traversal",
			roomId: "../../../etc/passwd",
			status: http.StatusBadRequest,
		},
		{
			name:   "Extremely long room ID",
			roomId: strings.Repeat("a", 1000),
			status: http.StatusBadRequest,
		},
		{
			name:   "Room ID with null bytes",
			roomId: "room\x00id",
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range invalidRoomTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/events/"+tt.roomId, nil)
			req.Header.Set("Accept", "text/event-stream")
			req.AddCookie(&http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			})
			w := httptest.NewRecorder()

			// Mock handler - will be replaced
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("SSE handler not implemented yet - room validation test should fail")
				w.WriteHeader(http.StatusNotFound)
			}

			mockHandler(w, req)

			if w.Code != tt.status {
				t.Errorf("Expected status %d for invalid room ID %s, got %d", tt.status, tt.roomId, w.Code)
			}
		})
	}
}

// TestEventsSSEConnectionManagement tests connection lifecycle
func TestEventsSSEConnectionManagement(t *testing.T) {
	// Test connection cleanup and proper resource management
	req := httptest.NewRequest(http.MethodGet, "/events/room-123", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})

	// Create context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("SSE handler not implemented yet - connection management test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	// Start handler
	go func() {
		mockHandler(w, req)
	}()

	// Cancel context after short delay
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Give handler time to cleanup
	time.Sleep(50 * time.Millisecond)

	// Connection should be properly cleaned up
	// (This would be tested with actual implementation)
}

// TestEventsSSEEventDelivery tests message delivery over SSE
func TestEventsSSEEventDelivery(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/events/room-123", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("SSE handler not implemented yet - event delivery test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	body := w.Body.String()

	// Should deliver various event types
	expectedEventTypes := []struct {
		eventType string
		dataType  string
	}{
		{"message", "HTML fragment with message"},
		{"user-joined", "User status update"},
		{"user-left", "User status update"},
		{"typing", "Typing indicator"},
		{"room-updated", "Room metadata change"},
	}

	for _, eventType := range expectedEventTypes {
		eventPattern := "event: " + eventType.eventType
		if !strings.Contains(body, eventPattern) {
			t.Errorf("SSE stream should include %s events", eventType.eventType)
		}
	}
}

// TestEventsSSEErrorHandling tests error scenarios for SSE
func TestEventsSSEErrorHandling(t *testing.T) {
	errorTests := []struct {
		name           string
		authToken      string
		expectedStatus int
	}{
		{
			name:           "Malformed JWT token",
			authToken:      "malformed.jwt.token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Empty JWT token",
			authToken:      "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "JWT with invalid signature",
			authToken:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid_signature",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Expired JWT token",
			authToken:      "expired_jwt_token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/events/room-123", nil)
			req.Header.Set("Accept", "text/event-stream")
			if tt.authToken != "" {
				req.AddCookie(&http.Cookie{
					Name:  "auth_token",
					Value: tt.authToken,
				})
			}
			w := httptest.NewRecorder()

			// Mock handler - will be replaced
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("SSE handler not implemented yet - error handling test should fail")
				w.WriteHeader(http.StatusNotFound)
			}

			mockHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

// TestEventsSSEConcurrency tests concurrent SSE connections
func TestEventsSSEConcurrency(t *testing.T) {
	// Test multiple concurrent SSE connections to same room
	const numConnections = 10

	results := make(chan int, numConnections)

	for i := 0; i < numConnections; i++ {
		go func(id int) {
			req := httptest.NewRequest(http.MethodGet, "/events/room-123", nil)
			req.Header.Set("Accept", "text/event-stream")
			req.AddCookie(&http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			})
			w := httptest.NewRecorder()

			// Mock handler - will be replaced
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("SSE handler not implemented yet - concurrency test should fail")
				w.WriteHeader(http.StatusNotFound)
			}

			mockHandler(w, req)
			results <- w.Code
		}(i)
	}

	// Collect results
	statusCodes := make(map[int]int)
	for i := 0; i < numConnections; i++ {
		code := <-results
		statusCodes[code]++
	}

	// All connections should either succeed or fail consistently
	if len(statusCodes) > 2 { // Only success/failure expected
		t.Errorf("Inconsistent status codes in concurrent SSE connections: %v", statusCodes)
	}
}

// TestEventsSSEReconnection tests SSE reconnection functionality
func TestEventsSSEReconnection(t *testing.T) {
	// Test SSE reconnection with Last-Event-ID
	lastEventID := "msg-789"

	req := httptest.NewRequest(http.MethodGet, "/events/room-123", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Last-Event-ID", lastEventID)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("SSE handler not implemented yet - reconnection test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	// Should handle Last-Event-ID for reconnection
	body := w.Body.String()
	if strings.Contains(body, "id: "+lastEventID) {
		t.Error("Should not resend events with IDs <= Last-Event-ID")
	}

	// Should include retry directive for reconnection
	if !strings.Contains(body, "retry:") {
		t.Error("SSE stream should include retry directive for reconnection")
	}
}

// TestEventsSSEStreamParsing tests SSE stream parsing
func TestEventsSSEStreamParsing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/events/room-123", nil)
	req.Header.Set("Accept", "text/event-stream")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("SSE handler not implemented yet - stream parsing test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	body := w.Body.String()

	// Parse SSE stream for proper format
	scanner := bufio.NewScanner(strings.NewReader(body))
	events := 0
	hasEventType := false
	hasEventID := false
	hasEventData := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event:") {
			hasEventType = true
		} else if strings.HasPrefix(line, "id:") {
			hasEventID = true
		} else if strings.HasPrefix(line, "data:") {
			hasEventData = true
		} else if line == "" {
			// End of event
			if hasEventType && hasEventID && hasEventData {
				events++
			}
			hasEventType = false
			hasEventID = false
			hasEventData = false
		}
	}

	if events == 0 {
		t.Error("SSE stream should contain properly formatted events")
	}
}

