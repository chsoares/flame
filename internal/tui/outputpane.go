package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// OutputPane is a scrollable viewport that displays shell output or menu output.
type OutputPane struct {
	viewport viewport.Model
	content  strings.Builder
	width    int
	follow   bool // Auto-scroll to bottom
}

func NewOutputPane(width, height int) OutputPane {
	vp := viewport.New(width, height)
	return OutputPane{
		viewport: vp,
		width:    width,
		follow:   true,
	}
}

func (o *OutputPane) SetSize(width, height int) {
	o.width = width
	o.viewport.Width = width
	o.viewport.Height = height
	// Re-wrap existing content
	o.viewport.SetContent(o.wrapContent(o.content.String()))
	if o.follow {
		o.viewport.GotoBottom()
	}
}

// Append adds text to the output and optionally auto-scrolls.
func (o *OutputPane) Append(text string) {
	o.content.WriteString(text)
	o.viewport.SetContent(o.wrapContent(o.content.String()))
	if o.follow {
		o.viewport.GotoBottom()
	}
}

// SetContent replaces all content (used when switching sessions).
func (o *OutputPane) SetContent(text string) {
	o.content.Reset()
	o.content.WriteString(text)
	o.viewport.SetContent(o.wrapContent(text))
	o.viewport.GotoBottom()
	o.follow = true
}

// Clear removes all content.
func (o *OutputPane) Clear() {
	o.content.Reset()
	o.viewport.SetContent("")
}

func (o *OutputPane) Update(msg tea.Msg) (*OutputPane, tea.Cmd) {
	var cmd tea.Cmd
	o.viewport, cmd = o.viewport.Update(msg)

	if o.viewport.AtBottom() {
		o.follow = true
	} else {
		o.follow = false
	}

	return o, cmd
}

func (o *OutputPane) View() string {
	return o.viewport.View()
}

// wrapContent wraps long lines to fit the viewport width.
func (o *OutputPane) wrapContent(text string) string {
	if o.width <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var wrapped []string
	for _, line := range lines {
		w := lipgloss.Width(line)
		if w <= o.width {
			wrapped = append(wrapped, line)
			continue
		}
		// Hard wrap: split at viewport width boundary
		for len(line) > 0 {
			// Use lipgloss.Width for ANSI-aware width
			if lipgloss.Width(line) <= o.width {
				wrapped = append(wrapped, line)
				break
			}
			// Find split point: walk rune by rune
			cut := 0
			currentWidth := 0
			for i, r := range line {
				rw := lipgloss.Width(string(r))
				if currentWidth+rw > o.width {
					cut = i
					break
				}
				currentWidth += rw
				cut = i + len(string(r))
			}
			if cut == 0 {
				cut = 1 // Prevent infinite loop
			}
			wrapped = append(wrapped, line[:cut])
			line = line[cut:]
		}
	}
	return strings.Join(wrapped, "\n")
}
