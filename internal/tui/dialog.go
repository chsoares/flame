package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DialogAction identifies what the dialog is confirming.
type DialogAction int

const (
	DialogQuit DialogAction = iota
	DialogKill              // Future: kill session confirmation
)

// Dialog is a modal overlay for confirmations (quit, kill, etc.).
type Dialog struct {
	Title      string
	Message    string
	SubMessage string       // Secondary info (muted)
	Action     DialogAction
	Selected   int // 0 = confirm (left), 1 = cancel (right)
}

// Toggle swaps between confirm and cancel.
func (d *Dialog) Toggle() {
	d.Selected = 1 - d.Selected
}

// View renders the dialog centered on a dark background.
func (d *Dialog) View(termW, termH int, _ string) string {
	// --- Dialog width: generous like Crush ---
	dialogW := 52
	if dialogW > termW-4 {
		dialogW = termW - 4
	}
	innerW := dialogW - 6 // 2 border + 4 padding (2 each side)

	// --- Buttons: "Yep!" and "Nope" with underlined Y/N ---
	btnW := 14

	selectedBg := colorMagenta
	unselectedBg := colorDim // 237, matches our layout palette

	makeBtn := func(label string, underlineIdx int, selected bool) string {
		bg := unselectedBg
		fg := colorMuted
		if selected {
			bg = selectedBg
			fg = lipgloss.Color("255")
		}

		base := lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(selected)
		ul := lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(selected).Underline(true)

		// Build label with underlined character
		before := label[:underlineIdx]
		char := string(label[underlineIdx])
		after := label[underlineIdx+1:]
		text := base.Render(before) + ul.Render(char) + base.Render(after)

		// Pad to button width: center the text
		textW := lipgloss.Width(text)
		padTotal := btnW - textW
		if padTotal < 0 {
			padTotal = 0
		}
		padL := padTotal / 2
		padR := padTotal - padL
		padStyle := lipgloss.NewStyle().Background(bg)
		return padStyle.Render(strings.Repeat(" ", padL)) + text + padStyle.Render(strings.Repeat(" ", padR))
	}

	confirmBtn := makeBtn("Yep!", 0, d.Selected == 0)
	cancelBtn := makeBtn("Nope", 0, d.Selected == 1)

	buttonRow := confirmBtn + "  " + cancelBtn

	// --- Hint line (same style as status bar) ---
	dot := styleSubtle.Render(" • ")
	hint := func(key, desc string) string {
		return styleMuted.Bold(true).Render(key) + " " + styleSubtle.Render(desc)
	}
	hintRow := hint("Tab", "toggle") + dot + hint("Enter", "confirm") + dot + hint("Esc", "cancel")

	// --- Build content ---
	var content []string
	content = append(content, "")
	content = append(content, styleBase.Render(d.Title))
	if d.SubMessage != "" {
		content = append(content, styleMuted.Render(d.SubMessage))
	}
	content = append(content, "")
	content = append(content, buttonRow)
	content = append(content, "")
	content = append(content, hintRow)
	content = append(content, "")

	inner := strings.Join(content, "\n")

	// --- Box with rounded cyan border ---
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorCyan).
		Padding(0, 2).
		Width(innerW)

	box := boxStyle.Render(inner)

	// Center on dark background
	return lipgloss.Place(termW, termH, lipgloss.Center, lipgloss.Center, box)
}

// confirmQuitDialog builds the dialog for quit with active sessions.
func confirmQuitDialog(sessionCount int) *Dialog {
	sessWord := "session"
	if sessionCount > 1 {
		sessWord = "sessions"
	}
	return &Dialog{
		Title:      "Are you sure you want to quit?",
		SubMessage: fmt.Sprintf("%d active %s will be lost.", sessionCount, sessWord),
		Action:     DialogQuit,
		Selected:   1, // Default to Nope (safe)
	}
}
