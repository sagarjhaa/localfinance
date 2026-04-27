// Package monthreview owns the "Month in Review" feature: it generates a
// per-period summary (insights + narrative), caches it, and exposes both an
// internal trigger (called by Logos after upload) and a public read endpoint.
package monthreview

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sagarjhaa/localfinance/services/sophia/insights"
	"github.com/sagarjhaa/localfinance/services/sophia/models"
)

// Period identifies a calendar month in UTC.
type Period struct {
	Year  int
	Month int // 1-12
}

// String formats the period as "YYYY-MM".
func (p Period) String() string {
	return fmt.Sprintf("%04d-%02d", p.Year, p.Month)
}

// ParsePeriod parses a "YYYY-MM" string into a Period. Whitespace is trimmed.
// Returns an error for malformed input or out-of-range months.
func ParsePeriod(s string) (Period, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "-")
	if len(parts) != 2 {
		return Period{}, fmt.Errorf("invalid period %q: expected YYYY-MM", s)
	}
	year, err := strconv.Atoi(parts[0])
	if err != nil || len(parts[0]) != 4 || year < 1900 || year > 9999 {
		return Period{}, fmt.Errorf("invalid period year in %q", s)
	}
	month, err := strconv.Atoi(parts[1])
	if err != nil || month < 1 || month > 12 {
		return Period{}, fmt.Errorf("invalid period month in %q", s)
	}
	return Period{Year: year, Month: month}, nil
}

// StartEnd returns the [start, end) UTC window covered by this period.
// End is exclusive (the first instant of the following month).
func (p Period) StartEnd() (time.Time, time.Time) {
	start := time.Date(p.Year, time.Month(p.Month), 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(p.Year, time.Month(p.Month)+1, 1, 0, 0, 0, 0, time.UTC)
	return start, end
}

// PeriodFromTime returns the Period that contains the given time (in UTC).
func PeriodFromTime(t time.Time) Period {
	t = t.UTC()
	return Period{Year: t.Year(), Month: int(t.Month())}
}

// MonthReview is the cached per-period summary returned to the UI.
type MonthReview struct {
	UserID      string                            `json:"user_id"`
	Period      string                            `json:"period"` // "YYYY-MM"
	GeneratedAt time.Time                         `json:"generated_at"`
	Insights    []models.FinancialInsight         `json:"insights"`
	Narrative   insights.MonthInReviewNarrative   `json:"narrative"`
}
