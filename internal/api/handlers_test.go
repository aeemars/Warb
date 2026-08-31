package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"opportunity-engine/internal/engine"
	"opportunity-engine/internal/models"
	"opportunity-engine/internal/store"
)

func setupTestServer(t *testing.T) (*http.ServeMux, *store.Store, func()) {
	tmpDir, err := os.MkdirTemp("", "api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")

	s, err := store.New(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create store: %v", err)
	}

	_ = s.Seed()

	eng := engine.New("test-key", "test-model", s)
	handlers := NewHandlers(s, eng, "test-client-id.apps.googleusercontent.com")
	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux)

	cleanup := func() {
		s.Close()
		os.RemoveAll(tmpDir)
	}
	return mux, s, cleanup
}

func TestAuthAndUserIsolationAPI(t *testing.T) {
	mux, s, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Unauthenticated request to /api/clients should be 401 Unauthorized
	req := httptest.NewRequest(http.MethodGet, "/api/clients", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for unauthenticated /api/clients, got %d", rec.Code)
	}

	// 2. Create User 1 and User 2 in Store
	u1, err := s.UpsertGoogleUser(&models.User{
		GoogleID: "gid-1",
		Email:    "tariq@warbabank.com",
		Name:     "Tariq Al-Rashid",
	})
	if err != nil {
		t.Fatalf("upsert user 1 failed: %v", err)
	}
	_ = s.SeedUserPortfolio(u1.ID, u1.Name)

	u2, err := s.UpsertGoogleUser(&models.User{
		GoogleID: "gid-2",
		Email:    "fatima@warbabank.com",
		Name:     "Fatima Al-Rashidi",
	})
	if err != nil {
		t.Fatalf("upsert user 2 failed: %v", err)
	}

	token1 := "token-u1"
	token2 := "token-u2"
	_ = s.CreateSession(token1, u1.ID, time.Now().Add(24*time.Hour))
	_ = s.CreateSession(token2, u2.ID, time.Now().Add(24*time.Hour))

	// 3. User 1 calls GET /api/clients (should get 20 clients)
	req = httptest.NewRequest(http.MethodGet, "/api/clients", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token1})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for user 1 /api/clients, got %d: %s", rec.Code, rec.Body.String())
	}

	var clientsResp models.APIResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &clientsResp)
	clientsBytes, _ := json.Marshal(clientsResp.Data)
	var clients []models.Client
	_ = json.Unmarshal(clientsBytes, &clients)
	if len(clients) != 20 {
		t.Errorf("expected 20 clients for user 1, got %d", len(clients))
	}

	// 4. User 2 calls GET /api/clients (should get 0 clients before seeding)
	req = httptest.NewRequest(http.MethodGet, "/api/clients", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token2})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for user 2 /api/clients, got %d", rec.Code)
	}

	_ = json.Unmarshal(rec.Body.Bytes(), &clientsResp)
	clientsBytes, _ = json.Marshal(clientsResp.Data)
	_ = json.Unmarshal(clientsBytes, &clients)
	if len(clients) != 0 {
		t.Errorf("expected 0 clients for user 2, got %d", len(clients))
	}

	// 5. User 1 calls GET /api/portfolio/summary
	req = httptest.NewRequest(http.MethodGet, "/api/portfolio/summary", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token1})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for user 1 /api/portfolio/summary, got %d", rec.Code)
	}

	// 6. User 1 logs out
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token1})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for logout, got %d", rec.Code)
	}

	// 7. Requesting /api/clients after logout should now fail with 401
	req = httptest.NewRequest(http.MethodGet, "/api/clients", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token1})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized after logout, got %d", rec.Code)
	}
}
