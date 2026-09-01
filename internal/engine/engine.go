package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"opportunity-engine/internal/models"
	"opportunity-engine/internal/store"

	openai "github.com/sashabaranov/go-openai"
)

// FINDING-03 FIX: Strip HTML/script tags from AI output before persistence.
var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

func sanitizeAIField(s string) string {
	// Remove any HTML tags
	s = htmlTagRegex.ReplaceAllString(s, "")
	// Remove javascript: URLs
	s = strings.ReplaceAll(s, "javascript:", "")
	// Limit field length to prevent DB bloat
	if len(s) > 2000 {
		s = s[:2000] + "..."
	}
	return strings.TrimSpace(s)
}

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
func (e *Engine) AnalyzeClient(ctx context.Context, userID, clientID string) ([]models.Opportunity, error) {
	// Fetch client with full context
	client, err := e.store.GetClient(userID, clientID)
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
		MaxTokens:   8000,
	})
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI model")
	}

	choice := resp.Choices[0]
	content := choice.Message.Content
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("AI returned empty content (finish_reason: %s). Increase token budget or switch model", choice.FinishReason)
	}
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
		// FINDING-03 FIX: Sanitize all AI-generated text fields before persistence.
		opp := models.Opportunity{
			ID:           fmt.Sprintf("opp-%s-%s-%d", clientID, s.ProductID, now.UnixMilli()),
			UserID:       userID,
			ClientID:     clientID,
			ClientName:   client.Name,
			ProductID:    s.ProductID,
			Confidence:   s.Confidence,
			Urgency:      models.Urgency(sanitizeAIField(s.Urgency)),
			Reasoning:    sanitizeAIField(s.Reasoning),
			NextAction:   sanitizeAIField(s.NextAction),
			ShariahNotes: sanitizeAIField(s.ShariahNotes),
			Status:       models.OpportunityNew,
			CreatedAt:    now,
		}

		// Validate confidence range
		if opp.Confidence < 0 {
			opp.Confidence = 0
		} else if opp.Confidence > 1 {
			opp.Confidence = 1
		}

		// Validate product_id exists in catalog
		prod, _ := e.store.GetProduct(s.ProductID)
		if prod != nil {
			opp.ProductName = prod.Name
		} else {
			log.Printf("[Engine] Warning: AI suggested unknown product_id %q, skipping", s.ProductID)
			continue
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
func (e *Engine) ScanPortfolio(ctx context.Context, userID string) ([]models.Opportunity, error) {
	clients, err := e.store.ListClients(userID)
	if err != nil {
		return nil, fmt.Errorf("listing clients: %w", err)
	}

	log.Printf("[Engine] Scanning portfolio: %d clients", len(clients))

	// Build portfolio summary for AI
	var clientSummaries []string
	for _, c := range clients {
		// Fetch full client data
		full, err := e.store.GetClient(userID, c.ID)
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
		MaxTokens:   8000,
	})
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI model")
	}

	choice := resp.Choices[0]
	content := choice.Message.Content
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("AI returned empty content (finish_reason: %s). Increase token budget or switch model", choice.FinishReason)
	}
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
		// FINDING-03 FIX: Sanitize all AI-generated text fields before persistence.
		opp := models.Opportunity{
			ID:           fmt.Sprintf("opp-%s-%s-%d", s.ClientID, s.ProductID, now.UnixMilli()),
			UserID:       userID,
			ClientID:     s.ClientID,
			ProductID:    s.ProductID,
			Confidence:   s.Confidence,
			Urgency:      models.Urgency(sanitizeAIField(s.Urgency)),
			Reasoning:    sanitizeAIField(s.Reasoning),
			NextAction:   sanitizeAIField(s.NextAction),
			ShariahNotes: sanitizeAIField(s.ShariahNotes),
			Status:       models.OpportunityNew,
			CreatedAt:    now,
		}

		// Validate confidence range
		if opp.Confidence < 0 {
			opp.Confidence = 0
		} else if opp.Confidence > 1 {
			opp.Confidence = 1
		}

		// Look up and validate names
		client, _ := e.store.GetClient(userID, s.ClientID)
		if client != nil {
			opp.ClientName = client.Name
		}
		prod, _ := e.store.GetProduct(s.ProductID)
		if prod != nil {
			opp.ProductName = prod.Name
		} else {
			log.Printf("[Engine] Warning: AI suggested unknown product_id %q, skipping", s.ProductID)
			continue
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

// cleanJSON strips reasoning/thinking tags, markdown code blocks, and isolates the JSON array.
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)

	// Remove thinking tags if present
	for {
		start := strings.Index(s, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(s, "</think>")
		if end == -1 {
			s = s[start+7:]
			break
		}
		s = s[:start] + s[end+8:]
	}

	// Remove markdown code fences if present
	if start := strings.Index(s, "```json"); start != -1 {
		rest := s[start+7:]
		if end := strings.Index(rest, "```"); end != -1 {
			s = rest[:end]
		}
	} else if start := strings.Index(s, "```"); start != -1 {
		rest := s[start+3:]
		if end := strings.Index(rest, "```"); end != -1 {
			s = rest[:end]
		}
	}

	s = strings.TrimSpace(s)

	// Isolate JSON array from first '[' to last ']'
	startIdx := strings.Index(s, "[")
	endIdx := strings.LastIndex(s, "]")
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		s = s[startIdx : endIdx+1]
	}

	return strings.TrimSpace(s)
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
