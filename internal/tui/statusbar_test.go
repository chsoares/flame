package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestStatusBarMenuHintsIncludeF1Help(t *testing.T) {
	s := NewStatusBar(120)
	s.Context = ContextMenu
	got := ansi.Strip(s.View())
	if strings.Contains(got, "Tab complete") {
		t.Fatalf("expected menu hints without tab complete, got %q", got)
	}
	if !strings.Contains(got, "F1 help") {
		t.Fatalf("expected menu hints to include F1 help, got %q", got)
	}
}

func TestStatusBarShellHintsIncludeF1HelpAfterBang(t *testing.T) {
	s := NewStatusBar(120)
	s.Context = ContextShell
	got := ansi.Strip(s.View())
	if !strings.Contains(got, "! flame cmd") {
		t.Fatalf("expected shell hints to start with bang command, got %q", got)
	}
	if !strings.Contains(got, "F1 help") {
		t.Fatalf("expected shell hints to include F1 help, got %q", got)
	}
	if strings.Index(got, "F1 help") > strings.Index(got, "F11 sidebar") {
		t.Fatalf("expected F1 help before F11 sidebar, got %q", got)
	}
}
