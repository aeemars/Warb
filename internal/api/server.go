package api

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
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
func NewServer(port string, s *store.Store, e *engine.Engine, googleClientID string) *Server {
	mux := http.NewServeMux()

	// Register API routes
	handlers := NewHandlers(s, e, googleClientID)
	handlers.RegisterRoutes(mux)

	// Serve static frontend files
	fs := http.FileServer(http.Dir("web"))
	mux.Handle("/", fs)

	// Wrap with middleware (innermost first)
	handler := corsMiddleware(
		securityHeadersMiddleware(
			rateLimitMiddleware(
				bodySizeLimitMiddleware(
					loggingMiddleware(
						recoveryMiddleware(mux),
					),
				),
			),
		),
	)

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

// FINDING-01 FIX: Replace wildcard CORS with explicit origin allowlist.
func corsMiddleware(next http.Handler) http.Handler {
	// Build allowlist from environment or default to localhost development origins.
	allowedOrigins := map[string]bool{
		"http://localhost:8080":  true,
		"http://localhost:3000":  true,
		"https://localhost:8080": true,
	}
	if extra := os.Getenv("CORS_ALLOWED_ORIGINS"); extra != "" {
		for _, o := range strings.Split(extra, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				allowedOrigins[o] = true
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Vary", "Origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// FINDING-09 FIX: Add comprehensive security headers.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' https://accounts.google.com https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://accounts.google.com; "+
				"font-src https://fonts.gstatic.com; "+
				"img-src 'self' https: data:; "+
				"frame-src https://accounts.google.com; "+
				"connect-src 'self' https://accounts.google.com")

		// HSTS only over TLS; detect via X-Forwarded-Proto or TLS connection.
		if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}

// FINDING-06 FIX: Simple per-IP rate limiter.
// Auth endpoints: 10 req/min. AI endpoints: 5 req/min. General: 60 req/min.
func rateLimitMiddleware(next http.Handler) http.Handler {
	type visitor struct {
		tokens   float64
		lastSeen time.Time
	}

	var (
		mu       sync.Mutex
		visitors = make(map[string]*visitor)
	)

	// Cleanup stale entries every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			mu.Lock()
			now := time.Now()
			for ip, v := range visitors {
				if now.Sub(v.lastSeen) > 10*time.Minute {
					delete(visitors, ip)
				}
			}
			mu.Unlock()
		}
	}()

	allow := func(ip string, limit float64, burst float64) bool {
		mu.Lock()
		defer mu.Unlock()

		v, exists := visitors[ip]
		now := time.Now()
		if !exists {
			visitors[ip] = &visitor{tokens: burst - 1, lastSeen: now}
			return true
		}

		elapsed := now.Sub(v.lastSeen).Seconds()
		v.lastSeen = now
		v.tokens += elapsed * (limit / 60.0) // refill rate per second
		if v.tokens > burst {
			v.tokens = burst
		}

		if v.tokens >= 1 {
			v.tokens--
			return true
		}
		return false
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip == "" {
			ip = r.RemoteAddr
		}

		path := r.URL.Path

		var allowed bool
		switch {
		case strings.HasPrefix(path, "/api/auth/"):
			// Auth endpoints: 10 per minute per IP
			allowed = allow(ip+":auth", 10, 15)
		case strings.HasPrefix(path, "/api/portfolio/scan") || strings.Contains(path, "/analyze"):
			// AI endpoints: 5 per minute per IP
			allowed = allow(ip+":ai", 5, 8)
		case strings.HasPrefix(path, "/api/"):
			// General API: 60 per minute per IP
			allowed = allow(ip+":api", 60, 80)
		default:
			// Static assets: no limiting
			allowed = true
		}

		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "10")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"success":false,"error":"Rate limit exceeded. Please try again shortly."}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// FINDING-05 FIX: Limit request body size to 1MB for all API endpoints.
func bodySizeLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") && r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
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
