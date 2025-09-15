package ui

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"duckchat/internal/auth"
	"duckchat/internal/ui/models"
	"duckchat/web/templates/components"
	"duckchat/web/templates/layouts"
)

// Handler manages UI-related HTTP requests
type Handler struct {
	attemptTracker *models.LoginAttemptTracker
	authService    *auth.Service
}

// NewHandler creates a new UI handler
func NewHandler(authService *auth.Service) *Handler {
	return &Handler{
		attemptTracker: models.NewLoginAttemptTracker(),
		authService:    authService,
	}
}

// LoginGetHandler handles GET /login requests
func (h *Handler) LoginGetHandler(w http.ResponseWriter, r *http.Request) {
	// Check if user is already authenticated
	if h.isAuthenticated(r) {
		http.Redirect(w, r, "/rooms", http.StatusFound)
		return
	}

	// Generate CSRF token
	csrfToken := h.generateCSRFToken(r)

	// Create login form state
	formState := models.NewLoginFormState(csrfToken)

	// Set redirect URL from query parameter
	if redirectURL := r.URL.Query().Get("redirect"); redirectURL != "" {
		formState.RedirectURL = redirectURL
	}

	// Create layout data
	layoutData := layouts.NewBaseLayoutData("DuckChat - Login", nil, csrfToken)
	layoutData.BodyClass = "login-page"
	layoutData.LayoutState = models.NewResponsiveLayoutStateFromUserAgent(r.UserAgent())

	// Create login form data
	formData := components.NewLoginFormData(formState)

	// Set security headers
	h.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render the login page
	err := components.LoginPage(formData, layoutData).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render login page", http.StatusInternalServerError)
		return
	}
}

// LoginPostHandler handles POST /login requests
func (h *Handler) LoginPostHandler(w http.ResponseWriter, r *http.Request) {
	// Get client IP for rate limiting
	clientIP := h.getClientIP(r)

	// Check rate limiting
	if h.attemptTracker.IsRateLimited(clientIP) {
		retryAfter := h.attemptTracker.GetRetryAfter(clientIP)
		w.Header().Set("Retry-After", retryAfter.String())

		csrfToken := h.generateCSRFToken(r)
		formState := models.LoginFormStateWithError(csrfToken, "Too many login attempts, please try again later")

		h.renderLoginWithError(w, r, formState, http.StatusTooManyRequests)
		return
	}

	// Parse form data
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Create form state from request (guest mode)
	nickname := r.FormValue("nickname")
	if nickname == "" {
		nickname = r.FormValue("username") // fallback for compatibility
	}

	formState := &models.LoginFormState{
		Username:         nickname, // Store nickname as username for compatibility
		Password:         "",       // No password needed for guest
		CSRFToken:        r.FormValue("csrf_token"),
		RedirectURL:      r.FormValue("redirect_url"),
		ValidationErrors: make(map[string]string),
		IsSubmitting:     false,
		IsGuestMode:      true, // Enable guest mode
	}

	// Sanitize input
	formState.SanitizeInput()

	// Validate input
	if err := formState.ValidateInput(); err != nil {
		h.renderLoginWithError(w, r, formState, http.StatusBadRequest)
		return
	}

	// Guest authentication - just validate nickname
	if !h.validateGuestNickname(formState.Username) {
		formState.SetGeneralError(errors.New("Invalid nickname. Please use 2-30 characters (Korean, English, numbers, spaces allowed)"))
		h.renderLoginWithError(w, r, formState, http.StatusBadRequest)
		return
	}

	// Clear failed attempts on successful login
	h.attemptTracker.ClearAttempts(clientIP)

	// Set authentication cookie for guest
	h.setGuestAuthCookie(w, formState.Username)

	// Redirect to intended destination
	redirectURL := formState.RedirectURL
	if redirectURL == "" {
		redirectURL = "/rooms"
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// RoomsGetHandler handles GET /rooms requests
func (h *Handler) RoomsGetHandler(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	user := h.getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Create mock room list data
	rooms := []*components.RoomListItem{
		{
			ID:               "room1",
			Name:             "General",
			Description:      func() *string { s := "Main discussion room"; return &s }(),
			ParticipantCount: 5,
			UnreadCount:      0,
			IsPrivate:        false,
		},
	}

	// Create room list data
	createState := models.NewRoomCreationState(h.generateCSRFToken(r))
	roomListData := components.NewRoomListData(user, createState)
	roomListData.Rooms = rooms

	// Create layout data
	layoutData := layouts.NewBaseLayoutData("DuckChat - Rooms", user, h.generateCSRFToken(r))
	layoutData.BodyClass = "rooms-page"
	layoutData.LayoutState = models.NewResponsiveLayoutStateFromUserAgent(r.UserAgent())

	// Create navigation data
	navData := h.CreateNavigationComponentData(user, r.URL.Path)
	layoutData.NavigationComponent = components.Navigation(navData)

	// Set security headers
	h.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Render the rooms page
	err := components.RoomListPage(roomListData, navData, layoutData).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render rooms page", http.StatusInternalServerError)
		return
	}
}

// Helper methods

func (h *Handler) isAuthenticated(r *http.Request) bool {
	// Check for auth_token cookie (dynamically set by active tab)
	cookie, err := r.Cookie("auth_token")
	return err == nil && cookie.Value != ""
}

func (h *Handler) getCurrentUser(r *http.Request) *models.UserContext {
	// Extract JWT token from cookie (consistent with isAuthenticated)
	cookie, err := r.Cookie("auth_token")
	if err != nil || cookie.Value == "" {
		return nil
	}

	token := cookie.Value

	// Validate JWT token using auth service
	claims, err := h.authService.ValidateToken(r.Context(), token)
	if err != nil {
		return nil
	}

	// Extract username from user ID for guest users
	userID := claims.UserID
	var username string

	if strings.HasPrefix(userID, "guest_") {
		// For guest users: "guest_username_timestamp" -> extract username
		parts := strings.Split(userID, "_")
		if len(parts) >= 2 {
			username = parts[1] // Get the username part
		} else {
			username = strings.TrimPrefix(userID, "guest_")
		}
	} else {
		// For other users, use userID as username for now
		username = userID
	}

	return &models.UserContext{
		ID:       userID,
		Username: username,
		IsOnline: true,
	}
}

func (h *Handler) generateCSRFToken(r *http.Request) string {
	// Mock CSRF token generation for TDD
	return "mock_csrf_token_" + time.Now().Format("20060102150405")
}

func (h *Handler) authenticateUser(username, password string) bool {
	// Mock authentication for TDD
	return username == "testuser" && password == "password123"
}

func (h *Handler) validateGuestNickname(nickname string) bool {
	// Validate nickname length
	if len(nickname) < 2 || len(nickname) > 30 {
		return false
	}

	// Check for empty/whitespace-only nickname
	if strings.TrimSpace(nickname) == "" {
		return false
	}

	// For now, allow all valid nicknames
	// In the future, you might want to check against reserved names or profanity
	return true
}

func (h *Handler) setAuthCookie(w http.ResponseWriter, username string) {
	// Mock JWT cookie for TDD
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    "mock_jwt_" + username,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
	}
	http.SetCookie(w, cookie)
}

func (h *Handler) setGuestAuthCookie(w http.ResponseWriter, nickname string) {
	// Guest auth cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    "guest_" + nickname,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Allow HTTP for development
		SameSite: http.SameSiteLaxMode,
		MaxAge:   24 * 60 * 60, // 1 day for guests
	}
	http.SetCookie(w, cookie)
}

func (h *Handler) setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

func (h *Handler) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
		return xForwardedFor
	}

	// Check X-Real-IP header
	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

func (h *Handler) renderLoginWithError(w http.ResponseWriter, r *http.Request, formState *models.LoginFormState, statusCode int) {
	// Create layout data
	layoutData := layouts.NewBaseLayoutData("DuckChat - Login", nil, formState.CSRFToken)
	layoutData.BodyClass = "login-page error"
	layoutData.LayoutState = models.NewResponsiveLayoutStateFromUserAgent(r.UserAgent())

	// Create login form data
	formData := components.NewLoginFormData(formState)

	// Set security headers and status
	h.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)

	// Render the login page with errors
	err := components.LoginPage(formData, layoutData).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render login page", http.StatusInternalServerError)
		return
	}
}

// Navigation helper methods

// GetNavigationData creates navigation data for the current user and URL
func (h *Handler) GetNavigationData(user *models.UserContext, currentURL string) *models.NavigationData {
	navData := models.NewNavigationData()

	// Determine active section based on current URL
	switch {
	case strings.HasPrefix(currentURL, "/rooms") || strings.HasPrefix(currentURL, "/chat"):
		navData.SetActiveSection(models.NavigationSectionChat)
	case strings.HasPrefix(currentURL, "/profile"):
		navData.SetActiveSection(models.NavigationSectionProfile)
	default:
		navData.SetActiveSection(models.NavigationSectionChat) // default to chat
	}

	// Update badge counts if needed (e.g., unread messages)
	if user != nil {
		// TODO: Get actual unread counts from services
		// For now, mock some data
		navData.UpdateBadgeCount(models.NavigationSectionChat, 0)
		navData.UpdateBadgeCount(models.NavigationSectionProfile, 0)
	}

	return navData
}

// CreateNavigationComponentData creates the data structure for navigation component
func (h *Handler) CreateNavigationComponentData(user *models.UserContext, currentURL string) *components.NavigationData {
	return &components.NavigationData{
		Navigation: h.GetNavigationData(user, currentURL),
		CurrentURL: currentURL,
	}
}

// ProfileGetHandler handles GET /profile requests
func (h *Handler) ProfileGetHandler(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	user := h.getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Create layout data
	layoutData := layouts.NewBaseLayoutData("DuckChat - Profile", user, h.generateCSRFToken(r))
	layoutData.BodyClass = "profile-page"
	layoutData.LayoutState = models.NewResponsiveLayoutStateFromUserAgent(r.UserAgent())

	// Create navigation data
	navData := h.CreateNavigationComponentData(user, r.URL.Path)
	layoutData.NavigationComponent = components.Navigation(navData)

	// Set security headers
	h.setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// For now, render a simple profile page
	err := components.ProfilePage(user, navData, layoutData).Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Failed to render profile page", http.StatusInternalServerError)
		return
	}
}
