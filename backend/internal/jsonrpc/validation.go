package jsonrpc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// JSONRPCValidationMiddleware provides JSON-RPC 2.0 format validation
type JSONRPCValidationMiddleware struct {
	next http.Handler
}

// JSONRPCValidationConfig holds configuration for JSON-RPC validation
type JSONRPCValidationConfig struct {
	// Optional custom error messages
	InvalidMethodMessage     string
	InvalidContentType      string
	InvalidJSONMessage      string
	InvalidVersionMessage   string
	MissingMethodMessage    string
}

// NewJSONRPCValidationMiddleware creates a new JSON-RPC 2.0 validation middleware
func NewJSONRPCValidationMiddleware(next http.Handler) *JSONRPCValidationMiddleware {
	return &JSONRPCValidationMiddleware{
		next: next,
	}
}

// NewJSONRPCValidationMiddlewareWithConfig creates a new JSON-RPC 2.0 validation middleware with custom config
func NewJSONRPCValidationMiddlewareWithConfig(config JSONRPCValidationConfig, next http.Handler) *JSONRPCValidationMiddleware {
	return &JSONRPCValidationMiddleware{
		next: next,
	}
}

// ServeHTTP implements http.Handler interface
func (j *JSONRPCValidationMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set response content type to JSON
	w.Header().Set("Content-Type", "application/json")

	// Validate HTTP method
	if r.Method != http.MethodPost {
		j.writeErrorResponse(w, &Error{
			Code:    InvalidRequest,
			Message: "Only POST method allowed for JSON-RPC",
			Data:    fmt.Sprintf("Received method: %s", r.Method),
		}, nil)
		return
	}

	// Validate Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		j.writeErrorResponse(w, &Error{
			Code:    InvalidRequest,
			Message: "Content-Type must be application/json",
			Data:    fmt.Sprintf("Received Content-Type: %s", contentType),
		}, nil)
		return
	}

	// Read and validate request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		j.writeErrorResponse(w, &Error{
			Code:    ParseError,
			Message: "Failed to read request body",
			Data:    err.Error(),
		}, nil)
		return
	}

	// Close original body and create new one for next handler
	r.Body.Close()
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	// Parse JSON to validate JSON-RPC 2.0 format
	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		j.writeErrorResponse(w, &Error{
			Code:    ParseError,
			Message: "Invalid JSON",
			Data:    err.Error(),
		}, nil)
		return
	}

	// Validate JSON-RPC version
	if req.JSONRpc != "2.0" {
		j.writeErrorResponse(w, &Error{
			Code:    InvalidRequest,
			Message: "Invalid JSON-RPC version, must be '2.0'",
			Data:    fmt.Sprintf("Received version: %s", req.JSONRpc),
		}, req.ID)
		return
	}

	// Validate required method field
	if req.Method == "" {
		j.writeErrorResponse(w, &Error{
			Code:    InvalidRequest,
			Message: "Method field is required",
			Data:    "Missing 'method' field in JSON-RPC request",
		}, req.ID)
		return
	}

	// Validate ID field exists (can be null, string, number, but must be present)
	if !j.hasIDField(body) {
		j.writeErrorResponse(w, &Error{
			Code:    InvalidRequest,
			Message: "ID field is required",
			Data:    "Missing 'id' field in JSON-RPC request",
		}, nil)
		return
	}

	// All validations passed, continue to next handler
	j.next.ServeHTTP(w, r)
}

// hasIDField checks if the JSON contains an 'id' field
func (j *JSONRPCValidationMiddleware) hasIDField(body []byte) bool {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return false
	}
	_, exists := raw["id"]
	return exists
}

// writeErrorResponse writes a JSON-RPC 2.0 error response
func (j *JSONRPCValidationMiddleware) writeErrorResponse(w http.ResponseWriter, err *Error, id interface{}) {
	// Always return HTTP 200 for JSON-RPC as per spec
	w.WriteHeader(http.StatusOK)

	response := Response{
		JSONRpc: "2.0",
		Error:   err,
		ID:      id,
	}

	if jsonErr := json.NewEncoder(w).Encode(response); jsonErr != nil {
		// Last resort: write plain text error
		fmt.Fprintf(w, `{"jsonrpc":"2.0","error":{"code":%d,"message":"Internal encoding error"},"id":null}`, InternalError)
	}
}

// Optional: Batch request validation middleware
type JSONRPCBatchValidationMiddleware struct {
	next http.Handler
}

// NewJSONRPCBatchValidationMiddleware creates middleware that also supports batch requests
func NewJSONRPCBatchValidationMiddleware(next http.Handler) *JSONRPCBatchValidationMiddleware {
	return &JSONRPCBatchValidationMiddleware{
		next: next,
	}
}

// ServeHTTP implements http.Handler interface for batch validation
func (j *JSONRPCBatchValidationMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Validate HTTP method
	if r.Method != http.MethodPost {
		j.writeErrorResponse(w, &Error{
			Code:    InvalidRequest,
			Message: "Only POST method allowed for JSON-RPC",
		}, nil)
		return
	}

	// Validate Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		j.writeErrorResponse(w, &Error{
			Code:    InvalidRequest,
			Message: "Content-Type must be application/json",
		}, nil)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		j.writeErrorResponse(w, &Error{
			Code:    ParseError,
			Message: "Failed to read request body",
		}, nil)
		return
	}

	r.Body.Close()
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	// Try to parse as single request first
	var singleReq Request
	if err := json.Unmarshal(body, &singleReq); err == nil {
		// Single request validation
		if err := j.validateSingleRequest(&singleReq); err != nil {
			j.writeErrorResponse(w, err, singleReq.ID)
			return
		}
	} else {
		// Try to parse as batch request
		var batchReq []Request
		if err := json.Unmarshal(body, &batchReq); err != nil {
			j.writeErrorResponse(w, &Error{
				Code:    ParseError,
				Message: "Invalid JSON",
				Data:    err.Error(),
			}, nil)
			return
		}

		// Validate batch request
		if len(batchReq) == 0 {
			j.writeErrorResponse(w, &Error{
				Code:    InvalidRequest,
				Message: "Batch request cannot be empty",
			}, nil)
			return
		}

		// Validate each request in batch
		for i, req := range batchReq {
			if err := j.validateSingleRequest(&req); err != nil {
				err.Data = fmt.Sprintf("Error in batch request at index %d: %v", i, err.Data)
				j.writeErrorResponse(w, err, req.ID)
				return
			}
		}
	}

	// All validations passed
	j.next.ServeHTTP(w, r)
}

// validateSingleRequest validates a single JSON-RPC request
func (j *JSONRPCBatchValidationMiddleware) validateSingleRequest(req *Request) *Error {
	if req.JSONRpc != "2.0" {
		return &Error{
			Code:    InvalidRequest,
			Message: "Invalid JSON-RPC version, must be '2.0'",
			Data:    fmt.Sprintf("Received version: %s", req.JSONRpc),
		}
	}

	if req.Method == "" {
		return &Error{
			Code:    InvalidRequest,
			Message: "Method field is required",
			Data:    "Missing 'method' field in JSON-RPC request",
		}
	}

	return nil
}

// writeErrorResponse for batch middleware
func (j *JSONRPCBatchValidationMiddleware) writeErrorResponse(w http.ResponseWriter, err *Error, id interface{}) {
	w.WriteHeader(http.StatusOK)

	response := Response{
		JSONRpc: "2.0",
		Error:   err,
		ID:      id,
	}

	if jsonErr := json.NewEncoder(w).Encode(response); jsonErr != nil {
		fmt.Fprintf(w, `{"jsonrpc":"2.0","error":{"code":%d,"message":"Internal encoding error"},"id":null}`, InternalError)
	}
}