package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"opportunity-engine/internal/models"
	"opportunity-engine/internal/store"

	openai "github.com/sashabaranov/go-openai"
)

// openRouterTransport adds OpenRouter-specific headers to requests.
type openRouterTransport struct {
	base http.RoundTripper
}

func (t *openRouterTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("HTTP-Referer", "https://warba-opportunity-engine.demo")
	req.Header.Set("X-Title", "Warba Proactive Opportunity Engine")
	return t.base.RoundTrip(req)
}

// Engine is the AI-powered opportunity detection engine.
type Engine struct {
	client *openai.Client
	model  string
	store  *store.Store
}

// New creates a new Engine connected to OpenRouter.
func New(apiKey, model string, s *store.Store) *Engine {
	httpClient := &http.Client{
		Timeout: 120 * time.Second,
		Transport: &openRouterTransport{
			base: http.DefaultTransport,
		},
	}

	config := openai.DefaultConfig(apiKey)
	config.BaseURL = "https://openrouter.ai/api/v1"
	config.HTTPClient = httpClient

	return &Engine{
		client: openai.NewClientWithConfig(config),
		model:  model,
		store:  s,
	}
}

// AnalyzeClient performs AI analysis on a single client and returns opportunity suggestions.
func (e *Engine) AnalyzeClient(ctx context.Context, clientID string) ([]models.Opportunity, error) {
	// Fetch client with full context
	client, err := e.store.GetClient(clientID)
	if err != nil {
		return nil, fmt.Errorf("fetching client: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	// Format current products
	var prodLines []string
	for _, cp := range client.CurrentProducts {
		prodLines = append(prodLines, fmt.Sprintf("- %s (%s): KWD %s, Status: %s, Since: %s",
			cp.ProductName, cp.ProductID, formatAmount(cp.AmountKWD), cp.Status, cp.StartDate))
	}
	productsStr := "None"
	if len(prodLines) > 0 {
		productsStr = strings.Join(prodLines, "\n")
	}

	// Format interactions
	var intLines []string
	for _, i := range client.Interactions {
		intLines = append(intLines, fmt.Sprintf("- [%s] %s: %s (Outcome: %s)",
			i.Date, i.Type, i.Summary, i.Outcome))
	}
	interactionsStr := "No recent interactions"
	if len(intLines) > 0 {
		interactionsStr = strings.Join(intLines, "\n")
	}

	// Build the prompt
	prompt := FormatClientAnalysisPrompt(
		client.Name, client.Industry, client.SubIndustry,
		formatAmount(client.RevenueKWD), client.EmployeeCount, client.IncorporationYear,
		client.Country, string(client.RiskRating), string(client.KYCStatus),
		client.RelationshipStart, productsStr, interactionsStr, client.Notes,
	)

	log.Printf("[Engine] Analyzing client: %s (%s)", client.Name, clientID)

	// Call OpenRouter
	resp, err := e.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: e.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: SystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0.3, // Low temperature for consistent, analytical output
		MaxTokens:   2000,
	})
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI model")
	}

	content := resp.Choices[0].Message.Content
	log.Printf("[Engine] AI response for %s: %s", clientID, truncate(content, 200))

	// Parse AI suggestions
	suggestions, err := parseAISuggestions(content)
	if err != nil {
		return nil, fmt.Errorf("parsing AI response: %w", err)
	}

	// Convert to opportunities and persist
	var opportunities []models.Opportunity
	now := time.Now()
	for _, s := range suggestions {
		opp := models.Opportunity{
			ID:           fmt.Sprintf("opp-%s-%s-%d", clientID, s.ProductID, now.UnixMilli()),
			ClientID:     clientID,
			ClientName:   client.Name,
			ProductID:    s.ProductID,
			Confidence:   s.Confidence,
			Urgency:      models.Urgency(s.Urgency),
			Reasoning:    s.Reasoning,
			NextAction:   s.NextAction,
			ShariahNotes: s.ShariahNotes,
			Status:       models.OpportunityNew,
			CreatedAt:    now,
		}

		// Look up product name
		prod, _ := e.store.GetProduct(s.ProductID)
		if prod != nil {
			opp.ProductName = prod.Name
		}

		// Persist
		if err := e.store.InsertOpportunity(opp); err != nil {
			log.Printf("[Engine] Warning: failed to persist opportunity: %v", err)
		}

		opportunities = append(opportunities, opp)
		now = now.Add(time.Millisecond) // Ensure unique IDs
	}

	log.Printf("[Engine] Generated %d opportunities for %s", len(opportunities), client.Name)
	return opportunities, nil
}

// ScanPortfolio analyzes all clients and returns prioritized opportunities.
func (e *Engine) ScanPortfolio(ctx context.Context) ([]models.Opportunity, error) {
	clients, err := e.store.ListClients()
	if err != nil {
		return nil, fmt.Errorf("listing clients: %w", err)
	}

	log.Printf("[Engine] Scanning portfolio: %d clients", len(clients))

	// Build portfolio summary for AI
	var clientSummaries []string
	for _, c := range clients {
		// Fetch full client data
		full, err := e.store.GetClient(c.ID)
		if err != nil || full == nil {
			continue
		}

		var prods []string
		for _, cp := range full.CurrentProducts {
			prods = append(prods, cp.ProductName)
		}
		prodsStr := "None"
		if len(prods) > 0 {
			prodsStr = strings.Join(prods, ", ")
		}

		var recentInt string
		if len(full.Interactions) > 0 {
			recentInt = full.Interactions[0].Summary
		}

		summary := fmt.Sprintf(`[%s] %s | Industry: %s | Revenue: KWD %s | Risk: %s | Products: %s | Recent: %s | Notes: %s`,
			c.ID, c.Name, c.Industry, formatAmount(c.RevenueKWD), c.RiskRating, prodsStr, recentInt, c.Notes)
		clientSummaries = append(clientSummaries, summary)
	}

	portfolioText := strings.Join(clientSummaries, "\n\n")
	prompt := fmt.Sprintf(PortfolioScanPrompt, portfolioText)

	resp, err := e.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: e.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: SystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   3000,
	})
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI model")
	}

	content := resp.Choices[0].Message.Content
	log.Printf("[Engine] Portfolio scan response: %s", truncate(content, 300))

	// Parse portfolio suggestions
	var portfolioSuggestions []PortfolioSuggestion
	cleaned := cleanJSON(content)
	if err := json.Unmarshal([]byte(cleaned), &portfolioSuggestions); err != nil {
		return nil, fmt.Errorf("parsing portfolio scan response: %w (content: %s)", err, truncate(content, 500))
	}

	// Convert to opportunities
	var opportunities []models.Opportunity
	now := time.Now()
	for _, s := range portfolioSuggestions {
		opp := models.Opportunity{
			ID:           fmt.Sprintf("opp-%s-%s-%d", s.ClientID, s.ProductID, now.UnixMilli()),
			ClientID:     s.ClientID,
			ProductID:    s.ProductID,
			Confidence:   s.Confidence,
			Urgency:      models.Urgency(s.Urgency),
			Reasoning:    s.Reasoning,
			NextAction:   s.NextAction,
			ShariahNotes: s.ShariahNotes,
			Status:       models.OpportunityNew,
			CreatedAt:    now,
		}

		// Look up names
		client, _ := e.store.GetClient(s.ClientID)
		if client != nil {
			opp.ClientName = client.Name
		}
		prod, _ := e.store.GetProduct(s.ProductID)
		if prod != nil {
			opp.ProductName = prod.Name
		}

		if err := e.store.InsertOpportunity(opp); err != nil {
			log.Printf("[Engine] Warning: failed to persist opportunity: %v", err)
		}

		opportunities = append(opportunities, opp)
		now = now.Add(time.Millisecond)
	}

	log.Printf("[Engine] Portfolio scan generated %d opportunities", len(opportunities))
	return opportunities, nil
}

// --- Helper functions ---

// parseAISuggestions extracts structured suggestions from the AI response.
func parseAISuggestions(content string) ([]models.AIOpportunitySuggestion, error) {
	cleaned := cleanJSON(content)

	var suggestions []models.AIOpportunitySuggestion
	if err := json.Unmarshal([]byte(cleaned), &suggestions); err != nil {
		return nil, fmt.Errorf("JSON unmarshal failed: %w (content: %s)", err, truncate(cleaned, 500))
	}
	return suggestions, nil
}

// cleanJSON strips markdown code blocks and whitespace from AI output.
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	// Remove markdown code fences
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	return s
}

// formatAmount formats a KWD amount with commas.
func formatAmount(amount float64) string {
	if amount >= 1000000 {
		return fmt.Sprintf("%.1fM", amount/1000000)
	}
	if amount >= 1000 {
		return fmt.Sprintf("%.0fK", amount/1000)
	}
	return fmt.Sprintf("%.0f", amount)
}

// truncate shortens a string to n characters.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
