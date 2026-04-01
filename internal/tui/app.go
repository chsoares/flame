package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	internal "github.com/chsoares/gummy/internal"
)

const (
	selectionClearDelay = 3 * time.Second // Clear highlight after copy
	statusClearDelay    = 2 * time.Second // Clear status bar message
	scrollbarHideDelay  = 1 * time.Second // Hide scrollbar after scroll stops
)

// CommandExecutor is the interface the Manager must satisfy for the TUI.
type CommandExecutor interface {
	ExecuteCommand(cmd string) string
	GetSelectedSessionID() int
	SessionCount() int
	GetSessionsForDisplay() string
	GetActiveSessionDisplay() (ip, whoami, platform string, ok bool)
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
	splash  bool // Show splash screen before first input
	focus   FocusMode
	context ContextMode

	// Scrollbar
	scrollbarVisible bool
	scrollbarID      int // Incremented on each scroll to invalidate stale hide timers

	// Backend
	executor CommandExecutor

	// Config
	listenerAddr string
	cwd          string
	sessionName  string // date-based session identifier
}

// New creates a new App model.
func New(executor CommandExecutor, listenerAddr string) App {
	output := NewOutputPane(80, 20)
	input := NewInput()

	cwd, _ := os.Getwd()
	sessionName := time.Now().Format("2006_01_02")

	return App{
		header:       NewHeader(listenerAddr),
		output:       &output,
		input:        &input,
		statusBar:    NewStatusBar(80),
		splash:       true,
		focus:        FocusInput,
		context:      ContextMenu,
		executor:     executor,
		listenerAddr: listenerAddr,
		cwd:          cwd,
		sessionName:  sessionName,
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
		if a.splash {
			if msg.String() == "enter" {
				cmd := a.input.Submit()
				a.splash = false
				if cmd != "" {
					return a.executeInput(cmd)
				}
				return a, nil
			}
			// Forward other keys to input (typing works during splash)
			if msg.String() == "ctrl+d" {
				return a, tea.Quit
			}
			var cmd tea.Cmd
			a.input, cmd = a.input.Update(msg)
			return a, cmd
		}
		switch a.focus {
		case FocusInput:
			return a.updateInputMode(msg)
		case FocusSidebar:
			return a.updateSidebarMode(msg)
		}

	case CommandOutputMsg:
		a.output.Append(msg.Output)
		a.syncSessionInfo()
		return a, nil

	case tea.MouseMsg:
		return a.handleMouse(msg)

	case clipboardCopiedMsg:
		a.statusBar.StatusMsg = "Copied to clipboard"
		return a, tea.Tick(statusClearDelay, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearSelectionMsg:
		a.output.ClearSelection()
		return a, nil

	case clearStatusMsg:
		a.statusBar.StatusMsg = ""
		return a, nil

	case hideScrollbarMsg:
		if msg.id == a.scrollbarID {
			a.scrollbarVisible = false
		}
		return a, nil
	}

	return a, nil
}

// handleMouse routes mouse events to the appropriate component.
func (a App) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	ox, oy := a.layout.Output.X, a.layout.Output.Y
	ow, oh := a.layout.Output.W, a.layout.Output.H

	// Translate to output-pane-relative coordinates
	viewX := msg.X - ox
	viewY := msg.Y - oy

	inOutput := viewX >= 0 && viewX < ow && viewY >= 0 && viewY < oh

	switch {
	case msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown:
		// Scroll: always forward to viewport (even if mouse is slightly outside)
		var cmd tea.Cmd
		a.output, cmd = a.output.Update(msg)
		hideCmd := a.showScrollbar()
		return a, tea.Batch(cmd, hideCmd)

	case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
		if !inOutput {
			// Click outside output pane clears selection
			a.output.ClearSelection()
			return a, nil
		}
		a.output.HandleMouseDown(viewX, viewY)
		return a, nil

	case msg.Action == tea.MouseActionMotion:
		if a.output.selection.Active {
			// Clamp to output pane bounds
			if viewX < 0 {
				viewX = 0
			}
			if viewX >= ow {
				viewX = ow - 1
			}
			if viewY < 0 {
				viewY = 0
			}
			if viewY >= oh {
				viewY = oh - 1
			}
			a.output.HandleMouseMotion(viewX, viewY)
		}
		return a, nil

	case msg.Action == tea.MouseActionRelease:
		if a.output.selection.Active {
			// Clamp to output pane bounds
			if viewX < 0 {
				viewX = 0
			}
			if viewX >= ow {
				viewX = ow - 1
			}
			if viewY < 0 {
				viewY = 0
			}
			if viewY >= oh {
				viewY = oh - 1
			}
			if a.output.HandleMouseUp(viewX, viewY) {
				// Selection made — copy and schedule cleanup
				text := a.output.CopySelection()
				if text != "" {
					return a, tea.Batch(
						func() tea.Msg { return clipboardCopiedMsg{Text: text} },
						tea.Tick(selectionClearDelay, func(time.Time) tea.Msg {
							return clearSelectionMsg{}
						}),
					)
				}
			}
		}
		return a, nil
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
			a.output.Append("\n" + styleMuted.Render("--- Detached from shell ---") + "\n\n")
			return a, nil
		}

	case "pgup", "pgdown", "home", "end":
		var cmd tea.Cmd
		a.output, cmd = a.output.Update(msg)
		hideCmd := a.showScrollbar()
		return a, tea.Batch(cmd, hideCmd)
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
		// Echo command with styled prompt
		prompt := styleMagentaBold.Render("❯") + " "
		a.output.Append(prompt + styleBase.Render(cmd) + "\n")

		switch {
		case cmd == "shell":
			a.output.Append(styleMuted.Render("Shell mode not yet implemented (Phase 3)") + "\n\n")
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
		a.output.Append(styleCyan.Render("$") + " " + cmd + "\n")
		a.output.Append(styleMuted.Render("Shell relay not yet implemented (Phase 3)") + "\n\n")
		return a, nil
	}

	return a, nil
}

// truncateLine truncates a line (which may contain ANSI codes) to maxWidth display columns.
func truncateLine(line string, maxWidth int) string {
	if lipgloss.Width(line) <= maxWidth {
		return line
	}
	// Walk rune by rune, tracking display width
	currentWidth := 0
	for i, r := range line {
		rw := lipgloss.Width(string(r))
		if currentWidth+rw > maxWidth {
			return line[:i]
		}
		currentWidth += rw
	}
	return line
}

// showScrollbar makes the scrollbar visible and returns a cmd to hide it after delay.
func (a *App) showScrollbar() tea.Cmd {
	a.scrollbarVisible = true
	a.scrollbarID++
	id := a.scrollbarID
	return tea.Tick(scrollbarHideDelay, func(time.Time) tea.Msg {
		return hideScrollbarMsg{id: id}
	})
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

	if a.splash {
		return a.viewSplash()
	}

	outputView := a.output.View()
	inputView := a.input.View()
	statusView := a.statusBar.View()

	// Truncate output to exact height
	outputLines := strings.Split(outputView, "\n")
	if len(outputLines) > a.layout.Output.H {
		outputLines = outputLines[len(outputLines)-a.layout.Output.H:]
	}
	for len(outputLines) < a.layout.Output.H {
		outputLines = append(outputLines, "")
	}

	// Truncate input to 1 line
	inputLines := strings.Split(inputView, "\n")
	inputView = inputLines[0]

	// Scrollbar thumb (shared by compact and wide)
	thumbStart, thumbEnd := -1, -1
	if a.scrollbarVisible {
		thumbStart, thumbEnd = a.output.ScrollbarThumb()
	}
	scrollThumb := lipgloss.NewStyle().Foreground(colorDim).Render("█")

	if a.layout.IsCompact() {
		// In compact mode, overlay scrollbar on last column of output lines
		if thumbStart >= 0 {
			for i := thumbStart; i < thumbEnd && i < len(outputLines); i++ {
				// Truncate line to width-1 then append thumb
				outputLines[i] = truncateLine(outputLines[i], a.layout.Output.W-1)
				lineW := lipgloss.Width(outputLines[i])
				if lineW < a.layout.Output.W-1 {
					outputLines[i] += strings.Repeat(" ", a.layout.Output.W-1-lineW)
				}
				outputLines[i] += scrollThumb
			}
		}

		headerView := a.header.View()
		return lipgloss.JoinVertical(lipgloss.Left,
			headerView,
			strings.Join(outputLines, "\n"),
			"",
			inputView,
			"",
			statusView,
		)
	}

	// Wide mode: sidebar with branding (no header bar)
	sidebarContent := a.renderSidebar()
	sidebarLines := strings.Split(sidebarContent, "\n")
	sidebarH := a.layout.Sidebar.H
	for len(sidebarLines) < sidebarH {
		sidebarLines = append(sidebarLines, "")
	}
	if len(sidebarLines) > sidebarH {
		sidebarLines = sidebarLines[:sidebarH]
	}

	// Pad line to exact width
	padLine := func(line string, width int) string {
		w := lipgloss.Width(line)
		if w >= width {
			return line
		}
		return line + strings.Repeat(" ", width-w)
	}

	// Build left column: output + blank + input + blank
	leftLines := append(outputLines, "", inputView, "")

	// Gap between main and sidebar
	gap := a.layout.Sidebar.X - a.layout.Output.W
	if gap < 1 {
		gap = 1
	}

	// Merge columns line by line
	totalH := len(leftLines)
	if sidebarH > totalH {
		totalH = sidebarH
	}
	var merged []string
	for i := 0; i < totalH; i++ {
		left := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		right := ""
		if i < len(sidebarLines) {
			right = sidebarLines[i]
		}

		// Build gap: scrollbar in the middle of the gap for output lines
		var gapStr string
		if i < len(outputLines) && thumbStart >= 0 && i >= thumbStart && i < thumbEnd {
			// Place scrollbar: 1 space + thumb + remaining spaces
			gapStr = " " + scrollThumb + strings.Repeat(" ", gap-2)
		} else {
			gapStr = strings.Repeat(" ", gap)
		}

		merged = append(merged, padLine(left, a.layout.Output.W)+gapStr+padLine(right, a.layout.Sidebar.W))
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		strings.Join(merged, "\n"),
		statusView,
	)
}

// viewSplash renders the splash/landing screen shown before first Enter.
// Input bar is active and functional. Banner has hatching only on the sides.
func (a App) viewSplash() string {
	banner := renderBannerSplash(a.width)
	inputView := strings.Split(a.input.View(), "\n")[0]
	statusView := a.statusBar.View()

	// Info lines below the banner
	info := []string{
		"",
		"  " + styleMuted.Render("Listening for connections on ") + styleCyan.Render(a.listenerAddr),
		"",
		"  " + styleSubtle.Render("Type 'help' for available commands"),
	}

	// Build content area: top padding + banner + info
	bannerLines := strings.Split(banner, "\n")
	var content []string

	// Push banner down ~1/3 of the screen
	topPad := 4
	for i := 0; i < topPad; i++ {
		content = append(content, "")
	}

	content = append(content, bannerLines...)
	content = append(content, info...)

	// Fill remaining space (height - blank - input - blank - status)
	contentH := a.height - 1 - 1 - 1 - 1 // blank + input + blank + status
	for len(content) < contentH {
		content = append(content, "")
	}
	if len(content) > contentH {
		content = content[:contentH]
	}

	return strings.Join(content, "\n") + "\n\n" + inputView + "\n\n" + statusView
}

// renderSidebar builds the sidebar content with adaptive branding.
func (a App) renderSidebar() string {
	w := a.layout.Sidebar.W
	if w <= 0 {
		return ""
	}

	var lines []string

	// --- Banner (adaptive based on layout mode) ---
	if a.layout.Mode == LayoutFull {
		banner := renderBannerFull(w)
		lines = append(lines, strings.Split(banner, "\n")...)
	} else {
		lines = append(lines, renderBannerCompact(w))
	}

	lines = append(lines, "")

	// --- Info: session name, cwd, listener ---
	lines = append(lines, styleMuted.Render(" "+a.sessionName))

	// Pretty CWD: replace home dir with ~
	prettyPath := a.cwd
	if home, err := os.UserHomeDir(); err == nil {
		if rel, err := filepath.Rel(home, a.cwd); err == nil && !strings.HasPrefix(rel, "..") {
			prettyPath = "~/" + rel
		}
	}
	lines = append(lines, styleMuted.Render(" "+prettyPath))

	lines = append(lines, "")

	// Listener
	lines = append(lines, " "+styleBase.Render("\uf095")+" "+styleMuted.Render(a.listenerAddr))

	lines = append(lines, "")

	// Binbag status
	binbagIcon := "\U000f059f" // nf-md-web 󰖟
	if internal.GlobalRuntimeConfig != nil && internal.GlobalRuntimeConfig.BinbagEnabled {
		prettyBinbag := internal.GlobalRuntimeConfig.BinbagPath
		if home, err := os.UserHomeDir(); err == nil {
			if rel, err := filepath.Rel(home, prettyBinbag); err == nil && !strings.HasPrefix(rel, "..") {
				prettyBinbag = "~/" + rel
			}
		}
		lines = append(lines, " "+styleBase.Render(binbagIcon)+" "+styleMuted.Render("binbag online"))
		lines = append(lines, "   "+styleSubtle.Render(prettyBinbag))
		lines = append(lines, "   "+styleSubtle.Render(fmt.Sprintf(":%d", internal.GlobalRuntimeConfig.HTTPPort)))
	} else {
		lines = append(lines, " "+styleSubtle.Render(binbagIcon)+" "+styleMuted.Render("binbag offline"))
	}

	lines = append(lines, "")

	// Session count
	sessionCount := 0
	if a.executor != nil {
		sessionCount = a.executor.SessionCount()
	}
	sessWord := "sessions"
	if sessionCount == 1 {
		sessWord = "session"
	}
	countStyle := styleBase
	if sessionCount == 0 {
		countStyle = styleSubtle
	}
	lines = append(lines, " "+countStyle.Render(fmt.Sprintf("%d", sessionCount))+" "+styleMuted.Render(sessWord))

	lines = append(lines, "")

	// --- Active session section ---
	lines = append(lines, sectionHeader("Active", w))

	if ip, whoami, platform, ok := a.executor.GetActiveSessionDisplay(); ok {
		// Platform icon
		platIcon := ""
		switch platform {
		case "linux":
			platIcon = " "
		case "windows":
			platIcon = " "
		default:
			platIcon = " "
		}

		// Name: whoami as fallback (future: user-set description)
		name := whoami
		if name == "" {
			name = ip
		}
		lines = append(lines, " "+styleCyan.Render(platIcon)+styleBase.Render(name))
		lines = append(lines, "  "+styleMuted.Render(ip))
		lines = append(lines, "  "+styleMuted.Render(platform))
	} else {
		lines = append(lines, styleSubtle.Render("  None"))
	}

	lines = append(lines, "")

	// --- Sessions section ---
	lines = append(lines, sectionHeader("Sessions", w))

	if a.executor != nil && sessionCount > 0 {
		sessionsDisplay := a.executor.GetSessionsForDisplay()
		for _, sl := range strings.Split(sessionsDisplay, "\n") {
			lines = append(lines, " "+sl)
		}
	} else {
		lines = append(lines, styleSubtle.Render("  None"))
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
