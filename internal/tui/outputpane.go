package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// OutputPane is a scrollable viewport that displays shell output or menu output.
type OutputPane struct {
	viewport viewport.Model
	content  strings.Builder
	follow   bool // Auto-scroll to bottom
}

func NewOutputPane(width, height int) OutputPane {
	vp := viewport.New(width, height)
	return OutputPane{
		viewport: vp,
		follow:   true,
	}
}

func (o *OutputPane) SetSize(width, height int) {
	o.viewport.Width = width
	o.viewport.Height = height
}

// Append adds text to the output and optionally auto-scrolls.
func (o *OutputPane) Append(text string) {
	o.content.WriteString(text)
	o.viewport.SetContent(o.content.String())
	if o.follow {
		o.viewport.GotoBottom()
	}
}

// SetContent replaces all content (used when switching sessions).
func (o *OutputPane) SetContent(text string) {
	o.content.Reset()
	o.content.WriteString(text)
	o.viewport.SetContent(text)
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

	// If user scrolled up, disable follow mode
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
