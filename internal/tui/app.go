package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// CommandExecutor is the interface the Manager must satisfy for the TUI.
type CommandExecutor interface {
	ExecuteCommand(cmd string) string
	GetSelectedSessionID() int
	SessionCount() int
	GetSessionsForDisplay() string
	SetSilent(silent bool)
	SetNotifyFunc(fn func(string))
}

// App is the root Bubble Tea model for the Gummy TUI.
type App struct {
	// Layout
	layout Layout
	width  int
	height int

	// Components (pointers — strings.Builder and viewport can't be copied by value)
	header    Header
	output    *OutputPane
	input     *Input
	statusBar StatusBar

	// State
	focus   FocusMode
	context ContextMode

	// Backend
	executor CommandExecutor

	// Config
	listenerAddr string
}

// New creates a new App model.
func New(executor CommandExecutor, listenerAddr string) App {
	output := NewOutputPane(80, 20)
	input := NewInput()
	return App{
		header:       NewHeader(listenerAddr),
		output:       &output,
		input:        &input,
		statusBar:    NewStatusBar(80),
		focus:        FocusInput,
		context:      ContextMenu,
		executor:     executor,
		listenerAddr: listenerAddr,
	}
}

func (a App) Init() tea.Cmd {
	return nil
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.layout = GenerateLayout(msg.Width, msg.Height)
		a.header.Width = msg.Width
		a.output.SetSize(a.layout.Output.W, a.layout.Output.H)
		a.input.SetWidth(a.layout.Input.W)
		a.statusBar.Width = msg.Width
		a.syncSessionInfo()
		return a, nil

	case tea.KeyMsg:
		switch a.focus {
		case FocusInput:
			return a.updateInputMode(msg)
		case FocusSidebar:
			return a.updateSidebarMode(msg)
		}

	case CommandOutputMsg:
		a.output.Append(msg.Output)
		a.syncSessionInfo() // Update sidebar/header after background events
		return a, nil

	case tea.MouseMsg:
		// Forward mouse events to viewport for scroll wheel support
		var cmd tea.Cmd
		a.output, cmd = a.output.Update(msg)
		return a, cmd
	}

	return a, nil
}

func (a App) updateInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		if a.context == ContextShell {
			// Phase 3: send interrupt to remote
			return a, nil
		}
		return a, nil

	case "ctrl+d":
		return a, tea.Quit

	case "enter":
		cmd := a.input.Submit()
		if cmd == "" {
			return a, nil
		}
		return a.executeInput(cmd)

	case "up":
		a.input.HistoryUp()
		return a, nil

	case "down":
		a.input.HistoryDown()
		return a, nil

	case "tab":
		// Phase 2: toggle sidebar
		return a, nil

	case "f12":
		if a.context == ContextShell {
			a.context = ContextMenu
			a.header.Context = ContextMenu
			a.statusBar.Context = ContextMenu
			a.input.SetContext(ContextMenu)
			a.output.Append("\n--- Detached from shell ---\n\n")
			return a, nil
		}

	case "pgup", "pgdown", "home", "end":
		// Forward scroll keys to output viewport
		var cmd tea.Cmd
		a.output, cmd = a.output.Update(msg)
		return a, cmd
	}

	// Forward to textinput for normal typing
	var cmd tea.Cmd
	a.input, cmd = a.input.Update(msg)
	return a, cmd
}

func (a App) updateSidebarMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "escape":
		a.focus = FocusInput
		a.input.Focus()
		return a, nil
	}
	return a, nil
}

func (a App) executeInput(cmd string) (tea.Model, tea.Cmd) {
	switch a.context {
	case ContextMenu:
		a.output.Append(fmt.Sprintf("gummy ❯ %s\n", cmd))

		switch {
		case cmd == "shell":
			a.output.Append("Shell mode not yet implemented (Phase 3)\n\n")
			return a, nil

		case cmd == "exit" || cmd == "quit" || cmd == "q":
			return a, tea.Quit

		case cmd == "clear" || cmd == "cls":
			a.output.Clear()
			return a, nil

		default:
			output := a.executor.ExecuteCommand(cmd)
			if output != "" {
				a.output.Append(output)
				if !strings.HasSuffix(output, "\n") {
					a.output.Append("\n")
				}
			}
			a.output.Append("\n")
			a.syncSessionInfo()
			return a, nil
		}

	case ContextShell:
		a.output.Append(fmt.Sprintf("$ %s\n", cmd))
		a.output.Append("Shell relay not yet implemented (Phase 3)\n\n")
		return a, nil
	}

	return a, nil
}

func (a *App) syncSessionInfo() {
	if a.executor == nil {
		return
	}
	a.header.SessionCount = a.executor.SessionCount()
	a.input.SetSessionID(a.executor.GetSelectedSessionID())
}

func (a App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Initializing..."
	}

	headerView := a.header.View()
	outputView := a.output.View()
	inputView := a.input.View()
	statusView := a.statusBar.View()

	// Truncate output to exact height (viewport handles width, but we enforce height)
	outputLines := strings.Split(outputView, "\n")
	if len(outputLines) > a.layout.Output.H {
		outputLines = outputLines[len(outputLines)-a.layout.Output.H:]
	}
	for len(outputLines) < a.layout.Output.H {
		outputLines = append(outputLines, "")
	}
	outputView = strings.Join(outputLines, "\n")

	// Truncate input to 1 line
	inputLines := strings.Split(inputView, "\n")
	inputView = inputLines[0]

	if a.layout.IsCompact() {
		return lipgloss.JoinVertical(lipgloss.Left,
			headerView,
			outputView,
			inputView,
			statusView,
		)
	}

	// Build sidebar with exact height
	sidebarContent := a.renderSidebar()
	sidebarLines := strings.Split(sidebarContent, "\n")
	sidebarH := a.layout.Output.H + a.layout.Input.H
	for len(sidebarLines) < sidebarH {
		sidebarLines = append(sidebarLines, "")
	}
	if len(sidebarLines) > sidebarH {
		sidebarLines = sidebarLines[:sidebarH]
	}

	// Pad each line to exact width
	padLine := func(line string, width int) string {
		w := lipgloss.Width(line)
		if w >= width {
			return line
		}
		return line + strings.Repeat(" ", width-w)
	}

	// Build left column (output + input)
	leftLines := append(outputLines, inputView)
	// Build separator
	sepChar := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")

	// Merge columns line by line
	var merged []string
	for i := 0; i < sidebarH; i++ {
		left := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		right := ""
		if i < len(sidebarLines) {
			right = sidebarLines[i]
		}
		merged = append(merged, padLine(left, a.layout.Output.W)+sepChar+padLine(right, a.layout.Sidebar.W))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		headerView,
		strings.Join(merged, "\n"),
		statusView,
	)
}

func (a App) renderSidebar() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("6")).
		Bold(true)

	lines := []string{
		titleStyle.Render(" Sessions"),
		"",
	}

	if a.executor != nil {
		sessionsDisplay := a.executor.GetSessionsForDisplay()
		if sessionsDisplay != "" {
			lines = append(lines, sessionsDisplay)
		}
	}

	return strings.Join(lines, "\n")
}

// Run starts the Bubble Tea TUI program.
func Run(executor CommandExecutor, listenerAddr string) error {
	executor.SetSilent(true)

	app := New(executor, listenerAddr)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	// Wire up background notifications → TUI via program.Send()
	executor.SetNotifyFunc(func(msg string) {
		p.Send(CommandOutputMsg{Output: msg + "\n"})
	})

	_, err := p.Run()

	executor.SetSilent(false)
	executor.SetNotifyFunc(nil)
	return err
}
