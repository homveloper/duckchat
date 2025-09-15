package contract

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// TestLoginPostEndpoint tests the POST /login endpoint contract
func TestLoginPostEndpoint(t *testing.T) {
	// This test MUST fail initially as the endpoint doesn't exist yet
	// Following TDD - RED phase

	tests := []struct {
		name            string
		formData        url.Values
		expectedStatus  int
		expectedHeaders map[string]string
		checkResponse   func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "Valid login credentials redirect to rooms",
			formData: url.Values{
				"username":   []string{"testuser"},
				"password":   []string{"password123"},
				"csrf_token": []string{"valid_token"},
			},
			expectedStatus: http.StatusFound, // 302
			expectedHeaders: map[string]string{
				"Location": "/rooms",
			},
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Should set auth cookie
				cookies := w.Result().Cookies()
				foundAuthCookie := false
				for _, cookie := range cookies {
					if cookie.Name == "auth_token" {
						foundAuthCookie = true
						if !cookie.HttpOnly {
							t.Error("Auth cookie must be HttpOnly")
						}
						if cookie.SameSite != http.SameSiteStrictMode {
							t.Error("Auth cookie must use SameSite=Strict")
						}
					}
				}
				if !foundAuthCookie {
					t.Error("Auth cookie not set on successful login")
				}
			},
		},
		{
			name: "Invalid username returns error",
			formData: url.Values{
				"username":   []string{""},
				"password":   []string{"password123"},
				"csrf_token": []string{"valid_token"},
			},
			expectedStatus: http.StatusBadRequest, // 400
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Username is required") {
					t.Error("Should show username validation error")
				}
				// Should return login form with error state
				if !contains(body, `class="error-message"`) {
					t.Error("Should show error styling")
				}
			},
		},
		{
			name: "Invalid password returns error",
			formData: url.Values{
				"username":   []string{"testuser"},
				"password":   []string{""},
				"csrf_token": []string{"valid_token"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Password is required") {
					t.Error("Should show password validation error")
				}
			},
		},
		{
			name: "Wrong credentials returns error",
			formData: url.Values{
				"username":   []string{"testuser"},
				"password":   []string{"wrongpassword"},
				"csrf_token": []string{"valid_token"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Invalid username or password") {
					t.Error("Should show login failure message")
				}
				// Should not leak which field was wrong
				if contains(body, "username not found") || contains(body, "password incorrect") {
					t.Error("Should not leak specific login failure details")
				}
			},
		},
		{
			name: "Missing CSRF token returns error",
			formData: url.Values{
				"username": []string{"testuser"},
				"password": []string{"password123"},
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
			formData: url.Values{
				"username":   []string{"testuser"},
				"password":   []string{"wrongpassword"},
				"csrf_token": []string{"valid_token"},
			},
			expectedStatus: http.StatusTooManyRequests, // 429
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				body := w.Body.String()
				if !contains(body, "Too many login attempts") {
					t.Error("Should show rate limit message")
				}
				// Should include retry-after information
				retryAfter := w.Header().Get("Retry-After")
				if retryAfter == "" {
					t.Error("Should include Retry-After header")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create POST request with form data
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			// Simulate rate limiting by adding multiple failed attempts
			if tt.expectedStatus == http.StatusTooManyRequests {
				req.Header.Set("X-Forwarded-For", "192.168.1.100") // Mock IP for rate limiting
			}

			w := httptest.NewRecorder()

			// This will fail until we implement the login handler
			// TODO: Replace with actual handler once implemented
			mockLoginPostHandler := func(w http.ResponseWriter, r *http.Request) {
				// This mock will be replaced with real implementation
				t.Errorf("Login POST handler not implemented yet - this test should fail (TDD RED phase)")
				w.WriteHeader(http.StatusNotFound)
			}

			// Call handler
			mockLoginPostHandler(w, req)

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

// TestLoginPostSecurity tests security requirements
func TestLoginPostSecurity(t *testing.T) {
	formData := url.Values{
		"username":   []string{"testuser"},
		"password":   []string{"password123"},
		"csrf_token": []string{"valid_token"},
	}

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Login POST handler not implemented yet - security test should fail")
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

// TestLoginPostValidation tests input validation
func TestLoginPostValidation(t *testing.T) {
	validationTests := []struct {
		name     string
		username string
		password string
		wantErr  string
	}{
		{
			name:     "Username too short",
			username: "ab",
			password: "password123",
			wantErr:  "Username must be at least 3 characters",
		},
		{
			name:     "Username too long",
			username: strings.Repeat("a", 51),
			password: "password123",
			wantErr:  "Username must be no more than 50 characters",
		},
		{
			name:     "Username invalid characters",
			username: "test@user",
			password: "password123",
			wantErr:  "Username can only contain letters, numbers, and underscores",
		},
		{
			name:     "Password too short",
			username: "testuser",
			password: "12345",
			wantErr:  "Password must be at least 6 characters",
		},
		{
			name:     "SQL injection attempt",
			username: "admin'; DROP TABLE users; --",
			password: "password",
			wantErr:  "Username can only contain letters, numbers, and underscores",
		},
		{
			name:     "XSS attempt",
			username: "<script>alert('xss')</script>",
			password: "password",
			wantErr:  "Username can only contain letters, numbers, and underscores",
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			formData := url.Values{
				"username":   []string{tt.username},
				"password":   []string{tt.password},
				"csrf_token": []string{"valid_token"},
			}

			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			// Mock handler - will be replaced
			mockHandler := func(w http.ResponseWriter, r *http.Request) {
				t.Errorf("Login POST handler not implemented yet - validation test should fail")
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

// TestLoginPostPerformance tests performance requirements
func TestLoginPostPerformance(t *testing.T) {
	formData := url.Values{
		"username":   []string{"testuser"},
		"password":   []string{"password123"},
		"csrf_token": []string{"valid_token"},
	}

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	// Mock handler - will be replaced
	mockHandler := func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Login POST handler not implemented yet - performance test should fail")
		w.WriteHeader(http.StatusNotFound)
	}

	// Measure response time
	start := time.Now()
	mockHandler(w, req)
	duration := time.Since(start)

	// Performance requirement: < 200ms for authentication
	if duration > 200*time.Millisecond {
		t.Errorf("Login processing took %v, expected < 200ms", duration)
	}
}
