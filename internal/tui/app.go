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
}

// App is the root Bubble Tea model for the Gummy TUI.
type App struct {
	// Layout
	layout Layout
	width  int
	height int

	// Components
	header    Header
	output    OutputPane
	input     Input
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
	return App{
		header:       NewHeader(listenerAddr),
		output:       NewOutputPane(80, 20),
		input:        NewInput(),
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
	}

	// Forward to textinput for normal typing
	var cmd tea.Cmd
	inputPtr, cmd := a.input.Update(msg)
	a.input = *inputPtr
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

	a.syncSessionInfo()

	headerView := a.header.View()
	outputView := a.output.View()
	inputView := a.input.View()
	statusView := a.statusBar.View()

	outputStyle := lipgloss.NewStyle().
		Width(a.layout.Output.W).
		Height(a.layout.Output.H)

	inputStyle := lipgloss.NewStyle().
		Width(a.layout.Input.W).
		Height(a.layout.Input.H)

	var contentRows []string

	if a.layout.IsCompact() {
		contentRows = []string{
			outputStyle.Render(outputView),
			inputStyle.Render(inputView),
		}
	} else {
		sidebarContent := a.renderSidebar()
		sidebarStyle := lipgloss.NewStyle().
			Width(a.layout.Sidebar.W).
			Height(a.layout.Sidebar.H)

		sep := strings.Repeat(
			lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")+"\n",
			a.layout.Sidebar.H,
		)
		sep = strings.TrimRight(sep, "\n")

		outputBlock := lipgloss.JoinVertical(lipgloss.Left,
			outputStyle.Render(outputView),
			inputStyle.Render(inputView),
		)
		sidebarBlock := sidebarStyle.Render(sidebarContent)

		contentRow := lipgloss.JoinHorizontal(lipgloss.Top,
			outputBlock,
			sep,
			sidebarBlock,
		)
		contentRows = []string{contentRow}
	}

	parts := []string{headerView}
	parts = append(parts, contentRows...)
	parts = append(parts, statusView)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
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
	app := New(executor, listenerAddr)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
