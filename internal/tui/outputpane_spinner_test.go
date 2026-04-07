package tui

import (
	"strings"
	"testing"
)

func TestStopSpinnerDoesNotLeaveBlankLine(t *testing.T) {
	op := NewOutputPane(80, 5)
	op.Append("before\n")
	op.StartSpinner(1, "SSH handoff via target...")
	op.StopSpinner(1)
	content := op.viewport.View()
	if content == "" {
		t.Fatal("expected output to remain visible")
	}
	if content[len(content)-1] == '\n' {
		t.Fatalf("expected no trailing blank line, got %q", content)
	}
}

func TestUpdateSpinnerReplacesSpinnerText(t *testing.T) {
	op := NewOutputPane(80, 5)
	op.StartSpinner(1, "SSH handoff via target...")
	op.UpdateSpinner(1, "ssh timed out waiting for reverse shell")
	content := op.viewport.View()
	if !strings.Contains(content, "ssh timed out waiting for reverse shell") {
		t.Fatalf("expected updated spinner text, got %q", content)
	}
	if strings.Contains(content, "SSH handoff via target...") {
		t.Fatalf("expected old spinner text replaced, got %q", content)
	}
}
