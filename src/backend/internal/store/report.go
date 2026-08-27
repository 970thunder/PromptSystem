package store

// Report reasons are a bounded, typed enum consumed by both the prompt-report
// and comment-report flows. Clients must send one of the constants below;
// anything else is rejected at the store layer with ErrInvalidReportReason.
// Keeping the set closed here (and mirrored in docs/API契约.md) lets the admin
// review UI render a fixed reason set and avoids free-form spam in reports.
const (
	// ReportReasonSpam reports unsolicited or deceptive content.
	ReportReasonSpam = "spam"
	// ReportReasonAbuse reports harassment, threats, or targeted abuse.
	ReportReasonAbuse = "abuse"
	// ReportReasonNsfw reports sexual or explicit content.
	ReportReasonNsfw = "nsfw"
	// ReportReasonOther is the catch-all for reasons that do not fit the above.
	ReportReasonOther = "other"
)

// MaxReportDetailRunes caps the free-form report detail text. It matches the
// MySQL `reports.detail VARCHAR(500)` column so a report that passes validation
// can never be truncated or rejected by the database.
const MaxReportDetailRunes = 500

var reportReasons = map[string]struct{}{
	ReportReasonSpam:  {},
	ReportReasonAbuse: {},
	ReportReasonNsfw:  {},
	ReportReasonOther: {},
}

// ValidReportReason reports whether reason is one of the bounded report reasons.
func ValidReportReason(reason string) bool {
	_, ok := reportReasons[reason]
	return ok
}
