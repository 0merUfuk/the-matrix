// Package cli provides shared terminal output formatting for all the-matrix tools.
// output.go contains the original public API (PrintHeader, PrintSuccess, etc.)
// backed by a lazily-initialized default Theme for backward compatibility.
package cli

import (
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
)

// defaultTheme is lazily initialized on first use.
var (
	defaultTheme *Theme
	themeOnce    sync.Once
)

// getTheme returns the package-level default theme, creating it on first call.
func getTheme() *Theme {
	themeOnce.Do(func() {
		defaultTheme = NewTheme()
	})
	return defaultTheme
}

// GetTheme returns the shared default theme for direct use by commands
// that want the full Theme API (StatusLine, Banner, KeyValue, etc.).
func GetTheme() *Theme {
	return getTheme()
}

// PrintHeader prints a tool header line using the theme's Banner component.
func PrintHeader(tool, detail string) {
	fmt.Print(getTheme().Banner(tool, detail))
	fmt.Println()
}

// PrintSuccess prints a green success message with a check icon prefix.
func PrintSuccess(msg string) {
	fmt.Printf("  %s %s\n", getTheme().Success.Render("✓"), msg)
}

// PrintWarning prints a yellow warning message with a warning icon prefix.
func PrintWarning(msg string) {
	fmt.Printf("  %s %s\n", getTheme().Warning.Render("⚠"), msg)
}

// PrintError prints a red error message with a cross icon prefix.
func PrintError(msg string) {
	fmt.Printf("  %s %s\n", getTheme().Error.Render("✗"), msg)
}

// PrintDim prints dimmed secondary text.
func PrintDim(msg string) {
	fmt.Printf("  %s\n", getTheme().Muted.Render(msg))
}

// PrintCyan prints cyan text (for next-step commands).
func PrintCyan(msg string) {
	fmt.Printf("  %s\n", getTheme().Info.Render(msg))
}

// PrintWhite prints white text (for code/paths).
func PrintWhite(msg string) {
	fmt.Printf("    %s\n", getTheme().CodeStyle.Render(msg))
}

// PrintBoldGreen prints bold green text.
func PrintBoldGreen(msg string) {
	fmt.Printf("  %s\n", getTheme().Success.Bold(true).Render(msg))
}

// Level represents pass/fail/warn for status lines.
type Level int

const (
	Pass Level = iota
	Fail
	Warn
)

// PrintLine prints a status line with icon, label, and optional detail.
// Delegates to the theme's StatusLine for consistent rendering.
func PrintLine(level Level, label, detail string) {
	fmt.Println(getTheme().StatusLine(level, label, detail))
}

// PathToHome replaces the user's home directory with ~ for display.
func PathToHome(p string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

// ProgressBar renders an ASCII progress bar.
func ProgressBar(done, total, width int) string {
	if total == 0 {
		return strings.Repeat("░", width)
	}
	filled := int(math.Round(float64(done) / float64(total) * float64(width)))
	if filled > width {
		filled = width
	}
	t := getTheme()
	return t.Info.Render(strings.Repeat("█", filled) + strings.Repeat("░", width-filled))
}

// FormatDuration formats milliseconds into "Xh Ym" or "Xm" or a dash.
func FormatDuration(ms int64) string {
	if ms == 0 {
		return "—"
	}
	totalSec := int(math.Round(float64(ms) / 1000))
	hours := totalSec / 3600
	minutes := (totalSec % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// Pct calculates a rounded percentage.
func Pct(part, whole float64) int {
	if whole == 0 {
		return 0
	}
	return int(math.Round(part / whole * 100))
}

// Bold returns bold-styled text.
func Bold(s string) string {
	return getTheme().BoldStyle.Render(s)
}

// Dim returns dim-styled text.
func Dim(s string) string {
	return getTheme().Muted.Render(s)
}

// Green returns green-styled text.
func Green(s string) string {
	return getTheme().Success.Render(s)
}

// Red returns red-styled text.
func Red(s string) string {
	return getTheme().Error.Render(s)
}

// Yellow returns yellow-styled text.
func Yellow(s string) string {
	return getTheme().Warning.Render(s)
}

// Cyan returns cyan-styled text.
func Cyan(s string) string {
	return getTheme().Info.Render(s)
}
