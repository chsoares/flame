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

// clipboardCopiedMsg signals that text was copied to clipboard.
type clipboardCopiedMsg struct {
	Text string
}

// clearSelectionMsg signals to clear the selection highlight after a delay.
type clearSelectionMsg struct{}

// clearStatusMsg signals to clear the status bar message after a delay.
type clearStatusMsg struct{}
