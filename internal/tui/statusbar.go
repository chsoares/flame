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
	Context        ContextMode
	TransferPct    int    // -1 = no transfer, 0-100 = progress
	TransferMsg    string // e.g., "Uploading CLAUDE.md"
	TransferRight  string // Right side text: "47%" or "15.2 KB"
	TransferUpload bool   // true=upload, false=download
	Width          int
	Notify         *Notification // Active notification (overlays entire bar)
}

func NewStatusBar(width int) StatusBar {
	return StatusBar{
		TransferPct: -1,
		Width:       width,
	}
}

func (s StatusBar) View() string {
	// Notification priority: error/important notifications override transfer progress
	if s.Notify != nil && s.Notify.Level >= NotifyImportant {
		return s.renderNotification()
	}

	// Transfer progress overlay
	if s.TransferPct >= 0 {
		return s.renderTransferProgress()
	}

	// Info notifications (clipboard copy, etc.)
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
		left = hint("!", "flame cmd") + dot +
			hint("F11", "sidebar") + dot +
			hint("F12", "detach") + dot +
			hint("PgUp/PgDn", "scroll") + dot +
			hint("Ctrl+C", "interrupt") + dot +
			hint("Ctrl+D", "quit")
	} else {
		left = hint("Tab", "complete") + dot +
			hint("F11", "sidebar") + dot +
			hint("F12", "attach") + dot +
			hint("PgUp/PgDn", "scroll") + dot +
			hint("Ctrl+C", "cancel") + dot +
			hint("Ctrl+D", "quit")
	}

	gap := s.Width - lipgloss.Width(left)
	if gap < 1 {
		gap = 1
	}

	return left + fmt.Sprintf("%*s", gap, "")
}

// renderTransferProgress renders a full-width progress bar overlay, same style as notifications.
// Upload:   " ⬆ Uploading CLAUDE.md ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱              47% "
// Download: " ⬇ Downloading passwd ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱         83% "
func (s StatusBar) renderTransferProgress() string {
	bg := lipgloss.Color("4") // Blue background (same family as notifications)

	icon := ui.SymbolUpload
	if !s.TransferUpload {
		icon = ui.SymbolDownload
	}

	label := icon + " " + s.TransferMsg
	pctStr := s.TransferRight
	if pctStr == "" {
		pctStr = fmt.Sprintf("%d%%", s.TransferPct)
	}

	// Style for text on colored background
	textStyle := lipgloss.NewStyle().
		Background(bg).
		Foreground(lipgloss.Color("0")).
		Bold(true)

	// Calculate bar width: total - label - pct - spacing
	labelW := lipgloss.Width(label) + 2 // " icon text "
	pctW := len(pctStr) + 2             // " NN% "
	barWidth := s.Width - labelW - pctW
	if barWidth < 5 {
		barWidth = 5
	}

	filled := barWidth * s.TransferPct / 100
	if filled > barWidth {
		filled = barWidth
	}
	empty := barWidth - filled

	// Build hatching bar on colored background
	var filledBar string
	for i := 0; i < filled; i++ {
		filledBar += "/"
	}

	barStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("0"))
	emptyStyle := lipgloss.NewStyle().Background(bg)

	rendered := textStyle.Render(" "+label+" ") +
		barStyle.Render(filledBar) +
		emptyStyle.Render(fmt.Sprintf("%*s", empty, "")) +
		textStyle.Render(" "+pctStr+" ")

	// Fill remaining width
	contentW := lipgloss.Width(rendered)
	if contentW < s.Width {
		fill := lipgloss.NewStyle().Background(bg).Render(
			fmt.Sprintf("%*s", s.Width-contentW, ""),
		)
		rendered += fill
	}

	return rendered
}

func (s StatusBar) renderNotification() string {
	var bg lipgloss.Color
	var icon, prefix string

	switch s.Notify.Level {
	case NotifyInfo:
		bg = lipgloss.Color("4") // Blue
		icon = ui.SymbolInfo
		prefix = "Done!"
	case NotifyImportant:
		bg = lipgloss.Color("6") // Cyan
		icon = ui.SymbolFire
		prefix = "Yay!"
	case NotifyError:
		bg = lipgloss.Color("1") // Red
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
