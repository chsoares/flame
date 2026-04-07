package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestHeaderUsesSingularSessionLabel(t *testing.T) {
	h := NewHeader("10.0.0.1:4444")
	h.Width = 80
	h.SessionCount = 1
	view := ansi.Strip(h.View())
	if !strings.Contains(view, "1 session") {
		t.Fatalf("expected singular session label, got %q", view)
	}
	if strings.Contains(view, "1 sessions") {
		t.Fatalf("did not expect plural label for one session, got %q", view)
	}
}

func TestHeaderUsesPluralSessionsLabel(t *testing.T) {
	h := NewHeader("10.0.0.1:4444")
	h.Width = 80
	h.SessionCount = 2
	view := ansi.Strip(h.View())
	if !strings.Contains(view, "2 sessions") {
		t.Fatalf("expected plural sessions label, got %q", view)
	}
}
