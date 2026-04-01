package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/chsoares/gummy/internal/ui"
)

// NotifyLevel determines the notification severity and color.
type NotifyLevel int

const (
	NotifyInfo      NotifyLevel = iota // Blue background (4) — clipboard, general info
	NotifyImportant                    // Cyan background (6) — new session, important events
	NotifyError                        // Magenta background (5) — session closed, errors
)

// Notification is a transient message that overlays the status bar.
type Notification struct {
	Message string
	Level   NotifyLevel
}

// StatusBar renders the bottom status line with hotkey hints, or a notification overlay.
type StatusBar struct {
	Context     ContextMode
	TransferPct int    // -1 = no transfer, 0-100 = progress
	TransferMsg string // e.g., "linpeas.sh"
	Width       int
	Notify      *Notification // Active notification (overlays entire bar)
}

func NewStatusBar(width int) StatusBar {
	return StatusBar{
		TransferPct: -1,
		Width:       width,
	}
}

func (s StatusBar) View() string {
	// Notification overlay takes over the entire bar
	if s.Notify != nil {
		return s.renderNotification()
	}

	// Normal mode: hotkey hints
	dot := styleSubtle.Render(" • ")
	hint := func(key, desc string) string {
		return styleMuted.Bold(true).Render(key) + " " + styleSubtle.Render(desc)
	}

	var left string
	if s.Context == ContextShell {
		left = hint("F12", "menu") + dot +
			hint("Tab", "sidebar") + dot +
			hint("Ctrl+C", "interrupt") + dot +
			hint("PgUp/PgDn", "scroll") + dot +
			hint("Ctrl+D", "quit")
	} else {
		left = hint("Tab", "sidebar") + dot +
			hint("PgUp/PgDn", "scroll") + dot +
			hint("Ctrl+D", "quit")
	}

	// Transfer progress on the right
	var right string
	if s.TransferPct >= 0 {
		right = styleMagenta.Render(fmt.Sprintf("⬆ %d%% %s", s.TransferPct, s.TransferMsg))
	}

	gap := s.Width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return left + fmt.Sprintf("%*s", gap, "") + right
}

func (s StatusBar) renderNotification() string {
	var bg lipgloss.Color
	var icon, prefix string

	switch s.Notify.Level {
	case NotifyInfo:
		bg = lipgloss.Color("4")  // Blue
		icon = ui.SymbolInfo
		prefix = "Done!"
	case NotifyImportant:
		bg = lipgloss.Color("6")  // Cyan
		icon = ui.SymbolFire
		prefix = "Yay!"
	case NotifyError:
		bg = lipgloss.Color("1")  // Red
		icon = ui.SymbolSkull
		prefix = "Oops!"
	}

	style := lipgloss.NewStyle().
		Background(bg).
		Foreground(lipgloss.Color("0")). // Black text
		Bold(true)

	msg := " " + icon + " " + prefix + " " + s.Notify.Message + " "
	rendered := style.Render(msg)

	// Fill remaining width with the same background
	contentW := lipgloss.Width(rendered)
	if contentW < s.Width {
		fill := lipgloss.NewStyle().Background(bg).Render(
			fmt.Sprintf("%*s", s.Width-contentW, ""),
		)
		rendered += fill
	}

	return rendered
}
