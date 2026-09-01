package engine

import (
	"fmt"
	"strings"

	"opportunity-engine/internal/models"
)

// BuildCatalogPrompt generates a rich, dynamic catalog representation from the database products.
func BuildCatalogPrompt(products []models.Product) string {
	if len(products) == 0 {
		return "Catalog unavailable."
	}

	var sb strings.Builder
	for _, p := range products {
		amountStr := fmt.Sprintf("KWD %s – %s", formatAmount(p.MinAmountKWD), formatAmount(p.MaxAmountKWD))
		if p.MinAmountKWD == 0 && p.MaxAmountKWD == 0 {
			amountStr = "Custom / Fee-based"
		}

		sb.WriteString(fmt.Sprintf("[%s] %s", p.ID, p.Name))
		if p.NameAr != "" {
			sb.WriteString(fmt.Sprintf(" (%s)", p.NameAr))
		}
		sb.WriteString(fmt.Sprintf("\n  Category: %s | Structure: %s\n", p.Category, p.ShariahStructure))
		sb.WriteString(fmt.Sprintf("  Facility Range: %s | Tenure: %d months\n", amountStr, p.TypicalTenureMonths))
		if p.TargetIndustries != "" {
			sb.WriteString(fmt.Sprintf("  Target Industries: %s\n", p.TargetIndustries))
		}
		sb.WriteString(fmt.Sprintf("  Use Case: %s\n\n", p.Description))
	}
	return sb.String()
}

// BuildSystemPrompt creates the system prompt incorporating the live Shariah product catalog.
func BuildSystemPrompt(catalogPrompt string) string {
	return fmt.Sprintf(`You are the Senior Corporate Banking Credit & Structuring AI Advisor for Warba Bank, Kuwait's leading Islamic corporate bank.
Your mandate is to analyze corporate client portfolios and formulate proactive, highly specific, deal-ready product suggestions for Relationship Managers (RMs).

CRITICAL DIRECTIVES:
1. PRODUCT SELECTION: You MUST ONLY select product_id values from the active catalog below. Use the exact product_id string (e.g. "prod-murabaha-wc", "prod-ijara-mb", "prod-trade-lc", "prod-trade-lg", "prod-project-istisna", "prod-pos-finance", "prod-treasury-wakala", "prod-fx-waad", "prod-syndication", "prod-payroll-wps", "prod-receivable-factoring").
2. NEVER suggest conventional (interest-bearing / non-Islamic) banking mechanisms under any circumstances.
3. ANTI-GENERIC MANDATE: Generic advice (e.g. "client needs money for growth" or "helps improve cash flow") is UNACCEPTABLE. Every recommendation MUST cite:
   - Specific quantitative and qualitative signals from the client's profile (e.g., specific revenue size, employee headcount, recent tender bids, supplier import corridors, FX currency risks).
   - Executive names and dates from interaction history when available.
   - Proposed deal size (KWD) appropriate to the client's revenue and product limits.
4. ACTIONABLE NEXT STEPS: Suggest concrete, immediate RM actions (e.g. "Schedule term sheet review with CFO Tariq Al-Ghanim proposing KWD 2.5M 3-year Ijara facility with 15%% advance payment guarantee").
5. SHARIAH GOVERNANCE: Detail the exact Islamic structure (e.g., AAOIFI Standard No. 9 for Ijara Muntahia Bittamleek, unilateral binding Wa'ad for FX hedging, Kafalah fee rules, Parallel Istisna'a milestone billing).
6. HOLDINGS EXCLUSION: Do NOT recommend products the client already actively holds unless there is a clear expansion/upsell trigger highlighted in recent interactions.
7. Return between 1 and 3 high-conviction opportunities.

WARBA BANK ACTIVE SHARIAH PRODUCT CATALOG:
%s`, catalogPrompt)
}

// ClientAnalysisPromptTemplate formats the analysis prompt with deep client signals.
const ClientAnalysisPromptTemplate = `Analyze the following corporate client profile and formulate up to 3 proactive, high-conviction product opportunities from Warba Bank's catalog.

=== CLIENT DOSSIER ===
• Company: %s (%s)
• Industry: %s | Sub-Industry: %s | Country: %s
• Annual Revenue: KWD %s | Workforce: %d employees | Established: %d
• Risk Rating: %s | KYC Status: %s | Client Relationship Since: %s
• Account Overview & Strategic Notes: %s

=== CURRENT PRODUCT HOLDINGS ===
%s

=== RECENT INTERACTION TIMELINE & SIGNALS ===
%s

=== UNMET PRODUCT CATEGORIES (GAP ANALYSIS) ===
%s

TASK:
1. Synthesize the client's financial profile, expansion needs, supplier/contractor dynamics, and recent meeting outcomes.
2. Select the top 1 to 3 distinct products from the catalog that best match this client's immediate operational or capital needs.
3. For each opportunity, provide deep, specific, non-generic reasoning tying their financial metrics and meeting notes directly to the product structure.

Respond with ONLY a valid JSON array of objects (no markdown outside the JSON, no extra text):
[
  {
    "product_id": "prod-exact-id",
    "confidence": 0.88,
    "urgency": "Critical|High|Medium|Low",
    "reasoning": "Deep, client-specific reasoning citing actual financial metrics, project names, or meeting notes...",
    "next_action": "Concrete, step-by-step next action for the Relationship Manager including proposed facility size...",
    "shariah_notes": "Specific Islamic finance structuring compliance and AAOIFI standards rationale..."
  }
]`

// FormatClientAnalysisPrompt formats the prompt with client data and computed holding gaps.
func FormatClientAnalysisPrompt(
	name, nameAr, industry, subIndustry, revenue string,
	employees, incorporationYear int,
	country, riskRating, kycStatus, relationshipStart, notes string,
	currentProducts, interactions, holdingGaps string,
) string {
	if nameAr == "" {
		nameAr = "N/A"
	}
	if notes == "" {
		notes = "Standard corporate relationship in good standing."
	}
	if currentProducts == "" {
		currentProducts = "None — No active credit or treasury facilities currently held."
	}
	if interactions == "" {
		interactions = "No recent interaction logs recorded. Base assessment on corporate profile, industry dynamics, and revenue scale."
	}
	if holdingGaps == "" {
		holdingGaps = "Full product suite available for cross-selling."
	}

	return fmt.Sprintf(
		ClientAnalysisPromptTemplate,
		name, nameAr, industry, subIndustry, country,
		revenue, employees, incorporationYear,
		riskRating, kycStatus, relationshipStart, notes,
		currentProducts, interactions, holdingGaps,
	)
}

// PortfolioScanPromptTemplate formats the prompt for scanning across all clients.
const PortfolioScanPromptTemplate = `You are scanning a Senior Relationship Manager's entire corporate client portfolio to identify the top 5 highest-priority proactive deal opportunities across all clients.

=== CLIENT PORTFOLIO OVERVIEW ===
%s

TASK:
1. Scan across the entire portfolio for high-impact opportunities (e.g. major contract tenders, expansion funding, import trade finance, FX hedging, or liquidity placement).
2. Rank and return the top 5 highest-value opportunities across the portfolio.
3. Ensure every recommendation references the exact client_id, the exact product_id from the catalog, and provides rich, client-specific reasoning.

Respond with ONLY a valid JSON array:
[
  {
    "client_id": "cli-xxx-xxx",
    "product_id": "prod-exact-id",
    "confidence": 0.90,
    "urgency": "High",
    "reasoning": "Client-specific justification citing their industry, revenue, and recent context...",
    "next_action": "Targeted RM next steps with proposed deal parameters...",
    "shariah_notes": "Islamic structure and Shariah compliance notes..."
  }
]`

// PortfolioSuggestion extends AIOpportunitySuggestion with client ID for batch scans.
type PortfolioSuggestion struct {
	ClientID     string  `json:"client_id"`
	ProductID    string  `json:"product_id"`
	Confidence   float64 `json:"confidence"`
	Urgency      string  `json:"urgency"`
	Reasoning    string  `json:"reasoning"`
	NextAction   string  `json:"next_action"`
	ShariahNotes string  `json:"shariah_notes"`
}
