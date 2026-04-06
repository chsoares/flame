package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type listSection struct {
	Title string
	Items []string
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

func renderHelpShell(base string, termW, termH int, title, inputLine string, body []string, footer string) string {
	dialogW := 52
	if dialogW > termW-4 {
		dialogW = termW - 4
	}
	innerW := dialogW - 6
	contentWidth := innerW - 4
	if contentWidth < 1 {
		contentWidth = 1
	}
	bodyRows := modalBoxHeight(termH) - 7
	if bodyRows < 1 {
		bodyRows = 1
	}
	headHatchW := contentWidth - lipgloss.Width(title) - 1
	if headHatchW < 1 {
		headHatchW = 1
	}
	lines := []string{styleMagentaBold.Render(title) + " " + hatching(headHatchW), "", inputLine, ""}
	body = padToHeight(body, bodyRows)
	lines = append(lines, body...)
	lines = append(lines, "", footer, "")
	lines = padToHeight(lines, modalBoxHeight(termH))
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMagenta).
		Padding(0, 2).
		Width(innerW).
		Render(strings.Join(lines, "\n"))
	return overlayCenteredBox(base, box, termW, termH)
}

func buildHelpListBodyLines(sections []listSection, selected int, termW, termH int) []string {
	dialogW := 52
	if dialogW > termW-4 {
		dialogW = termW - 4
	}
	innerW := dialogW - 6
	contentWidth := innerW - 4
	if contentWidth < 1 {
		contentWidth = 1
	}
	bodyRows := modalBoxHeight(termH) - 7
	if bodyRows < 1 {
		bodyRows = 1
	}
	selectedStyle := lipgloss.NewStyle().Background(colorCyan).Foreground(lipgloss.Color("0")).Bold(true).Width(contentWidth)
	plainStyle := lipgloss.NewStyle().Foreground(colorBase).Width(contentWidth)
	categoryStyle := lipgloss.NewStyle().Foreground(colorSubtle).Bold(true)
	categoryDivider := func(label string) string {
		left := categoryStyle.Render(label)
		lineW := contentWidth - lipgloss.Width(label) - 1
		if lineW < 3 {
			lineW = 3
		}
		return left + " " + styleSubtle.Render(strings.Repeat(separatorChar, lineW))
	}
	padRow := func(text string) string {
		w := lipgloss.Width(text)
		if w >= contentWidth {
			return text
		}
		return text + strings.Repeat(" ", contentWidth-w)
	}
	flat := make([]struct{ kind, value string }, 0)
	for _, section := range sections {
		flat = append(flat, struct{ kind, value string }{kind: "category", value: section.Title})
		for _, item := range section.Items {
			flat = append(flat, struct{ kind, value string }{kind: "item", value: item})
		}
	}
	selectedRow := -1
	if selected < 0 {
		selected = 0
	}
	itemIdx := 0
	for i, row := range flat {
		if row.kind != "item" {
			continue
		}
		if itemIdx == selected {
			selectedRow = i
			break
		}
		itemIdx++
	}
	if selectedRow < 0 {
		selectedRow = 0
	}
	visible := flat
	if len(visible) > bodyRows {
		offset := 0
		if selectedRow >= bodyRows {
			offset = selectedRow - bodyRows + 1
		}
		if offset < 0 {
			offset = 0
		}
		if offset > len(flat)-bodyRows {
			offset = len(flat) - bodyRows
		}
		if offset < 0 {
			offset = 0
		}
		visible = flat[offset:]
		selectedRow -= offset
	}
	if len(visible) > bodyRows {
		visible = visible[:bodyRows]
	}
	for len(visible) < bodyRows {
		visible = append(visible, struct{ kind, value string }{})
	}
	lines := make([]string, 0, len(visible))
	for i, row := range visible {
		switch row.kind {
		case "category":
			lines = append(lines, categoryDivider(row.value))
		case "item":
			if i == selectedRow {
				lines = append(lines, selectedStyle.Render(padRow("  "+row.value)))
			} else {
				lines = append(lines, plainStyle.Render("  "+row.value))
			}
		default:
			lines = append(lines, plainStyle.Render(""))
		}
	}
	return lines
}
