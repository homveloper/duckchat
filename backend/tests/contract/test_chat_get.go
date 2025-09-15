package contract

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestChatGetEndpoint tests the GET /chat/{roomId} endpoint contract
func TestChatGetEndpoint(t *testing.T) {
	// This test MUST fail initially as the endpoint doesn't exist yet
	// Following TDD - RED phase

	tests := []struct {
		name            string
		path            string
		authCookie      *http.Cookie
		expectedStatus  int
		expectedHeaders map[string]string
		checkContent    func(t *testing.T, body string)
	}{
		{
			name: "Authenticated user accesses public room",
			path: "/chat/room-123",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			},
			checkContent: func(t *testing.T, body string) {
				// Must contain chat interface elements
				requiredElements := []string{
					`<div class="chat-container"`,
					`<div class="messages-container"`,
					`<form class="message-form"`,
					`name="content"`,
					`name="room_id"`,
					`name="csrf_token"`,
					`type="submit"`,
					`hx-post="/messages"`,
					`hx-sse="connect:/events/room-123"`,
				}
				for _, element := range requiredElements {
					if !contains(body, element) {
						t.Errorf("Chat page missing required element: %s", element)
					}
				}

				// Must be mobile-responsive
				responsiveElements := []string{
					`<meta name="viewport"`,
					`class="chat-responsive"`,
					`class="mobile-chat"`,
				}
				for _, element := range responsiveElements {
					if !contains(body, element) {
						t.Errorf("Chat page missing responsive element: %s", element)
					}
				}

				// Should include HTMX attributes for real-time functionality
				htmxAttributes := []string{
					`hx-target=`,
					`hx-trigger=`,
					`hx-swap=`,
				}
				foundHtmx := false
				for _, attr := range htmxAttributes {
					if contains(body, attr) {
						foundHtmx = true
						break
					}
				}
				if !foundHtmx {
					t.Error("Chat interface should use HTMX attributes")
				}
			},
		},
		{
			name: "Authenticated user accesses private room they're member of",
			path: "/chat/private-room-456",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "member_jwt_token",
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			},
			checkContent: func(t *testing.T, body string) {
				// Should show private room indicators
				if !contains(body, `class="private-room-indicator"`) {
					t.Error("Private room should have visual indicators")
				}
			},
		},
		{
			name:           "Unauthenticated user redirected to login",
			path:           "/chat/room-123",
			authCookie:     nil,
			expectedStatus: http.StatusFound, // 302
			expectedHeaders: map[string]string{
				"Location": "/login",
			},
			checkContent: func(t *testing.T, body string) {
				// Should redirect, minimal body expected
			},
		},
		{
			name: "Invalid auth token redirected to login",
			path: "/chat/room-123",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "invalid_jwt_token",
			},
			expectedStatus: http.StatusFound,
			expectedHeaders: map[string]string{
				"Location": "/login",
			},
			checkContent: func(t *testing.T, body string) {
				// Should redirect, minimal body expected
			},
		},
		{
			name: "Non-existent room returns 404",
			path: "/chat/nonexistent-room",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			expectedStatus: http.StatusNotFound, // 404
			expectedHeaders: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			},
			checkContent: func(t *testing.T, body string) {
				if !contains(body, "Room not found") {
					t.Error("Should show room not found message")
				}
				if !contains(body, `class="error-page"`) {
					t.Error("Should render error page template")
				}
			},
		},
		{
			name: "Access denied to private room",
			path: "/chat/private-room-789",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "non_member_jwt_token",
			},
			expectedStatus: http.StatusForbidden, // 403
			expectedHeaders: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
			},
			checkContent: func(t *testing.T, body string) {
				if !contains(body, "Access denied") {
					t.Error("Should show access denied message")
				}
				if !contains(body, `class="error-page"`) {
					t.Error("Should render error page template")
				}
			},
		},
		{
			name: "Invalid room ID format returns 400",
			path: "/chat/invalid@room#id",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			expectedStatus: http.StatusBadRequest, // 400
			checkContent: func(t *testing.T, body string) {
				if !contains(body, "Invalid room ID") {
					t.Error("Should show invalid room ID message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			// Add auth cookie if provided
			if tt.authCookie != nil {
				req.AddCookie(tt.authCookie)
			}

			// Create response recorder
			w := httptest.NewRecorder()

			// This will fail until we implement the chat handler
			// TODO: Replace with actual handler once implemented
			mockChatGetHandler := func(w http.ResponseWriter, r *http.Request) {
				// This mock will be replaced with real implementation
				t.Errorf("Chat GET handler not implemented yet - this test should fail (TDD RED phase)")
				w.WriteHeader(http.StatusNotFound)
			}

			// Call handler
			mockChatGetHandler(w, req)

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

			// Check content
			if tt.checkContent != nil {
				tt.checkContent(t, w.Body.String())
			}
		})
	}
}

// TestChatGetWithMessages tests chat interface with existing messages
func TestChatGetWithMessages(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/chat/room-123", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Chat GET handler not implemented yet - messages test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	body := w.Body.String()

	// Should display existing messages
	expectedMessageElements := []string{
		`class="message-item"`,
		`class="message-author"`,
		`class="message-content"`,
		`class="message-timestamp"`,
		`data-message-id=`,
	}

	for _, element := range expectedMessageElements {
		if !contains(body, element) {
			t.Errorf("Chat interface missing message element: %s", element)
		}
	}
}

// TestChatGetOnlineUsers tests online users display
func TestChatGetOnlineUsers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/chat/room-123", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Chat GET handler not implemented yet - online users test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	body := w.Body.String()

	// Should display online users sidebar
	onlineUsersElements := []string{
		`class="online-users"`,
		`class="user-list"`,
		`class="user-item"`,
		`class="user-status"`,
		`hx-get="/fragments/online-users/room-123"`,
	}

	for _, element := range onlineUsersElements {
		if !contains(body, element) {
			t.Errorf("Chat interface missing online users element: %s", element)
		}
	}
}

// TestChatGetSecurity tests security requirements
func TestChatGetSecurity(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/chat/room-123", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Chat GET handler not implemented yet - security test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	// Security headers check
	securityHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Cache-Control":          "no-cache, no-store, must-revalidate",
	}

	for header, expectedValue := range securityHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Missing security header %s: expected %s, got %s", header, expectedValue, actualValue)
		}
	}
}

// TestChatGetPerformance tests that chat page loads quickly
func TestChatGetPerformance(t *testing.T) {
	// Performance requirement: < 200ms server response time
	req := httptest.NewRequest(http.MethodGet, "/chat/room-123", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Chat GET handler not implemented yet - performance test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	// Measure response time
	start := time.Now()
	mockHandler(w, req)
	duration := time.Since(start)

	// Check performance requirement
	if duration > 200*time.Millisecond {
		t.Errorf("Chat page took %v, expected < 200ms", duration)
	}
}

// TestChatGetAccessibility tests accessibility requirements
func TestChatGetAccessibility(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/chat/room-123", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Chat GET handler not implemented yet - accessibility test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	body := w.Body.String()

	// Accessibility checks
	accessibilityRequirements := []string{
		`<title>`,              // Page title for screen readers
		`<h1>`,                 // Main heading
		`<label for="content"`, // Message input label
		`aria-label`,           // ARIA labels
		`role="main"`,          // Semantic roles
		`role="complementary"`, // Sidebar role
		`aria-live="polite"`,   // Live region for messages
		`tabindex=`,            // Keyboard navigation
		`aria-describedby=`,    // Form descriptions
	}

	for _, requirement := range accessibilityRequirements {
		if !contains(body, requirement) {
			t.Errorf("Chat page missing accessibility requirement: %s", requirement)
		}
	}
}

// TestChatGetRoomValidation tests room ID validation
func TestChatGetRoomValidation(t *testing.T) {
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
		{
			name:   "Room ID with unicode",
			roomId: "room-🎉",
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range invalidRoomTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/chat/"+tt.roomId, nil)
			req.AddCookie(&http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			})
			w := httptest.NewRecorder()

			// Mock handler - will be replaced
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("Chat GET handler not implemented yet - room validation test should fail")
				w.WriteHeader(http.StatusNotFound)
			}

			mockHandler(w, req)

			if w.Code != tt.status {
				t.Errorf("Expected status %d for invalid room ID %s, got %d", tt.status, tt.roomId, w.Code)
			}

			body := w.Body.String()
			if !contains(body, "Invalid room ID") && tt.status == http.StatusBadRequest {
				t.Error("Should show invalid room ID message")
			}
		})
	}
}

// TestChatGetErrorHandling tests various error scenarios
func TestChatGetErrorHandling(t *testing.T) {
	errorTests := []struct {
		name           string
		authToken      string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "Malformed JWT token",
			authToken:      "malformed.jwt.token",
			expectedStatus: http.StatusFound, // Should redirect to login
			expectedError:  "",
		},
		{
			name:           "Empty JWT token",
			authToken:      "",
			expectedStatus: http.StatusFound, // Should redirect to login
			expectedError:  "",
		},
		{
			name:           "JWT with invalid signature",
			authToken:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid_signature",
			expectedStatus: http.StatusFound, // Should redirect to login
			expectedError:  "",
		},
		{
			name:           "Expired JWT token",
			authToken:      "expired_jwt_token",
			expectedStatus: http.StatusFound, // Should redirect to login
			expectedError:  "",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/chat/room-123", nil)
			if tt.authToken != "" {
				req.AddCookie(&http.Cookie{
					Name:  "auth_token",
					Value: tt.authToken,
				})
			}
			w := httptest.NewRecorder()

			// Mock handler - will be replaced
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("Chat GET handler not implemented yet - error handling test should fail")
				w.WriteHeader(http.StatusNotFound)
			}

			mockHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				body := w.Body.String()
				if !contains(body, tt.expectedError) {
					t.Errorf("Expected error message '%s' not found", tt.expectedError)
				}
			}
		})
	}
}

// TestChatGetHTMXIntegration tests HTMX-specific functionality
func TestChatGetHTMXIntegration(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/chat/room-123", nil)
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Chat GET handler not implemented yet - HTMX integration test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	body := w.Body.String()

	// Should include HTMX script
	if !contains(body, "htmx") {
		t.Error("Chat page should include HTMX library")
	}

	// Should configure SSE connection
	sseElements := []string{
		`hx-sse="connect:/events/room-123"`,
		`hx-trigger="sse:message"`,
		`hx-trigger="sse:user-joined"`,
		`hx-trigger="sse:user-left"`,
	}

	foundSSE := false
	for _, element := range sseElements {
		if contains(body, element) {
			foundSSE = true
			break
		}
	}
	if !foundSSE {
		t.Error("Chat interface should configure SSE connections")
	}

	// Should include message form with HTMX attributes
	formElements := []string{
		`hx-post="/messages"`,
		`hx-target="#message-input"`,
		`hx-swap="outerHTML"`,
		`hx-on::after-request="this.reset()"`,
	}

	for _, element := range formElements {
		if !contains(body, element) {
			t.Errorf("Message form missing HTMX attribute: %s", element)
		}
	}
}

