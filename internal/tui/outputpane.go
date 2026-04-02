package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// OutputPane is a scrollable viewport that displays shell output or menu output.
type OutputPane struct {
	viewport viewport.Model
	content  strings.Builder
	width    int
	follow   bool // Auto-scroll to bottom

	// Mouse selection
	selection    Selection
	clicks       ClickTracker
	wrappedLines []string // Cache of wrapped lines for selection
	wrapContin   []bool   // true if wrappedLines[i] is a continuation (not a real newline)

	// Mouse throttling
	lastMotion time.Time

	// Inline spinner (temporary last line, not part of permanent content)
	spinnerActive bool
	spinnerID     int
	spinnerText   string
	spinnerFrame  int
}

const mouseThrottle = 15 * time.Millisecond

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func NewOutputPane(width, height int) OutputPane {
	vp := viewport.New(width, height)
	return OutputPane{
		viewport: vp,
		width:    width,
		follow:   true,
	}
}

func (o *OutputPane) SetSize(width, height int) {
	o.width = width
	o.viewport.Width = width
	o.viewport.Height = height
	// Re-wrap existing content
	wrapped := o.wrapContent(o.content.String())
	o.wrappedLines = strings.Split(wrapped, "\n")
	o.viewport.SetContent(wrapped)
	if o.follow {
		o.viewport.GotoBottom()
	}
}

// Append adds text to the output and optionally auto-scrolls.
func (o *OutputPane) Append(text string) {
	o.content.WriteString(text)
	wrapped := o.wrapContent(o.content.String())
	o.wrappedLines = strings.Split(wrapped, "\n")
	o.viewport.SetContent(wrapped)
	if o.follow {
		o.viewport.GotoBottom()
	}
}

// GetContent returns the raw (unwrapped) content of the pane.
func (o *OutputPane) GetContent() string {
	return o.content.String()
}

// SetContent replaces all content (used when switching sessions).
func (o *OutputPane) SetContent(text string) {
	o.content.Reset()
	o.content.WriteString(text)
	wrapped := o.wrapContent(text)
	o.wrappedLines = strings.Split(wrapped, "\n")
	o.viewport.SetContent(wrapped)
	o.viewport.GotoBottom()
	o.follow = true
}

// Clear removes all content.
func (o *OutputPane) Clear() {
	o.content.Reset()
	o.wrappedLines = nil
	o.viewport.SetContent("")
}

// HandleMouseDown starts a selection or updates click count for multi-click.
func (o *OutputPane) HandleMouseDown(viewX, viewY int) {
	contentLine := viewY + o.viewport.YOffset
	contentCol := viewX

	clickCount := o.clicks.RegisterClick(viewX, viewY)

	switch clickCount {
	case 1:
		// Start drag selection
		o.selection = Selection{
			Active:    true,
			HasRange:  false,
			StartLine: contentLine,
			StartCol:  contentCol,
			EndLine:   contentLine,
			EndCol:    contentCol,
		}

	case 2:
		// Word selection
		o.selectWord(contentLine, contentCol)

	case 3:
		// Line selection
		o.selectLine(contentLine)
		o.clicks.clickCount = 0 // Reset after triple
	}
}

// HandleMouseMotion updates the drag endpoint.
func (o *OutputPane) HandleMouseMotion(viewX, viewY int) {
	if !o.selection.Active || o.selection.Locked {
		return
	}

	// Throttle motion events
	now := time.Now()
	if now.Sub(o.lastMotion) < mouseThrottle {
		return
	}
	o.lastMotion = now

	contentLine := viewY + o.viewport.YOffset
	contentCol := viewX

	o.selection.EndLine = contentLine
	o.selection.EndCol = contentCol
	o.selection.HasRange = true
}

// HandleMouseUp finalizes the selection and copies to clipboard.
// Returns true if a selection was made.
func (o *OutputPane) HandleMouseUp(viewX, viewY int) bool {
	if !o.selection.Active {
		return false
	}

	o.selection.Active = false

	// Multi-click selections (word/line) are already finalized — don't overwrite
	if !o.selection.Locked {
		contentLine := viewY + o.viewport.YOffset
		contentCol := viewX
		o.selection.EndLine = contentLine
		o.selection.EndCol = contentCol
		o.selection.HasRange = true
	}

	o.selection.Locked = false

	if o.selection.IsEmpty() {
		o.selection.Clear()
		return false
	}

	return true
}

// CopySelection extracts selected text and copies to clipboard.
func (o *OutputPane) CopySelection() string {
	if !o.selection.HasRange || o.selection.IsEmpty() {
		return ""
	}

	startLine, startCol, endLine, endCol := o.selection.Normalized()
	text := extractSelectedText(o.wrappedLines, o.wrapContin, startLine, startCol, endLine, endCol)

	if text != "" {
		copyToClipboard(text)
	}

	return text
}

// ClearSelection removes any active selection.
func (o *OutputPane) ClearSelection() {
	o.selection.Clear()
}

// HasSelection returns true if there's an active selection to render.
func (o *OutputPane) HasSelection() bool {
	return o.selection.HasRange && !o.selection.IsEmpty()
}

func (o *OutputPane) selectWord(contentLine, contentCol int) {
	if contentLine < 0 || contentLine >= len(o.wrappedLines) {
		return
	}

	plain := ansi.Strip(o.wrappedLines[contentLine])
	startCol, endCol := findWordBoundaries(plain, contentCol)

	if startCol == endCol {
		return
	}

	o.selection = Selection{
		Active:    true,
		HasRange:  true,
		Locked:    true,
		StartLine: contentLine,
		StartCol:  startCol,
		EndLine:   contentLine,
		EndCol:    endCol,
	}
}

func (o *OutputPane) selectLine(contentLine int) {
	if contentLine < 0 || contentLine >= len(o.wrappedLines) {
		return
	}

	plain := ansi.Strip(o.wrappedLines[contentLine])
	lineWidth := ansi.StringWidth(plain)

	o.selection = Selection{
		Active:    true,
		HasRange:  true,
		Locked:    true,
		StartLine: contentLine,
		StartCol:  0,
		EndLine:   contentLine,
		EndCol:    lineWidth,
	}
}

// StartSpinner begins an inline spinner at the bottom of the viewport.
func (o *OutputPane) StartSpinner(id int, text string) {
	o.spinnerActive = true
	o.spinnerID = id
	o.spinnerText = text
	o.spinnerFrame = 0
	o.refreshSpinnerContent()
}

// StopSpinner removes the spinner if the ID matches.
func (o *OutputPane) StopSpinner(id int) {
	if o.spinnerID == id {
		o.spinnerActive = false
		// Re-set content without spinner line
		wrapped := o.wrapContent(o.content.String())
		o.wrappedLines = strings.Split(wrapped, "\n")
		o.viewport.SetContent(wrapped)
		if o.follow {
			o.viewport.GotoBottom()
		}
	}
}

// TickSpinner advances the spinner animation frame.
func (o *OutputPane) TickSpinner(id int) {
	if o.spinnerActive && o.spinnerID == id {
		o.spinnerFrame = (o.spinnerFrame + 1) % len(spinnerFrames)
		o.refreshSpinnerContent()
	}
}

// refreshSpinnerContent re-renders the viewport content with the spinner as the last line.
func (o *OutputPane) refreshSpinnerContent() {
	wrapped := o.wrapContent(o.content.String())
	// Strip trailing empty lines so spinner sits right below content
	wrapped = strings.TrimRight(wrapped, "\n")
	spinnerLine := styleMagenta.Render(spinnerFrames[o.spinnerFrame]) + " " + styleMuted.Render(o.spinnerText)
	withSpinner := wrapped + "\n" + spinnerLine
	o.wrappedLines = strings.Split(withSpinner, "\n")
	o.viewport.SetContent(withSpinner)
	if o.follow {
		o.viewport.GotoBottom()
	}
}

func (o *OutputPane) Update(msg tea.Msg) (*OutputPane, tea.Cmd) {
	var cmd tea.Cmd
	o.viewport, cmd = o.viewport.Update(msg)

	if o.viewport.AtBottom() {
		o.follow = true
	} else {
		o.follow = false
	}

	return o, cmd
}

// View renders the viewport, applying selection highlights.
func (o *OutputPane) View() string {
	if !o.HasSelection() {
		return o.viewport.View()
	}

	yOffset := o.viewport.YOffset
	height := o.viewport.Height

	viewLines := strings.Split(o.viewport.View(), "\n")

	startLine, startCol, endLine, endCol := o.selection.Normalized()

	for i := 0; i < len(viewLines) && i < height; i++ {
		contentIdx := i + yOffset

		if contentIdx < startLine || contentIdx > endLine {
			continue
		}

		lineColStart := 0
		lineColEnd := o.width
		if contentIdx >= 0 && contentIdx < len(o.wrappedLines) {
			lineColEnd = ansi.StringWidth(ansi.Strip(o.wrappedLines[contentIdx]))
		}

		if contentIdx == startLine {
			lineColStart = startCol
		}
		if contentIdx == endLine {
			lineColEnd = endCol
		}

		if lineColStart < lineColEnd {
			viewLines[i] = highlightLine(viewLines[i], lineColStart, lineColEnd)
		}
	}

	return strings.Join(viewLines, "\n")
}

// ScrollbarThumb returns the start and end view-line indices for the scrollbar thumb.
// Returns (-1, -1) if no scrollbar is needed (content fits in viewport).
func (o *OutputPane) ScrollbarThumb() (int, int) {
	totalLines := len(o.wrappedLines)
	height := o.viewport.Height

	if totalLines <= height || height <= 0 {
		return -1, -1
	}

	// Thumb size: proportional to visible fraction, minimum 1 line
	thumbSize := height * height / totalLines
	if thumbSize < 1 {
		thumbSize = 1
	}

	// Thumb position: proportional to scroll offset
	maxOffset := totalLines - height
	if maxOffset < 1 {
		maxOffset = 1
	}
	thumbStart := o.viewport.YOffset * (height - thumbSize) / maxOffset

	// Clamp
	if thumbStart < 0 {
		thumbStart = 0
	}
	if thumbStart+thumbSize > height {
		thumbStart = height - thumbSize
	}

	return thumbStart, thumbStart + thumbSize
}

// wrapContent wraps long lines to fit the viewport width using ANSI-aware
// hard wrapping. Preserves escape sequences across line breaks.
// Also builds o.wrapContin: contin[i] = true means wrappedLines[i] is a
// continuation of the previous original line (word-wrap break, not a real \n).
func (o *OutputPane) wrapContent(text string) string {
	if o.width <= 0 {
		o.wrapContin = nil
		return text
	}

	origLines := strings.Split(text, "\n")
	var wrapped []string
	var contin []bool
	for _, line := range origLines {
		// ansi.Hardwrap handles ANSI escape sequences correctly
		hw := ansi.Hardwrap(line, o.width, false)
		parts := strings.Split(hw, "\n")
		for i, part := range parts {
			wrapped = append(wrapped, part)
			contin = append(contin, i > 0) // First part is original, rest are continuations
		}
	}
	o.wrapContin = contin
	return strings.Join(wrapped, "\n")
}
