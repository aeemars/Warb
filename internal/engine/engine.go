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
	req.Header.Set("HTTP-Referer", "https://warb-opportunity-engine.demo")
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
	// 1. Fetch client with full context
	client, err := e.store.GetClient(userID, clientID)
	if err != nil {
		return nil, fmt.Errorf("fetching client: %w", err)
	}
	if client == nil {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	// 2. Load active product catalog from database
	products, err := e.store.ListProducts()
	if err != nil || len(products) == 0 {
		return nil, fmt.Errorf("loading product catalog: %w", err)
	}

	// 3. Format current products
	var prodLines []string
	for _, cp := range client.CurrentProducts {
		pName := cp.ProductName
		if pName == "" {
			if resolved := resolveProduct(products, cp.ProductID); resolved != nil {
				pName = resolved.Name
			} else {
				pName = cp.ProductID
			}
		}
		prodLines = append(prodLines, fmt.Sprintf("- %s (%s): KWD %s, Status: %s, Since: %s",
			pName, cp.ProductID, formatAmount(cp.AmountKWD), cp.Status, cp.StartDate))
	}
	productsStr := "None"
	if len(prodLines) > 0 {
		productsStr = strings.Join(prodLines, "\n")
	}

	// 4. Format interaction logs
	var intLines []string
	for _, i := range client.Interactions {
		outStr := ""
		if i.Outcome != "" {
			outStr = fmt.Sprintf(" | Outcome: %s", i.Outcome)
		}
		intLines = append(intLines, fmt.Sprintf("[%s] %s (%s): %s%s", i.Date, i.Type, i.ID, i.Summary, outStr))
	}
	interactionsStr := "None"
	if len(intLines) > 0 {
		interactionsStr = strings.Join(intLines, "\n")
	}

	// 5. Compute holding gaps
	holdingGapsStr := computeHoldingGaps(products, client.CurrentProducts)

	// 6. Build dynamic system & user prompts
	catalogPrompt := BuildCatalogPrompt(products)
	systemPrompt := BuildSystemPrompt(catalogPrompt)
	userPrompt := FormatClientAnalysisPrompt(
		client.Name, client.NameAr, client.Industry, client.SubIndustry,
		formatAmount(client.RevenueKWD), client.EmployeeCount, client.IncorporationYear,
		client.Country, string(client.RiskRating), string(client.KYCStatus),
		client.RelationshipStart, client.Notes, productsStr, interactionsStr, holdingGapsStr,
	)

	log.Printf("[Engine] Analyzing client %s (%s) using model %s", client.Name, clientID, e.model)

	// 7. Call OpenRouter API
	resp, err := e.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: e.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
		Temperature: 0.2, // Low temperature for high precision & compliance
		MaxTokens:   4000,
	})
	if err != nil {
		log.Printf("[Engine] OpenRouter API call failed for %s: %v", clientID, err)
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI model")
	}

	choice := resp.Choices[0]
	content := choice.Message.Content
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("AI returned empty content (finish_reason: %s)", choice.FinishReason)
	}
	log.Printf("[Engine] AI response for %s: %s", clientID, truncate(content, 200))

	// 8. Parse AI suggestions
	suggestions, err := parseAISuggestions(content)
	if err != nil {
		log.Printf("[Engine] Failed to parse AI JSON for %s: %v (raw: %s)", clientID, err, truncate(content, 300))
		return nil, fmt.Errorf("parsing AI response: %w", err)
	}

	// 9. Convert to opportunities, resolve products against DB catalog, and persist
	var opportunities []models.Opportunity
	now := time.Now()
	for _, s := range suggestions {
		// Resolve product identifier (handles exact IDs, aliases, and fuzzy names)
		matchedProduct := resolveProduct(products, s.ProductID)
		if matchedProduct == nil {
			log.Printf("[Engine] Warning: Unrecognized product %q for client %s, skipping", s.ProductID, clientID)
			continue
		}

		urgency := normalizeUrgency(s.Urgency)
		confidence := s.Confidence
		if confidence <= 0 {
			confidence = 0.75
		} else if confidence > 1 {
			confidence = 1.0
		}

		opp := models.Opportunity{
			ID:           fmt.Sprintf("opp-%s-%s-%d", clientID, matchedProduct.ID, now.UnixMilli()),
			UserID:       userID,
			ClientID:     clientID,
			ClientName:   client.Name,
			ProductID:    matchedProduct.ID,
			ProductName:  matchedProduct.Name,
			Confidence:   confidence,
			Urgency:      urgency,
			Reasoning:    sanitizeAIField(s.Reasoning),
			NextAction:   sanitizeAIField(s.NextAction),
			ShariahNotes: sanitizeAIField(s.ShariahNotes),
			Status:       models.OpportunityNew,
			CreatedAt:    now,
		}

		if err := e.store.InsertOpportunity(opp); err != nil {
			log.Printf("[Engine] Warning: failed to persist opportunity: %v", err)
		}

		opportunities = append(opportunities, opp)
		now = now.Add(time.Millisecond) // Ensure unique timestamps
	}

	log.Printf("[Engine] Successfully generated and stored %d opportunities for %s", len(opportunities), client.Name)
	return opportunities, nil
}

// ScanPortfolio analyzes all clients in the RM's portfolio and generates ranked opportunities.
func (e *Engine) ScanPortfolio(ctx context.Context, userID string) ([]models.Opportunity, error) {
	clients, err := e.store.ListClients(userID)
	if err != nil {
		return nil, fmt.Errorf("listing clients: %w", err)
	}
	if len(clients) == 0 {
		return []models.Opportunity{}, nil
	}

	products, err := e.store.ListProducts()
	if err != nil || len(products) == 0 {
		return nil, fmt.Errorf("loading product catalog: %w", err)
	}

	log.Printf("[Engine] Scanning portfolio: %d clients for user %s", len(clients), userID)

	// Build rich client summaries for portfolio scan
	var clientSummaries []string
	for _, c := range clients {
		full, err := e.store.GetClient(userID, c.ID)
		if err != nil || full == nil {
			full = &c
		}

		var prods []string
		for _, cp := range full.CurrentProducts {
			pName := cp.ProductName
			if pName == "" {
				if r := resolveProduct(products, cp.ProductID); r != nil {
					pName = r.Name
				} else {
					pName = cp.ProductID
				}
			}
			prods = append(prods, fmt.Sprintf("%s (%s)", pName, formatAmount(cp.AmountKWD)))
		}
		prodsStr := "None"
		if len(prods) > 0 {
			prodsStr = strings.Join(prods, ", ")
		}

		var recentInt string
		if len(full.Interactions) > 0 {
			recentInt = fmt.Sprintf("%s (%s): %s", full.Interactions[0].Date, full.Interactions[0].Type, full.Interactions[0].Summary)
			if full.Interactions[0].Outcome != "" {
				recentInt += " -> " + full.Interactions[0].Outcome
			}
		} else {
			recentInt = "No recent interaction"
		}

		summary := fmt.Sprintf(`• Client ID: %s | %s (%s)
  Industry: %s (%s) | Revenue: KWD %s | Employees: %d | Risk: %s | KYC: %s
  Active Holdings: %s
  Latest Interaction: %s
  Notes: %s`,
			c.ID, c.Name, c.NameAr, c.Industry, c.SubIndustry, formatAmount(c.RevenueKWD),
			c.EmployeeCount, c.RiskRating, c.KYCStatus, prodsStr, recentInt, c.Notes)

		clientSummaries = append(clientSummaries, summary)
	}

	catalogPrompt := BuildCatalogPrompt(products)
	systemPrompt := BuildSystemPrompt(catalogPrompt)
	portfolioText := strings.Join(clientSummaries, "\n\n")
	prompt := fmt.Sprintf(PortfolioScanPromptTemplate, portfolioText)

	resp, err := e.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: e.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
		Temperature: 0.2,
		MaxTokens:   4000,
	})
	if err != nil {
		log.Printf("[Engine] Portfolio scan API call failed: %v", err)
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI model")
	}

	choice := resp.Choices[0]
	content := choice.Message.Content
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("AI returned empty content (finish_reason: %s)", choice.FinishReason)
	}
	log.Printf("[Engine] Portfolio scan response: %s", truncate(content, 300))

	// Parse portfolio suggestions
	var portfolioSuggestions []PortfolioSuggestion
	cleaned := cleanJSON(content)
	if err := json.Unmarshal([]byte(cleaned), &portfolioSuggestions); err != nil {
		return nil, fmt.Errorf("parsing portfolio scan response: %w (content: %s)", err, truncate(content, 500))
	}

	// Map clients by ID for quick lookup
	clientMap := make(map[string]models.Client)
	for _, cl := range clients {
		clientMap[cl.ID] = cl
	}

	// Convert and persist opportunities
	var opportunities []models.Opportunity
	now := time.Now()
	for _, s := range portfolioSuggestions {
		matchedProduct := resolveProduct(products, s.ProductID)
		if matchedProduct == nil {
			log.Printf("[Engine] Warning: Unrecognized product %q in portfolio scan, skipping", s.ProductID)
			continue
		}

		targetClientID := s.ClientID
		clientObj, exists := clientMap[targetClientID]
		if !exists {
			// Try fuzzy client match by name
			for cID, c := range clientMap {
				if strings.Contains(strings.ToLower(s.ClientID), strings.ToLower(c.Name)) ||
					strings.Contains(strings.ToLower(c.Name), strings.ToLower(s.ClientID)) {
					targetClientID = cID
					clientObj = c
					exists = true
					break
				}
			}
		}

		clientName := clientObj.Name
		if clientName == "" {
			clientName = targetClientID
		}

		urgency := normalizeUrgency(s.Urgency)
		confidence := s.Confidence
		if confidence <= 0 {
			confidence = 0.80
		} else if confidence > 1 {
			confidence = 1.0
		}

		opp := models.Opportunity{
			ID:           fmt.Sprintf("opp-%s-%s-%d", targetClientID, matchedProduct.ID, now.UnixMilli()),
			UserID:       userID,
			ClientID:     targetClientID,
			ClientName:   clientName,
			ProductID:    matchedProduct.ID,
			ProductName:  matchedProduct.Name,
			Confidence:   confidence,
			Urgency:      urgency,
			Reasoning:    sanitizeAIField(s.Reasoning),
			NextAction:   sanitizeAIField(s.NextAction),
			ShariahNotes: sanitizeAIField(s.ShariahNotes),
			Status:       models.OpportunityNew,
			CreatedAt:    now,
		}

		if err := e.store.InsertOpportunity(opp); err != nil {
			log.Printf("[Engine] Warning: failed to persist opportunity: %v", err)
		}

		opportunities = append(opportunities, opp)
		now = now.Add(time.Millisecond)
	}

	log.Printf("[Engine] Portfolio scan generated %d valid opportunities", len(opportunities))
	return opportunities, nil
}

// --- Helper Functions ---

// resolveProduct matches an AI-suggested product identifier to an active catalog product.
// Handles exact IDs, legacy numbered IDs (prod-001..prod-010), exact names, and fuzzy structure keywords.
func resolveProduct(products []models.Product, raw string) *models.Product {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	rawLower := strings.ToLower(raw)

	// 1. Direct ID match
	for i := range products {
		if strings.EqualFold(products[i].ID, raw) {
			return &products[i]
		}
	}

	// 2. Legacy / prompt template number aliases
	aliasMap := map[string]string{
		"prod-001": "prod-murabaha-wc",
		"prod-002": "prod-ijara-mb",
		"prod-003": "prod-ijara-mb",
		"prod-004": "prod-ijara-mb",
		"prod-005": "prod-trade-lc",
		"prod-006": "prod-trade-lg",
		"prod-007": "prod-syndication",
		"prod-008": "prod-treasury-wakala",
		"prod-009": "prod-treasury-wakala",
		"prod-010": "prod-receivable-factoring",
		"prod-1":   "prod-murabaha-wc",
		"prod-2":   "prod-ijara-mb",
		"prod-3":   "prod-ijara-mb",
		"prod-4":   "prod-ijara-mb",
		"prod-5":   "prod-trade-lc",
		"prod-6":   "prod-trade-lg",
		"prod-7":   "prod-syndication",
		"prod-8":   "prod-treasury-wakala",
		"prod-9":   "prod-treasury-wakala",
		"prod-10":  "prod-receivable-factoring",
	}
	if targetID, ok := aliasMap[rawLower]; ok {
		for i := range products {
			if products[i].ID == targetID {
				return &products[i]
			}
		}
	}

	// 3. Exact Name match (case-insensitive)
	for i := range products {
		if strings.EqualFold(products[i].Name, raw) || (products[i].NameAr != "" && strings.EqualFold(products[i].NameAr, raw)) {
			return &products[i]
		}
	}

	// 4. Substring Name match
	for i := range products {
		pNameLower := strings.ToLower(products[i].Name)
		if strings.Contains(pNameLower, rawLower) || strings.Contains(rawLower, pNameLower) {
			return &products[i]
		}
	}

	// 5. Keyword / Shariah Structure match
	keywords := map[string]string{
		"murabaha":   "prod-murabaha-wc",
		"working":    "prod-murabaha-wc",
		"ijara":      "prod-ijara-mb",
		"lease":      "prod-ijara-mb",
		"equipment":  "prod-ijara-mb",
		"lc":         "prod-trade-lc",
		"letter of":  "prod-trade-lc",
		"lg":         "prod-trade-lg",
		"guarantee":  "prod-trade-lg",
		"istisna":    "prod-project-istisna",
		"construct":  "prod-project-istisna",
		"pos":        "prod-pos-finance",
		"merchant":   "prod-pos-finance",
		"wakala":     "prod-treasury-wakala",
		"deposit":    "prod-treasury-wakala",
		"treasury":   "prod-treasury-wakala",
		"waad":       "prod-fx-waad",
		"fx":         "prod-fx-waad",
		"hedging":    "prod-fx-waad",
		"syndicat":   "prod-syndication",
		"payroll":    "prod-payroll-wps",
		"wps":        "prod-payroll-wps",
		"salary":     "prod-payroll-wps",
		"factoring":  "prod-receivable-factoring",
		"receivable": "prod-receivable-factoring",
		"invoice":    "prod-receivable-factoring",
	}

	for kw, targetID := range keywords {
		if strings.Contains(rawLower, kw) {
			for i := range products {
				if products[i].ID == targetID {
					return &products[i]
				}
			}
		}
	}

	return nil
}

// computeHoldingGaps calculates which product categories the client does NOT currently hold.
func computeHoldingGaps(allProducts []models.Product, holdings []models.ClientProduct) string {
	heldIDs := make(map[string]bool)
	for _, h := range holdings {
		heldIDs[h.ProductID] = true
	}

	var missing []string
	for _, p := range allProducts {
		if !heldIDs[p.ID] {
			missing = append(missing, fmt.Sprintf("• [%s] %s (%s — %s)", p.ID, p.Name, p.Category, p.ShariahStructure))
		}
	}

	if len(missing) == 0 {
		return "Client currently holds facilities across all main catalog products. Focus on facility limit extensions or renewal upsells."
	}
	return strings.Join(missing, "\n")
}

// normalizeUrgency ensures urgency is a valid enum value.
func normalizeUrgency(raw string) models.Urgency {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "critical":
		return models.UrgencyCritical
	case "high":
		return models.UrgencyHigh
	case "medium":
		return models.UrgencyMedium
	case "low":
		return models.UrgencyLow
	default:
		return models.UrgencyMedium
	}
}

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
