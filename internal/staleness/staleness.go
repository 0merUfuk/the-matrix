// Package staleness provides frontmatter date parsing and age calculation
// for knowledge doc freshness checks.
package staleness

import (
	"fmt"
	"math"
	"regexp"
	"time"
)

var (
	createdRe     = regexp.MustCompile(`\*\*Created\*\*:\s*(\d{4}-\d{2}-\d{2})`)
	lastUpdatedRe = regexp.MustCompile(`\*\*Last Updated\*\*:\s*(\d{4}-\d{2}-\d{2})`)
)

const avgMonthDays = 30.44

// ParseCreatedDate extracts the **Created** date from markdown frontmatter.
// Falls back to **Last Updated** if Created is not found.
// Returns the parsed time and true if found, or zero time and false.
func ParseCreatedDate(content string) (time.Time, bool) {
	// Try **Created** first
	if m := createdRe.FindStringSubmatch(content); m != nil {
		if t, err := time.Parse("2006-01-02", m[1]); err == nil {
			return t, true
		}
	}
	// Fallback to **Last Updated**
	if m := lastUpdatedRe.FindStringSubmatch(content); m != nil {
		if t, err := time.Parse("2006-01-02", m[1]); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// MonthsAgo calculates the number of months elapsed since the given time.
func MonthsAgo(t time.Time) float64 {
	days := time.Since(t).Hours() / 24
	return days / avgMonthDays
}

// FormatAge returns a human-readable age string.
func FormatAge(t time.Time) string {
	days := int(time.Since(t).Hours() / 24)
	if days < 1 {
		return "today"
	}
	if days == 1 {
		return "1 day ago"
	}
	if days < 30 {
		return fmt.Sprintf("%d days ago", days)
	}
	months := int(math.Floor(float64(days) / avgMonthDays))
	if months == 1 {
		return "1 month ago"
	}
	return fmt.Sprintf("%d months ago", months)
}

// IsStale returns true if the given time is older than maxMonths.
func IsStale(t time.Time, maxMonths float64) bool {
	return MonthsAgo(t) >= maxMonths
}
