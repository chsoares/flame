package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Header renders the compact top bar (only shown in compact/narrow mode).
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

// View renders the compact header: logo + hatching + info
func (h Header) View() string {
	fire := ""

	logo := styleMagentaBold.Render("flame " + fire)
	addr := styleMuted.Render(h.ListenerAddr)
	sessions := styleMuted.Render(fmt.Sprintf("%d sessions", h.SessionCount))

	// logo + hatching + addr + sessions
	miniHatch := hatching(2)
	right := addr + " " + miniHatch + " " + sessions
	rightW := lipgloss.Width(right)
	logoW := lipgloss.Width(logo)

	hatchW := h.Width - logoW - rightW - 4 // 4 for spaces around hatching
	if hatchW < 3 {
		hatchW = 3
	}

	hatch := hatching(hatchW)
	line := logo + " " + hatch + " " + right

	// Right-pad to fill width
	lineW := lipgloss.Width(line)
	if lineW < h.Width {
		line = line + fmt.Sprintf("%*s", h.Width-lineW, "")
	}

	return lipgloss.NewStyle().Width(h.Width).Render(line)
}
