package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestInputSuggestionUsesMostRecentPrefixMatch(t *testing.T) {
	input := NewInput()
	input.menuHistory = []string{
		"help",
		"run whoami",
		"run rev bash",
		"sessions",
	}
	input.SetValue("run")

	got, ok := input.Suggestion()
	if !ok {
		t.Fatal("expected suggestion for prefix run")
	}
	if got != "run rev bash" {
		t.Fatalf("expected most recent prefix match, got %q", got)
	}
}

func TestInputSuggestionHiddenWhenExactMatchOrEmpty(t *testing.T) {
	t.Run("empty input", func(t *testing.T) {
		input := NewInput()
		input.menuHistory = []string{"run whoami"}

		if _, ok := input.Suggestion(); ok {
			t.Fatal("expected no suggestion for empty input")
		}
	})

	t.Run("exact match", func(t *testing.T) {
		input := NewInput()
		input.menuHistory = []string{"run whoami"}
		input.SetValue("run whoami")

		if _, ok := input.Suggestion(); ok {
			t.Fatal("expected no suggestion when input already equals match")
		}
	})
}

func TestInputViewRendersInlineSuggestionSuffix(t *testing.T) {
	input := NewInput()
	input.SetWidth(40)
	input.menuHistory = []string{"run whoami", "run rev bash"}
	input.SetValue("run")

	view := ansi.Strip(input.View())
	if !strings.Contains(view, "run rev bash") {
		t.Fatalf("expected inline suggestion to render in view, got %q", view)
	}
}

func TestInputUpdateRightAcceptsSuggestion(t *testing.T) {
	input := NewInput()
	input.menuHistory = []string{"run whoami", "run rev bash"}
	input.SetValue("run")

	input.Update(tea.KeyMsg{Type: tea.KeyRight})

	if got := input.Value(); got != "run rev bash" {
		t.Fatalf("expected right to accept suggestion via Input.Update, got %q", got)
	}
}

func TestInputUpdateTabDoesNotAcceptSuggestion(t *testing.T) {
	input := NewInput()
	input.menuHistory = []string{"run whoami", "run rev bash"}
	input.SetValue("run")

	input.Update(tea.KeyMsg{Type: tea.KeyTab})

	if got := input.Value(); got != "run" {
		t.Fatalf("expected tab not to accept suggestion, got %q", got)
	}
}

func TestInputHistoryNavigationFiltersByPrefixAndRestoresTypedPrefix(t *testing.T) {
	input := NewInput()
	input.menuHistory = []string{
		"help",
		"run whoami",
		"sessions",
		"run rev bash",
		"use 2",
	}
	input.SetValue("run")

	input.HistoryUp()
	if got := input.Value(); got != "run rev bash" {
		t.Fatalf("expected newest run* entry, got %q", got)
	}

	input.HistoryUp()
	if got := input.Value(); got != "run whoami" {
		t.Fatalf("expected older run* entry, got %q", got)
	}

	input.HistoryDown()
	if got := input.Value(); got != "run rev bash" {
		t.Fatalf("expected newer run* entry, got %q", got)
	}

	input.HistoryDown()
	if got := input.Value(); got != "run" {
		t.Fatalf("expected original typed prefix restored, got %q", got)
	}
}

func TestInputHistoryNavigationUsesSessionHistoryInShellContext(t *testing.T) {
	input := NewInput()
	input.SetContext(ContextShell)
	input.SetSessionID(7)
	input.sessionHistory[7] = []string{
		"pwd",
		"ps aux",
		"python -c 'print(1)'",
		"ps -ef",
	}
	input.SetValue("ps")

	input.HistoryUp()
	if got := input.Value(); got != "ps -ef" {
		t.Fatalf("expected newest shell prefix match, got %q", got)
	}
}

func TestInputHistoryNavigationKeepsEmptyInputAsReturnPoint(t *testing.T) {
	input := NewInput()
	input.menuHistory = []string{
		"alpha",
		"beta",
		"gamma",
	}

	input.HistoryUp()
	if got := input.Value(); got != "gamma" {
		t.Fatalf("expected newest history entry, got %q", got)
	}

	input.HistoryUp()
	if got := input.Value(); got != "beta" {
		t.Fatalf("expected previous history entry on repeated up, got %q", got)
	}

	input.HistoryDown()
	if got := input.Value(); got != "gamma" {
		t.Fatalf("expected next history entry on down, got %q", got)
	}

	input.HistoryDown()
	if got := input.Value(); got != "" {
		t.Fatalf("expected empty input restored after history end, got %q", got)
	}
}

func TestInputSubmitDropsConsecutiveDuplicatesInMenuHistory(t *testing.T) {
	input := NewInput()
	input.menuHistory = []string{"run whoami"}
	input.SetValue("run whoami")

	input.Submit()

	if got := input.menuHistory; len(got) != 1 || got[0] != "run whoami" {
		t.Fatalf("expected consecutive duplicate not to be appended, got %#v", got)
	}
}

func TestInputHistoryNavigationSuppressesStaleSuggestion(t *testing.T) {
	input := NewInput()
	input.menuHistory = []string{"ls", "lalala"}
	input.SetValue("l")

	input.HistoryUp()
	input.HistoryUp()

	if got := input.Value(); got != "ls" {
		t.Fatalf("expected to land on older history entry, got %q", got)
	}
	if _, ok := input.Suggestion(); ok {
		t.Fatal("expected suggestion to be suppressed while navigating history")
	}
}
