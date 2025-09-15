package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"duckchat/internal/jsonrpc"
)

// TestImplementedRoomsListContract tests rooms.list with actual implementation
func TestImplementedRoomsListContract(t *testing.T) {
	middleware := setupMiddlewareWithHandlers(t)

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

				// Should have result, not error
				if resp["result"] == nil {
					t.Error("Expected result field")
				}
				if resp["error"] != nil {
					t.Errorf("Expected no error, got %v", resp["error"])
				}

				// Validate result structure
				if result, ok := resp["result"].(map[string]interface{}); ok {
					if _, exists := result["rooms"]; !exists {
						t.Error("Result should have 'rooms' field")
					}
					if _, exists := result["total_count"]; !exists {
						t.Error("Result should have 'total_count' field")
					}

					if rooms, ok := result["rooms"].([]interface{}); ok {
						// Should initially be empty
						if len(rooms) != 0 {
							t.Logf("Note: Found %d existing rooms (expected 0 for clean test)", len(rooms))
						}
					} else {
						t.Error("'rooms' should be an array")
					}

					if totalCount, ok := result["total_count"].(float64); ok {
						if totalCount < 0 {
							t.Error("total_count should not be negative")
						}
					} else {
						t.Error("'total_count' should be a number")
					}
				} else {
					t.Error("Expected result to be an object")
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
				if resp["error"] != nil {
					t.Errorf("Expected no error, got %v", resp["error"])
				}
				if resp["result"] == nil {
					t.Error("Expected result field")
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

// TestImplementedRoomsCreateContract tests rooms.create with actual implementation
func TestImplementedRoomsCreateContract(t *testing.T) {
	middleware := setupMiddlewareWithHandlers(t)

	tests := []struct {
		name           string
		request        string
		expectedStatus int
		validateResp   func(t *testing.T, resp map[string]interface{})
	}{
		{
			name: "valid rooms.create request",
			request: `{
				"jsonrpc": "2.0",
				"method": "rooms.create",
				"params": {
					"title": "Test Room Contract"
				},
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

				// Should have result, not error
				if resp["result"] == nil {
					t.Error("Expected result field")
				}
				if resp["error"] != nil {
					t.Errorf("Expected no error, got %v", resp["error"])
				}

				// Validate result structure
				if result, ok := resp["result"].(map[string]interface{}); ok {
					requiredFields := []string{"id", "title", "created_at", "created_by", "is_active", "participant_count"}
					for _, field := range requiredFields {
						if _, exists := result[field]; !exists {
							t.Errorf("Result missing required field: %s", field)
						}
					}

					if result["title"] != "Test Room Contract" {
						t.Errorf("Expected title 'Test Room Contract', got %v", result["title"])
					}

					if result["is_active"] != true {
						t.Error("New room should be active")
					}

					if result["participant_count"] != float64(1) {
						t.Error("Creator should be auto-participant, count should be 1")
					}

					// Verify room ID format (should be UUID-like)
					if roomID, ok := result["id"].(string); ok {
						if len(roomID) != 36 {
							t.Error("Room ID should be UUID format (36 characters)")
						}
					} else {
						t.Error("Room ID should be a string")
					}
				} else {
					t.Error("Expected result to be an object")
				}
			},
		},
		{
			name: "missing title parameter",
			request: `{
				"jsonrpc": "2.0",
				"method": "rooms.create",
				"params": {},
				"id": 2
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				// Should have error, not result
				if resp["error"] == nil {
					t.Error("Expected error response for missing title")
				}
				if resp["result"] != nil {
					t.Error("Expected no result for error case")
				}

				if errorObj, ok := resp["error"].(map[string]interface{}); ok {
					if code, ok := errorObj["code"].(float64); ok {
						if int(code) != jsonrpc.ValidationFailed {
							t.Errorf("Expected validation error code %d, got %d", jsonrpc.ValidationFailed, int(code))
						}
					} else {
						t.Error("Error should have numeric code")
					}
				} else {
					t.Error("Error should be an object")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/rooms.create", bytes.NewBufferString(tt.request))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			// Verify HTTP status
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
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

// TestImplementedRoomsLeaveContract tests rooms.leave with actual implementation
func TestImplementedRoomsLeaveContract(t *testing.T) {
	middleware := setupMiddlewareWithHandlers(t)

	tests := []struct {
		name           string
		request        string
		expectedStatus int
		validateResp   func(t *testing.T, resp map[string]interface{})
	}{
		{
			name: "missing room_id parameter",
			request: `{
				"jsonrpc": "2.0",
				"method": "rooms.leave",
				"params": {},
				"id": 1
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				if resp["error"] == nil {
					t.Error("Expected error response for missing room_id")
				}
				if errorObj, ok := resp["error"].(map[string]interface{}); ok {
					if code, ok := errorObj["code"].(float64); ok {
						if int(code) != jsonrpc.ValidationFailed {
							t.Errorf("Expected validation error code %d, got %d", jsonrpc.ValidationFailed, int(code))
						}
					}
				}
			},
		},
		{
			name: "non-existent room_id",
			request: `{
				"jsonrpc": "2.0",
				"method": "rooms.leave",
				"params": {
					"room_id": "00000000-0000-0000-0000-000000000000"
				},
				"id": 2
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				if resp["error"] == nil {
					t.Error("Expected error response for non-existent room")
				}
				if errorObj, ok := resp["error"].(map[string]interface{}); ok {
					if code, ok := errorObj["code"].(float64); ok {
						if int(code) != jsonrpc.ResourceNotFound {
							t.Errorf("Expected resource not found error code %d, got %d", jsonrpc.ResourceNotFound, int(code))
						}
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/rooms.leave", bytes.NewBufferString(tt.request))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			tt.validateResp(t, resp)
		})
	}
}