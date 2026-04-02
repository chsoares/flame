package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Input is the bottom input bar with dynamic prompt and command history.
type Input struct {
	textinput textinput.Model
	prompt    string
	context   ContextMode
	history   []string
	histIdx   int
	width     int
	sessionID int // Selected session NumID (0 = none)
}

func NewInput() Input {
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 4096
	ti.Prompt = "" // We render prompt ourselves
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorBase)
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(colorMagenta)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorSubtle)
	ti.Placeholder = "Type a command..."

	return Input{
		textinput: ti,
		prompt:    menuPrompt(0),
		context:   ContextMenu,
		histIdx:   -1,
	}
}

func (i *Input) SetWidth(w int) {
	i.width = w
	promptWidth := lipgloss.Width(i.prompt)
	inputWidth := w - promptWidth - 1
	if inputWidth < 10 {
		inputWidth = 10
	}
	i.textinput.Width = inputWidth
}

func (i *Input) SetContext(ctx ContextMode) {
	i.context = ctx
	i.updatePrompt()
}

func (i *Input) SetSessionID(id int) {
	i.sessionID = id
	i.updatePrompt()
}

func (i *Input) SetShellPrompt(prompt string) {
	i.prompt = prompt + " "
	i.SetWidth(i.width)
}

func (i *Input) Focus() {
	i.textinput.Focus()
}

func (i *Input) Blur() {
	i.textinput.Blur()
}

// Value returns the current input text.
func (i *Input) Value() string {
	return i.textinput.Value()
}

// Clear resets the input text.
func (i *Input) Clear() {
	i.textinput.SetValue("")
	i.histIdx = -1
}

// Submit returns the current value and adds it to history.
func (i *Input) Submit() string {
	val := i.textinput.Value()
	if val != "" {
		i.history = append(i.history, val)
	}
	i.Clear()
	return val
}

// HistoryUp navigates to the previous command in history.
func (i *Input) HistoryUp() {
	if len(i.history) == 0 {
		return
	}
	if i.histIdx == -1 {
		i.histIdx = len(i.history) - 1
	} else if i.histIdx > 0 {
		i.histIdx--
	}
	i.textinput.SetValue(i.history[i.histIdx])
	i.textinput.CursorEnd()
}

// HistoryDown navigates to the next command in history.
func (i *Input) HistoryDown() {
	if i.histIdx == -1 {
		return
	}
	if i.histIdx < len(i.history)-1 {
		i.histIdx++
		i.textinput.SetValue(i.history[i.histIdx])
		i.textinput.CursorEnd()
	} else {
		i.histIdx = -1
		i.textinput.SetValue("")
	}
}

func (i *Input) updatePrompt() {
	switch i.context {
	case ContextShell:
		if i.prompt == "" || i.prompt == menuPrompt(i.sessionID) {
			i.prompt = styleCyan.Render("$") + " "
		}
	case ContextMenu:
		i.prompt = menuPrompt(i.sessionID)
	}
	i.SetWidth(i.width)
}

func menuPrompt(sessionID int) string {
	droplet := "\U000F18FB"
	arrow := styleMagenta.Render("❯")

	if sessionID > 0 {
		return styleMagentaBold.Render(droplet+" gummy") +
			styleSubtle.Render("[") +
			styleCyan.Render(string(rune('0'+sessionID))) +
			styleSubtle.Render("]") +
			" " + arrow + " "
	}
	return styleMagentaBold.Render(droplet+" gummy") + " " + arrow + " "
}

func (i *Input) Update(msg tea.Msg) (*Input, tea.Cmd) {
	var cmd tea.Cmd
	i.textinput, cmd = i.textinput.Update(msg)
	return i, cmd
}

func (i *Input) View() string {
	return i.prompt + i.textinput.View()
}
