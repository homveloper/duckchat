package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// AuthToken represents a JWT authentication token
type AuthToken struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Claims represents JWT claims structure
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// TokenPair represents an access token pair
type TokenPair struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	UserID      string `json:"user_id"`
}

// TokenClaims represents validated token claims
type TokenClaims struct {
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
}

// AuthProvider represents different authentication providers
type AuthProvider string

const (
	ProviderGuest  AuthProvider = "guest"
	ProviderGoogle AuthProvider = "google"
	ProviderApple  AuthProvider = "apple"
)

// UserAccount represents a user account that can have multiple auth providers
type UserAccount struct {
	ID          string             `json:"id"`
	DisplayName string             `json:"display_name"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
	AuthLinks   []AuthLink         `json:"auth_links"`
}

// AuthLink represents a connection between a user account and an auth provider
type AuthLink struct {
	ID           string       `json:"id"`
	UserID       string       `json:"user_id"`
	Provider     AuthProvider `json:"provider"`
	ProviderID   string       `json:"provider_id"`   // Provider-specific user ID
	DisplayName  string       `json:"display_name"`  // Provider-specific display name
	Email        *string      `json:"email,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	IsActive     bool         `json:"is_active"`
	LastUsedAt   *time.Time   `json:"last_used_at"`
}

// LoginRequest represents a login request with multiple auth methods
type LoginRequest struct {
	Provider    AuthProvider `json:"provider"`
	Credentials interface{}  `json:"credentials"`
}

// GuestLoginCredentials for guest authentication
type GuestLoginCredentials struct {
	Username string `json:"username"`
}

// GoogleLoginCredentials for Google OAuth
type GoogleLoginCredentials struct {
	IDToken string `json:"id_token"`
}

// AppleLoginCredentials for Apple OAuth
type AppleLoginCredentials struct {
	IDToken string `json:"id_token"`
}

// LinkAuthRequest for linking additional auth providers to existing account
type LinkAuthRequest struct {
	Provider    AuthProvider `json:"provider"`
	Credentials interface{}  `json:"credentials"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token        string       `json:"token"`
	UserID       string       `json:"user_id"`
	Username     string       `json:"username"`
	Provider     AuthProvider `json:"provider"`
	IsNewAccount bool         `json:"is_new_account"`
}

// Domain errors
var (
	ErrInvalidUsername     = errors.New("username must be 1-50 characters")
	ErrTokenExpired        = errors.New("token has expired")
	ErrInvalidToken        = errors.New("invalid token")
	ErrMissingToken        = errors.New("missing authentication token")
	ErrInvalidClaims       = errors.New("invalid token claims")
	ErrTokenRevoked        = errors.New("token has been revoked")
	ErrInvalidProvider     = errors.New("invalid authentication provider")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrProviderNotFound    = errors.New("authentication provider not found")
	ErrAuthLinkExists      = errors.New("authentication link already exists")
	ErrUserNotFound        = errors.New("user not found")
)

// Constants
const (
	TokenExpirationTime = 24 * time.Hour
	JWTSecretKey       = "duckchat-secret-key-change-in-production" // TODO: Move to config
)

// ValidateUsername validates a username according to business rules
func ValidateUsername(username string) error {
	if len(username) < 1 || len(username) > 50 {
		return ErrInvalidUsername
	}
	return nil
}

// ValidateProvider validates authentication provider
func ValidateProvider(provider AuthProvider) error {
	switch provider {
	case ProviderGuest, ProviderGoogle, ProviderApple:
		return nil
	default:
		return ErrInvalidProvider
	}
}

// ValidateLoginRequest validates a login request based on provider
func ValidateLoginRequest(req *LoginRequest) error {
	if err := ValidateProvider(req.Provider); err != nil {
		return err
	}

	switch req.Provider {
	case ProviderGuest:
		if creds, ok := req.Credentials.(GuestLoginCredentials); ok {
			return ValidateUsername(creds.Username)
		}
		if credsMap, ok := req.Credentials.(map[string]interface{}); ok {
			if username, exists := credsMap["username"]; exists {
				if usernameStr, ok := username.(string); ok {
					return ValidateUsername(usernameStr)
				}
			}
		}
		return ErrInvalidCredentials

	case ProviderGoogle:
		if creds, ok := req.Credentials.(GoogleLoginCredentials); ok {
			if creds.IDToken == "" {
				return errors.New("Google ID token is required")
			}
			return nil
		}
		if credsMap, ok := req.Credentials.(map[string]interface{}); ok {
			if idToken, exists := credsMap["id_token"]; exists {
				if idTokenStr, ok := idToken.(string); ok && idTokenStr != "" {
					return nil
				}
			}
		}
		return ErrInvalidCredentials

	case ProviderApple:
		if creds, ok := req.Credentials.(AppleLoginCredentials); ok {
			if creds.IDToken == "" {
				return errors.New("Apple ID token is required")
			}
			return nil
		}
		if credsMap, ok := req.Credentials.(map[string]interface{}); ok {
			if idToken, exists := credsMap["id_token"]; exists {
				if idTokenStr, ok := idToken.(string); ok && idTokenStr != "" {
					return nil
				}
			}
		}
		return ErrInvalidCredentials

	default:
		return ErrInvalidProvider
	}
}

// NewAuthToken creates a new authentication token
func NewAuthToken(username string) (*AuthToken, error) {
	if err := ValidateUsername(username); err != nil {
		return nil, err
	}

	userID := uuid.New().String()
	expiresAt := time.Now().Add(TokenExpirationTime)

	// Create JWT claims
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
			Issuer:    "duckchat",
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(JWTSecretKey))
	if err != nil {
		return nil, err
	}

	return &AuthToken{
		Token:     tokenString,
		UserID:    userID,
		Username:  username,
		ExpiresAt: expiresAt,
	}, nil
}

// ValidateToken validates a JWT token and returns claims
func ValidateToken(tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, ErrMissingToken
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(JWTSecretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrInvalidClaims
}

// IsExpired checks if the token is expired
func (t *AuthToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// TimeToExpiry returns time until token expires
func (t *AuthToken) TimeToExpiry() time.Duration {
	return time.Until(t.ExpiresAt)
}

// NewGuestLoginRequest creates a guest login request with validation
func NewGuestLoginRequest(username string) (*LoginRequest, error) {
	if err := ValidateUsername(username); err != nil {
		return nil, err
	}

	return &LoginRequest{
		Provider: ProviderGuest,
		Credentials: GuestLoginCredentials{
			Username: username,
		},
	}, nil
}

// NewGoogleLoginRequest creates a Google OAuth login request
func NewGoogleLoginRequest(idToken string) (*LoginRequest, error) {
	if idToken == "" {
		return nil, errors.New("Google ID token is required")
	}

	return &LoginRequest{
		Provider: ProviderGoogle,
		Credentials: GoogleLoginCredentials{
			IDToken: idToken,
		},
	}, nil
}

// NewAppleLoginRequest creates an Apple OAuth login request
func NewAppleLoginRequest(idToken string) (*LoginRequest, error) {
	if idToken == "" {
		return nil, errors.New("Apple ID token is required")
	}

	return &LoginRequest{
		Provider: ProviderApple,
		Credentials: AppleLoginCredentials{
			IDToken: idToken,
		},
	}, nil
}

// ToResponse converts AuthToken to LoginResponse
func (t *AuthToken) ToResponse() *LoginResponse {
	return &LoginResponse{
		Token:    t.Token,
		UserID:   t.UserID,
		Username: t.Username,
	}
}

// Authentication context for requests
type Context struct {
	UserID   string
	Username string
	Claims   *Claims
}

// NewContext creates a new authentication context
func NewContext(claims *Claims) *Context {
	return &Context{
		UserID:   claims.UserID,
		Username: claims.Username,
		Claims:   claims,
	}
}

// IsAuthenticated checks if context has valid authentication
func (c *Context) IsAuthenticated() bool {
	return c != nil && c.UserID != "" && c.Claims != nil
}