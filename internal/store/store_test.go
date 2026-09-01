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
		t.Fatalf("failed to open store: %v", err)
	}

	cleanup := func() {
		s.Close()
		os.RemoveAll(tmpDir)
	}
	return s, cleanup
}

func TestStoreCRUDAndUserIsolation(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	// 1. Seed Global Products
	if err := s.Seed(); err != nil {
		t.Fatalf("seeding failed: %v", err)
	}

	products, err := s.ListProducts()
	if err != nil {
		t.Fatalf("listing products failed: %v", err)
	}
	if len(products) != 10 {
		t.Errorf("expected 10 products, got %d", len(products))
	}

	// 2. Create User 1 and User 2
	user1, _, err := s.UpsertGoogleUser(&models.User{
		GoogleID: "gid-user-1",
		Email:    "user1@warbabank.com",
		Name:     "Tariq Al-Rashid",
	})
	if err != nil {
		t.Fatalf("upsert user 1 failed: %v", err)
	}

	user2, _, err := s.UpsertGoogleUser(&models.User{
		GoogleID: "gid-user-2",
		Email:    "user2@warbabank.com",
		Name:     "Fatima Al-Rashidi",
	})
	if err != nil {
		t.Fatalf("upsert user 2 failed: %v", err)
	}

	// 3. Seed portfolio for User 1 only
	if err := s.SeedUserPortfolio(user1.ID, user1.Name); err != nil {
		t.Fatalf("seeding user 1 portfolio failed: %v", err)
	}

	// 4. Verify User 1 has 20 clients, but User 2 has 0 clients
	u1Clients, err := s.ListClients(user1.ID)
	if err != nil {
		t.Fatalf("listing user 1 clients failed: %v", err)
	}
	if len(u1Clients) != 20 {
		t.Errorf("expected 20 clients for user 1, got %d", len(u1Clients))
	}

	u2Clients, err := s.ListClients(user2.ID)
	if err != nil {
		t.Fatalf("listing user 2 clients failed: %v", err)
	}
	if len(u2Clients) != 0 {
		t.Errorf("expected 0 clients for user 2 before seeding, got %d", len(u2Clients))
	}

	// 5. Verify User 1 Opportunities & Summary
	summary1, err := s.GetPortfolioSummary(user1.ID)
	if err != nil {
		t.Fatalf("get user 1 summary failed: %v", err)
	}
	if summary1.TotalClients != 20 {
		t.Errorf("expected 20 total clients in summary, got %d", summary1.TotalClients)
	}
	if summary1.TotalOpportunities == 0 {
		t.Errorf("expected >0 opportunities for user 1")
	}

	summary2, err := s.GetPortfolioSummary(user2.ID)
	if err != nil {
		t.Fatalf("get user 2 summary failed: %v", err)
	}
	if summary2.TotalClients != 0 {
		t.Errorf("expected 0 total clients for user 2, got %d", summary2.TotalClients)
	}

	// 6. Test Opportunity status updates for User 1
	opps, err := s.ListOpportunities(user1.ID, "", "", "")
	if err != nil || len(opps) == 0 {
		t.Fatalf("failed to list user 1 opportunities: %v", err)
	}

	firstOpp := opps[0]
	if err := s.UpdateOpportunityStatus(user1.ID, firstOpp.ID, models.OpportunityAccepted); err != nil {
		t.Fatalf("updating opportunity status failed: %v", err)
	}

	// Verify User 2 cannot update User 1's opportunity
	if err := s.UpdateOpportunityStatus(user2.ID, firstOpp.ID, models.OpportunityConverted); err == nil {
		t.Errorf("expected error when user 2 updates user 1 opportunity, got nil")
	}
}

func TestUserAndSessionStore(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	// 1. Create a user
	u := &models.User{
		GoogleID: "test-google-id-12345",
		Email:    "ahmad.mutairi@warbabank.com",
		Name:     "Ahmad Al-Mutairi",
		Avatar:   "https://lh3.googleusercontent.com/a/test-avatar",
		Role:     "Senior Relationship Manager",
	}

	saved, _, err := s.UpsertGoogleUser(u)
	if err != nil {
		t.Fatalf("UpsertGoogleUser failed: %v", err)
	}
	if saved.ID == "" || saved.Email != u.Email {
		t.Errorf("unexpected saved user: %+v", saved)
	}

	// 2. Create a session
	token := "session-token-random-123456789"
	err = s.CreateSession(token, saved.ID, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// 3. Retrieve user by session
	fetched, err := s.GetUserBySession(token)
	if err != nil {
		t.Fatalf("GetUserBySession failed: %v", err)
	}
	if fetched == nil || fetched.ID != saved.ID || fetched.Email != saved.Email {
		t.Errorf("unexpected user from session: %+v", fetched)
	}

	// 4. Delete session and verify
	err = s.DeleteSession(token)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	fetchedAfterDelete, err := s.GetUserBySession(token)
	if err != nil {
		t.Fatalf("GetUserBySession after delete failed: %v", err)
	}
	if fetchedAfterDelete != nil {
		t.Errorf("expected nil user after session delete, got %+v", fetchedAfterDelete)
	}
}
