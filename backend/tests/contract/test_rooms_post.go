package contract

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestRoomsPostEndpoint tests the POST /rooms endpoint contract
func TestRoomsPostEndpoint(t *testing.T) {
	// This test MUST fail initially as the endpoint doesn't exist yet
	// Following TDD - RED phase

	tests := []struct {
		name            string
		authCookie      *http.Cookie
		formData        url.Values
		expectedStatus  int
		expectedHeaders map[string]string
		checkResponse   func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Valid room creation redirects to chat",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"room_name":   []string{"Test Room"},
				"description": []string{"A test room for testing"},
				"is_private":  []string{"false"},
				"csrf_token":  []string{"valid_csrf_token"},
			},
			expectedStatus: http.StatusFound, // 302
			expectedHeaders: map[string]string{
				"Location": "/chat/", // Should contain room ID
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should redirect to the newly created room
				location := w.Header().Get("Location")
				if !strings.HasPrefix(location, "/chat/") {
					t.Error("Should redirect to chat room")
				}
				if len(location) <= 6 { // "/chat/" is 6 characters
					t.Error("Should include room ID in redirect")
				}
			},
		},
		{
			name: "Valid private room creation",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"room_name":   []string{"Private Room"},
				"description": []string{"A private test room"},
				"is_private":  []string{"true"},
				"csrf_token":  []string{"valid_csrf_token"},
			},
			expectedStatus: http.StatusFound,
			expectedHeaders: map[string]string{
				"Location": "/chat/",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				location := w.Header().Get("Location")
				if !strings.HasPrefix(location, "/chat/") {
					t.Error("Should redirect to private chat room")
				}
			},
		},
		{
			name:       "Unauthenticated user gets forbidden",
			authCookie: nil,
			formData: url.Values{
				"room_name":   []string{"Test Room"},
				"description": []string{"A test room"},
				"csrf_token":  []string{"valid_csrf_token"},
			},
			expectedStatus: http.StatusUnauthorized, // 401
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should redirect to login or return 401
			},
		},
		{
			name: "Missing room name returns error",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"room_name":   []string{""},
				"description": []string{"A test room"},
				"csrf_token":  []string{"valid_csrf_token"},
			},
			expectedStatus: http.StatusBadRequest, // 400
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Room name is required") {
					t.Error("Should show room name validation error")
				}
				// Should return rooms form with error state
				if !contains(body, `class="error-message"`) {
					t.Error("Should show error styling")
				}
			},
		},
		{
			name: "Room name too short returns error",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"room_name":   []string{"AB"},
				"description": []string{"A test room"},
				"csrf_token":  []string{"valid_csrf_token"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Room name must be at least 3 characters") {
					t.Error("Should show room name length validation error")
				}
			},
		},
		{
			name: "Room name too long returns error",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"room_name":   []string{strings.Repeat("A", 31)}, // 31 characters
				"description": []string{"A test room"},
				"csrf_token":  []string{"valid_csrf_token"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Room name must be no more than 30 characters") {
					t.Error("Should show room name length validation error")
				}
			},
		},
		{
			name: "Invalid room name characters returns error",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"room_name":   []string{"Test@Room#123"},
				"description": []string{"A test room"},
				"csrf_token":  []string{"valid_csrf_token"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Room name contains invalid characters") {
					t.Error("Should show room name character validation error")
				}
			},
		},
		{
			name: "Description too long returns error",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"room_name":   []string{"Test Room"},
				"description": []string{strings.Repeat("A", 201)}, // 201 characters
				"csrf_token":  []string{"valid_csrf_token"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Description must be no more than 200 characters") {
					t.Error("Should show description length validation error")
				}
			},
		},
		{
			name: "Duplicate room name returns conflict",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"room_name":   []string{"Existing Room"},
				"description": []string{"A duplicate room"},
				"csrf_token":  []string{"valid_csrf_token"},
			},
			expectedStatus: http.StatusConflict, // 409
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Room name already exists") {
					t.Error("Should show room name conflict error")
				}
			},
		},
		{
			name: "Missing CSRF token returns forbidden",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"room_name":   []string{"Test Room"},
				"description": []string{"A test room"},
			},
			expectedStatus: http.StatusForbidden, // 403
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "CSRF token required") {
					t.Error("Should show CSRF error message")
				}
			},
		},
		{
			name: "Invalid CSRF token returns forbidden",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"room_name":   []string{"Test Room"},
				"description": []string{"A test room"},
				"csrf_token":  []string{"invalid_csrf_token"},
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Invalid CSRF token") {
					t.Error("Should show CSRF validation error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create POST request with form data
			req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Add auth cookie if provided
			if tt.authCookie != nil {
				req.AddCookie(tt.authCookie)
			}

			w := httptest.NewRecorder()

			// This will fail until we implement the room creation handler
			// TODO: Replace with actual handler once implemented
			mockRoomsPostHandler := func(w http.ResponseWriter, r *http.Request) {
				// This mock will be replaced with real implementation
				t.Errorf("Rooms POST handler not implemented yet - this test should fail (TDD RED phase)")
				w.WriteHeader(http.StatusNotFound)
			}

			// Call handler
			mockRoomsPostHandler(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check headers
			for key, expectedValue := range tt.expectedHeaders {
				actualValue := w.Header().Get(key)
				if !strings.HasPrefix(actualValue, expectedValue) && actualValue != expectedValue {
					t.Errorf("Expected header %s to start with %s, got: %s", key, expectedValue, actualValue)
				}
			}

			// Check response
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestRoomsPostSecurity tests security requirements
func TestRoomsPostSecurity(t *testing.T) {
	formData := url.Values{
		"room_name":   []string{"Test Room"},
		"description": []string{"A test room"},
		"csrf_token":  []string{"valid_csrf_token"},
	}

	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Rooms POST handler not implemented yet - security test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	// Security checks
	securityHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expectedValue := range securityHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Missing security header %s: expected %s, got %s", header, expectedValue, actualValue)
		}
	}
}

// TestRoomsPostValidation tests comprehensive input validation
func TestRoomsPostValidation(t *testing.T) {
	validationTests := []struct {
		name        string
		roomName    string
		description string
		wantErr     string
	}{
		{
			name:        "XSS attempt in room name",
			roomName:    "<script>alert('xss')</script>",
			description: "Safe description",
			wantErr:     "Room name contains invalid characters",
		},
		{
			name:        "XSS attempt in description",
			roomName:    "Safe Room",
			description: "<script>alert('xss')</script>",
			wantErr:     "Description contains invalid characters",
		},
		{
			name:        "SQL injection in room name",
			roomName:    "'; DROP TABLE rooms; --",
			description: "Safe description",
			wantErr:     "Room name contains invalid characters",
		},
		{
			name:        "Unicode characters in room name",
			roomName:    "Room 😀 Test",
			description: "Safe description",
			wantErr:     "Room name contains invalid characters",
		},
		{
			name:        "Path traversal in room name",
			roomName:    "../../../etc/passwd",
			description: "Safe description",
			wantErr:     "Room name contains invalid characters",
		},
		{
			name:        "Command injection in description",
			roomName:    "Safe Room",
			description: "Description $(rm -rf /)",
			wantErr:     "Description contains invalid characters",
		},
		{
			name:        "HTML injection in description",
			roomName:    "Safe Room",
			description: "<iframe src='javascript:alert(1)'></iframe>",
			wantErr:     "Description contains invalid characters",
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{
				"room_name":   []string{tt.roomName},
				"description": []string{tt.description},
				"csrf_token":  []string{"valid_csrf_token"},
			}

			req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			})
			w := httptest.NewRecorder()

			// Mock handler - will be replaced
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("Rooms POST handler not implemented yet - validation test should fail")
				w.WriteHeader(http.StatusNotFound)
			}

			mockHandler(w, req)

			// Should return validation error
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for validation error, got %d", w.Code)
			}

			body := w.Body.String()
			if !contains(body, tt.wantErr) {
				t.Errorf("Expected error message '%s' not found in response", tt.wantErr)
			}
		})
	}
}

// TestRoomsPostPerformance tests performance requirements
func TestRoomsPostPerformance(t *testing.T) {
	formData := url.Values{
		"room_name":   []string{"Test Room"},
		"description": []string{"A test room"},
		"csrf_token":  []string{"valid_csrf_token"},
	}

	req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Rooms POST handler not implemented yet - performance test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	// Measure response time
	start := time.Now()
	mockHandler(w, req)
	duration := time.Since(start)

	// Performance requirement: < 250ms for room creation
	if duration > 250*time.Millisecond {
		t.Errorf("Room creation took %v, expected < 250ms", duration)
	}
}

// TestRoomsPostConcurrency tests concurrent room creation
func TestRoomsPostConcurrency(t *testing.T) {
	// Test creating multiple rooms concurrently to ensure no race conditions
	const numGoroutines = 10

	results := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			formData := url.Values{
				"room_name":   []string{"Concurrent Room " + string(rune(id+'A'))},
				"description": []string{"Concurrent test room"},
				"csrf_token":  []string{"valid_csrf_token"},
			}

			req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.AddCookie(&http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			})
			w := httptest.NewRecorder()

			// Mock handler - will be replaced
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("Rooms POST handler not implemented yet - concurrency test should fail")
				w.WriteHeader(http.StatusNotFound)
			}

			mockHandler(w, req)
			results <- w.Code
		}(i)
	}

	// Collect results
	statusCodes := make(map[int]int)
	for i := 0; i < numGoroutines; i++ {
		code := <-results
		statusCodes[code]++
	}

	// All requests should either succeed or fail consistently
	if len(statusCodes) > 2 { // Only success/failure expected
		t.Errorf("Inconsistent status codes in concurrent requests: %v", statusCodes)
	}
}

// TestRoomsPostRateLimiting tests rate limiting for room creation
func TestRoomsPostRateLimiting(t *testing.T) {
	// Simulate rapid room creation attempts
	const numRequests = 20
	statusCodes := make([]int, numRequests)

	for i := 0; i < numRequests; i++ {
		formData := url.Values{
			"room_name":   []string{"Rate Limit Room " + string(rune(i+'A'))},
			"description": []string{"Rate limit test"},
			"csrf_token":  []string{"valid_csrf_token"},
		}

		req := httptest.NewRequest(http.MethodPost, "/rooms", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("X-Forwarded-For", "192.168.1.100") // Consistent IP for rate limiting
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: "valid_jwt_token",
		})
		w := httptest.NewRecorder()

		// Mock handler - will be replaced
		mockHandler := func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("Rooms POST handler not implemented yet - rate limiting test should fail")
			w.WriteHeader(http.StatusNotFound)
		}

		mockHandler(w, req)
		statusCodes[i] = w.Code
	}

	// Should eventually hit rate limit (429)
	rateLimited := false
	for _, code := range statusCodes {
		if code == http.StatusTooManyRequests {
			rateLimited = true
			break
		}
	}

	if !rateLimited {
		t.Error("Rate limiting should be applied to rapid room creation")
	}
}
