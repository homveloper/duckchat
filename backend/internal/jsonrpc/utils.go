package jsonrpc

import (
	"encoding/json"
	"net/http"
)

// WriteResponse sends a JSON-RPC 2.0 success response with any result type
func WriteResponse(w http.ResponseWriter, result interface{}, id interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := Response{
		JSONRpc: "2.0",
		Result:  result,
		ID:      id,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback error response if encoding fails
		fallbackError := `{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal encoding error"},"id":null}`
		w.Write([]byte(fallbackError))
	}
}

// WriteResponseT sends a JSON-RPC 2.0 success response with typed result
func WriteResponseT[R any](w http.ResponseWriter, result R, id interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := ResponseT[R]{
		JSONRpc: "2.0",
		Result:  result,
		ID:      id,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback error response if encoding fails
		fallbackError := `{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal encoding error"},"id":null}`
		w.Write([]byte(fallbackError))
	}
}

// WriteError sends a JSON-RPC 2.0 error response
func WriteError(w http.ResponseWriter, code int, message string, data interface{}, id interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC always returns HTTP 200

	response := Response{
		JSONRpc: "2.0",
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback error response if encoding fails
		fallbackError := `{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal encoding error"},"id":null}`
		w.Write([]byte(fallbackError))
	}
}

// WriteErrorT sends a JSON-RPC 2.0 error response with typed data
func WriteErrorT[D any](w http.ResponseWriter, code int, message string, data D, id interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC always returns HTTP 200

	response := ResponseT[interface{}]{
		JSONRpc: "2.0",
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		// Fallback error response if encoding fails
		fallbackError := `{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal encoding error"},"id":null}`
		w.Write([]byte(fallbackError))
	}
}


// WriteCustomError sends a JSON-RPC 2.0 error response with custom error details
func WriteCustomError(w http.ResponseWriter, err *Error, id interface{}) {
	if err == nil {
		WriteError(w, InternalError, "Internal error", "Internal JSON-RPC error", id)
		return
	}

	WriteError(w, err.Code, err.Message, err.Data, id)
}

// WriteBatchResponse sends a JSON-RPC 2.0 batch response
func WriteBatchResponse(w http.ResponseWriter, responses []interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(responses); err != nil {
		// Fallback error response if encoding fails
		fallbackError := `[{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal encoding error"},"id":null}]`
		w.Write([]byte(fallbackError))
	}
}

