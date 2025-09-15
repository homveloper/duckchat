package jsonrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Request represents a JSON-RPC 2.0 request
type Request struct {
	JSONRpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id"`
}

// Response represents a JSON-RPC 2.0 response
type Response struct {
	JSONRpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Error represents a JSON-RPC 2.0 error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Generic typed versions
// RequestT represents a typed JSON-RPC 2.0 request
type RequestT[P any] struct {
	JSONRpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  P           `json:"params,omitempty"`
	ID      interface{} `json:"id"`
}

// ResponseT represents a typed JSON-RPC 2.0 response
type ResponseT[R any] struct {
	JSONRpc string      `json:"jsonrpc"`
	Result  R           `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// ErrorT represents a typed JSON-RPC 2.0 error with structured data
type ErrorT[D any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    D      `json:"data,omitempty"`
}

// Standard JSON-RPC 2.0 error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603

	// Custom application error codes
	AuthenticationRequired = -32001
	ValidationFailed       = -32002
	ResourceNotFound       = -32003
	ResourceConflict       = -32004
	PermissionDenied       = -32005
	RateLimitExceeded      = -32006
)

// MethodHandler defines the signature for JSON-RPC method handlers
type MethodHandler func(params interface{}) (interface{}, *Error)

// Middleware handles JSON-RPC 2.0 protocol
type Middleware struct {
	methods map[string]MethodHandler
}

// NewMiddleware creates a new JSON-RPC 2.0 middleware
func NewMiddleware() *Middleware {
	return &Middleware{
		methods: make(map[string]MethodHandler),
	}
}

// RegisterMethod registers a method handler
func (m *Middleware) RegisterMethod(method string, handler MethodHandler) {
	m.methods[method] = handler
}

// ServeHTTP implements http.Handler interface
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Always return HTTP 200 as per JSON-RPC 2.0 spec
	w.WriteHeader(http.StatusOK)

	// Parse request
	req, parseErr := m.parseRequest(r)
	if parseErr != nil {
		m.writeErrorResponse(w, parseErr, nil)
		return
	}

	// Validate JSON-RPC version
	if req.JSONRpc != "2.0" {
		m.writeErrorResponse(w, &Error{
			Code:    InvalidRequest,
			Message: "Invalid JSON-RPC version, must be '2.0'",
		}, req.ID)
		return
	}

	// Find method handler
	handler, exists := m.methods[req.Method]
	if !exists {
		m.writeErrorResponse(w, &Error{
			Code:    MethodNotFound,
			Message: fmt.Sprintf("Method '%s' not found", req.Method),
		}, req.ID)
		return
	}

	// Execute handler
	result, err := handler(req.Params)
	if err != nil {
		m.writeErrorResponse(w, err, req.ID)
		return
	}

	// Write success response
	response := Response{
		JSONRpc: "2.0",
		Result:  result,
		ID:      req.ID,
	}

	if jsonErr := json.NewEncoder(w).Encode(response); jsonErr != nil {
		// This shouldn't happen, but if it does, log it
		fmt.Printf("Failed to encode JSON-RPC response: %v\n", jsonErr)
	}
}

// parseRequest parses HTTP request body into JSON-RPC request
func (m *Middleware) parseRequest(r *http.Request) (*Request, *Error) {
	if r.Method != http.MethodPost {
		return nil, &Error{
			Code:    InvalidRequest,
			Message: "Only POST method allowed for JSON-RPC",
		}
	}

	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		return nil, &Error{
			Code:    InvalidRequest,
			Message: "Content-Type must be application/json",
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, &Error{
			Code:    ParseError,
			Message: "Failed to read request body",
			Data:    err.Error(),
		}
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, &Error{
			Code:    ParseError,
			Message: "Invalid JSON",
			Data:    err.Error(),
		}
	}

	// Validate required fields
	if req.Method == "" {
		return nil, &Error{
			Code:    InvalidRequest,
			Message: "Method field is required",
		}
	}

	return &req, nil
}

// writeErrorResponse writes a JSON-RPC error response
func (m *Middleware) writeErrorResponse(w http.ResponseWriter, err *Error, id interface{}) {
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

// NewError creates a new JSON-RPC error
func NewError(code int, message string, data interface{}) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// ValidationError creates a validation error with field details
func ValidationError(field, message string) *Error {
	return &Error{
		Code:    ValidationFailed,
		Message: "Validation failed",
		Data: map[string]interface{}{
			"error_type": "VALIDATION_ERROR",
			"field":      field,
			"details":    message,
		},
	}
}

// AuthError creates an authentication error
func AuthError() *Error {
	return &Error{
		Code:    AuthenticationRequired,
		Message: "Authentication required",
		Data: map[string]interface{}{
			"error_type": "AUTHENTICATION_ERROR",
		},
	}
}

// NotFoundError creates a resource not found error
func NotFoundError(resource string) *Error {
	return &Error{
		Code:    ResourceNotFound,
		Message: fmt.Sprintf("%s not found", resource),
		Data: map[string]interface{}{
			"error_type": "RESOURCE_NOT_FOUND",
			"resource":   resource,
		},
	}
}

// ConflictError creates a resource conflict error
func ConflictError(message string) *Error {
	return &Error{
		Code:    ResourceConflict,
		Message: message,
		Data: map[string]interface{}{
			"error_type": "RESOURCE_CONFLICT",
		},
	}
}

// GetParamsAs unmarshals JSON-RPC params into a struct
func GetParamsAs(params interface{}, target interface{}) *Error {
	if params == nil {
		return nil
	}

	// Convert params to JSON and back to properly unmarshal
	jsonBytes, err := json.Marshal(params)
	if err != nil {
		return &Error{
			Code:    InvalidParams,
			Message: "Failed to process parameters",
			Data:    err.Error(),
		}
	}

	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return &Error{
			Code:    InvalidParams,
			Message: "Invalid parameter format",
			Data:    err.Error(),
		}
	}

	// Validate struct fields if it implements Validator interface
	if validator, ok := target.(interface{ Validate() error }); ok {
		if validationErr := validator.Validate(); validationErr != nil {
			return ValidationError("params", validationErr.Error())
		}
	}

	return nil
}

// ValidateStruct validates struct fields using reflection
func ValidateStruct(s interface{}) error {
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %T", s)
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Check required fields
		if tag := fieldType.Tag.Get("required"); tag == "true" {
			if field.Kind() == reflect.String && field.String() == "" {
				return fmt.Errorf("field '%s' is required", fieldType.Name)
			}
			if field.Kind() == reflect.Ptr && field.IsNil() {
				return fmt.Errorf("field '%s' is required", fieldType.Name)
			}
		}
	}

	return nil
}

// Now returns the current time (helper function for handlers)
func Now() time.Time {
	return time.Now().UTC()
}

// Generic constructors
// NewRequestT creates a new typed JSON-RPC request
func NewRequestT[P any](method string, params P, id interface{}) *RequestT[P] {
	return &RequestT[P]{
		JSONRpc: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}
}

// NewResponseT creates a new typed JSON-RPC response
func NewResponseT[R any](result R, id interface{}) *ResponseT[R] {
	return &ResponseT[R]{
		JSONRpc: "2.0",
		Result:  result,
		ID:      id,
	}
}

// NewErrorResponseT creates a new typed JSON-RPC error response
func NewErrorResponseT[R any](err *Error, id interface{}) *ResponseT[R] {
	var zero R
	return &ResponseT[R]{
		JSONRpc: "2.0",
		Result:  zero,
		Error:   err,
		ID:      id,
	}
}

// NewErrorT creates a new typed JSON-RPC error
func NewErrorT[D any](code int, message string, data D) *ErrorT[D] {
	return &ErrorT[D]{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// ClaimsParser is a callback function that parses JWT claims and returns context values
type ClaimsParser func(claims jwt.Claims) context.Context

// JWTAuthMiddleware provides JWT authentication middleware functionality
type JWTAuthMiddleware struct {
	secretKey     string
	claimsParser  ClaimsParser
	newClaimsFunc func() jwt.Claims
	optional      bool
	next          http.Handler
}

// JWTAuthConfig holds configuration for JWT authentication
type JWTAuthConfig struct {
	SecretKey     string
	ClaimsParser  ClaimsParser
	NewClaimsFunc func() jwt.Claims
	Optional      bool
}

// NewJWTAuthMiddleware creates a new JWT authentication middleware
func NewJWTAuthMiddleware(config JWTAuthConfig, next http.Handler) *JWTAuthMiddleware {
	return &JWTAuthMiddleware{
		secretKey:     config.SecretKey,
		claimsParser:  config.ClaimsParser,
		newClaimsFunc: config.NewClaimsFunc,
		optional:      config.Optional,
		next:          next,
	}
}

// ServeHTTP implements http.Handler interface
func (j *JWTAuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")

	// Handle missing auth header
	if authHeader == "" {
		if j.optional {
			j.next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Authorization header required", http.StatusUnauthorized)
		return
	}

	// Check for Bearer token format
	tokenParts := strings.SplitN(authHeader, " ", 2)
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		if j.optional {
			j.next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
		return
	}

	tokenString := tokenParts[1]

	// Create new claims instance using provided factory function
	claims := j.newClaimsFunc()

	// Parse and validate JWT token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		if j.optional {
			j.next.ServeHTTP(w, r)
			return
		}
		http.Error(w, fmt.Sprintf("Invalid token: %v", err), http.StatusUnauthorized)
		return
	}

	// Validate token
	if !token.Valid {
		if j.optional {
			j.next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Use callback to parse claims and get context
	ctx := j.claimsParser(token.Claims)
	if ctx == nil {
		if j.optional {
			j.next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Failed to parse token claims", http.StatusUnauthorized)
		return
	}

	// Add token to context
	ctx = context.WithValue(ctx, "token", tokenString)
	ctx = context.WithValue(ctx, "claims", token.Claims)

	// Merge with request context
	finalCtx := r.Context()
	if ctx != context.Background() {
		// Copy all values from parsed context to request context
		finalCtx = mergeContexts(finalCtx, ctx)
	}

	// Call next handler with updated context
	j.next.ServeHTTP(w, r.WithContext(finalCtx))
}

// mergeContexts merges source context values into target context
func mergeContexts(target, source context.Context) context.Context {
	// This is a simplified version - in practice, you might want to implement
	// a more sophisticated context merging strategy
	result := target

	// Try to extract known values from source and add to result
	if userID := source.Value("user_id"); userID != nil {
		result = context.WithValue(result, "user_id", userID)
	}
	if username := source.Value("username"); username != nil {
		result = context.WithValue(result, "username", username)
	}
	if token := source.Value("token"); token != nil {
		result = context.WithValue(result, "token", token)
	}
	if claims := source.Value("claims"); claims != nil {
		result = context.WithValue(result, "claims", claims)
	}

	return result
}

// Helper function to create a standard claims parser for common use cases
func StandardClaimsParser(userIDField, usernameField string) ClaimsParser {
	return func(claims jwt.Claims) context.Context {
		mapClaims, ok := claims.(jwt.MapClaims)
		if !ok {
			return nil
		}

		ctx := context.Background()

		if userID, exists := mapClaims[userIDField]; exists {
			if userIDStr, ok := userID.(string); ok {
				ctx = context.WithValue(ctx, "user_id", userIDStr)
			}
		}

		if username, exists := mapClaims[usernameField]; exists {
			if usernameStr, ok := username.(string); ok {
				ctx = context.WithValue(ctx, "username", usernameStr)
			}
		}

		return ctx
	}
}