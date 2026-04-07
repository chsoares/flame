package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestHelpModalSelectedTopicMovesWithFilter(t *testing.T) {
	m := newHelpModal()
	m.SetFilter("run ps1")
	if got := m.SelectedTopic(); got != "run ps1" {
		t.Fatalf("expected filtered selection to stay on run ps1, got %q", got)
	}
}

func TestHelpModalCurrentSelectionTracksIndex(t *testing.T) {
	m := newHelpModal()
	if got := m.currentSelection(); got != "rev" {
		t.Fatalf("expected first visible selection to be rev, got %q", got)
	}
	m.MoveDown()
	if got := m.currentSelection(); got == "" {
		t.Fatal("expected selection after moving down")
	}
}

func TestHelpModalOpenSelectedLoadsDetail(t *testing.T) {
	m := newHelpModal()
	m.SetFilter("run ps1")
	m.OpenSelected()
	if m.detail == nil {
		t.Fatal("expected selected help topic detail to open")
	}
	if m.detail.Topic != "run ps1" {
		t.Fatalf("expected run ps1 detail, got %q", m.detail.Topic)
	}
}

func TestHelpModalDetailViewKeepsHelpTitleAndCyanCommand(t *testing.T) {
	m := newHelpModal()
	m.SetFilter("run ps1")
	m.OpenSelected()
	got := m.View(80, 24, strings.Repeat("x", 80*24))
	plain := ansi.Strip(got)
	if !strings.Contains(plain, "help") {
		t.Fatalf("expected help title in detail view, got %q", plain)
	}
	if !strings.Contains(got, styleCyan.Render("run ps1")) {
		t.Fatalf("expected command line in cyan, got %q", got)
	}
	if !strings.Contains(plain, "Backspace back") {
		t.Fatalf("expected back hint in detail footer, got %q", plain)
	}
}

func TestHelpModalListViewHasSingleGapBeforeFooter(t *testing.T) {
	m := newHelpModal()
	got := ansi.Strip(m.View(80, 24, strings.Repeat("x", 80*24)))
	if strings.Contains(got, "\n\n\nEnter details") {
		t.Fatalf("expected a single blank line before the footer, got %q", got)
	}
}

func TestUpdateHelpBackspaceEditsFilter(t *testing.T) {
	a := App{help: func() *helpModal { m := newHelpModal(); return &m }()}
	a.help.SetFilter("run ps1")
	model, _ := a.updateHelp(tea.KeyMsg{Type: tea.KeyBackspace})
	a = model.(App)
	if got := a.help.input; got != "run ps" {
		t.Fatalf("expected backspace to trim filter input, got %q", got)
	}
}

func TestUpdateHelpEnterDoesNotCloseModalYet(t *testing.T) {
	a := App{help: func() *helpModal { m := newHelpModal(); return &m }()}
	model, _ := a.updateHelp(tea.KeyMsg{Type: tea.KeyEnter})
	a = model.(App)
	if a.help.detail == nil {
		t.Fatal("expected enter to open a help topic detail")
	}
}

func TestUpdateHelpEscClosesHelpModalFromDetail(t *testing.T) {
	a := App{help: func() *helpModal { m := newHelpModal(); return &m }()}
	a.help.SetFilter("run ps1")
	a.help.OpenSelected()
	model, _ := a.updateHelp(tea.KeyMsg{Type: tea.KeyEsc})
	a = model.(App)
	if a.help != nil {
		t.Fatal("expected esc to close the help modal from detail view")
	}
}
