package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

type BodyAlign int

const (
	BodyAlignLeft BodyAlign = iota
	BodyAlignCenter
)

type ModalShell struct {
	Title     string
	Width     int
	MaxHeight int
	Body      []string
	Footer    string
	Align     BodyAlign
}

func modalBoxHeight(termH int) int {
	height := termH - 2
	if height > 24 {
		height = 24
	}
	if height < 8 {
		height = 8
	}
	return height
}

func padToHeight(lines []string, height int) []string {
	if len(lines) > height {
		return lines[:height]
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func centerLine(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	padL := (width - w) / 2
	padR := width - w - padL
	return strings.Repeat(" ", padL) + s + strings.Repeat(" ", padR)
}

func wrapModalLine(line string, width int) []string {
	if width < 1 {
		width = 1
	}
	wrapped := ansi.Wrap(line, width, "")
	if wrapped == "" {
		return []string{""}
	}
	return strings.Split(wrapped, "\n")
}

func wrapModalLines(lines []string, width int) []string {
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapModalLine(line, width)...)
	}
	return wrapped
}

func RenderModalShell(base string, termW, termH int, shell ModalShell) string {
	dialogW := shell.Width
	if dialogW <= 0 {
		dialogW = 52
	}
	if dialogW > termW-4 {
		dialogW = termW - 4
	}
	if dialogW < 1 {
		dialogW = 1
	}

	innerW := dialogW - 6
	if innerW < 1 {
		innerW = 1
	}
	contentWidth := innerW - 4
	if contentWidth < 1 {
		contentWidth = 1
	}

	headHatchW := contentWidth - lipgloss.Width(shell.Title) - 1
	if headHatchW < 1 {
		headHatchW = 1
	}
	headerRow := styleMagentaBold.Render(shell.Title) + " " + hatching(headHatchW)

	maxH := modalBoxHeight(termH)
	if shell.MaxHeight > 0 && shell.MaxHeight < maxH {
		maxH = shell.MaxHeight
	}
	bodyRows := maxH - 2
	if shell.Footer != "" {
		bodyRows--
	}
	if bodyRows < 1 {
		bodyRows = 1
	}
	body := padToHeight(wrapModalLines(shell.Body, contentWidth), bodyRows)
	if shell.Align == BodyAlignCenter {
		for i, line := range body {
			body[i] = centerLine(line, contentWidth)
		}
	}

	lines := []string{headerRow, ""}
	lines = append(lines, body...)
	if shell.Footer != "" {
		lines = append(lines, shell.Footer)
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMagenta).
		Padding(0, 2).
		Width(innerW).
		Render(strings.Join(lines, "\n"))
	return overlayCenteredBox(base, box, termW, termH)
}

func overlayCenteredBox(base, box string, termW, termH int) string {
	if termW <= 0 || termH <= 0 {
		return base
	}

	lines := strings.Split(base, "\n")
	if len(lines) < termH {
		for len(lines) < termH {
			lines = append(lines, strings.Repeat(" ", termW))
		}
	} else if len(lines) > termH {
		lines = lines[:termH]
	}
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < termW {
			lines[i] = line + strings.Repeat(" ", termW-w)
		} else if w > termW {
			lines[i] = ansi.Truncate(line, termW, "")
		}
	}

	boxLines := strings.Split(box, "\n")
	boxH := len(boxLines)
	boxW := 0
	for _, line := range boxLines {
		if w := lipgloss.Width(line); w > boxW {
			boxW = w
		}
	}
	if boxW <= 0 || boxH <= 0 {
		return strings.Join(lines, "\n")
	}

	x := (termW - boxW) / 2
	if x < 0 {
		x = 0
	}
	y := (termH - boxH) / 2
	if y < 0 {
		y = 0
	}

	for i, boxLine := range boxLines {
		row := y + i
		if row < 0 || row >= len(lines) {
			continue
		}

		line := lines[row]
		if lipgloss.Width(line) < termW {
			line += strings.Repeat(" ", termW-lipgloss.Width(line))
		}

		left := ansi.Cut(line, 0, x)
		right := ansi.Cut(line, x+boxW, termW)
		placed := boxLine
		if bw := lipgloss.Width(placed); bw > boxW {
			placed = ansi.Cut(placed, 0, boxW)
		}
		lines[row] = left + placed + right
	}

	return strings.Join(lines, "\n")
}
