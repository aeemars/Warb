package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

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
	mux, _, cleanup := setupTestServer(t)
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

	// 3. POST /api/auth/google (Authenticate with Demo token)
	authPayload := models.GoogleAuthRequest{
		Credential: "demo:tariq.rashid@warbabank.com:Tariq Al-Rashid",
	}
	body, _ := json.Marshal(authPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/auth/google, got %d: %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "session_token" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected session_token cookie to be set")
	}

	// 4. GET /api/auth/me (Authenticated with cookie)
	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(sessionCookie)
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
	if authMe.User.Name != "Tariq Al-Rashid" {
		t.Errorf("expected name Tariq Al-Rashid, got %s", authMe.User.Name)
	}

	// 5. POST /api/auth/logout
	req = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/auth/logout, got %d", rec.Code)
	}

	// 6. GET /api/auth/me (After logout)
	req = httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(sessionCookie)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	_ = json.Unmarshal(rec.Body.Bytes(), &meResp)
	meData, _ = json.Marshal(meResp.Data)
	_ = json.Unmarshal(meData, &authMe)
	if authMe.Authenticated {
		t.Errorf("expected authenticated=false after logout, got true")
	}
}
