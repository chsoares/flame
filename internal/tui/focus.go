package tui

// FocusMode determines which component receives keyboard input.
type FocusMode int

const (
	FocusInput   FocusMode = iota // Input bar active (shell or menu context)
	FocusSidebar                  // Session list navigation active
)
