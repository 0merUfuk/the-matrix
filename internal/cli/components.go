// Package cli provides shared terminal output formatting for all the-matrix tools.
// components.go defines reusable rendering methods on *Theme.
package cli

import (
	"fmt"
	"strings"
)

// StatusLine renders a pass/fail/warn status line with icon, label, and detail.
// The icon is colored according to the level. Detail text appears dimmed after the label.
func (t *Theme) StatusLine(level Level, label, detail string) string {
	var icon string
	switch level {
	case Pass:
		icon = t.Success.Render("✓")
	case Fail:
		icon = t.Error.Render("✗")
	case Warn:
		icon = t.Warning.Render("⚠")
	}

	if detail != "" {
		return fmt.Sprintf("  %s  %s  %s", icon, label, t.Muted.Render(detail))
	}
	return fmt.Sprintf("  %s  %s", icon, label)
}

// SectionHeader renders a bold uppercase section title with consistent spacing.
// Returns the title with a blank line before it.
func (t *Theme) SectionHeader(title string) string {
	return fmt.Sprintf("\n  %s", t.SectionTitle.Render(strings.ToUpper(title)))
}

// badgeBox is the shared implementation for ErrorBox, SuccessBox, WarnBox, InfoBox.
// badgeWidth is the display width of the badge string (excluding ANSI codes).
func (t *Theme) badgeBox(badge string, badgeWidth int, msg string, details []string) string {
	var b strings.Builder

	// First line: 2-space indent + badge + 2-space gap + message
	fmt.Fprintf(&b, "  %s  %s", badge, msg)

	// Detail lines aligned with the message text.
	// Indent = 2 (indent) + badgeWidth + 2 (gap)
	if len(details) > 0 {
		indent := strings.Repeat(" ", 2+badgeWidth+2)
		for _, d := range details {
			fmt.Fprintf(&b, "\n%s%s", indent, t.Muted.Render(d))
		}
	}

	return b.String()
}

// All badges use 5 chars of text + 2 chars of padding = 7 display chars.
// This ensures consistent alignment across all box types.
const badgeDisplayWidth = 7

// ErrorBox renders an error with ERROR pill badge + message + optional detail lines.
func (t *Theme) ErrorBox(msg string, details ...string) string {
	return t.badgeBox(t.ErrorBadge.Render("ERROR"), badgeDisplayWidth, msg, details)
}

// SuccessBox renders success with OK pill badge + message + optional detail lines.
func (t *Theme) SuccessBox(msg string, details ...string) string {
	return t.badgeBox(t.SuccessBadge.Render(" OK  "), badgeDisplayWidth, msg, details)
}

// WarnBox renders a warning with WARN pill badge + message + optional detail lines.
func (t *Theme) WarnBox(msg string, details ...string) string {
	return t.badgeBox(t.WarnBadge.Render(" WARN"), badgeDisplayWidth, msg, details)
}

// InfoBox renders info with INFO pill badge + message + optional detail lines.
func (t *Theme) InfoBox(msg string, details ...string) string {
	return t.badgeBox(t.InfoBadge.Render(" INFO"), badgeDisplayWidth, msg, details)
}

// KV is a key-value pair for the KeyValue component.
type KV struct {
	Key   string
	Value string
}

// KeyValue renders aligned key-value pairs. Keys are bold-styled and
// right-padded to the width of the longest key for visual alignment.
func (t *Theme) KeyValue(pairs []KV) string {
	if len(pairs) == 0 {
		return ""
	}

	// Find max key length for alignment.
	maxKey := 0
	for _, p := range pairs {
		if len(p.Key) > maxKey {
			maxKey = len(p.Key)
		}
	}

	var b strings.Builder
	for i, p := range pairs {
		if i > 0 {
			b.WriteByte('\n')
		}
		padded := p.Key + strings.Repeat(" ", maxKey-len(p.Key))
		fmt.Fprintf(&b, "  %s   %s", t.KeyLabel.Render(padded), p.Value)
	}
	return b.String()
}

// Summary renders a colored pass/fail/warn summary line.
// Non-zero counts are included; zero counts are omitted.
func (t *Theme) Summary(pass, fail, warn int) string {
	var parts []string
	if fail > 0 {
		parts = append(parts, t.Error.Render(fmt.Sprintf("%d critical", fail)))
	}
	if warn > 0 {
		label := "warning"
		if warn > 1 {
			label = "warnings"
		}
		parts = append(parts, t.Warning.Render(fmt.Sprintf("%d %s", warn, label)))
	}
	if pass > 0 {
		parts = append(parts, t.Success.Render(fmt.Sprintf("%d passed", pass)))
	}

	if len(parts) == 0 {
		return "  Status: " + t.Muted.Render("no checks run")
	}
	return "  Status: " + strings.Join(parts, ", ")
}

// Banner renders a tool header line: tool name in accent + detail in secondary.
// Returns the banner with surrounding blank lines for visual separation.
func (t *Theme) Banner(tool, detail string) string {
	if detail != "" {
		return fmt.Sprintf("\n  %s\n", t.AccentBold.Render(fmt.Sprintf("%s — %s", tool, detail)))
	}
	return fmt.Sprintf("\n  %s\n", t.AccentBold.Render(tool))
}
