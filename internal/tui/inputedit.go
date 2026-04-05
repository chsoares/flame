package tui

import (
	"unicode"

	"github.com/charmbracelet/bubbles/textinput"
)

func applyLineEdit(input *textinput.Model, key string) bool {
	switch key {
	case "home":
		input.SetCursor(0)
		return true
	case "end":
		input.CursorEnd()
		return true
	case "ctrl+backspace":
		deletePreviousWord(input)
		return true
	case "ctrl+delete":
		deleteNextWord(input)
		return true
	case "ctrl+z":
		input.SetValue("")
		input.SetCursor(0)
		return true
	default:
		return false
	}
}

func deletePreviousWord(input *textinput.Model) {
	value := []rune(input.Value())
	pos := input.Position()
	if pos == 0 {
		return
	}

	start := pos
	for start > 0 && unicode.IsSpace(value[start-1]) {
		start--
	}
	for start > 0 && !unicode.IsSpace(value[start-1]) {
		start--
	}

	input.SetValue(string(append(value[:start], value[pos:]...)))
	input.SetCursor(start)
}

func deleteNextWord(input *textinput.Model) {
	value := []rune(input.Value())
	pos := input.Position()
	if pos >= len(value) {
		return
	}

	start := pos
	for start < len(value) && unicode.IsSpace(value[start]) {
		start++
	}

	end := start
	for end < len(value) && !unicode.IsSpace(value[end]) {
		end++
	}
	for end < len(value) && unicode.IsSpace(value[end]) {
		end++
	}

	left := value[:pos]
	right := value[end:]
	joined := append([]rune{}, left...)
	if len(left) > 0 && len(right) > 0 && !unicode.IsSpace(left[len(left)-1]) && !unicode.IsSpace(right[0]) {
		joined = append(joined, ' ')
	}
	joined = append(joined, right...)

	input.SetValue(string(joined))
	input.SetCursor(pos)
}
