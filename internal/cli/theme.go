// Package cli provides shared terminal output formatting for all the-matrix tools.
// theme.go defines the Theme struct and constructor for unified visual styling.
package cli

import (
	"os"

	"charm.land/lipgloss/v2"
)

// Theme holds all visual styling decisions for the-matrix CLI tools.
// Create one via NewTheme(), which auto-detects the terminal background.
type Theme struct {
	// Semantic text styles
	Accent    lipgloss.Style // tool names, active elements — matrix green
	Primary   lipgloss.Style // main body text
	Secondary lipgloss.Style // labels, metadata, dimmed info
	Muted     lipgloss.Style // timestamps, paths, tertiary text

	// Status styles
	Success lipgloss.Style // green
	Error   lipgloss.Style // rose red
	Warning lipgloss.Style // amber
	Info    lipgloss.Style // soft blue

	// Pill badges (white text on colored bg, bold, horizontal padding)
	ErrorBadge   lipgloss.Style
	SuccessBadge lipgloss.Style
	WarnBadge    lipgloss.Style
	InfoBadge    lipgloss.Style

	// Typography
	BoldStyle    lipgloss.Style
	AccentBold   lipgloss.Style // pre-built accent + bold for Banner
	SectionTitle lipgloss.Style // bold uppercase for section headers
	KeyLabel     lipgloss.Style // bold for key in key-value pairs
	CodeStyle    lipgloss.Style // for inline commands/paths

	// State
	HasDarkBG bool
}

// NewTheme creates a Theme with colors adapted to the terminal background.
// It detects whether the terminal has a dark or light background and selects
// appropriate color values accordingly.
func NewTheme() *Theme {
	hasDark := lipgloss.HasDarkBackground(os.Stdin, os.Stderr)
	ld := lipgloss.LightDark(hasDark)

	// Adaptive palette — first arg is light-bg color, second is dark-bg color.
	accent := ld(lipgloss.Color("#00B36B"), lipgloss.Color("#00D787"))
	success := lipgloss.Color("#31BB71")
	errColor := lipgloss.Color("#FF5F87")
	warn := lipgloss.Color("#FFAF00")
	info := ld(lipgloss.Color("#5F87D7"), lipgloss.Color("#87AFFF"))
	primary := ld(lipgloss.Color("#333333"), lipgloss.Color("#DDDDDD"))
	secondary := ld(lipgloss.Color("#777777"), lipgloss.Color("#888888"))
	muted := ld(lipgloss.Color("#AAAAAA"), lipgloss.Color("#555555"))
	badgeFg := lipgloss.Color("#F1F1F1")

	return &Theme{
		// Semantic text
		Accent:    lipgloss.NewStyle().Foreground(accent),
		Primary:   lipgloss.NewStyle().Foreground(primary),
		Secondary: lipgloss.NewStyle().Foreground(secondary),
		Muted:     lipgloss.NewStyle().Foreground(muted),

		// Status
		Success: lipgloss.NewStyle().Foreground(success),
		Error:   lipgloss.NewStyle().Foreground(errColor),
		Warning: lipgloss.NewStyle().Foreground(warn),
		Info:    lipgloss.NewStyle().Foreground(info),

		// Pill badges: white text on colored background, bold, horizontal padding
		ErrorBadge:   lipgloss.NewStyle().Foreground(badgeFg).Background(lipgloss.Color("#FF5F87")).Bold(true).PaddingLeft(1).PaddingRight(1),
		SuccessBadge: lipgloss.NewStyle().Foreground(badgeFg).Background(lipgloss.Color("#00B36B")).Bold(true).PaddingLeft(1).PaddingRight(1),
		WarnBadge:    lipgloss.NewStyle().Foreground(badgeFg).Background(lipgloss.Color("#D7AF00")).Bold(true).PaddingLeft(1).PaddingRight(1),
		InfoBadge:    lipgloss.NewStyle().Foreground(badgeFg).Background(lipgloss.Color("#6C50FF")).Bold(true).PaddingLeft(1).PaddingRight(1),

		// Typography
		BoldStyle:    lipgloss.NewStyle().Bold(true),
		AccentBold:   lipgloss.NewStyle().Foreground(accent).Bold(true),
		SectionTitle: lipgloss.NewStyle().Bold(true).Foreground(primary),
		KeyLabel:     lipgloss.NewStyle().Bold(true).Foreground(primary),
		CodeStyle:    lipgloss.NewStyle().Foreground(muted),

		// State
		HasDarkBG: hasDark,
	}
}
