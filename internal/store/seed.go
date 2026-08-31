package store

import (
	"fmt"
	"time"

	"opportunity-engine/internal/models"
)

// SeedGlobalProducts inserts the standard Warba Bank Shariah-compliant product catalog.
func (s *Store) SeedGlobalProducts() error {
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count); err == nil && count > 0 {
		return nil
	}

	for _, p := range seedProducts {
		if err := s.InsertProduct(p); err != nil {
			return fmt.Errorf("seeding product %s: %w", p.ID, err)
		}
	}
	return nil
}

// SeedUserPortfolio initializes a full corporate banking portfolio of 20 clients, holdings, interactions,
// and sample opportunities for a specific authenticated user (RM).
func (s *Store) SeedUserPortfolio(userID, userName string) error {
	if userID == "" {
		return fmt.Errorf("cannot seed portfolio without valid userID")
	}

	var clientCount int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM clients WHERE user_id = ?", userID).Scan(&clientCount); err == nil && clientCount > 0 {
		return nil // Already seeded
	}

	// 1. Seed 20 Corporate Clients for this user
	for i, c := range seedClients {
		cID := fmt.Sprintf("cli-%s-%03d", userID, i+1)
		client := models.Client{
			ID:                cID,
			UserID:            userID,
			Name:              c.Name,
			NameAr:            c.NameAr,
			Industry:          c.Industry,
			SubIndustry:       c.SubIndustry,
			RevenueKWD:        c.RevenueKWD,
			EmployeeCount:     c.EmployeeCount,
			IncorporationYear: c.IncorporationYear,
			RiskRating:        c.RiskRating,
			KYCStatus:         c.KYCStatus,
			RelationshipStart: c.RelationshipStart,
			RMID:              userID,
			RMName:            userName,
			Country:           c.Country,
			Notes:             c.Notes,
		}

		if err := s.InsertClient(client); err != nil {
			return fmt.Errorf("seeding client %s for user %s: %w", client.ID, userID, err)
		}

		// 2. Add product holdings for this client
		if holdings, ok := seedClientProductsTemplate[i]; ok {
			for _, cp := range holdings {
				cp.ClientID = cID
				if err := s.InsertClientProduct(cp); err != nil {
					return fmt.Errorf("seeding product holding for client %s: %w", cID, err)
				}
			}
		}

		// 3. Add interaction history for this client
		if interactions, ok := seedInteractionsTemplate[i]; ok {
			for j, ix := range interactions {
				ix.ID = fmt.Sprintf("ix-%s-%03d-%d", userID, i+1, j+1)
				ix.ClientID = cID
				ix.UserID = userID
				if err := s.InsertInteraction(ix); err != nil {
					return fmt.Errorf("seeding interaction for client %s: %w", cID, err)
				}
			}
		}
	}

	// 4. Seed initial realistic opportunities for this user's clients
	for _, opp := range seedOpportunitiesTemplate {
		clientIdx := opp.ClientIdx
		if clientIdx >= len(seedClients) {
			continue
		}
		cID := fmt.Sprintf("cli-%s-%03d", userID, clientIdx+1)
		o := models.Opportunity{
			ID:           fmt.Sprintf("opp-%s-%03d-%s", userID, clientIdx+1, opp.ProductID),
			UserID:       userID,
			ClientID:     cID,
			ProductID:    opp.ProductID,
			Confidence:   opp.Confidence,
			Urgency:      opp.Urgency,
			Reasoning:    opp.Reasoning,
			NextAction:   opp.NextAction,
			ShariahNotes: opp.ShariahNotes,
			Status:       opp.Status,
			CreatedAt:    time.Now().Add(-time.Duration(opp.HoursAgo) * time.Hour),
		}
		if err := s.InsertOpportunity(o); err != nil {
			return fmt.Errorf("seeding opportunity %s: %w", o.ID, err)
		}
	}

	return nil
}

// --- Static Template Data ---

var seedProducts = []models.Product{
	{
		ID:                  "prod-murabaha-wc",
		Name:                "Working Capital Murabaha",
		NameAr:              "مرابحة رأس المال العامل",
		Category:            "Financing",
		Description:         "Cost-plus financing facility for purchase of raw materials, goods, and inventory with deferred payment terms.",
		ShariahStructure:    "Murabaha (cost-plus sale)",
		MinAmountKWD:        50000,
		MaxAmountKWD:        5000000,
		TypicalTenureMonths: 12,
		TargetIndustries:    "Trading, Manufacturing, Retail, Food & Beverage, Healthcare",
		IsActive:            true,
	},
	{
		ID:                  "prod-ijara-mb",
		Name:                "Ijara Muntahia Bittamleek",
		NameAr:              "إجارة منتهية بالتمليك",
		Category:            "Financing",
		Description:         "Islamic lease-to-own facility for acquiring machinery, commercial vehicles, heavy equipment, and real estate assets.",
		ShariahStructure:    "Ijara with transfer of ownership (usufruct lease)",
		MinAmountKWD:        100000,
		MaxAmountKWD:        10000000,
		TypicalTenureMonths: 60,
		TargetIndustries:    "Construction, Logistics, Manufacturing, Oil & Gas, Healthcare",
		IsActive:            true,
	},
	{
		ID:                  "prod-pos-finance",
		Name:                "POS Merchant Financing",
		NameAr:              "تمويل نقاط البيع",
		Category:            "Financing",
		Description:         "Working capital line structured against historical POS merchant receivables with flexible automated daily/weekly repayments.",
		ShariahStructure:    "Tawaruq / Murabaha against cash flow",
		MinAmountKWD:        10000,
		MaxAmountKWD:        500000,
		TypicalTenureMonths: 18,
		TargetIndustries:    "Retail, Food & Beverage, Hospitality, Healthcare",
		IsActive:            true,
	},
	{
		ID:                  "prod-trade-lc",
		Name:                "Wakala Letter of Credit (LC)",
		NameAr:              "اعتماد مستندي بالوكالة",
		Category:            "Trade Finance",
		Description:         "Import/Export documentary letters of credit issued on agency (Wakala) basis facilitating global trade.",
		ShariahStructure:    "Wakala bi al-Ujrah (agency with fee)",
		MinAmountKWD:        25000,
		MaxAmountKWD:        15000000,
		TypicalTenureMonths: 6,
		TargetIndustries:    "Trading, Oil & Gas, Manufacturing, Food & Beverage, Automotive",
		IsActive:            true,
	},
	{
		ID:                  "prod-trade-lg",
		Name:                "Kafalah Letters of Guarantee (LG)",
		NameAr:              "خطابات الضمان (الكفالة)",
		Category:            "Trade Finance",
		Description:         "Bid bonds, performance bonds, advance payment guarantees, and retention guarantees issued on Shariah surety principles.",
		ShariahStructure:    "Kafalah (guarantee/suretyship)",
		MinAmountKWD:        10000,
		MaxAmountKWD:        25000000,
		TypicalTenureMonths: 24,
		TargetIndustries:    "Construction, Engineering, Technology, Oil & Gas, Logistics",
		IsActive:            true,
	},
	{
		ID:                  "prod-project-istisna",
		Name:                "Istisna'a Construction Financing",
		NameAr:              "تمويل الاستصناع",
		Category:            "Project Finance",
		Description:         "Contract-based manufacturing and construction finance for building commercial real estate, factories, and infrastructure.",
		ShariahStructure:    "Istisna'a & Parallel Istisna'a (manufacturing sale)",
		MinAmountKWD:        500000,
		MaxAmountKWD:        30000000,
		TypicalTenureMonths: 36,
		TargetIndustries:    "Real Estate, Construction, Manufacturing, Healthcare, Hospitality",
		IsActive:            true,
	},
	{
		ID:                  "prod-treasury-wakala",
		Name:                "Wakala Deposit Investment",
		NameAr:              "وديعة الوكالة الاستثمارية",
		Category:            "Treasury & Deposits",
		Description:         "Corporate term investment deposit where the bank acts as Wakeel (agent) investing liquidity into Shariah-compliant instruments.",
		ShariahStructure:    "Wakala bi al-Istithmar (investment agency)",
		MinAmountKWD:        100000,
		MaxAmountKWD:        50000000,
		TypicalTenureMonths: 12,
		TargetIndustries:    "All Industries, Financial Services, Real Estate, Oil & Gas",
		IsActive:            true,
	},
	{
		ID:                  "prod-fx-waad",
		Name:                "Islamic FX Hedging (Wa'ad)",
		NameAr:              "التحوط بالوعد في الصرف الأجنبي",
		Category:            "Treasury & Deposits",
		Description:         "Shariah-compliant unilateral promise (Wa'ad) structures providing forward currency hedging for import/export cash flows.",
		ShariahStructure:    "Wa'ad Mulzim (binding unilateral promise)",
		MinAmountKWD:        50000,
		MaxAmountKWD:        20000000,
		TypicalTenureMonths: 12,
		TargetIndustries:    "Trading, Logistics, Automotive, Manufacturing, Food & Beverage",
		IsActive:            true,
	},
	{
		ID:                  "prod-payroll-wps",
		Name:                "Corporate Payroll & WPS Suite",
		NameAr:              "نظام إدارة الرواتب والشركات",
		Category:            "Cash Management",
		Description:         "Automated Wage Protection System (WPS) compliant bulk salary processing, employee prepaid cards, and treasury pooling.",
		ShariahStructure:    "Khadamat bi Ujrah (service fee)",
		MinAmountKWD:        0,
		MaxAmountKWD:        0,
		TypicalTenureMonths: 12,
		TargetIndustries:    "All Industries, Construction, Logistics, Retail, Hospitality",
		IsActive:            true,
	},
	{
		ID:                  "prod-receivable-factoring",
		Name:                "Islamic Invoice Discounting",
		NameAr:              "تمويل الفواتير المتوافقة",
		Category:            "Trade Finance",
		Description:         "Monetization of certified corporate and government contract receivables through Bai Al-Dayn (sale of debt) or Murabaha.",
		ShariahStructure:    "Bai Al-Dayn bi al-Sila (commodity-backed debt sale)",
		MinAmountKWD:        100000,
		MaxAmountKWD:        8000000,
		TypicalTenureMonths: 12,
		TargetIndustries:    "Construction, Engineering, Healthcare, Technology, Logistics",
		IsActive:            true,
	},
}

type clientTemplate struct {
	Name              string
	NameAr            string
	Industry          string
	SubIndustry       string
	RevenueKWD        float64
	EmployeeCount     int
	IncorporationYear int
	RiskRating        models.RiskRating
	KYCStatus         models.KYCStatus
	RelationshipStart string
	Country           string
	Notes             string
}

var seedClients = []clientTemplate{
	{Name: "Al-Ahmadi Oil Field Services", NameAr: "الأحمدي لخدمات حقول النفط", Industry: "Oil & Gas", SubIndustry: "Drilling & Well Support", RevenueKWD: 45000000, EmployeeCount: 850, IncorporationYear: 1998, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2015-03-12", Country: "Kuwait", Notes: "Major contractor for KOC and KNPC. Won KWD 18M tender for deep drilling support in north Kuwait. High capex cycle approaching."},
	{Name: "Burj Al Kuwait Real Estate", NameAr: "برج الكويت العقارية", Industry: "Real Estate", SubIndustry: "Commercial Development", RevenueKWD: 32000000, EmployeeCount: 320, IncorporationYear: 2005, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2018-07-01", Country: "Kuwait", Notes: "Developing a mixed-use tower in Kuwait City. Phase 2 launching Q4. Looking for additional financing."},
	{Name: "Gulf Star Trading Co.", NameAr: "نجمة الخليج للتجارة", Industry: "Trading", SubIndustry: "General Trading & Import", RevenueKWD: 18500000, EmployeeCount: 180, IncorporationYear: 2001, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2019-01-15", Country: "Kuwait", Notes: "Importing electronics and consumer goods from East Asia. Seasonal cash flow challenges in Q3-Q4."},
	{Name: "Al Salhiya Construction Group", NameAr: "الصالحية للمقاولات", Industry: "Construction", SubIndustry: "Civil Engineering", RevenueKWD: 55000000, EmployeeCount: 2200, IncorporationYear: 1992, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2014-09-20", Country: "Kuwait", Notes: "Major government contractor. Recently awarded KWD 25M highway project. Needs performance guarantees and equipment financing."},
	{Name: "Kuwait Digital Solutions", NameAr: "الكويت للحلول الرقمية", Industry: "Technology", SubIndustry: "IT Services & Cloud", RevenueKWD: 8500000, EmployeeCount: 145, IncorporationYear: 2012, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2021-03-10", Country: "Kuwait", Notes: "Fast-growing fintech/IT company. Won government digitization contracts. Scaling headcount and infrastructure."},
	{Name: "Al Watan Healthcare", NameAr: "الوطن للرعاية الصحية", Industry: "Healthcare", SubIndustry: "Private Hospitals", RevenueKWD: 22000000, EmployeeCount: 650, IncorporationYear: 2003, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2017-05-22", Country: "Kuwait", Notes: "Operating 3 clinics. Planning to build a new 200-bed hospital in Hawally. Seeking long-term financing."},
	{Name: "National Marine Logistics", NameAr: "الوطنية للخدمات البحرية", Industry: "Logistics", SubIndustry: "Shipping & Maritime", RevenueKWD: 28000000, EmployeeCount: 430, IncorporationYear: 1995, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2016-11-05", Country: "Kuwait", Notes: "Fleet of 12 vessels. Looking to acquire 3 new cargo ships. Trade routes to India, UAE, and East Africa."},
	{Name: "Kuwait Food Industries", NameAr: "الكويت للصناعات الغذائية", Industry: "Food & Beverage", SubIndustry: "Food Manufacturing", RevenueKWD: 15000000, EmployeeCount: 520, IncorporationYear: 2000, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2019-08-14", Country: "Kuwait", Notes: "Largest local dairy producer. Importing raw materials from New Zealand and Europe. Expanding into Saudi market."},
	{Name: "Desert Technologies LLC", NameAr: "صحراء للتكنولوجيا", Industry: "Manufacturing", SubIndustry: "Solar & Renewable Energy", RevenueKWD: 12000000, EmployeeCount: 210, IncorporationYear: 2010, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2020-02-28", Country: "Kuwait", Notes: "Manufacturing solar panels. Awarded government renewable energy pilot. Needs industrial land and equipment financing."},
	{Name: "Al Jahra Automotive Group", NameAr: "الجهراء للسيارات", Industry: "Automotive", SubIndustry: "Vehicle Distribution", RevenueKWD: 35000000, EmployeeCount: 380, IncorporationYear: 1997, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2013-06-18", Country: "Kuwait", Notes: "Exclusive distributor for 3 auto brands. Opening 2 new showrooms. Large inventory import cycles."},
	{Name: "Pearl Commercial Investments", NameAr: "اللؤلؤة للاستثمارات التجارية", Industry: "Financial Services", SubIndustry: "Investment Holding", RevenueKWD: 60000000, EmployeeCount: 85, IncorporationYear: 2006, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2016-01-30", Country: "Kuwait", Notes: "Holding company with interests in real estate, hospitality, and healthcare. Significant cash reserves seeking investment returns."},
	{Name: "Failaka Engineering Services", NameAr: "فيلكا للخدمات الهندسية", Industry: "Engineering", SubIndustry: "MEP & Industrial", RevenueKWD: 9500000, EmployeeCount: 290, IncorporationYear: 2008, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2020-11-12", Country: "Kuwait", Notes: "MEP contractor for major projects. Bidding on a KWD 8M hospital HVAC project. Needs bid bonds."},
	{Name: "Kuwait Cement Industries", NameAr: "الكويت لصناعة الاسمنت", Industry: "Manufacturing", SubIndustry: "Building Materials", RevenueKWD: 40000000, EmployeeCount: 700, IncorporationYear: 1985, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2012-04-05", Country: "Kuwait", Notes: "Major cement producer. Plant modernization planned for next year. Exporting to Iraq and GCC."},
	{Name: "Al Hamra Hospitality Group", NameAr: "الحمراء للضيافة", Industry: "Hospitality", SubIndustry: "Hotels & Resorts", RevenueKWD: 25000000, EmployeeCount: 1100, IncorporationYear: 2002, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2018-03-20", Country: "Kuwait", Notes: "Operating 4 hotels in Kuwait. Planning a resort project on the Gulf coast. Seasonal cash flow patterns."},
	{Name: "Gulf Chemical Industries", NameAr: "الخليج للصناعات الكيميائية", Industry: "Petrochemicals", SubIndustry: "Chemical Manufacturing", RevenueKWD: 38000000, EmployeeCount: 560, IncorporationYear: 1990, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2015-08-10", Country: "Kuwait", Notes: "Petrochemical derivatives producer. Expanding production capacity. Exporting to Asia and Africa."},
	{Name: "Mishref Pharmaceuticals", NameAr: "مشرف للأدوية", Industry: "Healthcare", SubIndustry: "Pharmaceuticals", RevenueKWD: 11000000, EmployeeCount: 175, IncorporationYear: 2009, RiskRating: models.RiskMedium, KYCStatus: models.KYCPending, RelationshipStart: "2022-01-08", Country: "Kuwait", Notes: "Generic drug manufacturer. Importing APIs from India. Seeking GCC regulatory approvals for expansion. KYC renewal due."},
	{Name: "Salmiya Retail Holdings", NameAr: "السالمية للتجزئة", Industry: "Retail", SubIndustry: "Shopping Malls & Retail", RevenueKWD: 20000000, EmployeeCount: 450, IncorporationYear: 2004, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2017-09-14", Country: "Kuwait", Notes: "Operating 2 shopping malls and 15 retail outlets. Planning mall expansion project. Tenant receivables growing."},
	{Name: "Kuwait Agri-Foods Co.", NameAr: "الكويت للأغذية الزراعية", Industry: "Agriculture", SubIndustry: "Food Import & Distribution", RevenueKWD: 7500000, EmployeeCount: 130, IncorporationYear: 2011, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2021-06-22", Country: "Kuwait", Notes: "Importing fresh produce and grains from Turkey and India. Working capital needs peak during Ramadan season."},
	{Name: "Al Rawda Printing & Packaging", NameAr: "الروضة للطباعة والتغليف", Industry: "Manufacturing", SubIndustry: "Packaging Materials", RevenueKWD: 6800000, EmployeeCount: 160, IncorporationYear: 2007, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2021-09-01", Country: "Kuwait", Notes: "Supplying packaging to food and pharma companies. Buying new German printing press (EUR 1.5M). Needs import LC and asset financing."},
	{Name: "Shuwaikh Metal Works", NameAr: "الشويخ للصناعات المعدنية", Industry: "Manufacturing", SubIndustry: "Steel & Metal Fabrication", RevenueKWD: 16000000, EmployeeCount: 340, IncorporationYear: 1999, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2016-04-18", Country: "Kuwait", Notes: "Steel fabrication for construction projects. Raw material prices volatile. Supplier payments in USD. Needs FX hedging and working capital."},
}

var seedClientProductsTemplate = map[int][]models.ClientProduct{
	0: {
		{ProductID: "prod-murabaha-wc", StartDate: "2023-01-15", AmountKWD: 2500000, Status: "Active"},
		{ProductID: "prod-trade-lg", StartDate: "2024-03-01", AmountKWD: 5000000, Status: "Active"},
		{ProductID: "prod-payroll-wps", StartDate: "2019-06-01", AmountKWD: 0, Status: "Active"},
	},
	1: {
		{ProductID: "prod-project-istisna", StartDate: "2022-06-01", AmountKWD: 8000000, Status: "Active"},
		{ProductID: "prod-treasury-wakala", StartDate: "2023-09-15", AmountKWD: 3000000, Status: "Active"},
	},
	2: {
		{ProductID: "prod-trade-lc", StartDate: "2024-01-10", AmountKWD: 1500000, Status: "Active"},
		{ProductID: "prod-murabaha-wc", StartDate: "2023-06-20", AmountKWD: 800000, Status: "Active"},
	},
	3: {
		{ProductID: "prod-trade-lg", StartDate: "2023-04-15", AmountKWD: 10000000, Status: "Active"},
		{ProductID: "prod-ijara-mb", StartDate: "2022-11-01", AmountKWD: 3500000, Status: "Active"},
		{ProductID: "prod-payroll-wps", StartDate: "2018-01-01", AmountKWD: 0, Status: "Active"},
	},
	4: {
		{ProductID: "prod-pos-finance", StartDate: "2023-08-01", AmountKWD: 150000, Status: "Active"},
		{ProductID: "prod-payroll-wps", StartDate: "2022-04-01", AmountKWD: 0, Status: "Active"},
	},
	5: {
		{ProductID: "prod-ijara-mb", StartDate: "2021-03-15", AmountKWD: 2000000, Status: "Active"},
		{ProductID: "prod-trade-lc", StartDate: "2023-10-01", AmountKWD: 500000, Status: "Active"},
	},
	6: {
		{ProductID: "prod-ijara-mb", StartDate: "2020-07-01", AmountKWD: 6000000, Status: "Active"},
		{ProductID: "prod-trade-lc", StartDate: "2024-02-15", AmountKWD: 2000000, Status: "Active"},
		{ProductID: "prod-fx-waad", StartDate: "2023-11-01", AmountKWD: 1000000, Status: "Active"},
	},
	7: {
		{ProductID: "prod-trade-lc", StartDate: "2023-05-01", AmountKWD: 1200000, Status: "Active"},
		{ProductID: "prod-murabaha-wc", StartDate: "2024-01-15", AmountKWD: 750000, Status: "Active"},
	},
	8: {
		{ProductID: "prod-murabaha-wc", StartDate: "2022-09-01", AmountKWD: 500000, Status: "Active"},
	},
	9: {
		{ProductID: "prod-trade-lc", StartDate: "2023-03-01", AmountKWD: 4000000, Status: "Active"},
		{ProductID: "prod-trade-lg", StartDate: "2023-08-15", AmountKWD: 1000000, Status: "Active"},
		{ProductID: "prod-fx-waad", StartDate: "2024-01-01", AmountKWD: 2500000, Status: "Active"},
	},
}

var seedInteractionsTemplate = map[int][]models.Interaction{
	0: {
		{Type: "Meeting", Date: "2026-06-15", Summary: "Quarterly review with CFO Tariq Al-Ghanim. Discussed upcoming KOC drilling tender.", Outcome: "Client confirmed they will bid on KWD 18M deep drilling package. Will need new heavy drilling equipment."},
		{Type: "Call", Date: "2026-07-02", Summary: "Follow-up on tender submission. Client was shortlisted as lowest bidder.", Outcome: "Expected award date late August. Will require KWD 5M Ijara for 2 new rig packages."},
		{Type: "Email", Date: "2026-08-10", Summary: "Sent preliminary term sheet for Ijara facility.", Outcome: "CFO reviewing pricing and Shariah structure."},
	},
	1: {
		{Type: "Meeting", Date: "2026-05-20", Summary: "Site visit at Burj Al Kuwait site with Managing Director.", Outcome: "Phase 1 is 85% leased. Phase 2 foundation work begins in October. Budget KWD 12M."},
		{Type: "Call", Date: "2026-07-18", Summary: "Discussion on Phase 2 financing structure.", Outcome: "Client prefers Istisna'a structure over conventional bridge loan."},
	},
	2: {
		{Type: "Call", Date: "2026-06-28", Summary: "Discussed supply chain disruptions from Asia. Shipping times increased 3 weeks.", Outcome: "Needs additional working capital buffer for inventory holding."},
		{Type: "Meeting", Date: "2026-08-05", Summary: "Reviewed FX exposure. Majority of imports in USD and CNY.", Outcome: "Client interested in FX Wa'ad hedging to lock in rates."},
	},
	3: {
		{Type: "Meeting", Date: "2026-07-10", Summary: "Meeting with CEO regarding KWD 25M Highway contract award.", Outcome: "Requires KWD 2.5M performance bond, KWD 3.75M advance payment guarantee."},
		{Type: "Note", Date: "2026-07-25", Summary: "Client credit assessment updated: excellent track record with Ministry of Public Works.", Outcome: "Recommend full support on LG package."},
	},
}

type oppTemplate struct {
	ClientIdx    int
	ProductID    string
	Confidence   float64
	Urgency      models.Urgency
	Reasoning    string
	NextAction   string
	ShariahNotes string
	Status       models.OpportunityStatus
	HoursAgo     int
}

var seedOpportunitiesTemplate = []oppTemplate{
	{
		ClientIdx:    0,
		ProductID:    "prod-ijara-mb",
		Confidence:   0.92,
		Urgency:      models.UrgencyCritical,
		Reasoning:    "Client won KWD 18M KOC tender requiring 2 new drilling rig packages. Has existing Murabaha with excellent track record. Ijara Muntahia Bittamleek is optimal for high-value equipment acquisition.",
		NextAction:   "Schedule meeting with CFO Tariq Al-Ghanim to present Ijara term sheet with 5-year tenure and KWD 5M facility limit.",
		ShariahNotes: "Asset-backed lease with separate promise of ownership transfer at maturity. Compliant with AAOIFI Shariah Standard No. 9.",
		Status:       models.OpportunityNew,
		HoursAgo:     4,
	},
	{
		ClientIdx:    3,
		ProductID:    "prod-trade-lg",
		Confidence:   0.88,
		Urgency:      models.UrgencyHigh,
		Reasoning:    "Awarded KWD 25M Ministry of Public Works highway project. Needs performance guarantee (10% = KWD 2.5M) and advance payment guarantee (15% = KWD 3.75M) within 30 days.",
		NextAction:   "Issue credit approval for KWD 6.25M combined Kafalah LG package with reduced commission rates.",
		ShariahNotes: "Kafalah structure with fee charged for administrative and documentation services only, compliant with Shariah Board rulings.",
		Status:       models.OpportunityNew,
		HoursAgo:     12,
	},
	{
		ClientIdx:    1,
		ProductID:    "prod-project-istisna",
		Confidence:   0.85,
		Urgency:      models.UrgencyHigh,
		Reasoning:    "Burj Al Kuwait Phase 2 launching Q4 with KWD 12M construction budget. Phase 1 is 85% leased showing strong project viability. Existing Istisna'a performing well.",
		NextAction:   "Propose Parallel Istisna'a structure for Phase 2 construction financing with milestone-based disbursements.",
		ShariahNotes: "Manufacturing sale contract with specifications and delivery schedule. Bank contracts with developer and assigns construction to certified contractor.",
		Status:       models.OpportunityReviewed,
		HoursAgo:     36,
	},
	{
		ClientIdx:    2,
		ProductID:    "prod-fx-waad",
		Confidence:   0.78,
		Urgency:      models.UrgencyMedium,
		Reasoning:    "Import volume from Asia growing 25% YoY. USD/KWD and CNY exposure creates margin volatility. Client has no active FX hedging in place.",
		NextAction:   "Introduce Treasury FX team to structure Islamic FX Wa'ad forward cover for next 2 quarters of import cycles.",
		ShariahNotes: "Unilateral binding promise structure (Wa'ad) avoiding bilateral forward exchange contract prohibitions (Ina / Riba al-Fadl).",
		Status:       models.OpportunityNew,
		HoursAgo:     48,
	},
	{
		ClientIdx:    6,
		ProductID:    "prod-ijara-mb",
		Confidence:   0.81,
		Urgency:      models.UrgencyMedium,
		Reasoning:    "Client expanding cargo fleet with 3 new vessels. Trade route expansion to East Africa driving 30% revenue growth. Existing Ijara facility in final year.",
		NextAction:   "Structure maritime vessel financing under Marine Ijara with international registry mortgage.",
		ShariahNotes: "Vessel lease with takaful coverage requirement and maintenance obligations aligned with AAOIFI standard.",
		Status:       models.OpportunityAccepted,
		HoursAgo:     72,
	},
}
