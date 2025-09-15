package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Service handles authentication business logic
type Service struct {
	repo      Repository
	jwtSecret []byte
	tokenTTL  time.Duration
}

// NewService creates a new authentication service
func NewService(repo Repository, jwtSecret string) *Service {
	return &Service{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		tokenTTL:  24 * time.Hour, // Default 24 hours
	}
}

// GenerateToken creates a JWT token for a user
func (s *Service) GenerateToken(ctx context.Context, userID string) (*TokenPair, error) {
	// Create token claims
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(s.tokenTTL).Unix(),
		"iat":     time.Now().Unix(),
		"iss":     "duckchat",
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	// Store token in repository
	err = s.repo.StoreToken(ctx, userID, tokenString, s.tokenTTL)
	if err != nil {
		return nil, fmt.Errorf("failed to store token: %w", err)
	}

	return &TokenPair{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   int64(s.tokenTTL.Seconds()),
		UserID:      userID,
	}, nil
}

// ValidateToken validates and parses a JWT token
func (s *Service) ValidateToken(ctx context.Context, tokenString string) (*TokenClaims, error) {
	// Check if token is blacklisted
	isBlacklisted, err := s.repo.IsTokenBlacklisted(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to check token blacklist: %w", err)
	}

	if isBlacklisted {
		return nil, ErrTokenRevoked
	}

	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Extract user ID
	userIDInterface, exists := claims["user_id"]
	if !exists {
		return nil, ErrInvalidToken
	}

	userID, ok := userIDInterface.(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	// Extract expiration
	expInterface, exists := claims["exp"]
	if !exists {
		return nil, ErrInvalidToken
	}

	exp, ok := expInterface.(float64)
	if !ok {
		return nil, ErrInvalidToken
	}

	expirationTime := time.Unix(int64(exp), 0)
	if time.Now().After(expirationTime) {
		return nil, ErrTokenExpired
	}

	return &TokenClaims{
		UserID:    userID,
		ExpiresAt: expirationTime,
		IssuedAt:  time.Unix(int64(claims["iat"].(float64)), 0),
	}, nil
}

// RevokeToken adds a token to the blacklist
func (s *Service) RevokeToken(ctx context.Context, tokenString string) error {
	// Parse token to get expiration for blacklist TTL
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		// Even if parsing fails, we should blacklist the token
		err = s.repo.BlacklistToken(ctx, tokenString, s.tokenTTL)
		if err != nil {
			return fmt.Errorf("failed to blacklist invalid token: %w", err)
		}
		return nil
	}

	// Calculate remaining TTL for blacklist
	claims := token.Claims.(jwt.MapClaims)
	exp := claims["exp"].(float64)
	expirationTime := time.Unix(int64(exp), 0)
	remainingTTL := time.Until(expirationTime)

	if remainingTTL <= 0 {
		// Token already expired, no need to blacklist
		return nil
	}

	// Blacklist token
	err = s.repo.BlacklistToken(ctx, tokenString, remainingTTL)
	if err != nil {
		return fmt.Errorf("failed to blacklist token: %w", err)
	}

	return nil
}

// RefreshToken generates a new token (for future use)
func (s *Service) RefreshToken(ctx context.Context, userID string, oldToken string) (*TokenPair, error) {
	// Revoke old token
	err := s.RevokeToken(ctx, oldToken)
	if err != nil {
		return nil, fmt.Errorf("failed to revoke old token: %w", err)
	}

	// Generate new token
	return s.GenerateToken(ctx, userID)
}

// LoginWithProvider performs user authentication with different providers and returns tokens
func (s *Service) LoginWithProvider(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	switch req.Provider {
	case ProviderGuest:
		return s.loginGuest(ctx, req)
	case ProviderGoogle:
		return s.loginGoogle(ctx, req)
	case ProviderApple:
		return s.loginApple(ctx, req)
	default:
		return nil, ErrInvalidProvider
	}
}

// loginGuest handles guest authentication
func (s *Service) loginGuest(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	var creds GuestLoginCredentials

	// Parse credentials
	if credsTyped, ok := req.Credentials.(GuestLoginCredentials); ok {
		creds = credsTyped
	} else if credsMap, ok := req.Credentials.(map[string]interface{}); ok {
		if username, exists := credsMap["username"]; exists {
			if usernameStr, ok := username.(string); ok {
				creds.Username = usernameStr
			} else {
				return nil, ErrInvalidCredentials
			}
		} else {
			return nil, ErrInvalidCredentials
		}
	} else {
		return nil, ErrInvalidCredentials
	}

	// Validate username
	if err := ValidateUsername(creds.Username); err != nil {
		return nil, err
	}

	// For guest accounts, we create a new user ID based on username
	// In production, you might want to check if this guest user already exists
	userID := fmt.Sprintf("guest_%s_%d", creds.Username, time.Now().Unix())

	// Generate token
	tokenPair, err := s.GenerateToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		UserID:      userID,
		AccessToken: tokenPair.AccessToken,
		TokenType:   tokenPair.TokenType,
		ExpiresIn:   tokenPair.ExpiresIn,
	}, nil
}

// loginGoogle handles Google OAuth authentication
func (s *Service) loginGoogle(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	var creds GoogleLoginCredentials

	// Parse credentials
	if credsTyped, ok := req.Credentials.(GoogleLoginCredentials); ok {
		creds = credsTyped
	} else if credsMap, ok := req.Credentials.(map[string]interface{}); ok {
		if idToken, exists := credsMap["id_token"]; exists {
			if idTokenStr, ok := idToken.(string); ok {
				creds.IDToken = idTokenStr
			} else {
				return nil, ErrInvalidCredentials
			}
		} else {
			return nil, ErrInvalidCredentials
		}
	} else {
		return nil, ErrInvalidCredentials
	}

	// TODO: Implement Google ID token validation
	// For now, we'll just create a placeholder implementation
	if creds.IDToken == "" {
		return nil, fmt.Errorf("Google ID token is required")
	}

	// In production, you would:
	// 1. Validate the Google ID token
	// 2. Extract user info from the token
	// 3. Check if user exists or create new account
	// 4. Generate internal JWT token

	userID := fmt.Sprintf("google_%d", time.Now().Unix())

	// Generate token
	tokenPair, err := s.GenerateToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		UserID:      userID,
		AccessToken: tokenPair.AccessToken,
		TokenType:   tokenPair.TokenType,
		ExpiresIn:   tokenPair.ExpiresIn,
	}, nil
}

// loginApple handles Apple OAuth authentication
func (s *Service) loginApple(ctx context.Context, req *LoginRequest) (*AuthResponse, error) {
	var creds AppleLoginCredentials

	// Parse credentials
	if credsTyped, ok := req.Credentials.(AppleLoginCredentials); ok {
		creds = credsTyped
	} else if credsMap, ok := req.Credentials.(map[string]interface{}); ok {
		if idToken, exists := credsMap["id_token"]; exists {
			if idTokenStr, ok := idToken.(string); ok {
				creds.IDToken = idTokenStr
			} else {
				return nil, ErrInvalidCredentials
			}
		} else {
			return nil, ErrInvalidCredentials
		}
	} else {
		return nil, ErrInvalidCredentials
	}

	// TODO: Implement Apple ID token validation
	// For now, we'll just create a placeholder implementation
	if creds.IDToken == "" {
		return nil, fmt.Errorf("Apple ID token is required")
	}

	// In production, you would:
	// 1. Validate the Apple ID token
	// 2. Extract user info from the token
	// 3. Check if user exists or create new account
	// 4. Generate internal JWT token

	userID := fmt.Sprintf("apple_%d", time.Now().Unix())

	// Generate token
	tokenPair, err := s.GenerateToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		UserID:      userID,
		AccessToken: tokenPair.AccessToken,
		TokenType:   tokenPair.TokenType,
		ExpiresIn:   tokenPair.ExpiresIn,
	}, nil
}

// LinkAuthProvider links an additional authentication provider to an existing user account
func (s *Service) LinkAuthProvider(ctx context.Context, userID string, req *LinkAuthRequest) error {
	// TODO: Implement auth provider linking
	// This would involve:
	// 1. Validating the new provider credentials
	// 2. Checking if the provider account is already linked to another user
	// 3. Creating a new AuthLink record for the user
	// 4. Updating the user account

	// For now, return success (placeholder implementation)
	return nil
}

// LoginUser performs user authentication and returns tokens (legacy method for backward compatibility)
func (s *Service) LoginUser(ctx context.Context, userID string) (*AuthResponse, error) {
	// Validate user ID format
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// For MVP, we assume user exists - in production you'd validate against user service
	tokenPair, err := s.GenerateToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return &AuthResponse{
		UserID:      userID,
		AccessToken: tokenPair.AccessToken,
		TokenType:   tokenPair.TokenType,
		ExpiresIn:   tokenPair.ExpiresIn,
	}, nil
}

// LogoutUser revokes user's active token
func (s *Service) LogoutUser(ctx context.Context, userID, tokenString string) error {
	// Revoke the specific token
	err := s.RevokeToken(ctx, tokenString)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// ValidateUserAccess checks if user has valid authentication
func (s *Service) ValidateUserAccess(ctx context.Context, tokenString string) (*TokenClaims, error) {
	claims, err := s.ValidateToken(ctx, tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid authentication: %w", err)
	}

	return claims, nil
}

// SetTokenTTL updates the token time-to-live duration
func (s *Service) SetTokenTTL(ttl time.Duration) {
	s.tokenTTL = ttl
}

// GetTokenTTL returns the current token time-to-live duration
func (s *Service) GetTokenTTL() time.Duration {
	return s.tokenTTL
}

// AuthResponse represents authentication response
type AuthResponse struct {
	UserID      string `json:"user_id"`
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	UserID string `json:"user_id,omitempty"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	UserID   string `json:"user_id"`
	OldToken string `json:"old_token"`
}

