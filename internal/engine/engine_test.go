package engine

import (
	"strings"
	"testing"

	"opportunity-engine/internal/models"
)

func sampleProducts() []models.Product {
	return []models.Product{
		{
			ID:                  "prod-murabaha-wc",
			Name:                "Working Capital Murabaha",
			NameAr:              "مرابحة رأس المال العامل",
			Category:            "Financing",
			ShariahStructure:    "Murabaha (cost-plus sale)",
			MinAmountKWD:        50000,
			MaxAmountKWD:        5000000,
			TypicalTenureMonths: 12,
			TargetIndustries:    "Trading, Manufacturing, Retail",
			Description:         "Cost-plus financing facility for purchase of raw materials and goods.",
		},
		{
			ID:                  "prod-ijara-mb",
			Name:                "Ijara Muntahia Bittamleek",
			NameAr:              "إجارة منتهية بالتمليك",
			Category:            "Financing",
			ShariahStructure:    "Ijara with transfer of ownership",
			MinAmountKWD:        100000,
			MaxAmountKWD:        10000000,
			TypicalTenureMonths: 60,
			TargetIndustries:    "Construction, Logistics, Manufacturing, Oil & Gas",
			Description:         "Islamic lease-to-own facility for heavy equipment and property.",
		},
		{
			ID:                  "prod-trade-lc",
			Name:                "Documentary Letter of Credit",
			Category:            "Trade Finance",
			ShariahStructure:    "Wakala bi al-Ujrah",
			MinAmountKWD:        25000,
			MaxAmountKWD:        20000000,
			TypicalTenureMonths: 12,
			TargetIndustries:    "Trading, Manufacturing",
			Description:         "Import and export trade finance.",
		},
		{
			ID:                  "prod-treasury-wakala",
			Name:                "Corporate Wakala Deposit",
			Category:            "Treasury",
			ShariahStructure:    "Wakala bil Istithmar",
			MinAmountKWD:        100000,
			MaxAmountKWD:        50000000,
			TypicalTenureMonths: 12,
			Description:         "Surplus liquidity placement.",
		},
	}
}

func TestResolveProduct(t *testing.T) {
	products := sampleProducts()

	tests := []struct {
		input    string
		expected string // Expected Product ID
	}{
		// 1. Direct ID match
		{"prod-murabaha-wc", "prod-murabaha-wc"},
		{"prod-ijara-mb", "prod-ijara-mb"},
		{"PROD-TRADE-LC", "prod-trade-lc"},

		// 2. Legacy / prompt template number aliases
		{"prod-001", "prod-murabaha-wc"},
		{"prod-002", "prod-ijara-mb"},
		{"prod-005", "prod-trade-lc"},
		{"prod-008", "prod-treasury-wakala"},

		// 3. Exact Name match
		{"Working Capital Murabaha", "prod-murabaha-wc"},
		{"Ijara Muntahia Bittamleek", "prod-ijara-mb"},
		{"Documentary Letter of Credit", "prod-trade-lc"},

		// 4. Substring Name match
		{"Letter of Credit", "prod-trade-lc"},

		// 5. Keyword match
		{"murabaha financing", "prod-murabaha-wc"},
		{"ijara lease for rigs", "prod-ijara-mb"},
		{"wakala deposit placement", "prod-treasury-wakala"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			res := resolveProduct(products, tt.input)
			if res == nil {
				t.Fatalf("expected product for %q, got nil", tt.input)
			}
			if res.ID != tt.expected {
				t.Errorf("resolveProduct(%q) = %q, expected %q", tt.input, res.ID, tt.expected)
			}
		})
	}
}

func TestBuildCatalogAndSystemPrompt(t *testing.T) {
	products := sampleProducts()
	catPrompt := BuildCatalogPrompt(products)

	if !strings.Contains(catPrompt, "prod-murabaha-wc") {
		t.Errorf("expected catalog prompt to contain prod-murabaha-wc")
	}
	if !strings.Contains(catPrompt, "Ijara Muntahia Bittamleek") {
		t.Errorf("expected catalog prompt to contain Ijara Muntahia Bittamleek")
	}

	sysPrompt := BuildSystemPrompt(catPrompt)
	if !strings.Contains(sysPrompt, "ANTI-GENERIC MANDATE") {
		t.Errorf("expected system prompt to contain ANTI-GENERIC MANDATE")
	}
	if !strings.Contains(sysPrompt, "prod-murabaha-wc") {
		t.Errorf("expected system prompt to contain catalog IDs")
	}
}

func TestComputeHoldingGaps(t *testing.T) {
	products := sampleProducts()
	holdings := []models.ClientProduct{
		{ProductID: "prod-murabaha-wc", ProductName: "Working Capital Murabaha"},
	}

	gaps := computeHoldingGaps(products, holdings)
	if strings.Contains(gaps, "prod-murabaha-wc") {
		t.Errorf("gaps should not contain already held prod-murabaha-wc")
	}
	if !strings.Contains(gaps, "prod-ijara-mb") {
		t.Errorf("gaps should include missing prod-ijara-mb")
	}
	if !strings.Contains(gaps, "prod-trade-lc") {
		t.Errorf("gaps should include missing prod-trade-lc")
	}
}

func TestCleanJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "With thinking tags and markdown",
			input:    "<think>Evaluating client balance sheet...</think>```json\n[{\"product_id\":\"prod-ijara-mb\"}]\n```",
			expected: "[{\"product_id\":\"prod-ijara-mb\"}]",
		},
		{
			name:     "With leading and trailing commentary",
			input:    "Here are the top opportunities:\n[{\"product_id\":\"prod-murabaha-wc\"}]\nHope this helps!",
			expected: "[{\"product_id\":\"prod-murabaha-wc\"}]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := cleanJSON(tt.input)
			if out != tt.expected {
				t.Errorf("cleanJSON() = %q, expected %q", out, tt.expected)
			}
		})
	}
}
