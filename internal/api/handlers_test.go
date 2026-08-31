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

func TestAuthFlow(t *testing.T) {
	mux, s, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. GET /api/auth/config
	req := httptest.NewRequest(http.MethodGet, "/api/auth/config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/auth/config, got %d", rec.Code)
	}

	var configResp models.APIResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &configResp); err != nil {
		t.Fatalf("failed to parse auth config response: %v", err)
	}
	if !configResp.Success {
		t.Fatalf("expected success true in auth config")
	}

	// 2. GET /api/auth/me (Unauthenticated)
	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/auth/me, got %d", rec.Code)
	}
	var meResp models.APIResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &meResp)
	meData, _ := json.Marshal(meResp.Data)
	var authMe models.AuthUserResponse
	_ = json.Unmarshal(meData, &authMe)
	if authMe.Authenticated {
		t.Errorf("expected authenticated=false initially, got true")
	}

	// 3. Create a real User and Session in Store
	testUser := &models.User{
		GoogleID: "google-123456789",
		Email:    "tariq.rashid@warbabank.com",
		Name:     "Tariq Al-Rashid",
		Avatar:   "https://lh3.googleusercontent.com/test",
		Role:     "Senior Relationship Manager",
	}
	savedUser, err := s.UpsertGoogleUser(testUser)
	if err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	token := "session-test-token-xyz"
	if err := s.CreateSession(token, savedUser.ID, time.Now().Add(24*time.Hour)); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// 4. GET /api/auth/me (Authenticated with Cookie)
	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/auth/me with session, got %d", rec.Code)
	}

	_ = json.Unmarshal(rec.Body.Bytes(), &meResp)
	meData, _ = json.Marshal(meResp.Data)
	_ = json.Unmarshal(meData, &authMe)
	if !authMe.Authenticated || authMe.User == nil {
		t.Fatalf("expected authenticated=true with user, got %v", authMe)
	}
	if authMe.User.Email != "tariq.rashid@warbabank.com" {
		t.Errorf("expected email tariq.rashid@warbabank.com, got %s", authMe.User.Email)
	}

	// 5. POST /api/auth/logout
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/auth/logout, got %d", rec.Code)
	}

	// 6. GET /api/auth/me (After logout)
	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "session_token", Value: token})
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	_ = json.Unmarshal(rec.Body.Bytes(), &meResp)
	meData, _ = json.Marshal(meResp.Data)
	_ = json.Unmarshal(meData, &authMe)
	if authMe.Authenticated {
		t.Errorf("expected authenticated=false after logout, got true")
	}
}
