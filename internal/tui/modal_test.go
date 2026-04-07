package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderModalShellKeepsFixedHeightWithWrappedLines(t *testing.T) {
	base := strings.Repeat("x", 80*24)
	short := RenderModalShell(base, 80, 24, ModalShell{
		Title: "help",
		Body:  []string{"short line"},
		Width: 52,
	})
	wrapped := RenderModalShell(base, 80, 24, ModalShell{
		Title: "help",
		Body:  []string{"this is a very long line that should wrap in the modal body and not change the shell height"},
		Width: 52,
	})

	if short == wrapped {
		t.Fatal("expected wrapped and short modal renders to differ")
	}
	if strings.Count(ansi.Strip(short), "\n") != strings.Count(ansi.Strip(wrapped), "\n") {
		t.Fatalf("expected same number of visible lines, got short=%d wrapped=%d", strings.Count(ansi.Strip(short), "\n"), strings.Count(ansi.Strip(wrapped), "\n"))
	}
}
