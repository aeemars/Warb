package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"opportunity-engine/internal/engine"
	"opportunity-engine/internal/models"
	"opportunity-engine/internal/store"
)

// Handlers holds the HTTP handler methods.
type Handlers struct {
	store  *store.Store
	engine *engine.Engine
}

// NewHandlers creates new Handlers.
func NewHandlers(s *store.Store, e *engine.Engine) *Handlers {
	return &Handlers{store: s, engine: e}
}

// RegisterRoutes registers all API routes on the given mux.
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/clients", h.handleListClients)
	mux.HandleFunc("/api/clients/", h.handleClientRoutes)
	mux.HandleFunc("/api/opportunities", h.handleListOpportunities)
	mux.HandleFunc("/api/opportunities/", h.handleOpportunityRoutes)
	mux.HandleFunc("/api/portfolio/scan", h.handlePortfolioScan)
	mux.HandleFunc("/api/portfolio/summary", h.handlePortfolioSummary)
	mux.HandleFunc("/api/products", h.handleListProducts)
}

// --- Client Handlers ---

func (h *Handlers) handleListClients(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	clients, err := h.store.ListClients()
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
	// Parse: /api/clients/{id} or /api/clients/{id}/analyze
	path := strings.TrimPrefix(r.URL.Path, "/api/clients/")
	parts := strings.SplitN(path, "/", 2)
	clientID := parts[0]

	if clientID == "" {
		h.badRequest(w, "Client ID required")
		return
	}

	// /api/clients/{id}/analyze
	if len(parts) > 1 && parts[1] == "analyze" {
		h.handleAnalyzeClient(w, r, clientID)
		return
	}

	// /api/clients/{id}
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	client, err := h.store.GetClient(clientID)
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

func (h *Handlers) handleAnalyzeClient(w http.ResponseWriter, r *http.Request, clientID string) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w)
		return
	}

	log.Printf("[API] Analyzing client: %s", clientID)

	opportunities, err := h.engine.AnalyzeClient(r.Context(), clientID)
	if err != nil {
		h.serverError(w, "AI analysis failed", err)
		return
	}
	if opportunities == nil {
		opportunities = []models.Opportunity{}
	}

	h.jsonOK(w, opportunities)
}

// --- Opportunity Handlers ---

func (h *Handlers) handleListOpportunities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.methodNotAllowed(w)
		return
	}

	status := r.URL.Query().Get("status")
	urgency := r.URL.Query().Get("urgency")
	clientID := r.URL.Query().Get("client_id")

	opps, err := h.store.ListOpportunities(status, urgency, clientID)
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

	// Validate status
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

	if err := h.store.UpdateOpportunityStatus(oppID, req.Status); err != nil {
		h.serverError(w, "Failed to update opportunity", err)
		return
	}

	h.jsonOK(w, map[string]string{"status": "updated", "id": oppID, "new_status": string(req.Status)})
}

// --- Portfolio Handlers ---

func (h *Handlers) handlePortfolioScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.methodNotAllowed(w)
		return
	}

	log.Printf("[API] Starting portfolio scan")

	opportunities, err := h.engine.ScanPortfolio(r.Context())
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

	summary, err := h.store.GetPortfolioSummary()
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
