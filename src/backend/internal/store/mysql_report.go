package store

import (
	"database/sql"
	"errors"
	"time"
)

// scanReportRow keeps report reads consistent when the write and the
// idempotency lookup run inside the same transaction. A report that does not
// exist is represented by found=false so callers can map it to a stable error.
func scanReportRow(scan func(dest ...any) error) (Report, bool, error) {
	var (
		report    Report
		createdAt time.Time
	)
	if err := scan(
		&report.ID,
		&report.UserID,
		&report.TargetType,
		&report.TargetID,
		&report.Reason,
		&report.Detail,
		&report.Status,
		&createdAt,
	); errors.Is(err, sql.ErrNoRows) {
		return Report{}, false, nil
	} else if err != nil {
		return Report{}, false, err
	}

	report.CreatedAt = createdAt.UTC().Format(time.RFC3339)
	return report, true, nil
}
