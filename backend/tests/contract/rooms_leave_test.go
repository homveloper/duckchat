package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"duckchat/internal/jsonrpc"
)

func TestRoomsLeaveContract(t *testing.T) {
	middleware := jsonrpc.NewMiddleware()

	tests := []struct {
		name           string
		request        string
		expectedStatus int
		validateResp   func(t *testing.T, resp map[string]interface{})
	}{
		{
			name: "valid rooms.leave request",
			request: `{
				"jsonrpc": "2.0",
				"method": "rooms.leave",
				"params": {
					"room_id": "room_123"
				},
				"id": 1
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				if resp["jsonrpc"] != "2.0" {
					t.Errorf("Expected jsonrpc 2.0, got %v", resp["jsonrpc"])
				}
				if resp["id"] != float64(1) {
					t.Errorf("Expected id 1, got %v", resp["id"])
				}

				if resp["result"] == nil && resp["error"] == nil {
					t.Error("Response must have either result or error")
				}

				if result, ok := resp["result"].(map[string]interface{}); ok {
					if result["success"] != true {
						t.Error("Expected success to be true")
					}

					if result["room_id"] != "room_123" {
						t.Errorf("Expected room_id 'room_123', got %v", result["room_id"])
					}

					if _, exists := result["participant_count"]; exists {
						if count, ok := result["participant_count"].(float64); ok {
							if count < 0 {
								t.Error("participant_count should not be negative")
							}
						}
					}
				}
			},
		},
		{
			name: "missing room_id parameter",
			request: `{
				"jsonrpc": "2.0",
				"method": "rooms.leave",
				"params": {},
				"id": 2
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
			name: "empty room_id parameter",
			request: `{
				"jsonrpc": "2.0",
				"method": "rooms.leave",
				"params": {
					"room_id": ""
				},
				"id": 3
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				if resp["error"] == nil {
					t.Error("Expected error response for empty room_id")
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
					"room_id": "nonexistent_room"
				},
				"id": 4
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
		{
			name: "user not in room",
			request: `{
				"jsonrpc": "2.0",
				"method": "rooms.leave",
				"params": {
					"room_id": "room_456"
				},
				"id": 5
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				if resp["error"] == nil {
					t.Error("Expected error response for user not in room")
				}
				if errorObj, ok := resp["error"].(map[string]interface{}); ok {
					if code, ok := errorObj["code"].(float64); ok {
						if int(code) != jsonrpc.PermissionDenied {
							t.Errorf("Expected permission denied error code %d, got %d", jsonrpc.PermissionDenied, int(code))
						}
					}
				}
			},
		},
		{
			name: "invalid JSON",
			request: `{
				"jsonrpc": "2.0"
				"method": "rooms.leave"
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
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
			name: "wrong JSON-RPC version",
			request: `{
				"jsonrpc": "1.0",
				"method": "rooms.leave",
				"params": {
					"room_id": "room_123"
				},
				"id": 6
			}`,
			expectedStatus: http.StatusOK,
			validateResp: func(t *testing.T, resp map[string]interface{}) {
				if resp["error"] == nil {
					t.Error("Expected error response for wrong JSON-RPC version")
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

			if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("Expected content type application/json, got %s", contentType)
			}

			var resp map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			tt.validateResp(t, resp)
		})
	}
}

func TestRoomsLeaveMethodNotImplemented(t *testing.T) {
	middleware := jsonrpc.NewMiddleware()

	request := `{
		"jsonrpc": "2.0",
		"method": "rooms.leave",
		"params": {
			"room_id": "test_room"
		},
		"id": 1
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/rooms.leave", bytes.NewBufferString(request))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

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