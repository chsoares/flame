package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func newTestInput(value string, cursor int) textinput.Model {
	ti := textinput.New()
	ti.SetValue(value)
	ti.SetCursor(cursor)
	return ti
}

func syntheticShortcutKey(shortcut string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(shortcut)}
}

func TestApplyLineEditHomeAndEnd(t *testing.T) {
	t.Run("home", func(t *testing.T) {
		ti := newTestInput("hello world", 5)
		if !applyLineEdit(&ti, "home") {
			t.Fatal("expected home to be handled")
		}
		if ti.Position() != 0 {
			t.Fatalf("expected cursor 0, got %d", ti.Position())
		}
	})

	t.Run("end", func(t *testing.T) {
		ti := newTestInput("hello world", 2)
		if !applyLineEdit(&ti, "end") {
			t.Fatal("expected end to be handled")
		}
		if ti.Position() != len("hello world") {
			t.Fatalf("expected cursor at end, got %d", ti.Position())
		}
	})
}

func TestApplyLineEditDeletePreviousWord(t *testing.T) {
	ti := newTestInput("alpha beta gamma", len("alpha beta "))
	if !applyLineEdit(&ti, "ctrl+backspace") {
		t.Fatal("expected ctrl+backspace to be handled")
	}
	if got := ti.Value(); got != "alpha gamma" {
		t.Fatalf("expected previous word deleted, got %q", got)
	}
	if got := ti.Position(); got != len("alpha ") {
		t.Fatalf("expected cursor at %d, got %d", len("alpha "), got)
	}
}

func TestApplyLineEditDeleteNextWord(t *testing.T) {
	ti := newTestInput("alpha beta gamma", len("alpha "))
	if !applyLineEdit(&ti, "ctrl+delete") {
		t.Fatal("expected ctrl+delete to be handled")
	}
	if got := ti.Value(); got != "alpha gamma" {
		t.Fatalf("expected next word deleted, got %q", got)
	}
	if got := ti.Position(); got != len("alpha ") {
		t.Fatalf("expected cursor at %d, got %d", len("alpha "), got)
	}
}

func TestApplyLineEditDeleteNextWordAfterSpaces(t *testing.T) {
	ti := newTestInput("alpha   beta gamma", len("alpha"))
	if !applyLineEdit(&ti, "ctrl+delete") {
		t.Fatal("expected ctrl+delete to be handled")
	}
	if got := ti.Value(); got != "alpha gamma" {
		t.Fatalf("expected spaces before next word and that word deleted, got %q", got)
	}
	if got := ti.Position(); got != len("alpha") {
		t.Fatalf("expected cursor at %d, got %d", len("alpha"), got)
	}
}

func TestApplyLineEditBoundariesNoOp(t *testing.T) {
	t.Run("ctrl+backspace at start", func(t *testing.T) {
		ti := newTestInput("alpha", 0)
		applyLineEdit(&ti, "ctrl+backspace")
		if got := ti.Value(); got != "alpha" || ti.Position() != 0 {
			t.Fatalf("expected no-op at start, got %q / %d", got, ti.Position())
		}
	})

	t.Run("ctrl+delete at end", func(t *testing.T) {
		ti := newTestInput("alpha", len("alpha"))
		applyLineEdit(&ti, "ctrl+delete")
		if got := ti.Value(); got != "alpha" || ti.Position() != len("alpha") {
			t.Fatalf("expected no-op at end, got %q / %d", got, ti.Position())
		}
	})
}

func TestApplyLineEditClearInput(t *testing.T) {
	ti := newTestInput("alpha beta", len("alpha"))
	if !applyLineEdit(&ti, "ctrl+z") {
		t.Fatal("expected ctrl+z to be handled")
	}
	if got := ti.Value(); got != "" || ti.Position() != 0 {
		t.Fatalf("expected cleared input, got %q / %d", got, ti.Position())
	}
}

func TestApplyLineEditUnsupportedKey(t *testing.T) {
	ti := newTestInput("alpha beta", len("alpha"))
	if applyLineEdit(&ti, "ctrl+x") {
		t.Fatal("expected unsupported key to fall through")
	}
	if got := ti.Value(); got != "alpha beta" || ti.Position() != len("alpha") {
		t.Fatalf("expected unsupported key to be a no-op, got %q / %d", got, ti.Position())
	}
}

func TestInputUpdateUsesSharedLineEditHelper(t *testing.T) {
	t.Run("home and end", func(t *testing.T) {
		input := NewInput()
		input.SetValue("alpha beta gamma")
		input.textinput.SetCursor(len("alpha "))

		input.Update(tea.KeyMsg{Type: tea.KeyHome})
		if got := input.textinput.Position(); got != 0 {
			t.Fatalf("expected home to move cursor to 0 via shared helper, got %d", got)
		}

		input.Update(tea.KeyMsg{Type: tea.KeyEnd})
		if got := input.textinput.Position(); got != len("alpha beta gamma") {
			t.Fatalf("expected end to move cursor to %d via shared helper, got %d", len("alpha beta gamma"), got)
		}
	})

	t.Run("ctrl+z", func(t *testing.T) {
		input := NewInput()
		input.SetValue("alpha beta gamma")
		input.textinput.SetCursor(len("alpha "))

		input.Update(tea.KeyMsg{Type: tea.KeyCtrlZ})

		if got := input.Value(); got != "" {
			t.Fatalf("expected Input.Update to clear input via shared helper, got %q", got)
		}
		if got := input.textinput.Position(); got != 0 {
			t.Fatalf("expected cursor at 0, got %d", got)
		}
		if got := input.textinput.Err; got != nil {
			t.Fatalf("expected no text input error, got %v", got)
		}
	})

	t.Run("app input mode forwards home and end to input update", func(t *testing.T) {
		app := New(nil, "127.0.0.1:4444")
		app.splash = false
		app.context = ContextMenu
		app.input.SetValue("alpha beta gamma")
		app.input.textinput.SetCursor(len("alpha "))

		app.updateInputMode(tea.KeyMsg{Type: tea.KeyHome})
		if got := app.input.textinput.Position(); got != 0 {
			t.Fatalf("expected app input mode to forward home to input update, got cursor %d", got)
		}

		app.updateInputMode(tea.KeyMsg{Type: tea.KeyEnd})
		if got := app.input.textinput.Position(); got != len("alpha beta gamma") {
			t.Fatalf("expected app input mode to forward end to input update, got cursor %d", got)
		}
	})

	t.Run("app input mode forwards ctrl+backspace to input update", func(t *testing.T) {
		app := New(nil, "127.0.0.1:4444")
		app.splash = false
		app.context = ContextMenu
		app.input.SetValue("alpha beta gamma")
		app.input.textinput.SetCursor(len("alpha beta "))

		app.updateInputMode(syntheticShortcutKey("ctrl+backspace"))

		if got := app.input.Value(); got != "alpha gamma" {
			t.Fatalf("expected app input mode to forward ctrl+backspace to input update, got %q", got)
		}
		if got := app.input.textinput.Position(); got != len("alpha ") {
			t.Fatalf("expected cursor at %d, got %d", len("alpha "), got)
		}
	})

	t.Run("app input mode forwards ctrl+delete to input update", func(t *testing.T) {
		app := New(nil, "127.0.0.1:4444")
		app.splash = false
		app.context = ContextMenu
		app.input.SetValue("alpha beta gamma")
		app.input.textinput.SetCursor(len("alpha "))

		app.updateInputMode(syntheticShortcutKey("ctrl+delete"))

		if got := app.input.Value(); got != "alpha gamma" {
			t.Fatalf("expected app input mode to forward ctrl+delete to input update, got %q", got)
		}
		if got := app.input.textinput.Position(); got != len("alpha ") {
			t.Fatalf("expected cursor at %d, got %d", len("alpha "), got)
		}
	})
}
