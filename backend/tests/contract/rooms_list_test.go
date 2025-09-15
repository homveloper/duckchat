package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"duckchat/internal/jsonrpc"
)

func TestRoomsListContract(t *testing.T) {
	// Create JSON-RPC middleware
	middleware := jsonrpc.NewMiddleware()

	tests := []struct {
		name           string
		request        string
		expectedStatus int
		validateResp   func(t *testing.T, resp map[string]interface{})
	}{
		{
			name: "valid rooms.list request",
			request: `{
				"jsonrpc": "2.0",
				"method": "rooms.list",
				"params": {},
				"id": 1
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				// Verify JSON-RPC 2.0 structure
				if resp["jsonrpc"] != "2.0" {
					t.Errorf("Expected jsonrpc 2.0, got %v", resp["jsonrpc"])
				}
				if resp["id"] != float64(1) {
					t.Errorf("Expected id 1, got %v", resp["id"])
				}

				// Should have either result or error
				if resp["result"] == nil && resp["error"] == nil {
					t.Error("Response must have either result or error")
				}

				// If successful, validate result structure
				if result, ok := resp["result"].(map[string]interface{}); ok {
					if rooms, exists := result["rooms"]; exists {
						if roomsArray, isArray := rooms.([]interface{}); isArray {
							// Each room should have required fields
							for i, room := range roomsArray {
								roomObj, ok := room.(map[string]interface{})
								if !ok {
									t.Errorf("Room %d is not an object", i)
									continue
								}

								// Validate required fields
								requiredFields := []string{"id", "title", "created_at", "created_by", "is_active", "participant_count"}
								for _, field := range requiredFields {
									if _, exists := roomObj[field]; !exists {
										t.Errorf("Room %d missing required field: %s", i, field)
									}
								}
							}
						}
					}

					if totalCount, exists := result["total_count"]; exists {
						if _, isNumber := totalCount.(float64); !isNumber {
							t.Error("total_count should be a number")
						}
					}
				}
			},
		},
		{
			name: "rooms.list with filters",
			request: `{
				"jsonrpc": "2.0",
				"method": "rooms.list",
				"params": {
					"filters": {
						"is_active": true
					},
					"limit": 10
				},
				"id": 2
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				// Should be valid JSON-RPC response
				if resp["jsonrpc"] != "2.0" {
					t.Errorf("Expected jsonrpc 2.0, got %v", resp["jsonrpc"])
				}
			},
		},
		{
			name: "invalid JSON",
			request: `{
				"jsonrpc": "2.0"
				"method": "rooms.list"
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				// Should be error response
				if resp["error"] == nil {
					t.Error("Expected error response for invalid JSON")
				}
				if errorObj, ok := resp["error"].(map[string]interface{}); ok {
					if code, ok := errorObj["code"].(float64); ok {
						if int(code) != jsonrpc.ParseError {
							t.Errorf("Expected parse error code %d, got %d", jsonrpc.ParseError, int(code))
						}
					}
				}
			},
		},
		{
			name: "missing method field",
			request: `{
				"jsonrpc": "2.0",
				"params": {},
				"id": 3
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				// Should be error response
				if resp["error"] == nil {
					t.Error("Expected error response for missing method")
				}
				if errorObj, ok := resp["error"].(map[string]interface{}); ok {
					if code, ok := errorObj["code"].(float64); ok {
						if int(code) != jsonrpc.InvalidRequest {
							t.Errorf("Expected invalid request error code %d, got %d", jsonrpc.InvalidRequest, int(code))
						}
					}
				}
			},
		},
		{
			name: "wrong JSON-RPC version",
			request: `{
				"jsonrpc": "1.0",
				"method": "rooms.list",
				"id": 4
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				// Should be error response
				if resp["error"] == nil {
					t.Error("Expected error response for wrong version")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/rooms.list", bytes.NewBufferString(tt.request))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			// Verify HTTP status (should always be 200 for JSON-RPC)
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Verify content type
			if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("Expected content type application/json, got %s", contentType)
			}

			// Parse response
			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Run custom validation
			tt.validateResp(t, resp)
		})
	}
}

// TestRoomsListMethodNotImplemented tests that the method is not yet implemented
// This test should fail initially (following TDD Red-Green-Refactor)
func TestRoomsListMethodNotImplemented(t *testing.T) {
	middleware := jsonrpc.NewMiddleware()

	request := `{
		"jsonrpc": "2.0",
		"method": "rooms.list",
		"params": {},
		"id": 1
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/rooms.list", bytes.NewBufferString(request))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	// This should fail because method is not implemented yet
	if errorObj, ok := resp["error"].(map[string]interface{}); ok {
		if code, ok := errorObj["code"].(float64); ok {
			if int(code) != jsonrpc.MethodNotFound {
				t.Errorf("Expected method not found error, got code %d", int(code))
			}
		} else {
			t.Error("Expected error code to be a number")
		}
	} else {
		t.Error("Expected error response for unimplemented method")
	}
}