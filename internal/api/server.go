package api

import (
	"log"
	"net/http"
	"time"

	"opportunity-engine/internal/engine"
	"opportunity-engine/internal/store"
)

// Server is the HTTP server for the Opportunity Engine.
type Server struct {
	mux    *http.ServeMux
	server *http.Server
}

// NewServer creates and configures a new HTTP server.
func NewServer(port string, s *store.Store, e *engine.Engine) *Server {
	mux := http.NewServeMux()

	// Register API routes
	handlers := NewHandlers(s, e)
	handlers.RegisterRoutes(mux)

	// Serve static frontend files
	fs := http.FileServer(http.Dir("web"))
	mux.Handle("/", fs)

	// Wrap with middleware
	handler := corsMiddleware(loggingMiddleware(recoveryMiddleware(mux)))

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 180 * time.Second, // Long timeout for AI analysis
		IdleTimeout:  60 * time.Second,
	}

	return &Server{mux: mux, server: srv}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	log.Printf("[Server] Starting on %s", s.server.Addr)
	return s.server.ListenAndServe()
}

// --- Middleware ---

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[HTTP] %s %s — %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %s %s — %v", r.Method, r.URL.Path, err)
				http.Error(w, `{"success":false,"error":"Internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
