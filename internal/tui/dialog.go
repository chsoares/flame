package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
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
	// --- Dialog width: generous like Crush ---
	dialogW := 52
	if dialogW > termW-4 {
		dialogW = termW - 4
	}
	innerW := dialogW - 6 // 2 border + 4 padding (2 each side)
	contentW := innerW - 4
	if contentW < 1 {
		contentW = 1
	}
	centerRow := func(s string) string {
		w := lipgloss.Width(s)
		if w >= contentW {
			return s
		}
		padL := (contentW - w) / 2
		padR := contentW - w - padL
		return strings.Repeat(" ", padL) + s + strings.Repeat(" ", padR)
	}

	shellName := "quit"
	if d.Action == DialogKill {
		shellName = "kill"
	}
	headerW := contentW - lipgloss.Width(shellName) - 1
	if headerW < 1 {
		headerW = 1
	}
	headerRow := styleMagentaBold.Render(shellName) + " " + hatching(headerW)

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

	// --- Build content ---
	var content []string
	content = append(content, headerRow)
	content = append(content, "")
	content = append(content, centerRow(styleBase.Render(d.Title)))
	if d.SubMessage != "" {
		content = append(content, centerRow(styleMuted.Render(d.SubMessage)))
	}
	content = append(content, "")
	content = append(content, centerRow(buttonRow))
	content = append(content, "")
	content = append(content, centerRow(hintRow))
	content = append(content, "")

	inner := strings.Join(content, "\n")

	// --- Box with rounded cyan border ---
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMagenta).
		Padding(0, 2).
		Width(innerW)

	box := boxStyle.Render(inner)

	return overlayCenteredBox(base, box, termW, termH)
}

func overlayCenteredBox(base, box string, termW, termH int) string {
	if termW <= 0 || termH <= 0 {
		return base
	}

	lines := strings.Split(base, "\n")
	if len(lines) < termH {
		for len(lines) < termH {
			lines = append(lines, strings.Repeat(" ", termW))
		}
	} else if len(lines) > termH {
		lines = lines[:termH]
	}
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < termW {
			lines[i] = line + strings.Repeat(" ", termW-w)
		} else if w > termW {
			lines[i] = ansi.Truncate(line, termW, "")
		}
	}

	boxLines := strings.Split(box, "\n")
	boxH := len(boxLines)
	boxW := 0
	for _, line := range boxLines {
		if w := lipgloss.Width(line); w > boxW {
			boxW = w
		}
	}
	if boxW <= 0 || boxH <= 0 {
		return strings.Join(lines, "\n")
	}

	x := (termW - boxW) / 2
	if x < 0 {
		x = 0
	}
	y := (termH - boxH) / 2
	if y < 0 {
		y = 0
	}

	for i, boxLine := range boxLines {
		row := y + i
		if row < 0 || row >= len(lines) {
			continue
		}

		line := lines[row]
		if lipgloss.Width(line) < termW {
			line += strings.Repeat(" ", termW-lipgloss.Width(line))
		}

		left := ansi.Cut(line, 0, x)
		right := ansi.Cut(line, x+boxW, termW)
		placed := boxLine
		if bw := lipgloss.Width(placed); bw > boxW {
			placed = ansi.Cut(placed, 0, boxW)
		}
		lines[row] = left + placed + right
	}

	return strings.Join(lines, "\n")
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
