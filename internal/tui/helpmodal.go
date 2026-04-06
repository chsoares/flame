package tui

import (
	"strings"

	internal "github.com/chsoares/flame/internal"
)

type helpModal struct {
	topics     []string
	categories []string
	index      int
	offset     int
	filter     string
	input      string
}

func newHelpModal() helpModal {
	return helpModal{topics: internal.HelpTopicsForModal(), categories: internal.HelpCategoriesForModal()}
}

func (h helpModal) SelectedTopic() string {
	if len(h.topics) == 0 {
		return ""
	}
	if h.index < 0 || h.index >= len(h.topics) {
		return h.topics[0]
	}
	return h.topics[h.index]
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
	h.offset = 0
}

func (h *helpModal) OpenSelected() {}

func (h *helpModal) CloseDetail() {}

func (h *helpModal) BackspaceFilter() {
	if h.input == "" {
		return
	}
	h.input = h.input[:len(h.input)-1]
	h.filter = h.input
	h.index = 0
	h.offset = 0
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

func (h *helpModal) View(termW, termH int, base string) string {
	sections := h.groupedTopics()
	selectable := h.selectableTopics()
	if len(selectable) == 0 {
		h.index = 0
	} else if h.index >= len(selectable) {
		h.index = len(selectable) - 1
	}
	filterLine := styleMagentaBold.Render("❯") + " "
	if h.input == "" {
		filterLine += styleSubtle.Render("Type to filter")
	} else {
		filterLine += styleBase.Render(h.input)
	}
	footer := styleMuted.Bold(true).Render("Enter") + " " + styleSubtle.Render("details") +
		styleSubtle.Render(" • ") + styleMuted.Bold(true).Render("Esc") + " " + styleSubtle.Render("close")
	body := buildHelpListBodyLines(sections, h.index, termW, termH)
	return renderHelpShell(base, termW, termH, "help", filterLine, body, footer)
}
