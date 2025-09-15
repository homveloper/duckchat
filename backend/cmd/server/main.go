package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"duckchat/internal/auth"
	"duckchat/internal/message"
	"duckchat/internal/room"
	"duckchat/internal/sse"
	"duckchat/internal/ui"
	"duckchat/internal/user"

	"github.com/redis/go-redis/v9"
)

// DuckChatApp contains all application dependencies
type DuckChatApp struct {
	// Redis client
	RedisClient *redis.Client

	// Repositories
	UserRepo    user.Repository
	AuthRepo    auth.Repository
	RoomRepo    room.Repository
	MessageRepo message.Repository

	// Services
	UserService    *user.Service
	AuthService    *auth.Service
	RoomService    *room.Service
	MessageService *message.Service
	SSEService     *sse.Service

	// Handlers
	UIHandler      *ui.Handler
	AuthHandler    *auth.Handler
	UserHandler    *user.Handler
	RoomHandler    *room.Handler
	MessageHandler *message.Handler

	// Configuration
	StaticDir string
	Port      string
}

// NewDuckChatApp creates and initializes the application with all dependencies
func NewDuckChatApp(staticDir, port string) *DuckChatApp {
	app := &DuckChatApp{
		StaticDir: staticDir,
		Port:      port,
	}

	// Initialize Redis client
	app.RedisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password
		DB:       0,  // default DB
	})

	// Initialize repositories with Redis client
	app.UserRepo = user.NewRepository(app.RedisClient)
	app.AuthRepo = auth.NewRepository(app.RedisClient)
	app.RoomRepo = room.NewRepository(app.RedisClient)
	app.MessageRepo = message.NewRepository(app.RedisClient)

	// Initialize services
	app.UserService = user.NewService(app.UserRepo)
	app.AuthService = auth.NewService(app.AuthRepo, "duckchat-jwt-secret-key")
	app.RoomService = room.NewService(app.RoomRepo)
	app.MessageService = message.NewService(app.MessageRepo, app.RoomService)
	app.SSEService = sse.NewService(app.AuthService)

	// Initialize handlers
	app.UIHandler = ui.NewHandler(app.AuthService)
	app.AuthHandler = auth.NewHandler(app.AuthService)
	app.UserHandler = user.NewHandler(app.UserService)
	app.RoomHandler = room.NewHandler(app.RoomService)
	app.MessageHandler = message.NewHandler(app.MessageService)

	return app
}

// SetupRoutes configures all HTTP routes
func (app *DuckChatApp) SetupRoutes() {
	// UI Routes (SSR)
	http.HandleFunc("/login", loggingMiddleware("LOGIN", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			app.UIHandler.LoginGetHandler(w, r)
		case http.MethodPost:
			app.UIHandler.LoginPostHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	http.HandleFunc("/rooms", loggingMiddleware("ROOMS", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			app.UIHandler.RoomsGetHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// API Routes (JSON-RPC)
	http.HandleFunc("/api/auth", loggingMiddleware("API-AUTH", app.AuthHandler.HandleAuth))
	http.HandleFunc("/api/users", loggingMiddleware("API-USERS", app.requireAuth(app.UserHandler.HandleUser)))
	http.HandleFunc("/api/rooms", loggingMiddleware("API-ROOMS", app.requireAuth(app.RoomHandler.HandleRoom)))
	http.HandleFunc("/api/messages", loggingMiddleware("API-MESSAGES", app.requireAuth(app.MessageHandler.HandleMessage)))

	// SSE Routes (use cookie auth for SSE)
	http.HandleFunc("/sse/events", loggingMiddleware("SSE-EVENTS", app.requireCookieAuth(app.SSEService.HandleSSE)))

	// Static file serving
	http.HandleFunc("/static/", app.handleStatic)

	// Root redirect
	http.HandleFunc("/", loggingMiddleware("ROOT", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/login", http.StatusFound)
		} else {
			http.NotFound(w, r)
		}
	}))
}

// requireAuth wraps handlers that require JWT authentication via Authorization header
func (app *DuckChatApp) requireAuth(handler http.HandlerFunc) http.HandlerFunc {
	return app.AuthHandler.AuthMiddleware(handler)
}

// requireCookieAuth wraps handlers that require JWT authentication via cookies
func (app *DuckChatApp) requireCookieAuth(handler http.HandlerFunc) http.HandlerFunc {
	return app.AuthHandler.CookieAuthMiddleware(handler)
}

// handleStatic serves static files with proper MIME types
func (app *DuckChatApp) handleStatic(w http.ResponseWriter, r *http.Request) {
	// Remove /static/ prefix
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	fullPath := filepath.Join(app.StaticDir, path)

	// Log the request
	log.Printf("[STATIC] %s %s -> %s", r.Method, r.URL.Path, fullPath)

	// Check if file exists
	if _, err := filepath.Abs(fullPath); err != nil {
		log.Printf("[STATIC] ERROR: Invalid path %s: %v", fullPath, err)
		http.NotFound(w, r)
		return
	}

	// Set proper MIME type based on file extension
	ext := filepath.Ext(path)
	switch ext {
	case ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		log.Printf("[STATIC] Setting MIME type: text/css for %s", path)
	case ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	case ".ico":
		w.Header().Set("Content-Type", "image/x-icon")
	case ".woff":
		w.Header().Set("Content-Type", "font/woff")
	case ".woff2":
		w.Header().Set("Content-Type", "font/woff2")
	default:
		log.Printf("[STATIC] Using default MIME type for %s (ext: %s)", path, ext)
	}

	// Serve the file and log the result
	http.ServeFile(w, r, fullPath)
	log.Printf("[STATIC] Served %s successfully", fullPath)
}

// Start starts the HTTP server
func (app *DuckChatApp) Start() error {
	app.SetupRoutes()
	log.Printf("Server starting on :%s", app.Port)
	return http.ListenAndServe(":"+app.Port, nil)
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// loggingMiddleware wraps an http.HandlerFunc with request and error logging
func loggingMiddleware(name string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Log request with body size for POST requests
		bodyInfo := ""
		if r.Method == "POST" && r.ContentLength > 0 {
			bodyInfo = fmt.Sprintf(" (body: %d bytes)", r.ContentLength)
		}
		log.Printf("[%s] %s %s from %s%s", name, r.Method, r.URL.Path, r.RemoteAddr, bodyInfo)

		// Wrap the response writer to capture status code
		rw := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Handle panics
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[%s] PANIC in %s %s: %v", name, r.Method, r.URL.Path, err)
				if !rw.written {
					http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
				}
			}
		}()

		// Call the handler
		handler(rw, r)

		// Log completion with status code
		duration := time.Since(start)
		statusText := http.StatusText(rw.statusCode)
		if rw.statusCode >= 400 {
			log.Printf("[%s] ERROR %s %s -> %d %s in %v", name, r.Method, r.URL.Path, rw.statusCode, statusText, duration)
		} else {
			log.Printf("[%s] %s %s -> %d %s in %v", name, r.Method, r.URL.Path, rw.statusCode, statusText, duration)
		}
	}
}

func main() {
	// Command line flags
	var (
		staticDir = flag.String("static", "web/static/", "Path to static files directory")
		port      = flag.String("port", "8080", "Port to listen on")
	)
	flag.Parse()

	log.Printf("Static files directory: %s", *staticDir)
	log.Printf("Server will listen on port: %s", *port)

	// Create and initialize the DuckChat application
	app := NewDuckChatApp(*staticDir, *port)

	// Start the server
	log.Fatal(app.Start())
}
