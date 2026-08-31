package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"

	"opportunity-engine/internal/models"
)

// Store wraps SQLite database access for the Opportunity Engine.
type Store struct {
	db *sql.DB
}

// New opens or creates the SQLite database and runs migrations.
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening sqlite db: %w", err)
	}

	// SQLite settings for concurrency
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode & busy timeout: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying *sql.DB.
func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) IsEmpty() (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM products").Scan(&count)
	return count == 0, err
}

// Seed initializes global products if empty.
func (s *Store) Seed() error {
	return s.SeedGlobalProducts()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		google_id TEXT UNIQUE,
		email TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		avatar TEXT DEFAULT '',
		role TEXT DEFAULT 'Senior Relationship Manager',
		created_at TEXT NOT NULL,
		last_login TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sessions (
		token TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		expires_at TEXT NOT NULL
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
		typical_tenure_months INTEGER NOT NULL DEFAULT 12,
		target_industries TEXT DEFAULT '',
		is_active INTEGER NOT NULL DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS clients (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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
		country TEXT NOT NULL DEFAULT 'Kuwait',
		notes TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS client_products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		client_id TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
		product_id TEXT NOT NULL REFERENCES products(id),
		product_name TEXT DEFAULT '',
		start_date TEXT NOT NULL,
		amount_kwd REAL NOT NULL DEFAULT 0,
		status TEXT NOT NULL DEFAULT 'Active'
	);

	CREATE TABLE IF NOT EXISTS interactions (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		client_id TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
		type TEXT NOT NULL,
		date TEXT NOT NULL,
		summary TEXT NOT NULL,
		outcome TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS opportunities (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		client_id TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
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

	CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
	CREATE INDEX IF NOT EXISTS idx_clients_user ON clients(user_id);
	CREATE INDEX IF NOT EXISTS idx_interactions_user ON interactions(user_id);
	CREATE INDEX IF NOT EXISTS idx_interactions_client ON interactions(client_id);
	CREATE INDEX IF NOT EXISTS idx_client_products_client ON client_products(client_id);
	CREATE INDEX IF NOT EXISTS idx_opportunities_user ON opportunities(user_id);
	CREATE INDEX IF NOT EXISTS idx_opportunities_client ON opportunities(client_id);
	CREATE INDEX IF NOT EXISTS idx_opportunities_status ON opportunities(status);
	`
	_, err := s.db.Exec(schema)
	return err
}

// --- Products ---

func (s *Store) InsertProduct(p models.Product) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO products (id, name, name_ar, category, description, shariah_structure,
			min_amount_kwd, max_amount_kwd, typical_tenure_months, target_industries, is_active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.NameAr, p.Category, p.Description, p.ShariahStructure,
		p.MinAmountKWD, p.MaxAmountKWD, p.TypicalTenureMonths, p.TargetIndustries, p.IsActive)
	return err
}

func (s *Store) ListProducts() ([]models.Product, error) {
	rows, err := s.db.Query(`
		SELECT id, name, name_ar, category, description, shariah_structure,
			min_amount_kwd, max_amount_kwd, typical_tenure_months, target_industries, is_active
		FROM products WHERE is_active = 1 ORDER BY category, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prods []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.NameAr, &p.Category, &p.Description,
			&p.ShariahStructure, &p.MinAmountKWD, &p.MaxAmountKWD,
			&p.TypicalTenureMonths, &p.TargetIndustries, &p.IsActive); err != nil {
			return nil, err
		}
		prods = append(prods, p)
	}
	return prods, rows.Err()
}

func (s *Store) GetProduct(id string) (*models.Product, error) {
	var p models.Product
	err := s.db.QueryRow(`
		SELECT id, name, name_ar, category, description, shariah_structure,
			min_amount_kwd, max_amount_kwd, typical_tenure_months, target_industries, is_active
		FROM products WHERE id = ?
	`, id).Scan(&p.ID, &p.Name, &p.NameAr, &p.Category, &p.Description,
		&p.ShariahStructure, &p.MinAmountKWD, &p.MaxAmountKWD,
		&p.TypicalTenureMonths, &p.TargetIndustries, &p.IsActive)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

// --- Clients (User-Scoped) ---

func (s *Store) InsertClient(c models.Client) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO clients (id, user_id, name, name_ar, industry, sub_industry,
			revenue_kwd, employee_count, incorporation_year, risk_rating, kyc_status,
			relationship_start, country, notes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.UserID, c.Name, c.NameAr, c.Industry, c.SubIndustry,
		c.RevenueKWD, c.EmployeeCount, c.IncorporationYear, c.RiskRating,
		c.KYCStatus, c.RelationshipStart, c.Country, c.Notes)
	return err
}

func (s *Store) ListClients(userID string) ([]models.Client, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.user_id, c.name, c.name_ar, c.industry, c.sub_industry,
			c.revenue_kwd, c.employee_count, c.incorporation_year, c.risk_rating,
			c.kyc_status, c.relationship_start, c.country, c.notes,
			u.name as rm_name
		FROM clients c
		JOIN users u ON u.id = c.user_id
		WHERE c.user_id = ?
		ORDER BY c.revenue_kwd DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []models.Client
	for rows.Next() {
		var c models.Client
		if err := rows.Scan(&c.ID, &c.UserID, &c.Name, &c.NameAr, &c.Industry, &c.SubIndustry,
			&c.RevenueKWD, &c.EmployeeCount, &c.IncorporationYear, &c.RiskRating,
			&c.KYCStatus, &c.RelationshipStart, &c.Country, &c.Notes,
			&c.RMName); err != nil {
			return nil, err
		}
		c.RMID = c.UserID
		clients = append(clients, c)
	}
	return clients, rows.Err()
}

func (s *Store) GetClient(userID, clientID string) (*models.Client, error) {
	var c models.Client
	err := s.db.QueryRow(`
		SELECT c.id, c.user_id, c.name, c.name_ar, c.industry, c.sub_industry,
			c.revenue_kwd, c.employee_count, c.incorporation_year, c.risk_rating,
			c.kyc_status, c.relationship_start, c.country, c.notes,
			u.name as rm_name
		FROM clients c
		JOIN users u ON u.id = c.user_id
		WHERE c.user_id = ? AND c.id = ?
	`, userID, clientID).Scan(&c.ID, &c.UserID, &c.Name, &c.NameAr, &c.Industry, &c.SubIndustry,
		&c.RevenueKWD, &c.EmployeeCount, &c.IncorporationYear, &c.RiskRating,
		&c.KYCStatus, &c.RelationshipStart, &c.Country, &c.Notes,
		&c.RMName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.RMID = c.UserID

	// Current products
	pRows, err := s.db.Query(`
		SELECT cp.client_id, cp.product_id, COALESCE(p.name, cp.product_name), cp.start_date, cp.amount_kwd, cp.status
		FROM client_products cp
		LEFT JOIN products p ON p.id = cp.product_id
		WHERE cp.client_id = ?
	`, clientID)
	if err == nil {
		for pRows.Next() {
			var cp models.ClientProduct
			if err := pRows.Scan(&cp.ClientID, &cp.ProductID, &cp.ProductName, &cp.StartDate, &cp.AmountKWD, &cp.Status); err == nil {
				c.CurrentProducts = append(c.CurrentProducts, cp)
			}
		}
		pRows.Close()
	}

	// Interactions
	iRows, err := s.db.Query(`
		SELECT id, client_id, user_id, type, date, summary, outcome
		FROM interactions WHERE client_id = ? AND user_id = ? ORDER BY date DESC
	`, clientID, userID)
	if err == nil {
		for iRows.Next() {
			var ix models.Interaction
			if err := iRows.Scan(&ix.ID, &ix.ClientID, &ix.UserID, &ix.Type, &ix.Date, &ix.Summary, &ix.Outcome); err == nil {
				ix.RMID = ix.UserID
				c.Interactions = append(c.Interactions, ix)
			}
		}
		iRows.Close()
	}

	return &c, nil
}

// --- Client Products & Interactions ---

func (s *Store) InsertClientProduct(cp models.ClientProduct) error {
	_, err := s.db.Exec(`
		INSERT INTO client_products (client_id, product_id, product_name, start_date, amount_kwd, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`, cp.ClientID, cp.ProductID, cp.ProductName, cp.StartDate, cp.AmountKWD, cp.Status)
	return err
}

func (s *Store) InsertInteraction(i models.Interaction) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO interactions (id, user_id, client_id, type, date, summary, outcome)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, i.ID, i.UserID, i.ClientID, i.Type, i.Date, i.Summary, i.Outcome)
	return err
}

// --- Opportunities (User-Scoped) ---

func (s *Store) InsertOpportunity(o models.Opportunity) error {
	if o.ID == "" {
		o.ID = "opp-" + uuid.New().String()[:8]
	}
	now := time.Now().Format(time.RFC3339)
	if o.CreatedAt.IsZero() {
		o.CreatedAt = time.Now()
	}
	if o.Status == "" {
		o.Status = models.OpportunityNew
	}

	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO opportunities (id, user_id, client_id, product_id, confidence,
			urgency, reasoning, next_action, shariah_notes, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, o.ID, o.UserID, o.ClientID, o.ProductID, o.Confidence,
		o.Urgency, o.Reasoning, o.NextAction, o.ShariahNotes,
		o.Status, o.CreatedAt.Format(time.RFC3339), now)
	return err
}

func (s *Store) ListOpportunities(userID, status, urgency, clientID string) ([]models.Opportunity, error) {
	query := `
		SELECT o.id, o.user_id, o.client_id, c.name, o.product_id, p.name,
			o.confidence, o.urgency, o.reasoning, o.next_action, o.shariah_notes,
			o.status, o.created_at, o.updated_at
		FROM opportunities o
		JOIN clients c ON c.id = o.client_id
		JOIN products p ON p.id = o.product_id
		WHERE o.user_id = ?
	`
	var args []interface{}
	args = append(args, userID)

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
		if err := rows.Scan(&o.ID, &o.UserID, &o.ClientID, &o.ClientName, &o.ProductID, &o.ProductName,
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

func (s *Store) UpdateOpportunityStatus(userID, id string, status models.OpportunityStatus) error {
	now := time.Now().Format(time.RFC3339)
	result, err := s.db.Exec(
		"UPDATE opportunities SET status = ?, updated_at = ? WHERE user_id = ? AND id = ?",
		status, now, userID, id,
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

// --- Portfolio Summary (User-Scoped) ---

func (s *Store) GetPortfolioSummary(userID string) (*models.PortfolioSummary, error) {
	summary := &models.PortfolioSummary{
		UrgencyBreakdown: make(map[string]int),
		TopIndustries:    []models.IndustryStat{},
		ProductBreakdown: []models.ProductStat{},
	}

	// Total clients
	if err := s.db.QueryRow("SELECT COUNT(*) FROM clients WHERE user_id = ?", userID).Scan(&summary.TotalClients); err != nil {
		return nil, fmt.Errorf("counting clients: %w", err)
	}

	// Opportunity counts
	if err := s.db.QueryRow("SELECT COUNT(*) FROM opportunities WHERE user_id = ?", userID).Scan(&summary.TotalOpportunities); err != nil {
		return nil, fmt.Errorf("counting opportunities: %w", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM opportunities WHERE user_id = ? AND status = 'New'", userID).Scan(&summary.NewOpportunities); err != nil {
		return nil, fmt.Errorf("counting new opportunities: %w", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM opportunities WHERE user_id = ? AND status = 'Accepted'", userID).Scan(&summary.AcceptedOpps); err != nil {
		return nil, fmt.Errorf("counting accepted opportunities: %w", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM opportunities WHERE user_id = ? AND status = 'Converted'", userID).Scan(&summary.ConvertedOpps); err != nil {
		return nil, fmt.Errorf("counting converted opportunities: %w", err)
	}
	if err := s.db.QueryRow("SELECT COUNT(*) FROM opportunities WHERE user_id = ? AND status = 'Dismissed'", userID).Scan(&summary.DismissedOpps); err != nil {
		return nil, fmt.Errorf("counting dismissed opportunities: %w", err)
	}

	// Average confidence
	if err := s.db.QueryRow("SELECT COALESCE(AVG(confidence), 0) FROM opportunities WHERE user_id = ? AND status != 'Dismissed'", userID).Scan(&summary.AvgConfidence); err != nil {
		return nil, fmt.Errorf("getting avg confidence: %w", err)
	}

	// Pipeline value
	if err := s.db.QueryRow(`
		SELECT COALESCE(SUM(p.max_amount_kwd * o.confidence), 0)
		FROM opportunities o
		JOIN products p ON p.id = o.product_id
		WHERE o.user_id = ? AND o.status IN ('New', 'Accepted', 'Reviewed')
	`, userID).Scan(&summary.PipelineValueKWD); err != nil {
		return nil, fmt.Errorf("getting pipeline value: %w", err)
	}

	// Top industries
	rows, err := s.db.Query(`
		SELECT c.industry, COUNT(*) as cnt, COALESCE(SUM(c.revenue_kwd), 0) as rev
		FROM clients c
		WHERE c.user_id = ?
		GROUP BY c.industry
		ORDER BY rev DESC
		LIMIT 5
	`, userID)
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
		WHERE user_id = ? AND status NOT IN ('Dismissed', 'Converted')
		GROUP BY urgency
	`, userID)
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
		WHERE o.user_id = ? AND o.status NOT IN ('Dismissed')
		GROUP BY o.product_id
		ORDER BY cnt DESC
	`, userID)
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

// --- Users & Sessions ---

func (s *Store) UpsertGoogleUser(u *models.User) (*models.User, error) {
	now := time.Now().Format(time.RFC3339)
	if u.Role == "" {
		u.Role = "Senior Relationship Manager"
	}

	var existing models.User
	var createdAtStr, lastLoginStr string
	err := s.db.QueryRow(`
		SELECT id, google_id, email, name, avatar, role, created_at, last_login
		FROM users
		WHERE google_id = ? OR email = ?
	`, u.GoogleID, u.Email).Scan(
		&existing.ID, &existing.GoogleID, &existing.Email, &existing.Name,
		&existing.Avatar, &existing.Role, &createdAtStr, &lastLoginStr,
	)

	if err == sql.ErrNoRows {
		if u.ID == "" {
			u.ID = "usr-" + uuid.New().String()[:8]
		}
		u.CreatedAt = time.Now()
		u.LastLogin = time.Now()
		_, err := s.db.Exec(`
			INSERT INTO users (id, google_id, email, name, avatar, role, created_at, last_login)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, u.ID, u.GoogleID, u.Email, u.Name, u.Avatar, u.Role, now, now)
		if err != nil {
			return nil, fmt.Errorf("inserting user: %w", err)
		}
		return u, nil
	} else if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}

	existing.GoogleID = u.GoogleID
	existing.Name = u.Name
	if u.Avatar != "" {
		existing.Avatar = u.Avatar
	}
	existing.LastLogin = time.Now()
	existing.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

	_, err = s.db.Exec(`
		UPDATE users
		SET google_id = ?, name = ?, avatar = ?, last_login = ?
		WHERE id = ?
	`, existing.GoogleID, existing.Name, existing.Avatar, now, existing.ID)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	return &existing, nil
}

func (s *Store) CreateSession(token, userID string, expiresAt time.Time) error {
	_, err := s.db.Exec(`
		INSERT INTO sessions (token, user_id, expires_at)
		VALUES (?, ?, ?)
		ON CONFLICT(token) DO UPDATE SET user_id = excluded.user_id, expires_at = excluded.expires_at
	`, token, userID, expiresAt.Format(time.RFC3339))
	return err
}

func (s *Store) GetUserBySession(token string) (*models.User, error) {
	var u models.User
	var createdAtStr, lastLoginStr, expiresAtStr string
	err := s.db.QueryRow(`
		SELECT u.id, u.google_id, u.email, u.name, u.avatar, u.role, u.created_at, u.last_login, s.expires_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token = ?
	`, token).Scan(
		&u.ID, &u.GoogleID, &u.Email, &u.Name, &u.Avatar, &u.Role,
		&createdAtStr, &lastLoginStr, &expiresAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresAtStr)
	if err == nil && time.Now().After(expiresAt) {
		_ = s.DeleteSession(token)
		return nil, nil
	}

	u.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	u.LastLogin, _ = time.Parse(time.RFC3339, lastLoginStr)
	return &u, nil
}

func (s *Store) DeleteSession(token string) error {
	_, err := s.db.Exec("DELETE FROM sessions WHERE token = ?", token)
	return err
}
