package store

import (
	"opportunity-engine/internal/models"
)

// Seed populates the database with realistic synthetic data modelling
// Warba Bank's corporate banking portfolio in Kuwait.
func (s *Store) Seed() error {
	// --- Relationship Managers ---
	rms := []models.RelationshipManager{
		{ID: "rm-001", Name: "Ahmad Al-Mutairi", Title: "Senior Relationship Manager", Email: "a.mutairi@warbabank.com", Department: "Corporate Banking"},
		{ID: "rm-002", Name: "Fatima Al-Rashidi", Title: "Relationship Manager", Email: "f.rashidi@warbabank.com", Department: "Corporate Banking"},
		{ID: "rm-003", Name: "Khalid Al-Sabah", Title: "VP Corporate Banking", Email: "k.sabah@warbabank.com", Department: "Corporate Banking"},
	}
	for _, rm := range rms {
		if err := s.InsertRM(rm); err != nil {
			return err
		}
	}

	// --- Shariah-Compliant Product Catalog ---
	products := []models.Product{
		{ID: "prod-001", Name: "Commodity Murabaha", NameAr: "مرابحة سلعية", Category: "Working Capital", Description: "Working capital financing through commodity Murabaha structure providing liquidity for operating expenses and cash payments.", ShariahStructure: "Murabaha (cost-plus sale)", MinAmountKWD: 50000, MaxAmountKWD: 10000000, TypicalTenureMonths: 6, TargetIndustries: "All", IsActive: true},
		{ID: "prod-002", Name: "Forward Murabaha", NameAr: "مرابحة آجلة", Category: "Asset Finance", Description: "Asset financing for the purchase of machinery, equipment, production lines, furniture, decoration, or land through deferred sale.", ShariahStructure: "Murabaha (deferred sale)", MinAmountKWD: 100000, MaxAmountKWD: 25000000, TypicalTenureMonths: 36, TargetIndustries: "Manufacturing, Construction, Oil & Gas, Industrial", IsActive: true},
		{ID: "prod-003", Name: "Real Estate Ijara", NameAr: "إجارة عقارية", Category: "Real Estate", Description: "Property financing ending with ownership transfer, suitable for purchasing real estate or new project development.", ShariahStructure: "Ijara Muntahia Bittamleek (lease-to-own)", MinAmountKWD: 500000, MaxAmountKWD: 50000000, TypicalTenureMonths: 84, TargetIndustries: "Real Estate, Hospitality, Retail, All", IsActive: true},
		{ID: "prod-004", Name: "Industrial Plot Ijara", NameAr: "إجارة أراضي صناعية", Category: "Real Estate", Description: "Financing for trading or investing in the usufruct of industrial plots leased from the state.", ShariahStructure: "Ijara (usufruct lease)", MinAmountKWD: 200000, MaxAmountKWD: 15000000, TypicalTenureMonths: 48, TargetIndustries: "Manufacturing, Industrial, Oil & Gas, Logistics", IsActive: true},
		{ID: "prod-005", Name: "Documentary Letter of Credit", NameAr: "اعتماد مستندي", Category: "Trade Finance", Description: "Trade finance tool for import/export transactions enabling secure international trade settlement.", ShariahStructure: "Wakala (agency)", MinAmountKWD: 25000, MaxAmountKWD: 20000000, TypicalTenureMonths: 6, TargetIndustries: "Trading, Manufacturing, Food & Beverage, All", IsActive: true},
		{ID: "prod-006", Name: "Letter of Guarantee", NameAr: "خطاب ضمان", Category: "Trade Finance", Description: "Bid bonds, performance guarantees, and advance payment guarantees for commercial and industrial companies.", ShariahStructure: "Kafalah (guarantee)", MinAmountKWD: 10000, MaxAmountKWD: 30000000, TypicalTenureMonths: 12, TargetIndustries: "Construction, Oil & Gas, Government Contracting, All", IsActive: true},
		{ID: "prod-007", Name: "Syndication Finance", NameAr: "تمويل مشترك", Category: "Structured Finance", Description: "Large-scale project financing for infrastructure and mega-projects through participatory structures.", ShariahStructure: "Musharakah/Murabaha (participatory)", MinAmountKWD: 5000000, MaxAmountKWD: 100000000, TypicalTenureMonths: 84, TargetIndustries: "Oil & Gas, Real Estate, Infrastructure, Telecom", IsActive: true},
		{ID: "prod-008", Name: "Corporate Wakala Deposit", NameAr: "وديعة وكالة", Category: "Treasury", Description: "Short-term surplus fund placement through Wakala agency investment structure.", ShariahStructure: "Wakala (agency investment)", MinAmountKWD: 100000, MaxAmountKWD: 50000000, TypicalTenureMonths: 6, TargetIndustries: "All", IsActive: true},
		{ID: "prod-009", Name: "Mudaraba Investment Account", NameAr: "حساب استثمار مضاربة", Category: "Investment", Description: "Medium-term profit-sharing investment arrangement for corporate surplus funds.", ShariahStructure: "Mudaraba (profit-sharing)", MinAmountKWD: 250000, MaxAmountKWD: 25000000, TypicalTenureMonths: 18, TargetIndustries: "All", IsActive: true},
		{ID: "prod-010", Name: "Invoice Discounting", NameAr: "خصم فواتير", Category: "Receivables Finance", Description: "Accelerating receivables collection and improving cash flow through sale of receivables.", ShariahStructure: "Bai Al-Dayn (sale of debt)", MinAmountKWD: 50000, MaxAmountKWD: 5000000, TypicalTenureMonths: 3, TargetIndustries: "Trading, Manufacturing, Services, Construction", IsActive: true},
	}
	for _, p := range products {
		if err := s.InsertProduct(p); err != nil {
			return err
		}
	}

	// --- Corporate Clients ---
	clients := []models.Client{
		{ID: "cli-001", Name: "Al Khaleej Petroleum Services", NameAr: "الخليج لخدمات البترول", Industry: "Oil & Gas", SubIndustry: "Oilfield Services", RevenueKWD: 45000000, EmployeeCount: 850, IncorporationYear: 1998, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2015-03-12", RMID: "rm-001", Country: "Kuwait", Notes: "Expanding into offshore drilling. Won a KOC contract in Q1 2026. Considering new equipment acquisition."},
		{ID: "cli-002", Name: "Burj Al Kuwait Real Estate", NameAr: "برج الكويت العقارية", Industry: "Real Estate", SubIndustry: "Commercial Development", RevenueKWD: 32000000, EmployeeCount: 320, IncorporationYear: 2005, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2018-07-01", RMID: "rm-001", Country: "Kuwait", Notes: "Developing a mixed-use tower in Kuwait City. Phase 2 launching Q4 2026. Looking for additional financing."},
		{ID: "cli-003", Name: "Gulf Star Trading Co.", NameAr: "نجمة الخليج للتجارة", Industry: "Trading", SubIndustry: "General Trading & Import", RevenueKWD: 18500000, EmployeeCount: 180, IncorporationYear: 2001, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2019-01-15", RMID: "rm-002", Country: "Kuwait", Notes: "Importing electronics and consumer goods from East Asia. Seasonal cash flow challenges in Q3-Q4."},
		{ID: "cli-004", Name: "Al Salhiya Construction Group", NameAr: "الصالحية للمقاولات", Industry: "Construction", SubIndustry: "Civil Engineering", RevenueKWD: 55000000, EmployeeCount: 2200, IncorporationYear: 1992, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2014-09-20", RMID: "rm-003", Country: "Kuwait", Notes: "Major government contractor. Recently awarded KWD 25M highway project. Needs performance guarantees and equipment financing."},
		{ID: "cli-005", Name: "Kuwait Digital Solutions", NameAr: "الكويت للحلول الرقمية", Industry: "Technology", SubIndustry: "IT Services & Cloud", RevenueKWD: 8500000, EmployeeCount: 145, IncorporationYear: 2012, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2021-03-10", RMID: "rm-002", Country: "Kuwait", Notes: "Fast-growing fintech/IT company. Won government digitization contracts. Scaling headcount and infrastructure."},
		{ID: "cli-006", Name: "Al Watan Healthcare", NameAr: "الوطن للرعاية الصحية", Industry: "Healthcare", SubIndustry: "Private Hospitals", RevenueKWD: 22000000, EmployeeCount: 650, IncorporationYear: 2003, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2017-05-22", RMID: "rm-001", Country: "Kuwait", Notes: "Operating 3 clinics. Planning to build a new 200-bed hospital in Hawally. Seeking long-term financing."},
		{ID: "cli-007", Name: "National Marine Logistics", NameAr: "الوطنية للخدمات البحرية", Industry: "Logistics", SubIndustry: "Shipping & Maritime", RevenueKWD: 28000000, EmployeeCount: 430, IncorporationYear: 1995, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2016-11-05", RMID: "rm-003", Country: "Kuwait", Notes: "Fleet of 12 vessels. Looking to acquire 3 new cargo ships. Trade routes to India, UAE, and East Africa."},
		{ID: "cli-008", Name: "Kuwait Food Industries", NameAr: "الكويت للصناعات الغذائية", Industry: "Food & Beverage", SubIndustry: "Food Manufacturing", RevenueKWD: 15000000, EmployeeCount: 520, IncorporationYear: 2000, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2019-08-14", RMID: "rm-002", Country: "Kuwait", Notes: "Largest local dairy producer. Importing raw materials from New Zealand and Europe. Expanding into Saudi market."},
		{ID: "cli-009", Name: "Desert Technologies LLC", NameAr: "صحراء للتكنولوجيا", Industry: "Manufacturing", SubIndustry: "Solar & Renewable Energy", RevenueKWD: 12000000, EmployeeCount: 210, IncorporationYear: 2010, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2020-02-28", RMID: "rm-001", Country: "Kuwait", Notes: "Manufacturing solar panels. Awarded government renewable energy pilot. Needs industrial land and equipment financing."},
		{ID: "cli-010", Name: "Al Jahra Automotive Group", NameAr: "الجهراء للسيارات", Industry: "Automotive", SubIndustry: "Vehicle Distribution", RevenueKWD: 35000000, EmployeeCount: 380, IncorporationYear: 1997, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2013-06-18", RMID: "rm-003", Country: "Kuwait", Notes: "Exclusive distributor for 3 auto brands. Opening 2 new showrooms. Large inventory import cycles."},
		{ID: "cli-011", Name: "Pearl Commercial Investments", NameAr: "اللؤلؤة للاستثمارات التجارية", Industry: "Financial Services", SubIndustry: "Investment Holding", RevenueKWD: 60000000, EmployeeCount: 85, IncorporationYear: 2006, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2016-01-30", RMID: "rm-003", Country: "Kuwait", Notes: "Holding company with interests in real estate, hospitality, and healthcare. Significant cash reserves seeking investment returns."},
		{ID: "cli-012", Name: "Failaka Engineering Services", NameAr: "فيلكا للخدمات الهندسية", Industry: "Engineering", SubIndustry: "MEP & Industrial", RevenueKWD: 9500000, EmployeeCount: 290, IncorporationYear: 2008, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2020-11-12", RMID: "rm-002", Country: "Kuwait", Notes: "MEP contractor for major projects. Bidding on a KWD 8M hospital HVAC project. Needs bid bonds."},
		{ID: "cli-013", Name: "Kuwait Cement Industries", NameAr: "الكويت لصناعة الاسمنت", Industry: "Manufacturing", SubIndustry: "Building Materials", RevenueKWD: 40000000, EmployeeCount: 700, IncorporationYear: 1985, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2012-04-05", RMID: "rm-001", Country: "Kuwait", Notes: "Major cement producer. Plant modernization planned for 2027. Exporting to Iraq and GCC."},
		{ID: "cli-014", Name: "Al Hamra Hospitality Group", NameAr: "الحمراء للضيافة", Industry: "Hospitality", SubIndustry: "Hotels & Resorts", RevenueKWD: 25000000, EmployeeCount: 1100, IncorporationYear: 2002, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2018-03-20", RMID: "rm-001", Country: "Kuwait", Notes: "Operating 4 hotels in Kuwait. Planning a resort project on the Gulf coast. Seasonal cash flow patterns."},
		{ID: "cli-015", Name: "Gulf Chemical Industries", NameAr: "الخليج للصناعات الكيميائية", Industry: "Petrochemicals", SubIndustry: "Chemical Manufacturing", RevenueKWD: 38000000, EmployeeCount: 560, IncorporationYear: 1990, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2015-08-10", RMID: "rm-003", Country: "Kuwait", Notes: "Petrochemical derivatives producer. Expanding production capacity. Exporting to Asia and Africa."},
		{ID: "cli-016", Name: "Mishref Pharmaceuticals", NameAr: "مشرف للأدوية", Industry: "Healthcare", SubIndustry: "Pharmaceuticals", RevenueKWD: 11000000, EmployeeCount: 175, IncorporationYear: 2009, RiskRating: models.RiskMedium, KYCStatus: models.KYCPending, RelationshipStart: "2022-01-08", RMID: "rm-002", Country: "Kuwait", Notes: "Generic drug manufacturer. Importing APIs from India. Seeking GCC regulatory approvals for expansion. KYC renewal due."},
		{ID: "cli-017", Name: "Salmiya Retail Holdings", NameAr: "السالمية للتجزئة", Industry: "Retail", SubIndustry: "Shopping Malls & Retail", RevenueKWD: 20000000, EmployeeCount: 450, IncorporationYear: 2004, RiskRating: models.RiskLow, KYCStatus: models.KYCActive, RelationshipStart: "2017-09-14", RMID: "rm-003", Country: "Kuwait", Notes: "Operating 2 shopping malls and 15 retail outlets. Planning mall expansion project. Tenant receivables growing."},
		{ID: "cli-018", Name: "Kuwait Agri-Foods Co.", NameAr: "الكويت للأغذية الزراعية", Industry: "Agriculture", SubIndustry: "Food Import & Distribution", RevenueKWD: 7500000, EmployeeCount: 130, IncorporationYear: 2011, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2021-06-22", RMID: "rm-002", Country: "Kuwait", Notes: "Importing fresh produce and grains from Turkey and India. Working capital needs peak during Ramadan season."},
		{ID: "cli-019", Name: "Al Bidaa Contracting", NameAr: "البدع للمقاولات", Industry: "Construction", SubIndustry: "Building Construction", RevenueKWD: 16000000, EmployeeCount: 800, IncorporationYear: 2007, RiskRating: models.RiskHigh, KYCStatus: models.KYCActive, RelationshipStart: "2023-02-01", RMID: "rm-002", Country: "Kuwait", Notes: "Mid-size contractor. Recent cash flow pressures due to delayed government payments. Needs receivables acceleration."},
		{ID: "cli-020", Name: "Digital Gulf Media", NameAr: "الخليج الرقمي للإعلام", Industry: "Media", SubIndustry: "Digital Advertising", RevenueKWD: 4500000, EmployeeCount: 65, IncorporationYear: 2015, RiskRating: models.RiskMedium, KYCStatus: models.KYCActive, RelationshipStart: "2023-09-05", RMID: "rm-001", Country: "Kuwait", Notes: "Digital-first media agency. Growing rapidly. Minimal banking products currently. Looking to establish corporate treasury."},
	}
	for _, c := range clients {
		if err := s.InsertClient(c); err != nil {
			return err
		}
	}

	// --- Client Product Holdings ---
	clientProducts := []models.ClientProduct{
		// Al Khaleej Petroleum — has working capital + trade finance
		{ClientID: "cli-001", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2023-06-15", AmountKWD: 5000000, Status: "Active"},
		{ClientID: "cli-001", ProductID: "prod-006", ProductName: "Letter of Guarantee", StartDate: "2024-01-10", AmountKWD: 8000000, Status: "Active"},
		// Burj Al Kuwait — real estate financing
		{ClientID: "cli-002", ProductID: "prod-003", ProductName: "Real Estate Ijara", StartDate: "2020-03-20", AmountKWD: 18000000, Status: "Active"},
		{ClientID: "cli-002", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-05-01", AmountKWD: 3000000, Status: "Active"},
		// Gulf Star Trading — trade finance
		{ClientID: "cli-003", ProductID: "prod-005", ProductName: "Documentary Letter of Credit", StartDate: "2023-09-01", AmountKWD: 2500000, Status: "Active"},
		// Al Salhiya Construction — guarantees + working capital
		{ClientID: "cli-004", ProductID: "prod-006", ProductName: "Letter of Guarantee", StartDate: "2022-11-15", AmountKWD: 15000000, Status: "Active"},
		{ClientID: "cli-004", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-03-01", AmountKWD: 8000000, Status: "Active"},
		{ClientID: "cli-004", ProductID: "prod-002", ProductName: "Forward Murabaha", StartDate: "2023-07-20", AmountKWD: 4500000, Status: "Active"},
		// Kuwait Digital Solutions — only working capital
		{ClientID: "cli-005", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-01-15", AmountKWD: 1500000, Status: "Active"},
		// Al Watan Healthcare — real estate + working capital
		{ClientID: "cli-006", ProductID: "prod-003", ProductName: "Real Estate Ijara", StartDate: "2021-08-10", AmountKWD: 8000000, Status: "Active"},
		{ClientID: "cli-006", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-06-01", AmountKWD: 2000000, Status: "Active"},
		// National Marine Logistics — asset finance + trade
		{ClientID: "cli-007", ProductID: "prod-002", ProductName: "Forward Murabaha", StartDate: "2022-05-10", AmountKWD: 12000000, Status: "Active"},
		{ClientID: "cli-007", ProductID: "prod-005", ProductName: "Documentary Letter of Credit", StartDate: "2024-02-15", AmountKWD: 4000000, Status: "Active"},
		// Kuwait Food Industries — trade finance + working capital
		{ClientID: "cli-008", ProductID: "prod-005", ProductName: "Documentary Letter of Credit", StartDate: "2023-04-20", AmountKWD: 3500000, Status: "Active"},
		{ClientID: "cli-008", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-08-01", AmountKWD: 2000000, Status: "Active"},
		// Desert Technologies — working capital only
		{ClientID: "cli-009", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-04-10", AmountKWD: 1000000, Status: "Active"},
		// Al Jahra Automotive — trade finance + working capital
		{ClientID: "cli-010", ProductID: "prod-005", ProductName: "Documentary Letter of Credit", StartDate: "2022-08-15", AmountKWD: 8000000, Status: "Active"},
		{ClientID: "cli-010", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-01-20", AmountKWD: 5000000, Status: "Active"},
		// Pearl Commercial — deposits + investment
		{ClientID: "cli-011", ProductID: "prod-008", ProductName: "Corporate Wakala Deposit", StartDate: "2023-03-01", AmountKWD: 20000000, Status: "Active"},
		{ClientID: "cli-011", ProductID: "prod-009", ProductName: "Mudaraba Investment Account", StartDate: "2023-06-15", AmountKWD: 15000000, Status: "Active"},
		// Failaka Engineering — working capital
		{ClientID: "cli-012", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-02-01", AmountKWD: 1200000, Status: "Active"},
		// Kuwait Cement — asset finance + working capital
		{ClientID: "cli-013", ProductID: "prod-002", ProductName: "Forward Murabaha", StartDate: "2021-09-10", AmountKWD: 10000000, Status: "Active"},
		{ClientID: "cli-013", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-07-15", AmountKWD: 6000000, Status: "Active"},
		// Al Hamra Hospitality — real estate
		{ClientID: "cli-014", ProductID: "prod-003", ProductName: "Real Estate Ijara", StartDate: "2020-11-01", AmountKWD: 12000000, Status: "Active"},
		// Gulf Chemical Industries — working capital + trade
		{ClientID: "cli-015", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2023-12-01", AmountKWD: 7000000, Status: "Active"},
		{ClientID: "cli-015", ProductID: "prod-005", ProductName: "Documentary Letter of Credit", StartDate: "2024-03-10", AmountKWD: 5000000, Status: "Active"},
		// Mishref Pharma — trade finance
		{ClientID: "cli-016", ProductID: "prod-005", ProductName: "Documentary Letter of Credit", StartDate: "2024-05-01", AmountKWD: 1800000, Status: "Active"},
		// Salmiya Retail — real estate + working capital
		{ClientID: "cli-017", ProductID: "prod-003", ProductName: "Real Estate Ijara", StartDate: "2019-06-10", AmountKWD: 14000000, Status: "Active"},
		{ClientID: "cli-017", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-04-15", AmountKWD: 3000000, Status: "Active"},
		// Kuwait Agri-Foods — trade finance
		{ClientID: "cli-018", ProductID: "prod-005", ProductName: "Documentary Letter of Credit", StartDate: "2024-01-10", AmountKWD: 1500000, Status: "Active"},
		// Al Bidaa Contracting — working capital only
		{ClientID: "cli-019", ProductID: "prod-001", ProductName: "Commodity Murabaha", StartDate: "2024-06-20", AmountKWD: 2500000, Status: "Active"},
		// Digital Gulf Media — no products yet (new client)
	}
	for _, cp := range clientProducts {
		if err := s.InsertClientProduct(cp); err != nil {
			return err
		}
	}

	// --- Interaction History ---
	interactions := []models.Interaction{
		// Al Khaleej Petroleum
		{ID: "int-001", ClientID: "cli-001", Type: "Meeting", Date: "2026-08-15", Summary: "Quarterly review. Client mentioned KOC contract expansion worth KWD 12M. Discussing equipment procurement timeline.", Outcome: "Follow up on equipment financing needs", RMID: "rm-001"},
		{ID: "int-002", ClientID: "cli-001", Type: "Call", Date: "2026-07-20", Summary: "CFO called regarding upcoming LC renewal for drilling equipment from Houston supplier.", Outcome: "Prepare LC renewal documentation", RMID: "rm-001"},
		{ID: "int-003", ClientID: "cli-001", Type: "Email", Date: "2026-06-10", Summary: "Shared annual report. Revenue up 18% YoY. EBITDA margins improving.", Outcome: "Noted for review", RMID: "rm-001"},

		// Burj Al Kuwait Real Estate
		{ID: "int-004", ClientID: "cli-002", Type: "Meeting", Date: "2026-08-22", Summary: "Phase 2 tower project discussion. Client needs additional KWD 15M financing. Architect plans finalized.", Outcome: "Prepare real estate ijara proposal for Phase 2", RMID: "rm-001"},
		{ID: "int-005", ClientID: "cli-002", Type: "Transaction", Date: "2026-08-01", Summary: "Murabaha facility partially drawn down — KWD 1.5M for contractor payments.", Outcome: "Monitor utilization", RMID: "rm-001"},

		// Gulf Star Trading
		{ID: "int-006", ClientID: "cli-003", Type: "Call", Date: "2026-08-10", Summary: "Client experiencing delayed receivables from a major retail chain. Needs short-term liquidity support.", Outcome: "Explore invoice discounting options", RMID: "rm-002"},
		{ID: "int-007", ClientID: "cli-003", Type: "Meeting", Date: "2026-07-05", Summary: "Semi-annual review. Discussed expanding import sources to include Vietnam. New LC facilities may be needed.", Outcome: "Prepare new LC facility for Vietnam suppliers", RMID: "rm-002"},

		// Al Salhiya Construction
		{ID: "int-008", ClientID: "cli-004", Type: "Meeting", Date: "2026-08-25", Summary: "New highway project kick-off. Needs KWD 10M performance guarantee and KWD 5M advance payment guarantee.", Outcome: "Process LG applications immediately", RMID: "rm-003"},
		{ID: "int-009", ClientID: "cli-004", Type: "Call", Date: "2026-07-30", Summary: "Procurement team requesting equipment finance for 20 new excavators from Caterpillar.", Outcome: "Prepare Forward Murabaha quote", RMID: "rm-003"},

		// Kuwait Digital Solutions
		{ID: "int-010", ClientID: "cli-005", Type: "Meeting", Date: "2026-08-05", Summary: "Fast-growing client. Wants to set up corporate treasury management. Discussed Wakala deposits for surplus cash.", Outcome: "Present treasury solutions package", RMID: "rm-002"},
		{ID: "int-011", ClientID: "cli-005", Type: "Note", Date: "2026-07-15", Summary: "Won MoI digitization contract worth KWD 3M. Working capital needs expected to increase.", Outcome: "Monitor WC utilization", RMID: "rm-002"},

		// Al Watan Healthcare
		{ID: "int-012", ClientID: "cli-006", Type: "Meeting", Date: "2026-08-18", Summary: "Hospital construction project in Hawally. Pre-feasibility study complete. Seeking KWD 20M Ijara financing.", Outcome: "Schedule credit committee pre-approval meeting", RMID: "rm-001"},
		{ID: "int-013", ClientID: "cli-006", Type: "Call", Date: "2026-07-25", Summary: "Medical equipment procurement from Germany — KWD 3M. Needs asset financing.", Outcome: "Prepare Forward Murabaha for equipment", RMID: "rm-001"},

		// National Marine Logistics
		{ID: "int-014", ClientID: "cli-007", Type: "Meeting", Date: "2026-08-12", Summary: "Fleet expansion discussion. 3 new cargo vessels at KWD 5M each from South Korean shipyard.", Outcome: "Assess syndication vs direct financing", RMID: "rm-003"},

		// Kuwait Food Industries
		{ID: "int-015", ClientID: "cli-008", Type: "Call", Date: "2026-08-20", Summary: "Saudi expansion plans progressing. Need warehouse lease financing in Riyadh.", Outcome: "Explore cross-border real estate options", RMID: "rm-002"},
		{ID: "int-016", ClientID: "cli-008", Type: "Transaction", Date: "2026-08-05", Summary: "LC opened for NZD 2.1M dairy imports from Fonterra, New Zealand.", Outcome: "LC processed", RMID: "rm-002"},

		// Desert Technologies
		{ID: "int-017", ClientID: "cli-009", Type: "Meeting", Date: "2026-08-08", Summary: "Government renewable energy project awarded. Needs industrial land lease and KWD 4M production line.", Outcome: "Prepare combined Ijara + Forward Murabaha proposal", RMID: "rm-001"},

		// Al Jahra Automotive
		{ID: "int-018", ClientID: "cli-010", Type: "Meeting", Date: "2026-08-14", Summary: "New showroom openings in Q4. Real estate financing needed for 2 properties totaling KWD 6M.", Outcome: "Begin Ijara application process", RMID: "rm-003"},
		{ID: "int-019", ClientID: "cli-010", Type: "Transaction", Date: "2026-07-28", Summary: "LC for JPY 500M vehicle import from Toyota Motor Corporation.", Outcome: "LC processed and confirmed", RMID: "rm-003"},

		// Pearl Commercial Investments
		{ID: "int-020", ClientID: "cli-011", Type: "Meeting", Date: "2026-08-20", Summary: "Portfolio rebalancing discussion. KWD 10M Wakala deposit maturing in 30 days. Considering longer-term Mudaraba.", Outcome: "Present Mudaraba extension options", RMID: "rm-003"},

		// Failaka Engineering
		{ID: "int-021", ClientID: "cli-012", Type: "Call", Date: "2026-08-16", Summary: "Bidding on KWD 8M hospital HVAC project. Needs bid bond (LG) of KWD 800K.", Outcome: "Process LG application", RMID: "rm-002"},
		{ID: "int-022", ClientID: "cli-012", Type: "Note", Date: "2026-07-20", Summary: "Company growing — hired 30 new engineers. May need office space financing.", Outcome: "Discuss real estate needs at next meeting", RMID: "rm-002"},

		// Kuwait Cement Industries
		{ID: "int-023", ClientID: "cli-013", Type: "Meeting", Date: "2026-08-10", Summary: "Plant modernization project briefing. KWD 15M equipment upgrade planned for 2027. Exploring financing options.", Outcome: "Prepare Forward Murabaha proposal for equipment", RMID: "rm-001"},
		{ID: "int-024", ClientID: "cli-013", Type: "Transaction", Date: "2026-08-01", Summary: "Export LC to Iraq — KWD 3.5M cement shipment.", Outcome: "LC processed", RMID: "rm-001"},

		// Al Hamra Hospitality
		{ID: "int-025", ClientID: "cli-014", Type: "Meeting", Date: "2026-08-22", Summary: "Gulf coast resort project discussion. KWD 25M total cost. Seeking structured finance package.", Outcome: "Evaluate syndication vs bilateral", RMID: "rm-001"},
		{ID: "int-026", ClientID: "cli-014", Type: "Call", Date: "2026-07-10", Summary: "Q3 typically low season — cash flow management needed. Discussed Wakala deposit for off-season reserves.", Outcome: "Present seasonal treasury plan", RMID: "rm-001"},

		// Gulf Chemical Industries
		{ID: "int-027", ClientID: "cli-015", Type: "Meeting", Date: "2026-08-05", Summary: "Capacity expansion project. New production line KWD 8M. Also need industrial land plot in Shuaiba.", Outcome: "Package Ijara + Forward Murabaha", RMID: "rm-003"},

		// Mishref Pharmaceuticals
		{ID: "int-028", ClientID: "cli-016", Type: "Call", Date: "2026-08-18", Summary: "KYC renewal discussion. Documents submitted but pending review. Also exploring working capital increase.", Outcome: "Expedite KYC review; prepare Murabaha increase", RMID: "rm-002"},

		// Salmiya Retail Holdings
		{ID: "int-029", ClientID: "cli-017", Type: "Meeting", Date: "2026-08-12", Summary: "Mall expansion project — KWD 10M additional Ijara. Also mentioned growing tenant receivables needing acceleration.", Outcome: "Prepare Ijara amendment + invoice discounting proposal", RMID: "rm-003"},

		// Kuwait Agri-Foods
		{ID: "int-030", ClientID: "cli-018", Type: "Call", Date: "2026-08-15", Summary: "Ramadan season approaching. Working capital needs will spike. Current LC facility may be insufficient.", Outcome: "Assess WC limit increase", RMID: "rm-002"},

		// Al Bidaa Contracting
		{ID: "int-031", ClientID: "cli-019", Type: "Meeting", Date: "2026-08-20", Summary: "Cash flow pressures from delayed government payments. KWD 4M receivables outstanding > 90 days.", Outcome: "Propose invoice discounting facility urgently", RMID: "rm-002"},
		{ID: "int-032", ClientID: "cli-019", Type: "Note", Date: "2026-07-28", Summary: "Risk rating under review due to receivables aging. Close monitoring required.", Outcome: "Schedule bi-weekly check-ins", RMID: "rm-002"},

		// Digital Gulf Media
		{ID: "int-033", ClientID: "cli-020", Type: "Meeting", Date: "2026-08-25", Summary: "Onboarding meeting. New client. Growing rapidly. No banking products yet. Interested in corporate account + treasury.", Outcome: "Present full product suite introduction", RMID: "rm-001"},
		{ID: "int-034", ClientID: "cli-020", Type: "Note", Date: "2026-08-10", Summary: "Client referred by Pearl Commercial Investments. Good growth trajectory. Revenue doubled in 2 years.", Outcome: "Prioritize relationship development", RMID: "rm-001"},
	}
	for _, i := range interactions {
		if err := s.InsertInteraction(i); err != nil {
			return err
		}
	}

	return nil
}
