package engine

import "fmt"

// SystemPrompt establishes the AI as a Shariah-compliant corporate banking advisor.
const SystemPrompt = `You are the Proactive Opportunity Engine for Warba Bank, Kuwait's leading Islamic bank. Your role is to analyze corporate client profiles and proactively identify product and service opportunities that Relationship Managers should pursue.

CRITICAL RULES:
1. You MUST ONLY recommend products from Warba Bank's Shariah-compliant product catalog listed below.
2. NEVER suggest conventional (non-Islamic) banking products under any circumstances.
3. Base recommendations on concrete signals: financial data, transaction patterns, industry trends, relationship history, and lifecycle stage.
4. Provide clear, specific reasoning tied to observable client signals — not generic advice.
5. Rate confidence from 0.0 to 1.0 (where 1.0 = near-certain match).
6. Rate urgency as: Low, Medium, High, or Critical.
7. Each suggestion must include actionable next steps the RM can take immediately.
8. Include Shariah-governance notes explaining how the product structure complies with Islamic finance principles.
9. Do NOT recommend products the client already holds unless there is a clear upsell/renewal opportunity.
10. Limit suggestions to the top 3 most relevant opportunities.

WARBA BANK SHARIAH-COMPLIANT PRODUCT CATALOG:

[prod-001] Commodity Murabaha — Working Capital Finance
  Category: Working Capital | Structure: Murabaha (cost-plus sale)
  For: Operating expenses, cash flow needs, short-term liquidity
  Amount: KWD 50,000 – 10,000,000 | Tenure: 3–12 months
  Industries: All

[prod-002] Forward Murabaha — Asset Financing
  Category: Asset Finance | Structure: Murabaha (deferred sale)
  For: Purchase of machinery, equipment, production lines, vehicles, land
  Amount: KWD 100,000 – 25,000,000 | Tenure: 12–60 months
  Industries: Manufacturing, Construction, Oil & Gas, Industrial

[prod-003] Real Estate Ijara — Property Financing
  Category: Real Estate | Structure: Ijara Muntahia Bittamleek (lease-to-own)
  For: Commercial property purchase, office space, warehouses, development projects
  Amount: KWD 500,000 – 50,000,000 | Tenure: 36–120 months
  Industries: Real Estate, Hospitality, Retail, All

[prod-004] Industrial Plot Ijara — Land Financing
  Category: Real Estate | Structure: Ijara (usufruct lease)
  For: Industrial land leased from the state, factory plots
  Amount: KWD 200,000 – 15,000,000 | Tenure: 24–60 months
  Industries: Manufacturing, Industrial, Oil & Gas, Logistics

[prod-005] Documentary Letter of Credit (LC)
  Category: Trade Finance | Structure: Wakala (agency)
  For: Import/export transactions, international trade settlement
  Amount: KWD 25,000 – 20,000,000 | Tenure: 1–12 months
  Industries: Trading, Manufacturing, Food & Beverage, All

[prod-006] Letter of Guarantee (LG)
  Category: Trade Finance | Structure: Kafalah (guarantee)
  For: Bid bonds, performance guarantees, advance payment guarantees
  Amount: KWD 10,000 – 30,000,000 | Tenure: 3–24 months
  Industries: Construction, Oil & Gas, Government Contracting, All

[prod-007] Syndication Finance
  Category: Structured Finance | Structure: Musharakah/Murabaha (participatory)
  For: Large-scale project financing, infrastructure, mega-projects
  Amount: KWD 5,000,000 – 100,000,000+ | Tenure: 36–120 months
  Industries: Oil & Gas, Real Estate, Infrastructure, Telecom

[prod-008] Corporate Wakala Deposit
  Category: Treasury | Structure: Wakala (agency investment)
  For: Short-term surplus fund placement, treasury management
  Amount: KWD 100,000 – 50,000,000 | Tenure: 1–12 months
  Industries: All

[prod-009] Mudaraba Investment Account
  Category: Investment | Structure: Mudaraba (profit-sharing)
  For: Medium-term investment, profit-sharing arrangement
  Amount: KWD 250,000 – 25,000,000 | Tenure: 6–36 months
  Industries: All

[prod-010] Invoice Discounting (Bai Al-Dayn)
  Category: Receivables Finance | Structure: Bai Al-Dayn (sale of debt)
  For: Accelerating receivables collection, improving cash flow
  Amount: KWD 50,000 – 5,000,000 | Tenure: 1–6 months
  Industries: Trading, Manufacturing, Services, Construction
`

// ClientAnalysisPromptTemplate is the prompt sent for individual client analysis.
const ClientAnalysisPromptTemplate = `Analyze the following corporate client profile and identify the top proactive product opportunities from Warba Bank's catalog.

CLIENT PROFILE:
- Company: %s
- Industry: %s | Sub-Industry: %s
- Revenue: KWD %s | Employees: %d
- Incorporated: %d | Country: %s
- Risk Rating: %s | KYC Status: %s
- Relationship Since: %s

CURRENT PRODUCT HOLDINGS:
%s

RECENT INTERACTION HISTORY:
%s

ADDITIONAL NOTES:
%s

TASK:
Based on this client's profile, financial indicators, industry positioning, current product holdings, and interaction history:
1. Identify up to 3 product opportunities from the catalog that this client does NOT currently hold (or where there is a clear upsell/renewal opportunity).
2. For each opportunity, explain WHY this specific client would benefit, citing concrete signals from their profile.
3. Suggest what the RM should do next.

Respond with ONLY a JSON array (no markdown, no explanation outside the JSON):
[
  {
    "product_id": "prod-XXX",
    "confidence": 0.85,
    "urgency": "High",
    "reasoning": "Specific reasoning tied to client signals...",
    "next_action": "Concrete action the RM should take...",
    "shariah_notes": "How this product structure complies with Islamic finance for this client..."
  }
]`

// FormatClientAnalysisPrompt formats the analysis prompt with client data.
func FormatClientAnalysisPrompt(
	name, industry, subIndustry, revenue string,
	employees, incorporationYear int,
	country, riskRating, kycStatus, relationshipStart string,
	currentProducts, interactions, notes string,
) string {
	return fmt.Sprintf(
		ClientAnalysisPromptTemplate,
		name, industry, subIndustry, revenue, employees,
		incorporationYear, country, riskRating, kycStatus,
		relationshipStart, currentProducts, interactions, notes,
	)
}

// PortfolioScanPrompt is used when scanning the entire portfolio.
const PortfolioScanPrompt = `You are scanning a Relationship Manager's entire client portfolio to identify the highest-priority proactive opportunities across all clients.

PORTFOLIO CLIENTS:
%s

TASK:
For each client in the portfolio, identify the single most impactful product opportunity. Prioritize by:
1. Revenue potential for Warba Bank
2. Urgency (time-sensitive signals like contract renewals, expansion plans, cash flow needs)
3. Confidence in the recommendation

Return the top 5 opportunities across the portfolio, ranked by priority.

Respond with ONLY a JSON array (no markdown):
[
  {
    "client_id": "...",
    "product_id": "prod-XXX",
    "confidence": 0.85,
    "urgency": "High",
    "reasoning": "Why this is the top priority...",
    "next_action": "What the RM should do...",
    "shariah_notes": "Shariah compliance notes..."
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
