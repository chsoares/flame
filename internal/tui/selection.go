package tui

import (
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/x/ansi"
)

// Selection tracks mouse-based text selection state in the output pane.
type Selection struct {
	Active   bool // Mouse is currently held down (dragging)
	HasRange bool // A valid selection range exists
	Locked   bool // Multi-click selection — don't update on motion/release

	// Content-space coordinates (after scroll offset applied)
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

// Normalized returns start/end in forward order (start <= end).
func (s Selection) Normalized() (startLine, startCol, endLine, endCol int) {
	if s.StartLine > s.EndLine || (s.StartLine == s.EndLine && s.StartCol > s.EndCol) {
		return s.EndLine, s.EndCol, s.StartLine, s.StartCol
	}
	return s.StartLine, s.StartCol, s.EndLine, s.EndCol
}

// IsEmpty returns true if the selection covers zero characters.
func (s Selection) IsEmpty() bool {
	sl, sc, el, ec := s.Normalized()
	return sl == el && sc == ec
}

// Clear resets the selection.
func (s *Selection) Clear() {
	s.Active = false
	s.HasRange = false
	s.Locked = false
}

// ClickTracker detects single, double, and triple clicks.
type ClickTracker struct {
	lastTime   time.Time
	lastX      int
	lastY      int
	clickCount int
}

const (
	doubleClickThreshold = 400 * time.Millisecond
	clickTolerance       = 2
)

// RegisterClick updates click count and returns it (1=single, 2=double, 3=triple).
func (ct *ClickTracker) RegisterClick(x, y int) int {
	now := time.Now()

	if now.Sub(ct.lastTime) <= doubleClickThreshold &&
		abs(x-ct.lastX) <= clickTolerance &&
		abs(y-ct.lastY) <= clickTolerance {
		ct.clickCount++
		if ct.clickCount > 3 {
			ct.clickCount = 1
		}
	} else {
		ct.clickCount = 1
	}

	ct.lastTime = now
	ct.lastX = x
	ct.lastY = y
	return ct.clickCount
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// findWordBoundaries finds the start and end column of the word at col in a plain-text line.
// Returns (start, end) where end is exclusive. Operates on display columns.
func findWordBoundaries(line string, col int) (int, int) {
	if line == "" || col < 0 {
		return 0, 0
	}

	// Build a slice of runes with their display column positions
	runes := []rune(line)
	if len(runes) == 0 {
		return 0, 0
	}

	// Map: display column → rune index
	// Walk runes to find which rune is under col
	runeIdx := -1
	currentCol := 0
	for i, r := range runes {
		rw := ansi.StringWidth(string(r))
		if col >= currentCol && col < currentCol+rw {
			runeIdx = i
			break
		}
		currentCol += rw
	}

	if runeIdx < 0 {
		// Past end of line
		return col, col
	}

	// If the rune under cursor is whitespace, no selection
	if unicode.IsSpace(runes[runeIdx]) {
		return col, col
	}

	isWordChar := func(r rune) bool {
		return !unicode.IsSpace(r)
	}

	// Walk left to find word start
	startIdx := runeIdx
	for startIdx > 0 && isWordChar(runes[startIdx-1]) {
		startIdx--
	}

	// Walk right to find word end
	endIdx := runeIdx
	for endIdx < len(runes)-1 && isWordChar(runes[endIdx+1]) {
		endIdx++
	}
	endIdx++ // exclusive end

	// Convert rune indices back to display columns
	startCol := 0
	for i := 0; i < startIdx; i++ {
		startCol += ansi.StringWidth(string(runes[i]))
	}
	endCol := 0
	for i := 0; i < endIdx; i++ {
		endCol += ansi.StringWidth(string(runes[i]))
	}

	return startCol, endCol
}

// extractSelectedText pulls out the plain text within the selection from wrapped content lines.
// wrapContin[i] == true means line i is a word-wrap continuation (not a real newline).
func extractSelectedText(wrappedLines []string, wrapContin []bool, startLine, startCol, endLine, endCol int) string {
	if startLine < 0 || endLine < 0 {
		return ""
	}
	if startLine >= len(wrappedLines) {
		return ""
	}
	if endLine >= len(wrappedLines) {
		endLine = len(wrappedLines) - 1
		endCol = ansi.StringWidth(ansi.Strip(wrappedLines[endLine]))
	}

	var sb strings.Builder

	for i := startLine; i <= endLine; i++ {
		if i >= len(wrappedLines) {
			break
		}
		plain := ansi.Strip(wrappedLines[i])
		runes := []rune(plain)

		// Determine column range for this line
		colStart := 0
		if i == startLine {
			colStart = startCol
		}
		colEnd := ansi.StringWidth(plain)
		if i == endLine {
			colEnd = endCol
		}

		// Extract runes in the column range
		currentCol := 0
		for _, r := range runes {
			rw := ansi.StringWidth(string(r))
			nextCol := currentCol + rw
			if currentCol >= colStart && nextCol <= colEnd {
				sb.WriteRune(r)
			}
			currentCol = nextCol
			if currentCol >= colEnd {
				break
			}
		}

		// Between lines: only insert \n for real line breaks, not word-wrap breaks
		if i < endLine {
			isContinuation := i+1 < len(wrapContin) && wrapContin[i+1]
			if !isContinuation {
				sb.WriteRune('\n')
			}
		}
	}

	return sb.String()
}

// highlightLine applies ANSI reverse video to a portion of a line.
// colStart/colEnd are display columns. Returns the modified line.
func highlightLine(line string, colStart, colEnd int) string {
	if colStart >= colEnd {
		return line
	}

	const (
		reverseOn  = "\033[7m"
		reverseOff = "\033[27m"
	)

	plain := ansi.Strip(line)
	runes := []rune(plain)

	var result strings.Builder
	currentCol := 0
	inHighlight := false

	// We need to walk the original line (with ANSI codes) and insert reverse
	// at the right display column positions. Since the original line may have
	// existing ANSI codes, we need to be ANSI-aware.
	//
	// Simple approach: rebuild from plain text with highlight applied.
	// This strips existing colors but is reliable. For a shell handler output
	// pane this is acceptable since we're selecting to copy anyway.

	for _, r := range runes {
		rw := ansi.StringWidth(string(r))

		if currentCol >= colStart && currentCol < colEnd && !inHighlight {
			result.WriteString(reverseOn)
			inHighlight = true
		}
		if currentCol >= colEnd && inHighlight {
			result.WriteString(reverseOff)
			inHighlight = false
		}

		result.WriteRune(r)
		currentCol += rw
	}

	if inHighlight {
		result.WriteString(reverseOff)
	}

	// Pad trailing space for full-line selections (so highlight extends visually)
	if colEnd > currentCol && colEnd < 999 {
		result.WriteString(reverseOn)
		for i := currentCol; i < colEnd; i++ {
			result.WriteRune(' ')
		}
		result.WriteString(reverseOff)
	}

	return result.String()
}
