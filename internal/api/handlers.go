package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"opportunity-engine/internal/auth"
	"opportunity-engine/internal/engine"
	"opportunity-engine/internal/models"
	"opportunity-engine/internal/store"
)

// Handlers holds dependencies for HTTP route handling.
type Handlers struct {
	store          *store.Store
	engine         *engine.Engine
	googleClientID string
}

// NewHandlers creates a new Handlers instance.
func NewHandlers(s *store.Store, e *engine.Engine, googleClientID string) *Handlers {
	return &Handlers{
		store:          s,
		engine:         e,
		googleClientID: strings.TrimSpace(googleClientID),
	}
}

// RegisterRoutes registers all API routes on the given mux.
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	// API Root Index
	mux.HandleFunc("/api", h.handleAPIRoot)
	mux.HandleFunc("/api/", h.handleAPIRoot)

	// Auth routes
	mux.HandleFunc("/api/auth/config", h.handleAuthConfig)
	mux.HandleFunc("/api/auth/google", h.handleGoogleAuth)
	mux.HandleFunc("/api/auth/me", h.handleAuthMe)
	mux.HandleFunc("/api/auth/logout", h.handleAuthLogout)

	// Domain routes
	mux.HandleFunc("/api/clients", h.handleListClients)
	mux.HandleFunc("/api/clients/", h.handleClientRoutes)
	mux.HandleFunc("/api/opportunities", h.handleListOpportunities)
	mux.HandleFunc("/api/opportunities/", h.handleOpportunityRoutes)
	mux.HandleFunc("/api/portfolio/scan", h.handlePortfolioScan)
	mux.HandleFunc("/api/portfolio/summary", h.handlePortfolioSummary)
	mux.HandleFunc("/api/products", h.handleListProducts)
}

// --- API Root Handler ---

func (h *Handlers) handleAPIRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api" && r.URL.Path != "/api/" {
		h.notFound(w, "Endpoint not found")
		return
	}
	h.jsonOK(w, map[string]interface{}{
		"name":        "Warba Bank — Proactive Client Opportunity Engine API",
		"status":      "online",
		"dashboard":   "/",
		"auth_google": h.googleClientID != "",
		"endpoints": map[string]string{
			"GET  /api/auth/config":         "Get public auth configuration and Google Client ID",
			"POST /api/auth/google":         "Authenticate with Google ID token credential",
			"GET  /api/auth/me":             "Get currently authenticated user profile",
			"POST /api/auth/logout":         "Log out and invalidate session",
			"GET  /api/clients":             "List user's corporate clients",
			"GET  /api/clients/:id":         "Get single client with history and product holdings",
			"POST /api/clients/:id/analyze": "Trigger AI opportunity analysis for a client",
			"GET  /api/opportunities":      "List user's opportunities (filters: status, urgency, client_id)",
			"PATCH /api/opportunities/:id":  "Update opportunity status (New, Reviewed, Accepted, Dismissed, Converted)",
			"POST /api/portfolio/scan":      "Trigger AI scan across user's portfolio",
			"GET  /api/portfolio/summary":   "Get user's portfolio metrics and breakdown stats",
			"GET  /api/products":            "List Warba Bank Shariah-compliant product catalog",
		},
	})
}

// --- Authentication Handlers ---

func (h *Handlers) handleAuthConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}
	clientID := strings.TrimSpace(h.googleClientID)
	h.jsonOK(w, models.AuthConfigResponse{
		GoogleClientID: clientID,
		Enabled:        clientID != "",
	})
}

func (h *Handlers) handleGoogleAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w)
		return
	}

	var req models.GoogleAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "Invalid request body")
		return
	}

	if req.Credential == "" {
		h.badRequest(w, "Google credential (ID token) required")
		return
	}

	// Verify Google ID Token
	profile, err := auth.VerifyGoogleIDToken(req.Credential, h.googleClientID)
	if err != nil {
		log.Printf("[Auth] Google token verification failed: %v", err)
		h.badRequest(w, "Invalid Google ID token: "+err.Error())
		return
	}

	user := &models.User{
		GoogleID: profile.Sub,
		Email:    profile.Email,
		Name:     profile.Name,
		Avatar:   profile.Picture,
		Role:     "Senior Relationship Manager",
	}

	// Upsert user in database
	savedUser, err := h.store.UpsertGoogleUser(user)
	if err != nil {
		h.serverError(w, "Failed to save user account", err)
		return
	}

	// Auto-seed personalized client portfolio for first-time user
	if err := h.store.SeedUserPortfolio(savedUser.ID, savedUser.Name); err != nil {
		log.Printf("[Auth] Warning: portfolio seed failed for %s: %v", savedUser.ID, err)
	}

	// Create Session Token
	sessionToken := generateSecureToken(32)
	expiresAt := time.Now().Add(7 * 24 * time.Hour) // 7 days
	if err := h.store.CreateSession(sessionToken, savedUser.ID, expiresAt); err != nil {
		h.serverError(w, "Failed to create user session", err)
		return
	}

	// Set HTTP Session Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	log.Printf("[Auth] User successfully logged in: %s (%s)", savedUser.Name, savedUser.Email)
	h.jsonOK(w, map[string]interface{}{
		"user":          savedUser,
		"authenticated": true,
		"token":         sessionToken,
	})
}

func (h *Handlers) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	token := extractSessionToken(r)
	if token == "" {
		h.jsonOK(w, models.AuthUserResponse{User: nil, Authenticated: false})
		return
	}

	user, err := h.store.GetUserBySession(token)
	if err != nil || user == nil {
		h.jsonOK(w, models.AuthUserResponse{User: nil, Authenticated: false})
		return
	}

	h.jsonOK(w, models.AuthUserResponse{User: user, Authenticated: true})
}

func (h *Handlers) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w)
		return
	}

	token := extractSessionToken(r)
	if token != "" {
		_ = h.store.DeleteSession(token)
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	h.jsonOK(w, map[string]interface{}{
		"logged_out": true,
	})
}

func (h *Handlers) requireUser(w http.ResponseWriter, r *http.Request) (*models.User, bool) {
	token := extractSessionToken(r)
	if token == "" {
		h.unauthorized(w, "Authentication required. Please sign in with Google.")
		return nil, false
	}
	user, err := h.store.GetUserBySession(token)
	if err != nil || user == nil {
		h.unauthorized(w, "Invalid or expired session. Please sign in again.")
		return nil, false
	}
	return user, true
}

func extractSessionToken(r *http.Request) string {
	if c, err := r.Cookie("session_token"); err == nil && c.Value != "" {
		return c.Value
	}
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}
	return ""
}

func generateSecureToken(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// --- Client Handlers (User Scoped) ---

func (h *Handlers) handleListClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	clients, err := h.store.ListClients(user.ID)
	if err != nil {
		h.serverError(w, "Failed to list clients", err)
		return
	}
	if clients == nil {
		clients = []models.Client{}
	}
	h.jsonOK(w, clients)
}

func (h *Handlers) handleClientRoutes(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/clients/")
	parts := strings.SplitN(path, "/", 2)
	clientID := parts[0]

	if clientID == "" {
		h.badRequest(w, "Client ID required")
		return
	}

	if len(parts) > 1 && parts[1] == "analyze" {
		h.handleAnalyzeClient(w, r, user.ID, clientID)
		return
	}

	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	client, err := h.store.GetClient(user.ID, clientID)
	if err != nil {
		h.serverError(w, "Failed to get client", err)
		return
	}
	if client == nil {
		h.notFound(w, "Client not found")
		return
	}
	h.jsonOK(w, client)
}

func (h *Handlers) handleAnalyzeClient(w http.ResponseWriter, r *http.Request, userID, clientID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w)
		return
	}

	log.Printf("[API] Analyzing client %s for user %s", clientID, userID)

	opportunities, err := h.engine.AnalyzeClient(r.Context(), userID, clientID)
	if err != nil {
		h.serverError(w, "AI analysis failed", err)
		return
	}
	if opportunities == nil {
		opportunities = []models.Opportunity{}
	}

	h.jsonOK(w, opportunities)
}

// --- Opportunity Handlers (User Scoped) ---

func (h *Handlers) handleListOpportunities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	status := r.URL.Query().Get("status")
	urgency := r.URL.Query().Get("urgency")
	clientID := r.URL.Query().Get("client_id")

	opps, err := h.store.ListOpportunities(user.ID, status, urgency, clientID)
	if err != nil {
		h.serverError(w, "Failed to list opportunities", err)
		return
	}
	if opps == nil {
		opps = []models.Opportunity{}
	}
	h.jsonOK(w, opps)
}

func (h *Handlers) handleOpportunityRoutes(w http.ResponseWriter, r *http.Request) {
	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	oppID := strings.TrimPrefix(r.URL.Path, "/api/opportunities/")
	if oppID == "" {
		h.badRequest(w, "Opportunity ID required")
		return
	}

	if r.Method != http.MethodPatch {
		h.methodNotAllowed(w)
		return
	}

	var req models.StatusUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.badRequest(w, "Invalid request body")
		return
	}

	validStatuses := map[models.OpportunityStatus]bool{
		models.OpportunityNew:       true,
		models.OpportunityReviewed:  true,
		models.OpportunityAccepted:  true,
		models.OpportunityDismissed: true,
		models.OpportunityConverted: true,
	}
	if !validStatuses[req.Status] {
		h.badRequest(w, "Invalid status. Must be: New, Reviewed, Accepted, Dismissed, or Converted")
		return
	}

	if err := h.store.UpdateOpportunityStatus(user.ID, oppID, req.Status); err != nil {
		h.serverError(w, "Failed to update opportunity", err)
		return
	}

	h.jsonOK(w, map[string]string{"status": "updated", "id": oppID, "new_status": string(req.Status)})
}

// --- Portfolio Handlers (User Scoped) ---

func (h *Handlers) handlePortfolioScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w)
		return
	}

	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	log.Printf("[API] Starting portfolio scan for user: %s (%s)", user.Name, user.ID)

	opportunities, err := h.engine.ScanPortfolio(r.Context(), user.ID)
	if err != nil {
		h.serverError(w, "Portfolio scan failed", err)
		return
	}
	if opportunities == nil {
		opportunities = []models.Opportunity{}
	}

	h.jsonOK(w, opportunities)
}

func (h *Handlers) handlePortfolioSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	user, ok := h.requireUser(w, r)
	if !ok {
		return
	}

	summary, err := h.store.GetPortfolioSummary(user.ID)
	if err != nil {
		h.serverError(w, "Failed to get portfolio summary", err)
		return
	}
	h.jsonOK(w, summary)
}

// --- Product Handlers ---

func (h *Handlers) handleListProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	products, err := h.store.ListProducts()
	if err != nil {
		h.serverError(w, "Failed to list products", err)
		return
	}
	if products == nil {
		products = []models.Product{}
	}
	h.jsonOK(w, products)
}

// --- Response Helpers ---

func (h *Handlers) jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(models.APIResponse{Success: true, Data: data})
}

func (h *Handlers) unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(models.APIResponse{Success: false, Error: msg})
}

func (h *Handlers) badRequest(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(models.APIResponse{Success: false, Error: msg})
}

func (h *Handlers) notFound(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(models.APIResponse{Success: false, Error: msg})
}

func (h *Handlers) methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	json.NewEncoder(w).Encode(models.APIResponse{Success: false, Error: "Method not allowed"})
}

func (h *Handlers) serverError(w http.ResponseWriter, msg string, err error) {
	log.Printf("[API] Error: %s: %v", msg, err)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(models.APIResponse{Success: false, Error: msg})
}
