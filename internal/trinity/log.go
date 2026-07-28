package trinity

import (
	"fmt"
	"strings"
	"time"

	"github.com/0merUfuk/the-matrix/internal/config"
)

// BuildLogEntry creates a formatted log entry for a sync event.
// Format: "YYYY-MM-DD HH:MM | sync | {name} → {to} | {N} docs | {details}"
func BuildLogEntry(name, to string, total, newCount, updateCount int) string {
	now := time.Now()
	dateTime := now.Format("2006-01-02 15:04")

	var parts []string
	if newCount > 0 {
		parts = append(parts, fmt.Sprintf("%d new", newCount))
	}
	if updateCount > 0 {
		parts = append(parts, fmt.Sprintf("%d updated", updateCount))
	}

	return fmt.Sprintf("%s | sync | %s → %s | %d docs | %s",
		dateTime, name, to, total, strings.Join(parts, ", "))
}

// AppendLog appends a log entry to the trinity-log.md file.
// Creates the file with a header if it doesn't exist.
func AppendLog(logPath, entry string) error {
	if config.FileExists(logPath) {
		return config.AppendFileString(logPath, "\n"+entry)
	}

	// Create log file with header
	header := fmt.Sprintf("# Trinity Sync Log\n\n> Auto-maintained by `trinity sync`. Each line is one sync event.\n\n%s", entry)
	return config.WriteFileString(logPath, header)
}
