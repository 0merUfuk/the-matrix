package staleness

import (
	"testing"
	"time"
)

func TestParseCreatedDate(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantOK  bool
		wantDay int
	}{
		{
			name:    "standard Created field",
			content: "**Version**: 1.0\n**Created**: 2026-03-10\n**Last Updated**: 2026-03-12",
			wantOK:  true,
			wantDay: 10,
		},
		{
			name:    "fallback to Last Updated",
			content: "**Version**: 1.0\n**Last Updated**: 2026-03-15",
			wantOK:  true,
			wantDay: 15,
		},
		{
			name:    "no date found",
			content: "# Just a title\n\nSome content without dates.",
			wantOK:  false,
		},
		{
			name:    "invalid date format",
			content: "**Created**: not-a-date",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseCreatedDate(tt.content)
			if ok != tt.wantOK {
				t.Fatalf("ParseCreatedDate() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && got.Day() != tt.wantDay {
				t.Errorf("ParseCreatedDate() day = %d, want %d", got.Day(), tt.wantDay)
			}
		})
	}
}

func TestMonthsAgo(t *testing.T) {
	thirtyDaysAgo := time.Now().Add(-30 * 24 * time.Hour)
	months := MonthsAgo(thirtyDaysAgo)
	if months < 0.9 || months > 1.1 {
		t.Errorf("MonthsAgo(30 days) = %f, want ~1.0", months)
	}
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name string
		when time.Time
		want string
	}{
		{"today", time.Now(), "today"},
		{"1 day ago", time.Now().Add(-25 * time.Hour), "1 day ago"},
		{"5 days ago", time.Now().Add(-5 * 24 * time.Hour), "5 days ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatAge(tt.when)
			if got != tt.want {
				t.Errorf("FormatAge() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsStale(t *testing.T) {
	fresh := time.Now().Add(-2 * 30 * 24 * time.Hour) // ~2 months ago
	stale := time.Now().Add(-8 * 30 * 24 * time.Hour) // ~8 months ago

	if IsStale(fresh, 6) {
		t.Error("2-month-old doc should not be stale at 6-month threshold")
	}
	if !IsStale(stale, 6) {
		t.Error("8-month-old doc should be stale at 6-month threshold")
	}
}
