package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

)

// Header renders the top bar with listener info, session count, and mode.
type Header struct {
	ListenerAddr string
	SessionCount int
	Context      ContextMode
	Width        int
}

func NewHeader(addr string) Header {
	return Header{
		ListenerAddr: addr,
		Context:      ContextMenu,
	}
}

func (h Header) View() string {
	droplet := "󰗣"

	modeStr := "MENU"
	modeColor := lipgloss.Color("6") // Cyan
	if h.Context == ContextShell {
		modeStr = "SHELL"
		modeColor = lipgloss.Color("5") // Magenta
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("5")).
		Bold(true)

	addrStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6"))

	modeStyle := lipgloss.NewStyle().
		Foreground(modeColor).
		Bold(true)

	sepStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	sep := sepStyle.Render(" │ ")

	left := titleStyle.Render(fmt.Sprintf("gummy %s", droplet)) +
		sep +
		addrStyle.Render(h.ListenerAddr) +
		sep +
		countStyle.Render(fmt.Sprintf("%d sessions", h.SessionCount))

	right := modeStyle.Render(modeStr)

	// Pad middle to right-align mode indicator
	gap := h.Width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	line := left + fmt.Sprintf("%*s", gap, "") + right

	barStyle := lipgloss.NewStyle().
		Width(h.Width).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("255"))

	return barStyle.Render(line)
}
