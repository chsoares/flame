package tui

// All tea.Msg types for the Gummy TUI.

// --- Context ---

// ContextMode represents whether the user is in shell or menu context.
type ContextMode int

const (
	ContextMenu  ContextMode = iota // Gummy command input
	ContextShell                    // Shell command input to remote
)

// --- Session lifecycle events ---

// SessionConnectedMsg is published when a new reverse shell connects.
type SessionConnectedMsg struct {
	SessionID string
	NumID     int
	RemoteIP  string
}

// SessionInfoDetectedMsg is published after platform/user detection completes.
type SessionInfoDetectedMsg struct {
	SessionID string
	Whoami    string
	Platform  string
}

// SessionDisconnectedMsg is published when a session drops.
type SessionDisconnectedMsg struct {
	SessionID string
	NumID     int
	RemoteIP  string
}

// --- Shell I/O events ---

// ShellOutputMsg carries output data from a remote shell.
type ShellOutputMsg struct {
	SessionID string
	Data      []byte
}

// --- Transfer events ---

// TransferProgressMsg reports file transfer progress.
type TransferProgressMsg struct {
	SessionID  string
	Filename   string
	BytesDone  int64
	BytesTotal int64
	Done       bool
	Err        error
}

// --- Module events ---

// ModuleOutputMsg carries streaming output from a running module.
type ModuleOutputMsg struct {
	SessionID string
	Data      []byte
}

// ModuleFinishedMsg signals module execution completed.
type ModuleFinishedMsg struct {
	SessionID  string
	ModuleName string
	Err        error
}

// --- User actions ---

// SendCommandMsg is dispatched when user presses Enter in shell context.
type SendCommandMsg struct {
	SessionID string
	Command   string
}

// ExecuteGummyMsg is dispatched when user presses Enter in menu context.
type ExecuteGummyMsg struct {
	Command string
}

// SwitchSessionMsg requests switching to a different session.
type SwitchSessionMsg struct {
	NumID int
}

// EnterShellMsg requests entering shell context for the selected session.
type EnterShellMsg struct{}

// ExitShellMsg requests exiting shell context back to menu.
type ExitShellMsg struct{}

// --- Prompt tracking ---

// PromptDetectedMsg carries a detected remote prompt string.
type PromptDetectedMsg struct {
	SessionID string
	Prompt    string
}

// --- Internal ---

// CommandOutputMsg carries output from a gummy command execution.
type CommandOutputMsg struct {
	Output string
}

// clearSelectionMsg signals to clear the selection highlight after a delay.
type clearSelectionMsg struct{}

// showNotifyMsg triggers a notification overlay on the status bar.
type showNotifyMsg struct {
	Message string
	Level   NotifyLevel
}

// clearNotifyMsg dismisses the notification overlay.
type clearNotifyMsg struct {
	id int // Only clear if this matches current notification ID
}

// hideScrollbarMsg signals to hide the scrollbar after inactivity.
type hideScrollbarMsg struct {
	id int // Only hide if this matches current scrollbar ID (prevents stale timers)
}

// shellReadyMsg signals that the shell relay is ready (or failed).
type shellReadyMsg struct {
	err error
}

// spinnerStartMsg starts an animated spinner in the output pane.
type spinnerStartMsg struct {
	ID   int    // Unique spinner ID so we can stop the right one
	Text string // e.g., "Detecting session info..."
}

// spinnerStopMsg stops and removes the spinner from the output pane.
type spinnerStopMsg struct {
	ID int
}

// spinnerTickMsg drives the spinner animation.
type spinnerTickMsg struct {
	ID int
}

// clearQuitPendingMsg resets the double-press quit state after timeout.
type clearQuitPendingMsg struct {
	id int
}

// sendResizeMsg fires after debounce to send stty resize to remote PTY.
type sendResizeMsg struct {
	id   int
	cols int
	rows int
}
