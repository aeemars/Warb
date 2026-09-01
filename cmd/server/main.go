package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"opportunity-engine/internal/api"
	"opportunity-engine/internal/engine"
	"opportunity-engine/internal/store"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("=== Warba Bank — Proactive Client Opportunity Engine ===")

	// Load configuration from environment
	apiKey := getEnv("OPENROUTER_API_KEY", "")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY environment variable is required")
	}

	port := getEnv("PORT", "8080")
	model := getEnv("AI_MODEL", "inclusionai/ling-3.0-flash-fin:free")
	dbPath := getEnv("DB_PATH", "./data/opportunity.db")

	log.Printf("Config: port=%s, model=%s, db=%s", port, model, dbPath)

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize store
	s, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer s.Close()

	// Seed global Shariah products if empty
	if err := s.Seed(); err != nil {
		log.Fatalf("Failed to initialize product catalog: %v", err)
	}

	// FINDING-07 FIX: Google OAuth Client ID is required for secure authentication.
	googleClientID := getEnv("GOOGLE_CLIENT_ID", "")
	if googleClientID != "" {
		masked := googleClientID
		if len(masked) > 12 {
			masked = masked[:12] + "..."
		}
		log.Printf("Google Auth: Configured (Client ID: %s)", masked)
	} else {
		log.Printf("WARNING: GOOGLE_CLIENT_ID not set — authentication will be unavailable until configured")
	}

	// Initialize AI engine
	eng := engine.New(apiKey, model, s)
	log.Printf("AI Engine initialized with model: %s", model)

	// FINDING-17 FIX: Start background goroutine to purge expired sessions.
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			purged, err := s.PurgeExpiredSessions()
			if err != nil {
				log.Printf("[Session GC] Error purging expired sessions: %v", err)
			} else if purged > 0 {
				log.Printf("[Session GC] Purged %d expired sessions", purged)
			}
		}
	}()

	// Start server
	srv := api.NewServer(port, s, eng, googleClientID)
	log.Printf("Dashboard: http://localhost:%s", port)
	log.Printf("API Base:  http://localhost:%s/api", port)

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		v = strings.TrimPrefix(v, key+"=")
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"'`)
		if v != "" {
			return v
		}
	}
	return fallback
}
