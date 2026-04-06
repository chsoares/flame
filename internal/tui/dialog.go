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
	SubMessage string // Secondary info (muted)
	Action     DialogAction
	Selected   int // 0 = confirm (left), 1 = cancel (right)
}

// Toggle swaps between confirm and cancel.
func (d *Dialog) Toggle() {
	d.Selected = 1 - d.Selected
}

// View renders the dialog centered on a dark background.
func (d *Dialog) View(termW, termH int, base string) string {
	const contentWidth = 42

	centerRow := func(s string) string {
		w := lipgloss.Width(s)
		if w >= contentWidth {
			return s
		}
		padL := (contentWidth - w) / 2
		padR := contentWidth - w - padL
		return strings.Repeat(" ", padL) + s + strings.Repeat(" ", padR)
	}

	// --- Buttons: "Yep!" and "Nope" with underlined Y/N ---
	btnW := 14

	selectedBg := colorCyan
	unselectedBg := colorDim // 237, matches our layout palette

	makeBtn := func(label string, underlineIdx int, selected bool) string {
		bg := unselectedBg
		fg := colorMuted
		if selected {
			bg = selectedBg
			fg = lipgloss.Color("0")
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

	body := []string{"", centerRow(styleBase.Render(d.Title))}
	if d.SubMessage != "" {
		body = append(body, centerRow(styleMuted.Render(d.SubMessage)))
	}
	body = append(body, "", centerRow(buttonRow), "", centerRow(hintRow))

	return RenderModalShell(base, termW, termH, ModalShell{
		Title:     shellNameForDialog(d.Action),
		Width:     52,
		MaxHeight: 9,
		Body:      body,
		Footer:    "",
		Align:     BodyAlignCenter,
	})
}

func shellNameForDialog(action DialogAction) string {
	if action == DialogKill {
		return "kill"
	}
	return "quit"
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
