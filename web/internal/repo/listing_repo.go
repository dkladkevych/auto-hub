package repo

import (
	"database/sql"
	"fmt"
	"strings"

	"auto-hub/web/internal/models"
)

// ListingRepo provides data access for inventory listings.
type ListingRepo struct {
	db *sql.DB
}

// NewListingRepo creates a new ListingRepo.
func NewListingRepo(db *sql.DB) *ListingRepo {
	return &ListingRepo{db: db}
}

func scanListing(row *sql.Row) (*models.Listing, error) {
	var l models.Listing
	var year, price, mileageKm, accountID sql.NullInt64
	var make, model, location, condition, notes, transmission, drivetrain, sourceURL sql.NullString
	var publishedAt sql.NullTime

	err := row.Scan(
		&l.ID, &accountID, &l.Title, &price, &l.Description, &sourceURL,
		&l.Status, &year, &make, &model, &mileageKm, &location,
		&condition, &notes, &transmission, &drivetrain, &publishedAt,
	)
	if err != nil {
		return nil, err
	}
	l.AccountID = int(accountID.Int64)
	l.Price = int(price.Int64)
	l.Year = nullIntPtr(year)
	l.Make = nullStringPtr(make)
	l.Model = nullStringPtr(model)
	l.MileageKm = nullIntPtr(mileageKm)
	l.Location = nullStringPtr(location)
	l.Condition = nullStringPtr(condition)
	l.Notes = nullStringPtr(notes)
	l.Transmission = nullStringPtr(transmission)
	l.Drivetrain = nullStringPtr(drivetrain)
	l.SourceURL = nullStringPtr(sourceURL)
	l.PublishedAt = nullTimePtr(publishedAt)
	return &l, nil
}

func scanListings(rows *sql.Rows) ([]*models.Listing, error) {
	var listings []*models.Listing
	for rows.Next() {
		var l models.Listing
		var year, price, mileageKm, accountID sql.NullInt64
		var make, model, location, condition, notes, transmission, drivetrain, sourceURL sql.NullString
		var publishedAt sql.NullTime

		err := rows.Scan(
			&l.ID, &accountID, &l.Title, &price, &l.Description, &sourceURL,
			&l.Status, &year, &make, &model, &mileageKm, &location,
			&condition, &notes, &transmission, &drivetrain, &publishedAt,
		)
		if err != nil {
			return nil, err
		}
		l.AccountID = int(accountID.Int64)
		l.Price = int(price.Int64)
		l.Year = nullIntPtr(year)
		l.Make = nullStringPtr(make)
		l.Model = nullStringPtr(model)
		l.MileageKm = nullIntPtr(mileageKm)
		l.Location = nullStringPtr(location)
		l.Condition = nullStringPtr(condition)
		l.Notes = nullStringPtr(notes)
		l.Transmission = nullStringPtr(transmission)
		l.Drivetrain = nullStringPtr(drivetrain)
		l.SourceURL = nullStringPtr(sourceURL)
		l.PublishedAt = nullTimePtr(publishedAt)
		listings = append(listings, &l)
	}
	return listings, rows.Err()
}

// GetByID fetches a listing by ID.
func (r *ListingRepo) GetByID(id int) (*models.Listing, error) {
	row := r.db.QueryRow(`
		SELECT id, account_id, title, price, description, source_url, status,
			year, make, model, mileage_km, location, condition, notes,
			transmission, drivetrain, published_at
		FROM inventory WHERE id = ?`, id)
	return scanListing(row)
}

// FilterParams holds filter criteria for listings.
type FilterParams struct {
	Q            string
	PriceMin     *int
	PriceMax     *int
	YearMin      *int
	YearMax      *int
	MileageMin   *int
	MileageMax   *int
	Transmission string
	Drivetrain   string
	Location     string
}

// buildFilterQuery constructs WHERE clauses and parameters.
func buildFilterQuery(f FilterParams) (string, []interface{}) {
	where := []string{"status IN ('active', 'demo')"}
	var params []interface{}

	if f.Q != "" {
		like := "%" + f.Q + "%"
		where = append(where, `(
			title LIKE ? OR make LIKE ? OR model LIKE ?
			OR description LIKE ? OR location LIKE ?
		)`)
		params = append(params, like, like, like, like, like)
	}
	if f.PriceMin != nil {
		where = append(where, "price >= ?")
		params = append(params, *f.PriceMin)
	}
	if f.PriceMax != nil {
		where = append(where, "price <= ?")
		params = append(params, *f.PriceMax)
	}
	if f.YearMin != nil {
		where = append(where, "year >= ?")
		params = append(params, *f.YearMin)
	}
	if f.YearMax != nil {
		where = append(where, "year <= ?")
		params = append(params, *f.YearMax)
	}
	if f.MileageMin != nil {
		where = append(where, "mileage_km >= ?")
		params = append(params, *f.MileageMin)
	}
	if f.MileageMax != nil {
		where = append(where, "mileage_km <= ?")
		params = append(params, *f.MileageMax)
	}
	if f.Transmission != "" {
		where = append(where, "transmission = ?")
		params = append(params, f.Transmission)
	}
	if f.Drivetrain != "" {
		where = append(where, "drivetrain = ?")
		params = append(params, f.Drivetrain)
	}
	if f.Location != "" {
		tokens := strings.Fields(strings.ToLower(f.Location))
		if len(tokens) > 0 {
			var locClauses []string
			for _, t := range tokens {
				locClauses = append(locClauses, "LOWER(location) LIKE ?")
				params = append(params, "%"+t+"%")
			}
			where = append(where, "("+strings.Join(locClauses, " OR ")+")")
		}
	}
	return strings.Join(where, " AND "), params
}

// GetFiltered returns paginated listings matching filters.
func (r *ListingRepo) GetFiltered(f FilterParams, page, perPage int) ([]*models.Listing, error) {
	where, params := buildFilterQuery(f)
	offset := (page - 1) * perPage
	params = append(params, perPage, offset)

	rows, err := r.db.Query(`
		SELECT id, account_id, title, price, description, source_url, status,
			year, make, model, mileage_km, location, condition, notes,
			transmission, drivetrain, published_at
		FROM inventory
		WHERE `+where+`
		ORDER BY published_at DESC
		LIMIT ? OFFSET ?`, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanListings(rows)
}

// CountFiltered returns total count matching filters.
func (r *ListingRepo) CountFiltered(f FilterParams) (int, error) {
	where, params := buildFilterQuery(f)
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM inventory WHERE `+where, params...).Scan(&count)
	return count, err
}

// GetByIDs returns listings for a list of IDs.
func (r *ListingRepo) GetByIDs(ids []int) ([]*models.Listing, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	params := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		params[i] = id
	}
	query := fmt.Sprintf(`
		SELECT id, account_id, title, price, description, source_url, status,
			year, make, model, mileage_km, location, condition, notes,
			transmission, drivetrain, published_at
		FROM inventory
		WHERE id IN (%s) AND status IN ('active', 'demo')
		ORDER BY published_at DESC`, strings.Join(placeholders, ","))
	rows, err := r.db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanListings(rows)
}

// Create inserts a new listing and returns its ID.
func (r *ListingRepo) Create(l *models.Listing) (int, error) {
	res, err := r.db.Exec(`
		INSERT INTO inventory (account_id, title, price, description, source_url, status,
			year, make, model, mileage_km, location, condition, notes,
			transmission, drivetrain, published_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.AccountID, l.Title, l.Price, l.Description, ptrString(l.SourceURL), l.Status,
		ptrInt(l.Year), ptrString(l.Make), ptrString(l.Model), ptrInt(l.MileageKm),
		ptrString(l.Location), ptrString(l.Condition), ptrString(l.Notes),
		ptrString(l.Transmission), ptrString(l.Drivetrain), ptrTime(l.PublishedAt),
	)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// Update updates an existing listing.
func (r *ListingRepo) Update(l *models.Listing) error {
	_, err := r.db.Exec(`
		UPDATE inventory SET
			title = ?, price = ?, description = ?, source_url = ?, status = ?,
			year = ?, make = ?, model = ?, mileage_km = ?, location = ?,
			condition = ?, notes = ?, transmission = ?, drivetrain = ?,
			published_at = COALESCE(published_at, ?)
		WHERE id = ?`,
		l.Title, l.Price, l.Description, ptrString(l.SourceURL), l.Status,
		ptrInt(l.Year), ptrString(l.Make), ptrString(l.Model), ptrInt(l.MileageKm),
		ptrString(l.Location), ptrString(l.Condition), ptrString(l.Notes),
		ptrString(l.Transmission), ptrString(l.Drivetrain), ptrTime(l.PublishedAt),
		l.ID,
	)
	return err
}

// Delete removes a listing and its stats/view logs.
func (r *ListingRepo) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM view_log WHERE target_type = 'listing' AND target_id = ?", id)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("DELETE FROM stats WHERE target_type = 'listing' AND target_id = ?", id)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("DELETE FROM inventory WHERE id = ?", id)
	return err
}

// SetStatus updates a listing's status and sets published_at on first active/demo.
func (r *ListingRepo) SetStatus(id int, status string) error {
	_, err := r.db.Exec(`
		UPDATE inventory SET
			status = ?,
			published_at = COALESCE(published_at, datetime('now'))
		WHERE id = ? AND status != ?`, status, id, status)
	return err
}

// CountByStatus returns counts per status.
func (r *ListingRepo) CountByStatus() (total, draft, active, archived, demo int, err error) {
	rows, err := r.db.Query(`
		SELECT status, COUNT(*) FROM inventory GROUP BY status`)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var st string
		var c int
		if err := rows.Scan(&st, &c); err != nil {
			return 0, 0, 0, 0, 0, err
		}
		total += c
		switch st {
		case "draft":
			draft = c
		case "active":
			active = c
		case "archived":
			archived = c
		case "demo":
			demo = c
		}
	}
	return total, draft, active, archived, demo, rows.Err()
}

// GetDashboardListings returns paginated listings with view counts for admin dashboard.
func (r *ListingRepo) GetDashboardListings(page, perPage int) ([]*models.Listing, map[int]int, error) {
	offset := (page - 1) * perPage
	rows, err := r.db.Query(`
		SELECT i.id, i.account_id, i.title, i.price, i.description, i.source_url, i.status,
			i.year, i.make, i.model, i.mileage_km, i.location, i.condition, i.notes,
			i.transmission, i.drivetrain, i.published_at,
			COALESCE(s.view_count, 0) as view_count
		FROM inventory i
		LEFT JOIN stats s ON s.target_type = 'listing' AND s.target_id = i.id
		ORDER BY i.id DESC
		LIMIT ? OFFSET ?`, perPage, offset)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var listings []*models.Listing
	views := make(map[int]int)
	for rows.Next() {
		var l models.Listing
		var year, price, mileageKm, accountID sql.NullInt64
		var make, model, location, condition, notes, transmission, drivetrain, sourceURL sql.NullString
		var publishedAt sql.NullTime
		var viewCount int

		err := rows.Scan(
			&l.ID, &accountID, &l.Title, &price, &l.Description, &sourceURL,
			&l.Status, &year, &make, &model, &mileageKm, &location,
			&condition, &notes, &transmission, &drivetrain, &publishedAt,
			&viewCount,
		)
		if err != nil {
			return nil, nil, err
		}
		l.AccountID = int(accountID.Int64)
		l.Price = int(price.Int64)
		l.Year = nullIntPtr(year)
		l.Make = nullStringPtr(make)
		l.Model = nullStringPtr(model)
		l.MileageKm = nullIntPtr(mileageKm)
		l.Location = nullStringPtr(location)
		l.Condition = nullStringPtr(condition)
		l.Notes = nullStringPtr(notes)
		l.Transmission = nullStringPtr(transmission)
		l.Drivetrain = nullStringPtr(drivetrain)
		l.SourceURL = nullStringPtr(sourceURL)
		l.PublishedAt = nullTimePtr(publishedAt)
		listings = append(listings, &l)
		views[l.ID] = viewCount
	}
	return listings, views, rows.Err()
}
