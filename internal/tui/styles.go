package tui

import "github.com/charmbracelet/lipgloss"

// --- Color Palette (Terminal ANSI for maximum compatibility) ---

var (
	// Primary accents
	colorMagenta = lipgloss.Color("5")
	colorCyan    = lipgloss.Color("6")
	colorGreen   = lipgloss.Color("2")
	colorRed     = lipgloss.Color("1")
	colorYellow  = lipgloss.Color("3")

	// Text hierarchy
	colorBase   = lipgloss.Color("253") // Primary text (bright)
	colorMuted  = lipgloss.Color("245") // Secondary (gray)
	colorSubtle = lipgloss.Color("240") // Hints, decorative (dark gray)
	colorDim    = lipgloss.Color("237") // Separators, borders (very dark)

	// Backgrounds
	colorBgBar   = lipgloss.Color("235") // Header/status bar background
	colorBgPanel = lipgloss.Color("236") // Sidebar panel background
)

// --- Reusable Styles ---

var (
	// Text styles
	styleBase   = lipgloss.NewStyle().Foreground(colorBase)
	styleMuted  = lipgloss.NewStyle().Foreground(colorMuted)
	styleSubtle = lipgloss.NewStyle().Foreground(colorSubtle)

	// Accent styles
	styleMagenta     = lipgloss.NewStyle().Foreground(colorMagenta)
	styleMagentaBold = lipgloss.NewStyle().Foreground(colorMagenta).Bold(true)
	styleCyan        = lipgloss.NewStyle().Foreground(colorCyan)
	styleCyanBold    = lipgloss.NewStyle().Foreground(colorCyan).Bold(true)
	styleYellow      = lipgloss.NewStyle().Foreground(colorYellow)
	styleGreen       = lipgloss.NewStyle().Foreground(colorGreen)
	styleRed         = lipgloss.NewStyle().Foreground(colorRed)

	// Separator character
	separatorChar = "─"
	diagonalChar  = "╱"
	verticalSep   = lipgloss.NewStyle().Foreground(colorDim).Render("│")
)

// hatching generates a string of diagonal slashes in cyan (color 6).
func hatching(width int) string {
	if width <= 0 {
		return ""
	}
	s := ""
	for i := 0; i < width; i++ {
		s += diagonalChar
	}
	return styleCyan.Render(s)
}

// sectionHeader renders a labeled separator line: "Title ──────────"
func sectionHeader(title string, width int) string {
	rendered := styleMuted.Render(title)
	titleW := lipgloss.Width(rendered)
	lineW := width - titleW - 1 // 1 space between title and line
	if lineW < 3 {
		lineW = 3
	}
	line := ""
	for i := 0; i < lineW; i++ {
		line += separatorChar
	}
	return rendered + " " + styleSubtle.Render(line)
}
