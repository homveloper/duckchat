package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"duckchat/internal/auth"
)

// SSENotification represents a JSON-RPC 2.0 notification for SSE
type SSENotification struct {
	JSONRPC string      `json:"jsonrpc"` // Always "2.0"
	Method  string      `json:"method"`  // e.g., "room.message", "room.user_joined"
	Params  interface{} `json:"params"`  // Event data
}

// Legacy Event struct for backward compatibility
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// Method constants for JSON-RPC notifications
const (
	MethodRoomMessage     = "room.message"
	MethodRoomUserJoined  = "room.user_joined"
	MethodRoomUserLeft    = "room.user_left"
	MethodRoomUpdated     = "room.updated"
	MethodUserTyping      = "user.typing"
	MethodHeartbeat       = "heartbeat"
	MethodConnected       = "connected"
	MethodError           = "error"
)

// Client represents a connected SSE client
type Client struct {
	ID        string
	UserID    string
	Channel   chan SSENotification
	Request   *http.Request
	Writer    http.ResponseWriter
	LastPing  time.Time
	Active    bool
	UserRooms []string // List of rooms user has access to (for filtering)
	mutex     sync.RWMutex
}

// Service manages SSE connections and event broadcasting
type Service struct {
	clients     map[string]*Client
	authService *auth.Service
	mutex       sync.RWMutex
}

// NewService creates a new SSE service
func NewService(authService *auth.Service) *Service {
	service := &Service{
		clients:     make(map[string]*Client),
		authService: authService,
	}

	// Start cleanup routine
	go service.cleanup()

	return service
}

// HandleSSE handles SSE connection requests with JWT authentication
func (s *Service) HandleSSE(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("[SSE] New connection request from %s\n", r.RemoteAddr)

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		fmt.Printf("[SSE] No user ID found in context - auth middleware may have failed\n")
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	fmt.Printf("[SSE] User authenticated via middleware: %s\n", userID)

	// Allow multiple concurrent connections per user (for multiple tabs)

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Create unique client ID for this connection
	clientID := fmt.Sprintf("%s_%d", userID, time.Now().UnixNano())

	client := &Client{
		ID:        clientID,
		UserID:    userID,
		Channel:   make(chan SSENotification, 100), // Buffer for notifications
		Request:   r,
		Writer:    w,
		LastPing:  time.Now(),
		Active:    true,
		UserRooms: []string{}, // TODO: Populate from user's room memberships
	}

	// Register client
	s.registerClient(client)
	defer s.unregisterClient(client.ID)
	fmt.Printf("[SSE] Client %s registered for user %s\n", client.ID, userID)

	// Send initial connection notification
	s.sendNotification(client, SSENotification{
		JSONRPC: "2.0",
		Method:  MethodConnected,
		Params: map[string]interface{}{
			"client_id": client.ID,
			"user_id":   userID,
		},
	})

	// Keep connection alive and send events
	heartbeatTicker := time.NewTicker(30 * time.Second) // Heartbeat every 30 seconds
	connectionCheckTicker := time.NewTicker(5 * time.Second) // Check connection every 5 seconds
	defer heartbeatTicker.Stop()
	defer connectionCheckTicker.Stop()

	for {
		select {
		case notification := <-client.Channel:
			if !client.IsActive() {
				fmt.Printf("[SSE] Client %s marked as inactive, closing connection\n", client.ID)
				return
			}
			if err := s.sendNotification(client, notification); err != nil {
				fmt.Printf("[SSE] Failed to send notification to client %s: %v\n", client.ID, err)
				return
			}

		case <-heartbeatTicker.C:
			if !client.IsActive() {
				fmt.Printf("[SSE] Client %s inactive during heartbeat, closing connection\n", client.ID)
				return
			}
			// Send heartbeat
			if err := s.sendNotification(client, SSENotification{
				JSONRPC: "2.0",
				Method:  MethodHeartbeat,
				Params:  map[string]interface{}{"timestamp": time.Now().Unix()},
			}); err != nil {
				fmt.Printf("[SSE] Heartbeat failed for client %s: %v\n", client.ID, err)
				return
			}

		case <-connectionCheckTicker.C:
			// Check if client is still active and hasn't timed out
			if !client.IsActive() || time.Since(client.GetLastPing()) > time.Minute*2 {
				fmt.Printf("[SSE] Client %s timed out or inactive, closing connection\n", client.ID)
				return
			}

		case <-r.Context().Done():
			fmt.Printf("[SSE] Client %s context cancelled, closing connection\n", client.ID)
			return
		}
	}
}

// BroadcastToRoom sends a notification to all clients who have access to a specific room
func (s *Service) BroadcastToRoom(roomID string, notification SSENotification) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, client := range s.clients {
		if client.IsActive() && s.hasRoomAccess(client, roomID) {
			select {
			case client.Channel <- notification:
			default:
				// Channel full, skip this client
			}
		}
	}
}

// BroadcastToUser sends a notification to all clients of a specific user
func (s *Service) BroadcastToUser(userID string, notification SSENotification) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, client := range s.clients {
		if client.UserID == userID && client.IsActive() {
			select {
			case client.Channel <- notification:
			default:
				// Channel full, skip this client
			}
		}
	}
}

// BroadcastToAll sends a notification to all connected clients
func (s *Service) BroadcastToAll(notification SSENotification) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, client := range s.clients {
		if client.IsActive() {
			select {
			case client.Channel <- notification:
			default:
				// Channel full, skip this client
			}
		}
	}
}


// closeUserConnections closes all existing connections for a user
func (s *Service) closeUserConnections(userID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var toClose []string
	for clientID, client := range s.clients {
		if client.UserID == userID {
			fmt.Printf("[SSE] Closing existing connection for user %s: %s\n", userID, clientID)
			client.SetActive(false)
			if client.Channel != nil {
				close(client.Channel)
			}
			toClose = append(toClose, clientID)
		}
	}

	// Remove closed clients from map
	for _, clientID := range toClose {
		delete(s.clients, clientID)
	}

	if len(toClose) > 0 {
		fmt.Printf("[SSE] Closed %d existing connections for user %s\n", len(toClose), userID)
	}
}

// registerClient adds a client to the service
func (s *Service) registerClient(client *Client) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.clients[client.ID] = client
}

// unregisterClient removes a client from the service
func (s *Service) unregisterClient(clientID string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	client, exists := s.clients[clientID]
	if !exists {
		return
	}

	client.SetActive(false)
	close(client.Channel)

	// Remove from clients map
	delete(s.clients, clientID)
}

// sendNotification sends a JSON-RPC notification to a specific client
func (s *Service) sendNotification(client *Client, notification SSENotification) error {
	notificationJSON, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal notification: %w", err)
	}

	// Format as SSE
	_, err = fmt.Fprintf(client.Writer, "event: notification\ndata: %s\n\n", string(notificationJSON))
	if err != nil {
		client.SetActive(false)
		return fmt.Errorf("failed to write notification: %w", err)
	}

	// Flush if possible
	if flusher, ok := client.Writer.(http.Flusher); ok {
		flusher.Flush()
	}

	client.UpdateLastPing()
	return nil
}

// cleanup removes inactive clients periodically
func (s *Service) cleanup() {
	ticker := time.NewTicker(30 * time.Second) // More frequent cleanup
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanupInactiveClients()
		}
	}
}

// cleanupInactiveClients removes clients that haven't been active recently
func (s *Service) cleanupInactiveClients() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	now := time.Now()
	inactiveThreshold := 3 * time.Minute // Reduced from 5 to 3 minutes

	var removedCount int
	for clientID, client := range s.clients {
		if now.Sub(client.GetLastPing()) > inactiveThreshold || !client.IsActive() {
			fmt.Printf("[SSE] Cleaning up inactive client %s (user: %s, last ping: %v ago)\n",
				clientID, client.UserID, now.Sub(client.GetLastPing()))
			client.SetActive(false)
			if client.Channel != nil {
				close(client.Channel)
			}
			delete(s.clients, clientID)
			removedCount++
		}
	}

	if removedCount > 0 {
		fmt.Printf("[SSE] Cleanup completed: removed %d inactive clients, %d clients remaining\n",
			removedCount, len(s.clients))
	}
}

// GetStats returns service statistics
func (s *Service) GetStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return map[string]interface{}{
		"total_clients": len(s.clients),
	}
}


// hasRoomAccess checks if a client has access to a specific room
func (s *Service) hasRoomAccess(client *Client, roomID string) bool {
	// TODO: Implement proper room access checking
	// For now, allow access to all rooms
	// In the future, check against client.UserRooms
	for _, userRoomID := range client.UserRooms {
		if userRoomID == roomID {
			return true
		}
	}
	// Default to allow access for now
	return true
}

// Client methods

// IsActive checks if client is active
func (c *Client) IsActive() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.Active
}

// SetActive sets client active status
func (c *Client) SetActive(active bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.Active = active
}

// UpdateLastPing updates client's last ping time
func (c *Client) UpdateLastPing() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.LastPing = time.Now()
}

// GetLastPing gets client's last ping time
func (c *Client) GetLastPing() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.LastPing
}

// Helper functions to create JSON-RPC notifications
func NewMessageNotification(params interface{}) SSENotification {
	return SSENotification{
		JSONRPC: "2.0",
		Method:  MethodRoomMessage,
		Params:  params,
	}
}

func NewUserJoinedNotification(params interface{}) SSENotification {
	return SSENotification{
		JSONRPC: "2.0",
		Method:  MethodRoomUserJoined,
		Params:  params,
	}
}

func NewUserLeftNotification(params interface{}) SSENotification {
	return SSENotification{
		JSONRPC: "2.0",
		Method:  MethodRoomUserLeft,
		Params:  params,
	}
}

func NewRoomUpdatedNotification(params interface{}) SSENotification {
	return SSENotification{
		JSONRPC: "2.0",
		Method:  MethodRoomUpdated,
		Params:  params,
	}
}

func NewTypingNotification(params interface{}) SSENotification {
	return SSENotification{
		JSONRPC: "2.0",
		Method:  MethodUserTyping,
		Params:  params,
	}
}
