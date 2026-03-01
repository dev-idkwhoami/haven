package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderKV renders a label: value pair with consistent styling.
func renderKV(key, value string) string {
	return StyleLabel.Render(key) + " " + StyleValue.Render(value)
}

// renderBox renders a dashed-header box with title and content.
func renderBox(title, content string, width int) string {
	if width < 10 {
		width = 40
	}
	boxTitle := StyleBoxTitle.Render("── " + title + " ")
	titleLen := lipgloss.Width(boxTitle)
	remaining := width - titleLen
	if remaining < 0 {
		remaining = 0
	}
	header := boxTitle + StyleBoxTitle.Render(strings.Repeat("─", remaining))
	return lipgloss.JoinVertical(lipgloss.Left, header, content)
}

// truncate shortens a string to max characters, adding ellipsis if needed.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

// formatDuration formats a time.Duration as a human-readable string.
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// formatTimeAgo returns a human-readable relative time string.
func formatTimeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// actionIcon returns a character icon for an audit log action.
func actionIcon(action string) string {
	switch {
	case strings.Contains(action, "approve"):
		return "✓"
	case strings.Contains(action, "reject"):
		return "✕"
	case strings.Contains(action, "ban"):
		return "⊘"
	case strings.Contains(action, "kick"):
		return "←"
	default:
		return "●"
	}
}

// actionColor returns a lipgloss color for an audit log action.
func actionColor(action string) lipgloss.Color {
	switch {
	case strings.Contains(action, "approve"):
		return ColorOnline
	case strings.Contains(action, "reject"), strings.Contains(action, "ban"):
		return ColorError
	default:
		return ColorTextSecondary
	}
}
