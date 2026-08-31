package models

import "time"

// --- Enums ---

type RiskRating string

const (
	RiskLow    RiskRating = "Low"
	RiskMedium RiskRating = "Medium"
	RiskHigh   RiskRating = "High"
)

type KYCStatus string

const (
	KYCActive  KYCStatus = "Active"
	KYCPending KYCStatus = "Pending"
	KYCExpired KYCStatus = "Expired"
)

type OpportunityStatus string

const (
	OpportunityNew       OpportunityStatus = "New"
	OpportunityReviewed  OpportunityStatus = "Reviewed"
	OpportunityAccepted  OpportunityStatus = "Accepted"
	OpportunityDismissed OpportunityStatus = "Dismissed"
	OpportunityConverted OpportunityStatus = "Converted"
)

type Urgency string

const (
	UrgencyLow      Urgency = "Low"
	UrgencyMedium   Urgency = "Medium"
	UrgencyHigh     Urgency = "High"
	UrgencyCritical Urgency = "Critical"
)

// --- Domain Models ---

// RelationshipManager represents a corporate banking RM.
type RelationshipManager struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Title       string `json:"title"`
	Email       string `json:"email"`
	Department  string `json:"department"`
	ClientCount int    `json:"client_count,omitempty"`
}

// Client represents a corporate banking client.
type Client struct {
	ID                string          `json:"id"`
	UserID            string          `json:"user_id"`
	Name              string          `json:"name"`
	NameAr            string          `json:"name_ar,omitempty"`
	Industry          string          `json:"industry"`
	SubIndustry       string          `json:"sub_industry,omitempty"`
	RevenueKWD        float64         `json:"revenue_kwd"`
	EmployeeCount     int             `json:"employee_count"`
	IncorporationYear int             `json:"incorporation_year"`
	RiskRating        RiskRating      `json:"risk_rating"`
	KYCStatus         KYCStatus       `json:"kyc_status"`
	RelationshipStart string          `json:"relationship_start"`
	RMID              string          `json:"rm_id"`
	RMName            string          `json:"rm_name,omitempty"`
	Country           string          `json:"country"`
	Notes             string          `json:"notes,omitempty"`
	CurrentProducts   []ClientProduct `json:"current_products,omitempty"`
	Interactions      []Interaction   `json:"interactions,omitempty"`
}

// Product represents a Warba Bank Shariah-compliant product.
type Product struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	NameAr              string  `json:"name_ar,omitempty"`
	Category            string  `json:"category"`
	Description         string  `json:"description"`
	ShariahStructure    string  `json:"shariah_structure"`
	MinAmountKWD        float64 `json:"min_amount_kwd"`
	MaxAmountKWD        float64 `json:"max_amount_kwd"`
	TypicalTenureMonths int     `json:"typical_tenure_months"`
	TargetIndustries    string  `json:"target_industries"`
	IsActive            bool    `json:"is_active"`
}

// ClientProduct represents a product currently held by a client.
type ClientProduct struct {
	ClientID    string  `json:"client_id"`
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name,omitempty"`
	StartDate   string  `json:"start_date"`
	AmountKWD   float64 `json:"amount_kwd"`
	Status      string  `json:"status"`
}

// Interaction represents a recorded interaction with a client.
type Interaction struct {
	ID       string `json:"id"`
	UserID   string `json:"user_id"`
	ClientID string `json:"client_id"`
	Type     string `json:"type"` // Meeting, Call, Email, Transaction, Note
	Date     string `json:"date"`
	Summary  string `json:"summary"`
	Outcome  string `json:"outcome,omitempty"`
	RMID     string `json:"rm_id,omitempty"`
}

// Opportunity represents an AI-generated product suggestion for a client.
type Opportunity struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id"`
	ClientID     string            `json:"client_id"`
	ClientName   string            `json:"client_name,omitempty"`
	ProductID    string            `json:"product_id"`
	ProductName  string            `json:"product_name,omitempty"`
	Confidence   float64           `json:"confidence"`
	Urgency      Urgency           `json:"urgency"`
	Reasoning    string            `json:"reasoning"`
	NextAction   string            `json:"next_action"`
	ShariahNotes string            `json:"shariah_notes"`
	Status       OpportunityStatus `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    *time.Time        `json:"updated_at,omitempty"`
}

// AIOpportunitySuggestion is the structured output expected from the AI model.
type AIOpportunitySuggestion struct {
	ProductID    string  `json:"product_id"`
	Confidence   float64 `json:"confidence"`
	Urgency      string  `json:"urgency"`
	Reasoning    string  `json:"reasoning"`
	NextAction   string  `json:"next_action"`
	ShariahNotes string  `json:"shariah_notes"`
}

// PortfolioSummary provides aggregate stats for a RM's portfolio.
type PortfolioSummary struct {
	TotalClients       int                `json:"total_clients"`
	TotalOpportunities int                `json:"total_opportunities"`
	NewOpportunities   int                `json:"new_opportunities"`
	AcceptedOpps       int                `json:"accepted_opportunities"`
	ConvertedOpps      int                `json:"converted_opportunities"`
	DismissedOpps      int                `json:"dismissed_opportunities"`
	AvgConfidence      float64            `json:"avg_confidence"`
	PipelineValueKWD   float64            `json:"pipeline_value_kwd"`
	TopIndustries      []IndustryStat     `json:"top_industries"`
	UrgencyBreakdown   map[string]int     `json:"urgency_breakdown"`
	ProductBreakdown   []ProductStat      `json:"product_breakdown"`
}

// IndustryStat provides industry-level stats.
type IndustryStat struct {
	Industry string  `json:"industry"`
	Count    int     `json:"count"`
	Revenue  float64 `json:"revenue_kwd"`
}

// ProductStat provides product-level opportunity stats.
type ProductStat struct {
	ProductID   string `json:"product_id"`
	ProductName string `json:"product_name"`
	Count       int    `json:"count"`
}

// User represents an authenticated corporate banking user / RM.
type User struct {
	ID        string    `json:"id"`
	GoogleID  string    `json:"google_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Avatar    string    `json:"avatar"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	LastLogin time.Time `json:"last_login"`
}

// Session represents an authenticated user session.
type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

// --- API Request/Response Types ---

type GoogleAuthRequest struct {
	Credential string `json:"credential"`
}

type AuthConfigResponse struct {
	GoogleClientID string `json:"google_client_id"`
	Enabled        bool   `json:"enabled"`
}

type AuthUserResponse struct {
	User          *User `json:"user"`
	Authenticated bool  `json:"authenticated"`
}


type AnalyzeRequest struct {
	ClientID string `json:"client_id"`
}

type PortfolioScanRequest struct {
	RMID string `json:"rm_id"`
}

type StatusUpdateRequest struct {
	Status OpportunityStatus `json:"status"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
