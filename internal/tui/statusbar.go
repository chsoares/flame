package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar renders the bottom status line with hotkey hints and transfer progress.
type StatusBar struct {
	Context     ContextMode
	TransferPct int    // -1 = no transfer, 0-100 = progress
	TransferMsg string // e.g., "linpeas.sh"
	Width       int
}

func NewStatusBar(width int) StatusBar {
	return StatusBar{
		TransferPct: -1,
		Width:       width,
	}
}

func (s StatusBar) View() string {
	// Hotkey hints: bold key + subtle desc, separated by •
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
