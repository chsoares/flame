package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/charmbracelet/x/ansi"

	internal "github.com/chsoares/gummy/internal"
	"github.com/chsoares/gummy/internal/ui"
)

const (
	selectionClearDelay = 3 * time.Second       // Clear highlight after copy
	notifyDuration      = 2 * time.Second       // Notification overlay duration (important)
	notifyDurationLong  = 4 * time.Second       // Longer duration (info, error)
	scrollbarHideDelay  = 1 * time.Second       // Hide scrollbar after scroll stops
	quitPendingTimeout  = 3 * time.Second       // Double-press Ctrl+D timeout
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
	SetNotifyBarFunc(fn func(string, int))                      // message, level (0=info, 1=important, 2=error)
	SetSpinnerFunc(start func(int, string), stop func(int), update func(int, string))
	SetShellOutputFunc(fn func(string, int, []byte))            // shell relay output (sessionID, numID, data)
	SetSessionDisconnectFunc(fn func(int, string))             // session disconnect (numID, remoteIP)
	StartShellRelay(cols, rows int) error                       // start reading from selected session's conn
	StopShellRelay()                                           // stop relay goroutine
	WriteToShell(data string) error                            // write to selected session's conn
	ResizePTY(cols, rows int)                                  // send stty resize to remote PTY
	CompleteInput(line string) string                          // tab completion
	SetTransferProgressFunc(fn func(string, int, string, bool))                               // transfer progress (filename, pct, right, upload)
	StartUpload(localPath, remotePath string, progressFn func(string), doneFn func(error))   // async upload
	StartDownload(remotePath, localPath string, progressFn func(string), doneFn func(error)) // async download
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

	// Per-session buffers
	menuBuffer     *strings.Builder          // Menu output buffer
	sessionBuffers map[int]*strings.Builder   // Per-session shell output buffers
	activeSession  int                        // NumID of the session currently shown in viewport (0 = menu)

	// Notifications
	notifyID int // Incremented on each notification to invalidate stale clear timers

	// Scrollbar
	scrollbarVisible bool
	scrollbarID      int // Incremented on each scroll to invalidate stale hide timers

	// Quit confirmation
	quitPending   bool
	quitPendingID int
	dialog        *Dialog // Modal overlay (nil = no dialog)

	// PTY resize debounce
	resizeID int // Incremented on each resize to invalidate stale debounce timers

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
		header:         NewHeader(listenerAddr),
		output:         &output,
		input:          &input,
		statusBar:      NewStatusBar(80),
		splash:         true,
		focus:          FocusInput,
		context:        ContextMenu,
		menuBuffer:     &strings.Builder{},
		sessionBuffers: make(map[int]*strings.Builder),
		executor:       executor,
		listenerAddr:   listenerAddr,
		cwd:            cwd,
		sessionName:    sessionName,
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
		// Debounce remote PTY resize — only send stty after 150ms of no resizes
		if a.context == ContextShell && a.activeSession > 0 {
			a.resizeID++
			id := a.resizeID
			cols, rows := a.layout.Output.W, a.layout.Output.H
			return a, tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
				return sendResizeMsg{id: id, cols: cols, rows: rows}
			})
		}
		return a, nil

	case tea.KeyMsg:
		// Modal dialog intercepts all keys
		if a.dialog != nil {
			return a.updateDialog(msg)
		}
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
				return a.tryQuitCtrlD()
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
		if a.splash {
			a.splash = false
		}
		// Always accumulate in menu buffer
		a.menuBuffer.WriteString(msg.Output)
		if a.activeSession == 0 {
			// Currently viewing menu — update viewport
			a.output.Append(msg.Output)
		}
		// If viewing a session, menu buffer accumulates silently.
		// Notification bar handles important events.
		a.syncSessionInfo()
		return a, nil

	case tea.MouseMsg:
		if a.dialog != nil {
			return a, nil // Block mouse when dialog is open
		}
		return a.handleMouse(msg)

	case showNotifyMsg:
		a.notifyID++
		id := a.notifyID
		a.statusBar.Notify = &Notification{Message: msg.Message, Level: msg.Level}
		duration := notifyDuration
		if msg.Level == NotifyImportant || msg.Level == NotifyError {
			duration = notifyDurationLong
		}
		return a, tea.Tick(duration, func(time.Time) tea.Msg {
			return clearNotifyMsg{id: id}
		})

	case clearNotifyMsg:
		if msg.id == a.notifyID {
			a.statusBar.Notify = nil
		}
		return a, nil

	case clearSelectionMsg:
		a.output.ClearSelection()
		return a, nil

	case clearQuitPendingMsg:
		if msg.id == a.quitPendingID {
			a.quitPending = false
		}
		return a, nil

	case transferProgressMsg:
		a.statusBar.TransferPct = msg.Pct
		a.statusBar.TransferMsg = msg.Filename
		a.statusBar.TransferRight = msg.Right
		a.statusBar.TransferUpload = msg.Upload
		return a, nil

	case transferDoneMsg:
		// Clear progress bar
		a.statusBar.TransferPct = -1
		a.statusBar.TransferMsg = ""
		a.statusBar.TransferRight = ""
		action := "Download"
		if msg.Upload {
			action = "Upload"
		}
		if msg.Err != nil {
			a.menuAppend(ui.Error(fmt.Sprintf("%s failed: %v", action, msg.Err)) + "\n\n")
			return a, func() tea.Msg {
				return showNotifyMsg{
					Message: fmt.Sprintf("%s failed: %s", action, msg.Filename),
					Level:   NotifyError,
				}
			}
		}
		a.menuAppend(ui.Success(fmt.Sprintf("%s complete: %s", action, msg.Filename)) + "\n\n")
		return a, func() tea.Msg {
			return showNotifyMsg{
				Message: fmt.Sprintf("%s complete: %s", action, msg.Filename),
				Level:   NotifyInfo,
			}
		}

	case sendResizeMsg:
		if msg.id != a.resizeID {
			return a, nil // Stale — a newer resize superseded this one
		}
		a.executor.ResizePTY(msg.cols, msg.rows)
		return a, nil

	case hideScrollbarMsg:
		if msg.id == a.scrollbarID {
			a.scrollbarVisible = false
		}
		return a, nil

	case spinnerStartMsg:
		if a.context == ContextShell {
			// Don't show viewport spinner in shell mode — use notification bar
			return a, nil
		}
		a.output.StartSpinner(msg.ID, msg.Text)
		return a, tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
			return spinnerTickMsg{ID: msg.ID}
		})

	case spinnerTickMsg:
		if !a.output.spinnerActive || a.output.spinnerID != msg.ID {
			return a, nil
		}
		a.output.TickSpinner(msg.ID)
		return a, tea.Tick(80*time.Millisecond, func(time.Time) tea.Msg {
			return spinnerTickMsg{ID: msg.ID}
		})

	case spinnerStopMsg:
		a.output.StopSpinner(msg.ID)
		return a, nil

	case spinnerUpdateMsg:
		a.output.UpdateSpinner(msg.ID, msg.Text)
		return a, nil

	case shellReadyMsg:
		if msg.err != nil {
			a.menuAppend(styleRed.Render("  "+msg.err.Error()) + "\n\n")
			return a, nil
		}
		// Switch context and viewport to session buffer
		selectedID := a.executor.GetSelectedSessionID()

		// Append entering message with session number to menu buffer
		a.menuAppend(ui.Info(fmt.Sprintf("Entering interactive shell #%d", selectedID)) + "\n")
		a.menuAppend(ui.CommandHelp("Press F12 to detach") + "\n")
		a.context = ContextShell
		a.header.Context = ContextShell
		a.statusBar.Context = ContextShell
		a.input.SetContext(ContextShell)
		a.switchToSession(selectedID)

		// Trigger first prompt display
		a.executor.WriteToShell("\n")
		return a, func() tea.Msg {
			return showNotifyMsg{
				Message: fmt.Sprintf("Attached to shell #%d — Press F12 to detach", selectedID),
				Level:   NotifyInfo,
			}
		}

	case ShellOutputMsg:
		text := sanitizeShellOutput(string(msg.Data))
		// Always accumulate in session buffer
		a.appendToSessionBuffer(msg.NumID, text)
		// Only update viewport if this session is active
		if a.activeSession == msg.NumID {
			a.output.Append(text)
		}
		return a, nil

	case SessionDisconnectedMsg:
		if a.context == ContextShell && a.activeSession == msg.NumID {
			a.context = ContextMenu
			a.header.Context = ContextMenu
			a.statusBar.Context = ContextMenu
			a.input.SetContext(ContextMenu)
			a.switchToMenu()
			a.menuAppend("\n" + styleRed.Render("--- Session disconnected ---") + "\n\n")
		}
		a.syncSessionInfo()
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
						func() tea.Msg {
							return showNotifyMsg{Message: "Selected text copied to clipboard", Level: NotifyInfo}
						},
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

// tryQuitCtrlD implements double-press Ctrl+D with warning notification.
func (a App) tryQuitCtrlD() (tea.Model, tea.Cmd) {
	if a.executor.SessionCount() == 0 {
		return a, tea.Quit
	}
	if a.quitPending {
		return a, tea.Quit
	}
	a.quitPending = true
	a.quitPendingID++
	id := a.quitPendingID
	return a, tea.Batch(
		func() tea.Msg {
			return showNotifyMsg{
				Message: "Active sessions! Press Ctrl+D again to quit",
				Level:   NotifyError,
			}
		},
		tea.Tick(quitPendingTimeout, func(time.Time) tea.Msg {
			return clearQuitPendingMsg{id: id}
		}),
	)
}

// updateDialog handles key events when a modal dialog is active.
func (a App) updateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab", "left", "right", "h", "l":
		a.dialog.Toggle()
		return a, nil
	case "enter":
		if a.dialog.Selected == 0 {
			// Confirmed
			switch a.dialog.Action {
			case DialogQuit:
				return a, tea.Quit
			}
		}
		a.dialog = nil
		return a, nil
	case "escape", "n":
		a.dialog = nil
		return a, nil
	case "y":
		return a, tea.Quit
	}
	return a, nil
}

func (a App) updateInputMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		if a.context == ContextShell {
			a.executor.WriteToShell("\x03")
			return a, nil
		}
		return a, nil

	case "ctrl+d":
		return a.tryQuitCtrlD()

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
		if a.context == ContextMenu {
			// Tab completion in menu mode
			current := a.input.Value()
			completed := a.executor.CompleteInput(current)
			if completed != current {
				a.input.SetValue(completed)
			}
		}
		return a, nil

	case "f12":
		if a.context == ContextShell {
			a.context = ContextMenu
			a.header.Context = ContextMenu
			a.statusBar.Context = ContextMenu
			a.input.SetContext(ContextMenu)
			// Switch viewport back to menu buffer
			a.switchToMenu()
			a.menuAppend("\n" + ui.Info("Exiting interactive shell") + "\n\n")
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
		a.menuAppend(prompt + styleBase.Render(cmd) + "\n")

		switch {
		case cmd == "shell":
			if a.executor.GetSelectedSessionID() == 0 {
				a.menuAppend(styleMuted.Render("  No session selected. Use 'use <id>' first") + "\n\n")
				return a, nil
			}
			// Start relay async (PTY upgrade blocks ~1s)
			cols, rows := a.layout.Output.W, a.layout.Output.H
			return a, func() tea.Msg {
				err := a.executor.StartShellRelay(cols, rows)
				return shellReadyMsg{err: err}
			}

		case cmd == "exit" || cmd == "quit" || cmd == "q":
			if a.executor.SessionCount() == 0 {
				return a, tea.Quit
			}
			a.dialog = confirmQuitDialog(a.executor.SessionCount())
			return a, nil

		case cmd == "clear" || cmd == "cls":
			a.menuBuffer.Reset()
			a.output.Clear()
			return a, nil

		case strings.HasPrefix(cmd, "upload "):
			return a.handleUploadCmd(cmd)

		case strings.HasPrefix(cmd, "download "):
			return a.handleDownloadCmd(cmd)

		default:
			output := a.executor.ExecuteCommand(cmd)
			if output != "" {
				a.menuAppend(output)
				if !strings.HasSuffix(output, "\n") {
					a.menuAppend("\n")
				}
			}
			a.menuAppend("\n")
			a.syncSessionInfo()

			// Fire notification for kill commands (can't p.Send from inside ExecuteCommand)
			if strings.HasPrefix(cmd, "kill ") && output != "" && !strings.Contains(output, "not found") && !strings.Contains(output, "Error") {
				// Strip ANSI + leading icon from the cli output to get clean text
				notifyMsg := stripANSI(strings.TrimSpace(output))
				// Remove leading nerd font icon (multi-byte char + space)
				if idx := strings.Index(notifyMsg, " "); idx >= 0 && idx <= 4 {
					notifyMsg = strings.TrimSpace(notifyMsg[idx:])
				}
				return a, func() tea.Msg {
					return showNotifyMsg{
						Message: notifyMsg,
						Level:   NotifyError,
					}
				}
			}

			return a, nil
		}

	case ContextShell:
		if err := a.executor.WriteToShell(cmd + "\n"); err != nil {
			a.output.Append(styleRed.Render("  Write error: "+err.Error()) + "\n\n")
			a.context = ContextMenu
			a.header.Context = ContextMenu
			a.statusBar.Context = ContextMenu
			a.input.SetContext(ContextMenu)
		}
		return a, nil
	}

	return a, nil
}

// sanitizeShellOutput normalizes line endings and strips dangerous terminal
// control sequences from remote shell output. Keeps colors/styles but removes
// cursor movement, screen clear, alt screen, etc. that would corrupt the TUI.
// Also filters out stty resize commands that leak from PTY resize operations.
func sanitizeShellOutput(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "")

	// Strip dangerous CSI sequences (cursor movement, screen clear, scroll regions)
	// Keep color/style sequences (SGR: ends with 'm')
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) {
			if s[i+1] == '[' {
				// CSI sequence: \033[ ... <final byte>
				j := i + 2
				for j < len(s) && s[j] >= 0x20 && s[j] <= 0x3F {
					j++ // skip parameter bytes (0-9, ;, etc.)
				}
				if j < len(s) {
					finalByte := s[j]
					if finalByte == 'm' {
						// SGR (color/style) — keep it
						result = append(result, s[i:j+1]...)
					}
					// All other CSI sequences (A,B,C,D,H,J,K,L,M,S,T, etc.) — drop
					i = j + 1
					continue
				}
			} else if s[i+1] == ']' {
				// OSC sequence: \033] ... ST — drop entirely (title changes etc.)
				j := i + 2
				for j < len(s) {
					if s[j] == '\033' && j+1 < len(s) && s[j+1] == '\\' {
						j += 2
						break
					}
					if s[j] == '\007' { // BEL also terminates OSC
						j++
						break
					}
					j++
				}
				i = j
				continue
			}
			// Other escape sequences — drop the \033 and next byte
			i += 2
			continue
		}
		result = append(result, s[i])
		i++
	}
	return string(result)
}

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	var result []byte
	i := 0
	for i < len(s) {
		if s[i] == '\033' && i+1 < len(s) && s[i+1] == '[' {
			// Skip until we find a letter
			j := i + 2
			for j < len(s) && !((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				j++
			}
			i = j + 1
		} else {
			result = append(result, s[i])
			i++
		}
	}
	return string(result)
}

// truncateLine truncates a line (which may contain ANSI codes) to maxWidth display columns.
// Uses ansi.Truncate for proper handling of escape sequences.
func truncateLine(line string, maxWidth int) string {
	return ansi.Truncate(line, maxWidth, "")
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

// menuAppend appends text to the menu buffer and, if menu is active, also to the viewport.
func (a *App) menuAppend(text string) {
	a.menuBuffer.WriteString(text)
	if a.activeSession == 0 {
		a.output.Append(text)
	}
}

// expandTilde replaces a leading ~ with the user's home directory.
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[1:])
		}
	}
	return path
}

// handleUploadCmd parses and launches an async upload.
func (a App) handleUploadCmd(cmd string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		a.menuAppend(ui.CommandHelp("Usage: upload <local_path> [remote_path]") + "\n\n")
		return a, nil
	}
	localPath := expandTilde(parts[1])
	remotePath := ""
	if len(parts) >= 3 {
		remotePath = parts[2]
	}

	if a.executor.GetSelectedSessionID() == 0 {
		a.menuAppend(ui.Error("No session selected. Use 'use <id>' first") + "\n\n")
		return a, nil
	}

	filename := filepath.Base(localPath)

	// Launch async — spinner is managed by Manager.StartUpload
	return a, func() tea.Msg {
		done := make(chan error, 1)
		a.executor.StartUpload(localPath, remotePath, nil, func(err error) { done <- err })
		return transferDoneMsg{Err: <-done, Filename: filename, Upload: true}
	}
}

// handleDownloadCmd parses and launches an async download.
func (a App) handleDownloadCmd(cmd string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(cmd)
	if len(parts) < 2 {
		a.menuAppend(ui.CommandHelp("Usage: download <remote_path> [local_path]") + "\n\n")
		return a, nil
	}
	remotePath := parts[1]
	localPath := ""
	if len(parts) >= 3 {
		localPath = expandTilde(parts[2])
	}

	if a.executor.GetSelectedSessionID() == 0 {
		a.menuAppend(ui.Error("No session selected. Use 'use <id>' first") + "\n\n")
		return a, nil
	}

	filename := filepath.Base(remotePath)

	// Launch async — spinner is managed by Manager.StartDownload
	return a, func() tea.Msg {
		done := make(chan error, 1)
		a.executor.StartDownload(remotePath, localPath, nil, func(err error) { done <- err })
		return transferDoneMsg{Err: <-done, Filename: filename, Upload: false}
	}
}

// saveViewportToBuffer saves the current viewport content to the active buffer.
func (a *App) saveViewportToBuffer() {
	content := a.output.GetContent()
	if a.activeSession == 0 {
		a.menuBuffer.Reset()
		a.menuBuffer.WriteString(content)
	} else {
		buf := a.getSessionBuffer(a.activeSession)
		buf.Reset()
		buf.WriteString(content)
	}
}

// switchToMenu saves the current buffer and loads the menu buffer into viewport.
func (a *App) switchToMenu() {
	a.saveViewportToBuffer()
	a.activeSession = 0
	a.output.SetContent(a.menuBuffer.String())
}

// switchToSession saves the current buffer and loads a session buffer into viewport.
func (a *App) switchToSession(numID int) {
	a.saveViewportToBuffer()
	a.activeSession = numID
	buf := a.getSessionBuffer(numID)
	a.output.SetContent(buf.String())
}

// getSessionBuffer returns the buffer for a session, creating it if needed.
func (a *App) getSessionBuffer(numID int) *strings.Builder {
	if buf, ok := a.sessionBuffers[numID]; ok {
		return buf
	}
	buf := &strings.Builder{}
	a.sessionBuffers[numID] = buf
	return buf
}

// appendToSessionBuffer appends text to a session's buffer without affecting the viewport.
func (a *App) appendToSessionBuffer(numID int, text string) {
	buf := a.getSessionBuffer(numID)
	buf.WriteString(text)
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

	// Truncate output to exact height and width
	outputLines := strings.Split(outputView, "\n")
	if len(outputLines) > a.layout.Output.H {
		outputLines = outputLines[len(outputLines)-a.layout.Output.H:]
	}
	for len(outputLines) < a.layout.Output.H {
		outputLines = append(outputLines, "")
	}
	// Clamp each line to output width to prevent bleeding into sidebar on resize
	for i, line := range outputLines {
		if lipgloss.Width(line) > a.layout.Output.W {
			outputLines[i] = truncateLine(line, a.layout.Output.W)
		}
	}

	// Truncate input to 1 line and clamp width
	inputLines := strings.Split(inputView, "\n")
	inputView = inputLines[0]
	if lipgloss.Width(inputView) > a.layout.Input.W {
		inputView = truncateLine(inputView, a.layout.Input.W)
	}

	// Scrollbar thumb (shared by compact and wide)
	thumbStart, thumbEnd := -1, -1
	if a.scrollbarVisible {
		thumbStart, thumbEnd = a.output.ScrollbarThumb()
	}
	scrollThumb := lipgloss.NewStyle().Foreground(colorDim).Render("▐")
	scrollChar := func(_ int) string { return scrollThumb }

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
				outputLines[i] += scrollChar(i)
			}
		}

		headerView := a.header.View()
		result := lipgloss.JoinVertical(lipgloss.Left,
			headerView,
			strings.Join(outputLines, "\n"),
			"",
			inputView,
			"",
			statusView,
		)
		result = a.padViewLines(result)
		if a.dialog != nil {
			return a.dialog.View(a.width, a.height, result)
		}
		return result
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
			gapStr = " " + scrollChar(i) + strings.Repeat(" ", gap-2)
		} else {
			gapStr = strings.Repeat(" ", gap)
		}

		merged = append(merged, padLine(left, a.layout.Output.W)+gapStr+padLine(right, a.layout.Sidebar.W))
	}

	result := lipgloss.JoinVertical(lipgloss.Left,
		strings.Join(merged, "\n"),
		statusView,
	)
	result = a.padViewLines(result)
	if a.dialog != nil {
		return a.dialog.View(a.width, a.height, result)
	}
	return result
}

// padViewLines ensures every line is padded to exactly a.width display columns.
// This prevents rendering artifacts when the terminal shrinks (old longer lines
// would otherwise remain visible since the new shorter lines don't overwrite them).
func (a App) padViewLines(view string) string {
	lines := strings.Split(view, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < a.width {
			lines[i] = line + strings.Repeat(" ", a.width-w)
		} else if w > a.width {
			lines[i] = truncateLine(line, a.width)
		}
	}
	return strings.Join(lines, "\n")
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

	result := strings.Join(content, "\n") + "\n\n" + inputView + "\n\n" + statusView
	return a.padViewLines(result)
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
	executor.SetNotifyBarFunc(func(msg string, level int) {
		p.Send(showNotifyMsg{Message: msg, Level: NotifyLevel(level)})
	})
	executor.SetSpinnerFunc(
		func(id int, text string) { p.Send(spinnerStartMsg{ID: id, Text: text}) },
		func(id int) { p.Send(spinnerStopMsg{ID: id}) },
		func(id int, text string) { p.Send(spinnerUpdateMsg{ID: id, Text: text}) },
	)
	executor.SetTransferProgressFunc(func(filename string, pct int, right string, upload bool) {
		p.Send(transferProgressMsg{Filename: filename, Pct: pct, Right: right, Upload: upload})
	})
	executor.SetShellOutputFunc(func(sessionID string, numID int, data []byte) {
		p.Send(ShellOutputMsg{SessionID: sessionID, NumID: numID, Data: data})
	})
	executor.SetSessionDisconnectFunc(func(numID int, remoteIP string) {
		p.Send(SessionDisconnectedMsg{NumID: numID, RemoteIP: remoteIP})
	})

	_, err := p.Run()

	executor.StopShellRelay()
	executor.SetSilent(false)
	executor.SetNotifyFunc(nil)
	executor.SetNotifyBarFunc(nil)
	executor.SetSpinnerFunc(nil, nil, nil)
	executor.SetTransferProgressFunc(nil)
	executor.SetShellOutputFunc(nil)
	executor.SetSessionDisconnectFunc(nil)
	return err
}
