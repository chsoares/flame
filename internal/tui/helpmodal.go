package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	internal "github.com/chsoares/flame/internal"
)

type listSection struct {
	Title string
	Items []string
}

type helpModal struct {
	topics     []string
	categories []string
	index      int
	filter     string
	input      string
	detail     *internal.HelpEntry
}

func newHelpModal() helpModal {
	return helpModal{topics: internal.HelpTopicsForModal(), categories: internal.HelpCategoriesForModal()}
}

func (h helpModal) SelectedTopic() string {
	visible := h.selectableTopics()
	if len(visible) == 0 {
		return ""
	}
	if h.index < 0 || h.index >= len(visible) {
		return visible[0]
	}
	return visible[h.index]
}

func (h *helpModal) MoveDown() {
	visible := h.selectableTopics()
	if len(visible) == 0 {
		return
	}
	h.index = (h.index + 1) % len(visible)
}

func (h *helpModal) MoveUp() {
	visible := h.selectableTopics()
	if len(visible) == 0 {
		return
	}
	h.index--
	if h.index < 0 {
		h.index = len(visible) - 1
	}
}

func (h *helpModal) SetFilter(value string) {
	h.filter = value
	h.input = value
	h.index = 0
	h.detail = nil
}

func (h *helpModal) OpenSelected() {
	topic := h.SelectedTopic()
	if topic == "" {
		return
	}
	entry, ok := internal.LookupHelpTopic(strings.Fields(topic))
	if !ok {
		return
	}
	h.detail = &entry
}

func (h *helpModal) CloseDetail() {
	h.detail = nil
}

func (h *helpModal) BackspaceFilter() {
	if h.input == "" {
		return
	}
	h.input = h.input[:len(h.input)-1]
	h.filter = h.input
	h.index = 0
	h.detail = nil
}

func (h helpModal) filteredTopics() []string {
	if h.filter == "" {
		return h.topics
	}
	needle := strings.ToLower(h.filter)
	filtered := make([]string, 0, len(h.topics))
	for _, topic := range h.topics {
		if strings.Contains(strings.ToLower(topic), needle) {
			filtered = append(filtered, topic)
		}
	}
	return filtered
}

func (h helpModal) groupedTopics() []listSection {
	filtered := h.filteredTopics()
	byCategory := make(map[string][]string)
	for _, topic := range filtered {
		if entry, ok := internal.LookupHelpTopic(strings.Fields(topic)); ok {
			cat := entry.Category
			if cat == "" {
				cat = "other"
			}
			byCategory[cat] = append(byCategory[cat], topic)
		}
	}
	sections := make([]listSection, 0, len(h.categories))
	for _, cat := range h.categories {
		items := byCategory[cat]
		if len(items) == 0 {
			continue
		}
		sections = append(sections, listSection{Title: cat, Items: items})
	}
	return sections
}

func (h helpModal) selectableTopics() []string {
	filtered := h.filteredTopics()
	out := make([]string, 0, len(filtered))
	for _, topic := range filtered {
		if entry, ok := internal.LookupHelpTopic(strings.Fields(topic)); ok && entry.Category != "" {
			out = append(out, topic)
		}
	}
	return out
}

func (h helpModal) currentSelection() string {
	visible := h.selectableTopics()
	if len(visible) == 0 {
		return ""
	}
	if h.index < 0 || h.index >= len(visible) {
		return visible[0]
	}
	return visible[h.index]
}

func (h *helpModal) View(termW, termH int, base string) string {
	if h.detail != nil {
		footer := styleMuted.Bold(true).Render("Backspace") + " " + styleSubtle.Render("back") +
			styleSubtle.Render(" • ") + styleMuted.Bold(true).Render("Esc") + " " + styleSubtle.Render("close")
		return RenderModalShell(base, termW, termH, ModalShell{
			Title:  "help",
			Width:  52,
			Body:   append([]string{styleCyan.Render(h.detail.Topic), ""}, strings.Split(internal.RenderHelpEntryDetail(*h.detail), "\n")...),
			Footer: footer,
			Align:  BodyAlignLeft,
		})
	}

	sections := h.groupedTopics()
	selectedTopic := h.currentSelection()
	selectable := h.selectableTopics()
	if len(selectable) == 0 {
		h.index = 0
	} else if selectedTopic == "" {
		h.index = 0
	}
	filterLine := styleMagentaBold.Render("❯") + " "
	if h.input == "" {
		filterLine += styleSubtle.Render("Type to filter")
	} else {
		filterLine += styleBase.Render(h.input)
	}
	footer := styleMuted.Bold(true).Render("Enter") + " " + styleSubtle.Render("details") +
		styleSubtle.Render(" • ") + styleMuted.Bold(true).Render("Esc") + " " + styleSubtle.Render("close")
	body := append([]string{filterLine, ""}, buildHelpListBodyLines(sections, selectedTopic, termW, termH)...)
	return RenderModalShell(base, termW, termH, ModalShell{
		Title:  "help",
		Width:  52,
		Body:   body,
		Footer: footer,
		Align:  BodyAlignLeft,
	})
}

func buildHelpListBodyLines(sections []listSection, selectedTopic string, termW, termH int) []string {
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
	for i, row := range flat {
		if row.kind == "item" && row.value == selectedTopic {
			selectedRow = i
			break
		}
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
