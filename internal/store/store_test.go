package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"opportunity-engine/internal/models"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	tmpDir, err := os.MkdirTemp("", "store-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create store: %v", err)
	}

	cleanup := func() {
		s.Close()
		os.RemoveAll(tmpDir)
	}
	return s, cleanup
}

func TestStoreCRUDAndSeed(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	empty, err := s.IsEmpty()
	if err != nil {
		t.Fatalf("IsEmpty failed: %v", err)
	}
	if !empty {
		t.Errorf("expected empty store, got empty=false")
	}

	if err := s.Seed(); err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	empty, err = s.IsEmpty()
	if err != nil {
		t.Fatalf("IsEmpty after seed failed: %v", err)
	}
	if empty {
		t.Errorf("expected non-empty store after seed")
	}

	// Test ListClients
	clients, err := s.ListClients()
	if err != nil {
		t.Fatalf("ListClients failed: %v", err)
	}
	if len(clients) != 20 {
		t.Errorf("expected 20 clients, got %d", len(clients))
	}

	// Test GetClient
	client, err := s.GetClient("cli-001")
	if err != nil {
		t.Fatalf("GetClient failed: %v", err)
	}
	if client == nil {
		t.Fatalf("expected client cli-001, got nil")
	}
	if len(client.CurrentProducts) == 0 {
		t.Errorf("expected client products, got 0")
	}
	if len(client.Interactions) == 0 {
		t.Errorf("expected client interactions, got 0")
	}

	// Test Products
	products, err := s.ListProducts()
	if err != nil {
		t.Fatalf("ListProducts failed: %v", err)
	}
	if len(products) != 10 {
		t.Errorf("expected 10 products, got %d", len(products))
	}

	// Test Opportunities
	opp := models.Opportunity{
		ID:           "test-opp-1",
		ClientID:     "cli-001",
		ProductID:    "prod-001",
		Confidence:   0.92,
		Urgency:      models.UrgencyHigh,
		Reasoning:    "Test reasoning",
		NextAction:   "Test next action",
		ShariahNotes: "Test shariah notes",
		Status:       models.OpportunityNew,
		CreatedAt:    time.Now(),
	}
	if err := s.InsertOpportunity(opp); err != nil {
		t.Fatalf("InsertOpportunity failed: %v", err)
	}

	opps, err := s.ListOpportunities("", "", "")
	if err != nil {
		t.Fatalf("ListOpportunities failed: %v", err)
	}
	if len(opps) != 1 {
		t.Errorf("expected 1 opportunity, got %d", len(opps))
	}

	// Test UpdateOpportunityStatus
	if err := s.UpdateOpportunityStatus("test-opp-1", models.OpportunityAccepted); err != nil {
		t.Fatalf("UpdateOpportunityStatus failed: %v", err)
	}

	// Test Portfolio Summary
	summary, err := s.GetPortfolioSummary()
	if err != nil {
		t.Fatalf("GetPortfolioSummary failed: %v", err)
	}
	if summary.TotalClients != 20 {
		t.Errorf("expected 20 clients in summary, got %d", summary.TotalClients)
	}
	if summary.AcceptedOpps != 1 {
		t.Errorf("expected 1 accepted opportunity, got %d", summary.AcceptedOpps)
	}
	if len(summary.TopIndustries) == 0 {
		t.Errorf("expected top industries, got 0")
	}
}

func TestUserAndSessionStore(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	// 1. Test insert new Google User
	u := &models.User{
		GoogleID: "google-123456",
		Email:    "ahmad.mutairi@example.com",
		Name:     "Ahmad Al-Mutairi",
		Avatar:   "https://lh3.googleusercontent.com/a/test-avatar",
		Role:     "Senior Relationship Manager",
	}

	saved, err := s.UpsertGoogleUser(u)
	if err != nil {
		t.Fatalf("UpsertGoogleUser failed: %v", err)
	}
	if saved.ID == "" {
		t.Errorf("expected user ID to be generated, got empty")
	}
	if saved.Email != u.Email {
		t.Errorf("expected email %s, got %s", u.Email, saved.Email)
	}

	// 2. Test update existing Google User
	uUpdate := &models.User{
		GoogleID: "google-123456",
		Email:    "ahmad.mutairi@example.com",
		Name:     "Ahmad Al-Mutairi (Updated)",
		Avatar:   "https://lh3.googleusercontent.com/a/new-avatar",
	}
	updated, err := s.UpsertGoogleUser(uUpdate)
	if err != nil {
		t.Fatalf("UpsertGoogleUser update failed: %v", err)
	}
	if updated.ID != saved.ID {
		t.Errorf("expected same user ID %s, got %s", saved.ID, updated.ID)
	}
	if updated.Name != "Ahmad Al-Mutairi (Updated)" {
		t.Errorf("expected updated name, got %s", updated.Name)
	}

	// 3. Test Session creation and retrieval
	token := "sess-tok-12345"
	expiresAt := time.Now().Add(24 * time.Hour)
	if err := s.CreateSession(token, saved.ID, expiresAt); err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	sessionUser, err := s.GetUserBySession(token)
	if err != nil {
		t.Fatalf("GetUserBySession failed: %v", err)
	}
	if sessionUser == nil {
		t.Fatalf("expected session user, got nil")
	}
	if sessionUser.ID != saved.ID {
		t.Errorf("expected user ID %s, got %s", saved.ID, sessionUser.ID)
	}

	// 4. Test Session Deletion
	if err := s.DeleteSession(token); err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}
	sessionUserAfter, err := s.GetUserBySession(token)
	if err != nil {
		t.Fatalf("GetUserBySession after delete failed: %v", err)
	}
	if sessionUserAfter != nil {
		t.Errorf("expected nil after delete, got %v", sessionUserAfter)
	}
}
