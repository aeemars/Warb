package store

import (
	"database/sql"
	"fmt"
	"time"

	"opportunity-engine/internal/models"

	_ "modernc.org/sqlite"
)

// Store provides data access to the SQLite database.
type Store struct {
	db *sql.DB
}

// New creates a new Store and initializes the schema.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrency and busy timeout
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("setting WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		return nil, fmt.Errorf("setting busy timeout: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return nil, fmt.Errorf("enabling foreign keys: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}
	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS relationship_managers (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		title TEXT NOT NULL,
		email TEXT NOT NULL,
		department TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS clients (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		name_ar TEXT DEFAULT '',
		industry TEXT NOT NULL,
		sub_industry TEXT DEFAULT '',
		revenue_kwd REAL NOT NULL DEFAULT 0,
		employee_count INTEGER NOT NULL DEFAULT 0,
		incorporation_year INTEGER NOT NULL DEFAULT 2000,
		risk_rating TEXT NOT NULL DEFAULT 'Medium',
		kyc_status TEXT NOT NULL DEFAULT 'Active',
		relationship_start TEXT NOT NULL,
		rm_id TEXT NOT NULL REFERENCES relationship_managers(id),
		country TEXT NOT NULL DEFAULT 'Kuwait',
		notes TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS products (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		name_ar TEXT DEFAULT '',
		category TEXT NOT NULL,
		description TEXT NOT NULL,
		shariah_structure TEXT NOT NULL,
		min_amount_kwd REAL NOT NULL DEFAULT 0,
		max_amount_kwd REAL NOT NULL DEFAULT 0,
		typical_tenure_months INTEGER NOT NULL DEFAULT 0,
		target_industries TEXT DEFAULT '',
		is_active BOOLEAN NOT NULL DEFAULT TRUE
	);

	CREATE TABLE IF NOT EXISTS client_products (
		client_id TEXT NOT NULL REFERENCES clients(id),
		product_id TEXT NOT NULL REFERENCES products(id),
		product_name TEXT DEFAULT '',
		start_date TEXT NOT NULL,
		amount_kwd REAL NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'Active',
		PRIMARY KEY (client_id, product_id)
	);

	CREATE TABLE IF NOT EXISTS interactions (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL REFERENCES clients(id),
		type TEXT NOT NULL,
		date TEXT NOT NULL,
		summary TEXT NOT NULL,
		outcome TEXT DEFAULT '',
		rm_id TEXT DEFAULT '' REFERENCES relationship_managers(id)
	);

	CREATE TABLE IF NOT EXISTS opportunities (
		id TEXT PRIMARY KEY,
		client_id TEXT NOT NULL REFERENCES clients(id),
		product_id TEXT NOT NULL REFERENCES products(id),
		confidence REAL NOT NULL DEFAULT 0,
		urgency TEXT NOT NULL DEFAULT 'Medium',
		reasoning TEXT NOT NULL,
		next_action TEXT DEFAULT '',
		shariah_notes TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT 'New',
		created_at TEXT NOT NULL,
		updated_at TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_opportunities_client ON opportunities(client_id);
	CREATE INDEX IF NOT EXISTS idx_opportunities_status ON opportunities(status);
	CREATE INDEX IF NOT EXISTS idx_interactions_client ON interactions(client_id);
	CREATE INDEX IF NOT EXISTS idx_clients_rm ON clients(rm_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

// IsEmpty checks if the database has been seeded.
func (s *Store) IsEmpty() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&count)
	if err != nil {
		return true, err
	}
	return count == 0, nil
}

// --- Relationship Managers ---

func (s *Store) InsertRM(rm models.RelationshipManager) error {
	_, err := s.db.Exec(
		"INSERT OR IGNORE INTO relationship_managers (id, name, title, email, department) VALUES (?, ?, ?, ?, ?)",
		rm.ID, rm.Name, rm.Title, rm.Email, rm.Department,
	)
	return err
}

func (s *Store) ListRMs() ([]models.RelationshipManager, error) {
	rows, err := s.db.Query(`
		SELECT rm.id, rm.name, rm.title, rm.email, rm.department, COUNT(c.id) as client_count
		FROM relationship_managers rm
		LEFT JOIN clients c ON c.rm_id = rm.id
		GROUP BY rm.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rms []models.RelationshipManager
	for rows.Next() {
		var rm models.RelationshipManager
		if err := rows.Scan(&rm.ID, &rm.Name, &rm.Title, &rm.Email, &rm.Department, &rm.ClientCount); err != nil {
			return nil, err
		}
		rms = append(rms, rm)
	}
	return rms, rows.Err()
}

// --- Clients ---

func (s *Store) InsertClient(c models.Client) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO clients (id, name, name_ar, industry, sub_industry, revenue_kwd, employee_count,
			incorporation_year, risk_rating, kyc_status, relationship_start, rm_id, country, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.NameAr, c.Industry, c.SubIndustry, c.RevenueKWD, c.EmployeeCount,
		c.IncorporationYear, c.RiskRating, c.KYCStatus, c.RelationshipStart, c.RMID, c.Country, c.Notes,
	)
	return err
}

func (s *Store) ListClients() ([]models.Client, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.name_ar, c.industry, c.sub_industry, c.revenue_kwd, c.employee_count,
			c.incorporation_year, c.risk_rating, c.kyc_status, c.relationship_start, c.rm_id, c.country, c.notes,
			COALESCE(rm.name, '') as rm_name
		FROM clients c
		LEFT JOIN relationship_managers rm ON rm.id = c.rm_id
		ORDER BY c.revenue_kwd DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []models.Client
	for rows.Next() {
		var c models.Client
		if err := rows.Scan(&c.ID, &c.Name, &c.NameAr, &c.Industry, &c.SubIndustry, &c.RevenueKWD,
			&c.EmployeeCount, &c.IncorporationYear, &c.RiskRating, &c.KYCStatus,
			&c.RelationshipStart, &c.RMID, &c.Country, &c.Notes, &c.RMName); err != nil {
			return nil, err
		}
		clients = append(clients, c)
	}
	return clients, rows.Err()
}

func (s *Store) GetClient(id string) (*models.Client, error) {
	var c models.Client
	err := s.db.QueryRow(`
		SELECT c.id, c.name, c.name_ar, c.industry, c.sub_industry, c.revenue_kwd, c.employee_count,
			c.incorporation_year, c.risk_rating, c.kyc_status, c.relationship_start, c.rm_id, c.country, c.notes,
			COALESCE(rm.name, '') as rm_name
		FROM clients c
		LEFT JOIN relationship_managers rm ON rm.id = c.rm_id
		WHERE c.id = ?`, id).Scan(
		&c.ID, &c.Name, &c.NameAr, &c.Industry, &c.SubIndustry, &c.RevenueKWD,
		&c.EmployeeCount, &c.IncorporationYear, &c.RiskRating, &c.KYCStatus,
		&c.RelationshipStart, &c.RMID, &c.Country, &c.Notes, &c.RMName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Fetch current products
	products, err := s.GetClientProducts(id)
	if err != nil {
		return nil, err
	}
	c.CurrentProducts = products

	// Fetch interactions
	interactions, err := s.GetClientInteractions(id)
	if err != nil {
		return nil, err
	}
	c.Interactions = interactions

	return &c, nil
}

// --- Client Products ---

func (s *Store) InsertClientProduct(cp models.ClientProduct) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO client_products (client_id, product_id, product_name, start_date, amount_kwd, status)
		VALUES (?, ?, ?, ?, ?, ?)`,
		cp.ClientID, cp.ProductID, cp.ProductName, cp.StartDate, cp.AmountKWD, cp.Status,
	)
	return err
}

func (s *Store) GetClientProducts(clientID string) ([]models.ClientProduct, error) {
	rows, err := s.db.Query(`
		SELECT cp.client_id, cp.product_id, COALESCE(p.name, cp.product_name) as product_name,
			cp.start_date, cp.amount_kwd, cp.status
		FROM client_products cp
		LEFT JOIN products p ON p.id = cp.product_id
		WHERE cp.client_id = ?
		ORDER BY cp.start_date DESC`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.ClientProduct
	for rows.Next() {
		var cp models.ClientProduct
		if err := rows.Scan(&cp.ClientID, &cp.ProductID, &cp.ProductName, &cp.StartDate, &cp.AmountKWD, &cp.Status); err != nil {
			return nil, err
		}
		products = append(products, cp)
	}
	return products, rows.Err()
}

// --- Interactions ---

func (s *Store) InsertInteraction(i models.Interaction) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO interactions (id, client_id, type, date, summary, outcome, rm_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		i.ID, i.ClientID, i.Type, i.Date, i.Summary, i.Outcome, i.RMID,
	)
	return err
}

func (s *Store) GetClientInteractions(clientID string) ([]models.Interaction, error) {
	rows, err := s.db.Query(`
		SELECT id, client_id, type, date, summary, outcome, rm_id
		FROM interactions
		WHERE client_id = ?
		ORDER BY date DESC
		LIMIT 20`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []models.Interaction
	for rows.Next() {
		var i models.Interaction
		if err := rows.Scan(&i.ID, &i.ClientID, &i.Type, &i.Date, &i.Summary, &i.Outcome, &i.RMID); err != nil {
			return nil, err
		}
		interactions = append(interactions, i)
	}
	return interactions, rows.Err()
}

// --- Products ---

func (s *Store) InsertProduct(p models.Product) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO products (id, name, name_ar, category, description, shariah_structure,
			min_amount_kwd, max_amount_kwd, typical_tenure_months, target_industries, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.NameAr, p.Category, p.Description, p.ShariahStructure,
		p.MinAmountKWD, p.MaxAmountKWD, p.TypicalTenureMonths, p.TargetIndustries, p.IsActive,
	)
	return err
}

func (s *Store) ListProducts() ([]models.Product, error) {
	rows, err := s.db.Query(`
		SELECT id, name, name_ar, category, description, shariah_structure,
			min_amount_kwd, max_amount_kwd, typical_tenure_months, target_industries, is_active
		FROM products
		WHERE is_active = TRUE
		ORDER BY category, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.NameAr, &p.Category, &p.Description, &p.ShariahStructure,
			&p.MinAmountKWD, &p.MaxAmountKWD, &p.TypicalTenureMonths, &p.TargetIndustries, &p.IsActive); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (s *Store) GetProduct(id string) (*models.Product, error) {
	var p models.Product
	err := s.db.QueryRow(`
		SELECT id, name, name_ar, category, description, shariah_structure,
			min_amount_kwd, max_amount_kwd, typical_tenure_months, target_industries, is_active
		FROM products WHERE id = ?`, id).Scan(
		&p.ID, &p.Name, &p.NameAr, &p.Category, &p.Description, &p.ShariahStructure,
		&p.MinAmountKWD, &p.MaxAmountKWD, &p.TypicalTenureMonths, &p.TargetIndustries, &p.IsActive,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// --- Opportunities ---

func (s *Store) InsertOpportunity(o models.Opportunity) error {
	_, err := s.db.Exec(`
		INSERT INTO opportunities (id, client_id, product_id, confidence, urgency, reasoning,
			next_action, shariah_notes, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.ID, o.ClientID, o.ProductID, o.Confidence, o.Urgency, o.Reasoning,
		o.NextAction, o.ShariahNotes, o.Status, o.CreatedAt.Format(time.RFC3339),
	)
	return err
}

func (s *Store) ListOpportunities(status, urgency, clientID string) ([]models.Opportunity, error) {
	query := `
		SELECT o.id, o.client_id, COALESCE(c.name, '') as client_name,
			o.product_id, COALESCE(p.name, '') as product_name,
			o.confidence, o.urgency, o.reasoning, o.next_action, o.shariah_notes,
			o.status, o.created_at, o.updated_at
		FROM opportunities o
		LEFT JOIN clients c ON c.id = o.client_id
		LEFT JOIN products p ON p.id = o.product_id
		WHERE 1=1`

	var args []interface{}
	if status != "" {
		query += " AND o.status = ?"
		args = append(args, status)
	}
	if urgency != "" {
		query += " AND o.urgency = ?"
		args = append(args, urgency)
	}
	if clientID != "" {
		query += " AND o.client_id = ?"
		args = append(args, clientID)
	}
	query += " ORDER BY o.created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var opps []models.Opportunity
	for rows.Next() {
		var o models.Opportunity
		var createdAt string
		var updatedAt sql.NullString
		if err := rows.Scan(&o.ID, &o.ClientID, &o.ClientName, &o.ProductID, &o.ProductName,
			&o.Confidence, &o.Urgency, &o.Reasoning, &o.NextAction, &o.ShariahNotes,
			&o.Status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		o.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		if updatedAt.Valid {
			t, _ := time.Parse(time.RFC3339, updatedAt.String)
			o.UpdatedAt = &t
		}
		opps = append(opps, o)
	}
	return opps, rows.Err()
}

func (s *Store) UpdateOpportunityStatus(id string, status models.OpportunityStatus) error {
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.Exec(
		"UPDATE opportunities SET status = ?, updated_at = ? WHERE id = ?",
		status, now, id,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("opportunity not found: %s", id)
	}
	return nil
}

// --- Portfolio Summary ---

func (s *Store) GetPortfolioSummary() (*models.PortfolioSummary, error) {
	summary := &models.PortfolioSummary{
		UrgencyBreakdown: make(map[string]int),
		TopIndustries:    []models.IndustryStat{},
		ProductBreakdown: []models.ProductStat{},
	}

	// Total clients
	if err := s.db.QueryRow("SELECT COUNT(*) FROM clients").Scan(&summary.TotalClients); err != nil {
		return nil, fmt.Errorf("counting clients: %w", err)
	}

	// Opportunity counts
	if err := s.db.QueryRow("SELECT COUNT(*) FROM opportunities").Scan(&summary.TotalOpportunities); err != nil {
		return nil, fmt.Errorf("counting opportunities: %w", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM opportunities WHERE status = 'New'").Scan(&summary.NewOpportunities); err != nil {
		return nil, fmt.Errorf("counting new opportunities: %w", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM opportunities WHERE status = 'Accepted'").Scan(&summary.AcceptedOpps); err != nil {
		return nil, fmt.Errorf("counting accepted opportunities: %w", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM opportunities WHERE status = 'Converted'").Scan(&summary.ConvertedOpps); err != nil {
		return nil, fmt.Errorf("counting converted opportunities: %w", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM opportunities WHERE status = 'Dismissed'").Scan(&summary.DismissedOpps); err != nil {
		return nil, fmt.Errorf("counting dismissed opportunities: %w", err)
	}

	// Average confidence
	if err := s.db.QueryRow("SELECT COALESCE(AVG(confidence), 0) FROM opportunities WHERE status != 'Dismissed'").Scan(&summary.AvgConfidence); err != nil {
		return nil, fmt.Errorf("getting avg confidence: %w", err)
	}

	// Pipeline value (sum of max product amounts for active opportunities)
	if err := s.db.QueryRow(`
		SELECT COALESCE(SUM(p.max_amount_kwd * o.confidence), 0)
		FROM opportunities o
		JOIN products p ON p.id = o.product_id
		WHERE o.status IN ('New', 'Accepted', 'Reviewed')
	`).Scan(&summary.PipelineValueKWD); err != nil {
		return nil, fmt.Errorf("getting pipeline value: %w", err)
	}

	// Top industries
	rows, err := s.db.Query(`
		SELECT c.industry, COUNT(*) as cnt, COALESCE(SUM(c.revenue_kwd), 0) as rev
		FROM clients c
		GROUP BY c.industry
		ORDER BY rev DESC
		LIMIT 5
	`)
	if err != nil {
		return nil, fmt.Errorf("querying top industries: %w", err)
	}
	for rows.Next() {
		var is models.IndustryStat
		if err := rows.Scan(&is.Industry, &is.Count, &is.Revenue); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scanning industry stat: %w", err)
		}
		summary.TopIndustries = append(summary.TopIndustries, is)
	}
	rows.Close()

	// Urgency breakdown
	rows2, err := s.db.Query(`
		SELECT urgency, COUNT(*) FROM opportunities
		WHERE status NOT IN ('Dismissed', 'Converted')
		GROUP BY urgency
	`)
	if err != nil {
		return nil, fmt.Errorf("querying urgency breakdown: %w", err)
	}
	for rows2.Next() {
		var urg string
		var cnt int
		if err := rows2.Scan(&urg, &cnt); err != nil {
			rows2.Close()
			return nil, fmt.Errorf("scanning urgency stat: %w", err)
		}
		summary.UrgencyBreakdown[urg] = cnt
	}
	rows2.Close()

	// Product breakdown
	rows3, err := s.db.Query(`
		SELECT o.product_id, COALESCE(p.name, ''), COUNT(*) as cnt
		FROM opportunities o
		LEFT JOIN products p ON p.id = o.product_id
		WHERE o.status NOT IN ('Dismissed')
		GROUP BY o.product_id
		ORDER BY cnt DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("querying product breakdown: %w", err)
	}
	for rows3.Next() {
		var ps models.ProductStat
		if err := rows3.Scan(&ps.ProductID, &ps.ProductName, &ps.Count); err != nil {
			rows3.Close()
			return nil, fmt.Errorf("scanning product stat: %w", err)
		}
		summary.ProductBreakdown = append(summary.ProductBreakdown, ps)
	}
	rows3.Close()

	return summary, nil
}
