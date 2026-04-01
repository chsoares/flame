package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

)

// StatusBar renders the bottom status line with mode, hotkeys, and progress.
type StatusBar struct {
	Context     ContextMode
	Focus       FocusMode
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
	modeStr := "MENU"
	modeColor := lipgloss.Color("6")
	if s.Context == ContextShell {
		modeStr = "SHELL"
		modeColor = lipgloss.Color("5")
	}

	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(modeColor).
		Bold(true).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6"))

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	mode := modeStyle.Render(modeStr)

	// Build hotkey hints based on context
	var hints string
	if s.Context == ContextShell {
		hints = keyStyle.Render("F12") + descStyle.Render(":menu ") +
			keyStyle.Render("Tab") + descStyle.Render(":sidebar ") +
			keyStyle.Render("Ctrl+C") + descStyle.Render(":interrupt")
	} else {
		hints = keyStyle.Render("Tab") + descStyle.Render(":sidebar ") +
			keyStyle.Render("Ctrl+C") + descStyle.Render(":- ")
	}

	left := mode + " " + hints

	// Transfer progress on the right
	var right string
	if s.TransferPct >= 0 {
		progressStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("5"))
		right = progressStyle.Render(fmt.Sprintf("⬆ %d%% %s", s.TransferPct, s.TransferMsg))
	}

	gap := s.Width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	barStyle := lipgloss.NewStyle().
		Width(s.Width).
		Background(lipgloss.Color("235"))

	return barStyle.Render(left + fmt.Sprintf("%*s", gap, "") + right)
}
