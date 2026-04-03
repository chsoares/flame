package tui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Input is the bottom input bar with dynamic prompt and command history.
type Input struct {
	textinput textinput.Model
	prompt    string
	context   ContextMode
	width     int
	sessionID int // Selected session NumID (0 = none)
	bangMode  bool // In shell context, user typed ! to run gummy commands

	// Per-context history
	menuHistory    []string         // Menu command history
	sessionHistory map[int][]string // Per-session shell command history
	histIdx        int
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
		textinput:      ti,
		prompt:         menuPrompt(0),
		context:        ContextMenu,
		histIdx:        -1,
		sessionHistory: make(map[int][]string),
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
	i.bangMode = false
	i.histIdx = -1 // Reset history navigation on context switch
	i.textinput.TextStyle = lipgloss.NewStyle().Foreground(colorBase)
	if ctx == ContextShell {
		i.textinput.Placeholder = "Type a shell command..."
	} else {
		i.textinput.Placeholder = "Type a command..."
	}
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

// SetValue sets the input text and moves cursor to end.
func (i *Input) SetValue(s string) {
	i.textinput.SetValue(s)
	i.textinput.CursorEnd()
}

// Clear resets the input text.
func (i *Input) Clear() {
	i.textinput.SetValue("")
	i.histIdx = -1
}

// history returns the active history slice for the current context.
func (i *Input) history() []string {
	if i.bangMode {
		return i.menuHistory // Bang mode uses gummy command history
	}
	if i.context == ContextShell && i.sessionID > 0 {
		return i.sessionHistory[i.sessionID]
	}
	return i.menuHistory
}

// appendHistory adds a command to the active history.
func (i *Input) appendHistory(val string) {
	if i.bangMode {
		i.menuHistory = append(i.menuHistory, val)
		return
	}
	if i.context == ContextShell && i.sessionID > 0 {
		i.sessionHistory[i.sessionID] = append(i.sessionHistory[i.sessionID], val)
	} else {
		i.menuHistory = append(i.menuHistory, val)
	}
}

// Submit returns the current value and adds it to history.
func (i *Input) Submit() string {
	val := i.textinput.Value()
	if val != "" {
		i.appendHistory(val)
	}
	i.Clear()
	return val
}

// HistoryUp navigates to the previous command in history.
func (i *Input) HistoryUp() {
	hist := i.history()
	if len(hist) == 0 {
		return
	}
	if i.histIdx == -1 {
		i.histIdx = len(hist) - 1
	} else if i.histIdx > 0 {
		i.histIdx--
	}
	i.textinput.SetValue(hist[i.histIdx])
	i.textinput.CursorEnd()
}

// HistoryDown navigates to the next command in history.
func (i *Input) HistoryDown() {
	hist := i.history()
	if i.histIdx == -1 {
		return
	}
	if i.histIdx < len(hist)-1 {
		i.histIdx++
		i.textinput.SetValue(hist[i.histIdx])
		i.textinput.CursorEnd()
	} else {
		i.histIdx = -1
		i.textinput.SetValue("")
	}
}

// EnterBangMode switches the input to gummy command mode (! prefix in shell).
func (i *Input) EnterBangMode() {
	i.bangMode = true
	i.histIdx = -1
	i.prompt = styleMagenta.Bold(true).Render("!") + " "
	i.textinput.Placeholder = "upload, download, run, spawn..."
	i.textinput.TextStyle = lipgloss.NewStyle().Foreground(colorMagenta)
	i.SetWidth(i.width)
}

// ExitBangMode switches back to normal shell input.
func (i *Input) ExitBangMode() {
	i.bangMode = false
	i.histIdx = -1
	i.textinput.Placeholder = "Type a shell command..."
	i.textinput.TextStyle = lipgloss.NewStyle().Foreground(colorBase)
	i.prompt = styleCyan.Render("$") + " "
	i.SetWidth(i.width)
}

// InBangMode returns whether ! command mode is active.
func (i *Input) InBangMode() bool {
	return i.bangMode
}

func (i *Input) updatePrompt() {
	if i.bangMode {
		return // Don't override bang mode prompt
	}
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

// historyPath returns the path to the persistent menu history file.
func historyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gummy", "menu_history.txt")
}

// LoadHistory loads menu history from disk.
func (i *Input) LoadHistory() {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			i.menuHistory = append(i.menuHistory, line)
		}
	}
}

// SaveHistory persists menu history to disk (last 500 entries).
func (i *Input) SaveHistory() {
	hist := i.menuHistory
	if len(hist) > 500 {
		hist = hist[len(hist)-500:]
	}
	path := historyPath()
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, []byte(strings.Join(hist, "\n")+"\n"), 0644)
}
