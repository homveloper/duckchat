package contract

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestMessagesPostEndpoint tests the POST /messages HTMX endpoint contract
func TestMessagesPostEndpoint(t *testing.T) {
	// This test MUST fail initially as the endpoint doesn't exist yet
	// Following TDD - RED phase

	tests := []struct {
		name            string
		authCookie      *http.Cookie
		formData        url.Values
		headers         map[string]string
		expectedStatus  int
		expectedHeaders map[string]string
		checkResponse   func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Valid message sent successfully",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"content":    []string{"Hello, world!"},
				"room_id":    []string{"room-123"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectedStatus: http.StatusOK,
			expectedHeaders: map[string]string{
				"Content-Type": "text/html; charset=utf-8",
				"HX-Trigger":   "message-sent",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				// Should return empty response or success indicator for HTMX
				if len(body) > 100 { // Minimal response expected
					t.Error("HTMX message response should be minimal")
				}
			},
		},
		{
			name: "Valid message with emoji and special characters",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"content":    []string{"Hello! 👋 How are you? 😊"},
				"room_id":    []string{"room-123"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should accept unicode characters
			},
		},
		{
			name:       "Unauthenticated user gets unauthorized",
			authCookie: nil,
			formData: url.Values{
				"content":    []string{"Hello, world!"},
				"room_id":    []string{"room-123"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectedStatus: http.StatusUnauthorized, // 401
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Authentication required") {
					t.Error("Should show authentication error for HTMX")
				}
			},
		},
		{
			name: "Empty message content returns error",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"content":    []string{""},
				"room_id":    []string{"room-123"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectedStatus: http.StatusBadRequest, // 400
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Message content is required") {
					t.Error("Should show content validation error")
				}
				// Should return error fragment for HTMX
				if !contains(body, `class="error-message"`) {
					t.Error("Should return error fragment")
				}
			},
		},
		{
			name: "Message too long returns error",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"content":    []string{strings.Repeat("A", 1001)}, // 1001 characters
				"room_id":    []string{"room-123"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Message too long") {
					t.Error("Should show message length validation error")
				}
			},
		},
		{
			name: "Missing room ID returns error",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"content":    []string{"Hello, world!"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Room ID is required") {
					t.Error("Should show room ID validation error")
				}
			},
		},
		{
			name: "Invalid room ID returns error",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"content":    []string{"Hello, world!"},
				"room_id":    []string{"invalid@room#id"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Invalid room ID") {
					t.Error("Should show invalid room ID error")
				}
			},
		},
		{
			name: "Non-existent room returns not found",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"content":    []string{"Hello, world!"},
				"room_id":    []string{"nonexistent-room"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectedStatus: http.StatusNotFound, // 404
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Room not found") {
					t.Error("Should show room not found error")
				}
			},
		},
		{
			name: "Access denied to private room",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "non_member_jwt_token",
			},
			formData: url.Values{
				"content":    []string{"Hello, world!"},
				"room_id":    []string{"private-room-789"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers: map[string]string{
				"HX-Request": "true",
			},
			expectedStatus: http.StatusForbidden, // 403
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Access denied") {
					t.Error("Should show access denied error")
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
				"content": []string{"Hello, world!"},
				"room_id": []string{"room-123"},
			},
			headers: map[string]string{
				"HX-Request": "true",
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
			name: "Rate limit exceeded",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"content":    []string{"Spam message"},
				"room_id":    []string{"room-123"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers: map[string]string{
				"HX-Request":      "true",
				"X-Forwarded-For": "192.168.1.100", // For rate limiting
			},
			expectedStatus: http.StatusTooManyRequests, // 429
			expectedHeaders: map[string]string{
				"Retry-After": "60",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Rate limit exceeded") {
					t.Error("Should show rate limit error")
				}
				// Should return rate limit fragment for HTMX
				if !contains(body, `class="rate-limit-error"`) {
					t.Error("Should return rate limit error fragment")
				}
			},
		},
		{
			name: "Non-HTMX request should still work",
			authCookie: &http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			},
			formData: url.Values{
				"content":    []string{"Hello from regular form!"},
				"room_id":    []string{"room-123"},
				"csrf_token": []string{"valid_csrf_token"},
			},
			headers:        map[string]string{}, // No HX-Request header
			expectedStatus: http.StatusFound,    // Should redirect back to chat
			expectedHeaders: map[string]string{
				"Location": "/chat/room-123",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Regular form submission should redirect
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create POST request with form data
			req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Add custom headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Add auth cookie if provided
			if tt.authCookie != nil {
				req.AddCookie(tt.authCookie)
			}

			w := httptest.NewRecorder()

			// This will fail until we implement the messages handler
			// TODO: Replace with actual handler once implemented
			mockMessagesPostHandler := func(w http.ResponseWriter, r *http.Request) {
				// This mock will be replaced with real implementation
				t.Errorf("Messages POST handler not implemented yet - this test should fail (TDD RED phase)")
				w.WriteHeader(http.StatusNotFound)
			}

			// Call handler
			mockMessagesPostHandler(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check headers
			for key, expectedValue := range tt.expectedHeaders {
				actualValue := w.Header().Get(key)
				if !strings.Contains(actualValue, expectedValue) && actualValue != expectedValue {
					t.Errorf("Expected header %s to contain %s, got: %s", key, expectedValue, actualValue)
				}
			}

			// Check response
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}

// TestMessagesPostSecurity tests security requirements
func TestMessagesPostSecurity(t *testing.T) {
	formData := url.Values{
		"content":    []string{"Test message"},
		"room_id":    []string{"room-123"},
		"csrf_token": []string{"valid_csrf_token"},
	}

	req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Messages POST handler not implemented yet - security test should fail")
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

// TestMessagesPostValidation tests comprehensive input validation
func TestMessagesPostValidation(t *testing.T) {
	validationTests := []struct {
		name    string
		content string
		wantErr string
		status  int
	}{
		{
			name:    "XSS attempt in content",
			content: "<script>alert('xss')</script>",
			wantErr: "Message contains potentially dangerous content",
			status:  http.StatusBadRequest,
		},
		{
			name:    "HTML injection attempt",
			content: "<iframe src='javascript:alert(1)'></iframe>",
			wantErr: "Message contains potentially dangerous content",
			status:  http.StatusBadRequest,
		},
		{
			name:    "SQL injection attempt",
			content: "'; DROP TABLE messages; --",
			wantErr: "Message contains potentially dangerous content",
			status:  http.StatusBadRequest,
		},
		{
			name:    "Command injection attempt",
			content: "$(rm -rf /)",
			wantErr: "Message contains potentially dangerous content",
			status:  http.StatusBadRequest,
		},
		{
			name:    "Path traversal attempt",
			content: "../../../etc/passwd",
			wantErr: "Message contains potentially dangerous content",
			status:  http.StatusBadRequest,
		},
		{
			name:    "Null byte injection",
			content: "Message\x00injection",
			wantErr: "Message contains invalid characters",
			status:  http.StatusBadRequest,
		},
		{
			name:    "Control character injection",
			content: "Message\x1b[31mRED\x1b[0m",
			wantErr: "Message contains invalid characters",
			status:  http.StatusBadRequest,
		},
		{
			name:    "Very long single word (potential DoS)",
			content: strings.Repeat("a", 500),
			wantErr: "", // Should be allowed but watched
			status:  http.StatusOK,
		},
		{
			name:    "Repeated characters (spam detection)",
			content: strings.Repeat("AAAA ", 200),
			wantErr: "Message appears to be spam",
			status:  http.StatusBadRequest,
		},
		{
			name:    "Valid markdown should be allowed",
			content: "**bold** and *italic* text with `code`",
			wantErr: "",
			status:  http.StatusOK,
		},
		{
			name:    "Valid URLs should be allowed",
			content: "Check out https://example.com for more info",
			wantErr: "",
			status:  http.StatusOK,
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{
				"content":    []string{tt.content},
				"room_id":    []string{"room-123"},
				"csrf_token": []string{"valid_csrf_token"},
			}

			req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			req.AddCookie(&http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			})
			w := httptest.NewRecorder()

			// Mock handler - will be replaced
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("Messages POST handler not implemented yet - validation test should fail")
				w.WriteHeader(http.StatusNotFound)
			}

			mockHandler(w, req)

			// Should return expected status
			if w.Code != tt.status && tt.status != 0 {
				t.Errorf("Expected status %d, got %d", tt.status, w.Code)
			}

			body := w.Body.String()
			if tt.wantErr != "" && !contains(body, tt.wantErr) {
				t.Errorf("Expected error message '%s' not found in response", tt.wantErr)
			}
		})
	}
}

// TestMessagesPostPerformance tests performance requirements
func TestMessagesPostPerformance(t *testing.T) {
	formData := url.Values{
		"content":    []string{"Test message for performance"},
		"room_id":    []string{"room-123"},
		"csrf_token": []string{"valid_csrf_token"},
	}

	req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Messages POST handler not implemented yet - performance test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	// Measure response time
	start := time.Now()
	mockHandler(w, req)
	duration := time.Since(start)

	// Performance requirement: < 50ms for message sending
	if duration > 50*time.Millisecond {
		t.Errorf("Message sending took %v, expected < 50ms", duration)
	}
}

// TestMessagesPostRateLimiting tests rate limiting for messages
func TestMessagesPostRateLimiting(t *testing.T) {
	// Simulate rapid message sending
	const numRequests = 50
	statusCodes := make([]int, numRequests)

	for i := 0; i < numRequests; i++ {
		formData := url.Values{
			"content":    []string{"Message " + string(rune(i+'A'))},
			"room_id":    []string{"room-123"},
			"csrf_token": []string{"valid_csrf_token"},
		}

		req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("HX-Request", "true")
		req.Header.Set("X-Forwarded-For", "192.168.1.100") // Consistent IP for rate limiting
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: "valid_jwt_token",
		})
		w := httptest.NewRecorder()

		// Mock handler - will be replaced
		mockHandler := func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("Messages POST handler not implemented yet - rate limiting test should fail")
			w.WriteHeader(http.StatusNotFound)
		}

		mockHandler(w, req)
		statusCodes[i] = w.Code

		// Small delay to simulate real usage
		time.Sleep(10 * time.Millisecond)
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
		t.Error("Rate limiting should be applied to rapid message sending")
	}
}

// TestMessagesPostHTMXIntegration tests HTMX-specific functionality
func TestMessagesPostHTMXIntegration(t *testing.T) {
	formData := url.Values{
		"content":    []string{"Test HTMX message"},
		"room_id":    []string{"room-123"},
		"csrf_token": []string{"valid_csrf_token"},
	}

	req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "#message-input")
	req.Header.Set("HX-Trigger", "submit")
	req.AddCookie(&http.Cookie{
		Name:  "auth_token",
		Value: "valid_jwt_token",
	})
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Messages POST handler not implemented yet - HTMX integration test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	mockHandler(w, req)

	// Check HTMX-specific response headers
	htmxHeaders := []string{
		"HX-Trigger",
		"HX-Refresh", // May be used to refresh parts of the page
	}

	foundHTMXHeader := false
	for _, header := range htmxHeaders {
		if w.Header().Get(header) != "" {
			foundHTMXHeader = true
			break
		}
	}

	if !foundHTMXHeader {
		t.Error("HTMX request should include HTMX-specific response headers")
	}
}

// TestMessagesPostConcurrency tests concurrent message sending
func TestMessagesPostConcurrency(t *testing.T) {
	// Test sending multiple messages concurrently to ensure no race conditions
	const numGoroutines = 20

	results := make(chan int, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			formData := url.Values{
				"content":    []string{"Concurrent message " + string(rune(id+'A'))},
				"room_id":    []string{"room-123"},
				"csrf_token": []string{"valid_csrf_token"},
			}

			req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.Header.Set("HX-Request", "true")
			req.AddCookie(&http.Cookie{
				Name:  "auth_token",
				Value: "valid_jwt_token",
			})
			w := httptest.NewRecorder()

			// Mock handler - will be replaced
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("Messages POST handler not implemented yet - concurrency test should fail")
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
	if len(statusCodes) > 3 { // Success, rate limit, validation errors expected
		t.Errorf("Too many different status codes in concurrent requests: %v", statusCodes)
	}
}

