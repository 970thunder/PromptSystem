package store

import "database/sql"

// PolymorphicIntegrityReport contains rows that cannot be resolved to a
// supported target. Polymorphic target IDs cannot use a single SQL foreign key,
// so this report is the periodic guardrail for direct SQL/import regressions.
type PolymorphicIntegrityReport struct {
	OrphanComments       int `json:"orphanComments"`
	OrphanLikes          int `json:"orphanLikes"`
	OrphanFavorites      int `json:"orphanFavorites"`
	OrphanReports        int `json:"orphanReports"`
	UnsupportedComments  int `json:"unsupportedComments"`
	UnsupportedLikes     int `json:"unsupportedLikes"`
	UnsupportedFavorites int `json:"unsupportedFavorites"`
	UnsupportedReports   int `json:"unsupportedReports"`
}

// CounterIntegrityReport reports prompts whose denormalized counters no longer
// equal their detail rows. Anonymous views are tracked separately because they
// cannot have a user foreign key; the audit therefore compares views against
// view_histories plus prompts.anonymous_views.
type CounterIntegrityReport struct {
	LikeDrift     int `json:"likeDrift"`
	FavoriteDrift int `json:"favoriteDrift"`
	ViewDrift     int `json:"viewDrift"`
}

func (r CounterIntegrityReport) Total() int {
	return r.LikeDrift + r.FavoriteDrift + r.ViewDrift
}

func (r PolymorphicIntegrityReport) Total() int {
	return r.OrphanComments + r.OrphanLikes + r.OrphanFavorites + r.OrphanReports +
		r.UnsupportedComments + r.UnsupportedLikes + r.UnsupportedFavorites + r.UnsupportedReports
}

// AuditMySQLPolymorphicIntegrity scans all polymorphic target tables. It only
// reads data and is safe to run from a low-frequency systemd timer or cron job.
func AuditMySQLPolymorphicIntegrity(db *sql.DB) (PolymorphicIntegrityReport, error) {
	var report PolymorphicIntegrityReport
	queries := []struct {
		dest  *int
		query string
	}{
		{&report.OrphanComments, `SELECT COUNT(*) FROM comments c WHERE (c.target_type = 'prompt' AND NOT EXISTS (SELECT 1 FROM prompts p WHERE p.id = c.target_id)) OR (c.target_type = 'skill' AND NOT EXISTS (SELECT 1 FROM skills s WHERE s.id = c.target_id))`},
		{&report.OrphanLikes, `SELECT COUNT(*) FROM likes l WHERE (l.target_type = 'prompt' AND NOT EXISTS (SELECT 1 FROM prompts p WHERE p.id = l.target_id)) OR (l.target_type = 'skill' AND NOT EXISTS (SELECT 1 FROM skills s WHERE s.id = l.target_id)) OR (l.target_type = 'comment' AND NOT EXISTS (SELECT 1 FROM comments c WHERE c.id = l.target_id))`},
		{&report.OrphanFavorites, `SELECT COUNT(*) FROM favorites f WHERE (f.target_type = 'prompt' AND NOT EXISTS (SELECT 1 FROM prompts p WHERE p.id = f.target_id)) OR (f.target_type = 'skill' AND NOT EXISTS (SELECT 1 FROM skills s WHERE s.id = f.target_id))`},
		{&report.OrphanReports, `SELECT COUNT(*) FROM reports r WHERE (r.target_type = 'prompt' AND NOT EXISTS (SELECT 1 FROM prompts p WHERE p.id = r.target_id)) OR (r.target_type = 'skill' AND NOT EXISTS (SELECT 1 FROM skills s WHERE s.id = r.target_id)) OR (r.target_type = 'comment' AND NOT EXISTS (SELECT 1 FROM comments c WHERE c.id = r.target_id))`},
		{&report.UnsupportedComments, `SELECT COUNT(*) FROM comments WHERE target_type NOT IN ('prompt', 'skill')`},
		{&report.UnsupportedLikes, `SELECT COUNT(*) FROM likes WHERE target_type NOT IN ('prompt', 'skill', 'comment')`},
		{&report.UnsupportedFavorites, `SELECT COUNT(*) FROM favorites WHERE target_type NOT IN ('prompt', 'skill')`},
		{&report.UnsupportedReports, `SELECT COUNT(*) FROM reports WHERE target_type NOT IN ('prompt', 'skill', 'comment')`},
	}
	for _, item := range queries {
		if err := db.QueryRow(item.query).Scan(item.dest); err != nil {
			return PolymorphicIntegrityReport{}, err
		}
	}
	return report, nil
}

// AuditMySQLPromptCounters verifies the denormalized prompt counters against
// their detail tables. It is read-only and intended for the same one-shot
// maintenance schedule as AuditMySQLPolymorphicIntegrity.
func AuditMySQLPromptCounters(db *sql.DB) (CounterIntegrityReport, error) {
	var report CounterIntegrityReport
	queries := []struct {
		dest  *int
		query string
	}{
		{&report.LikeDrift, `SELECT COUNT(*) FROM prompts p LEFT JOIN (SELECT target_id, COUNT(*) AS cnt FROM likes WHERE target_type = 'prompt' GROUP BY target_id) d ON d.target_id = p.id WHERE p.likes <> COALESCE(d.cnt, 0)`},
		{&report.FavoriteDrift, `SELECT COUNT(*) FROM prompts p LEFT JOIN (SELECT target_id, COUNT(*) AS cnt FROM favorites WHERE target_type = 'prompt' GROUP BY target_id) d ON d.target_id = p.id WHERE p.favorites <> COALESCE(d.cnt, 0)`},
		{&report.ViewDrift, `SELECT COUNT(*) FROM prompts p LEFT JOIN (SELECT prompt_id, COUNT(*) AS cnt FROM view_histories GROUP BY prompt_id) d ON d.prompt_id = p.id WHERE p.views <> COALESCE(d.cnt, 0) + p.anonymous_views`},
	}
	for _, item := range queries {
		if err := db.QueryRow(item.query).Scan(item.dest); err != nil {
			return CounterIntegrityReport{}, err
		}
	}
	return report, nil
}
