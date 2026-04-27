package repo

import (
	"database/sql"
	"fmt"
	"math/rand"
)

// StatsRepo provides data access for stats and view logs.
type StatsRepo struct {
	db *sql.DB
}

// NewStatsRepo creates a new StatsRepo.
func NewStatsRepo(db *sql.DB) *StatsRepo {
	return &StatsRepo{db: db}
}

// LogView inserts a view log (deduplicated) and upserts the stats counter.
func (r *StatsRepo) LogView(targetType string, targetID int, fingerprint string) error {
	_, err := r.db.Exec(`
		INSERT OR IGNORE INTO view_log (target_type, target_id, fingerprint)
		VALUES (?, ?, ?)`, targetType, targetID, fingerprint)
	if err != nil {
		return fmt.Errorf("insert view_log: %w", err)
	}
	_, err = r.db.Exec(`
		INSERT INTO stats (target_type, target_id, view_count)
		VALUES (?, ?, 1)
		ON CONFLICT(target_type, target_id) DO UPDATE SET
			view_count = stats.view_count + 1`,
		targetType, targetID,
	)
	if err != nil {
		return fmt.Errorf("upsert stats: %w", err)
	}
	// 1% chance cleanup
	if rand.Intn(100) == 0 {
		_ = r.CleanupOld()
	}
	return nil
}

// GetViewCount returns the view count for a target.
func (r *StatsRepo) GetViewCount(targetType string, targetID int) (int, error) {
	var count int
	err := r.db.QueryRow(`
		SELECT COALESCE(view_count, 0) FROM stats
		WHERE target_type = ? AND target_id = ?`, targetType, targetID).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return count, err
}

// GetSiteVisits returns total site visits.
func (r *StatsRepo) GetSiteVisits() (int, error) {
	return r.GetViewCount("site", 0)
}

// GetTotalListingViews returns sum of all listing views.
func (r *StatsRepo) GetTotalListingViews() (int, error) {
	var total sql.NullInt64
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(view_count), 0) FROM stats WHERE target_type = 'listing'`).Scan(&total)
	return int(total.Int64), err
}

// CleanupOld removes view logs older than 24 hours.
func (r *StatsRepo) CleanupOld() error {
	_, err := r.db.Exec(`DELETE FROM view_log WHERE viewed_at < datetime('now', '-1 day')`)
	return err
}
