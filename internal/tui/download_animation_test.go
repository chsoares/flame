package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestStatusBarDownloadViewKeepsBytesAndBar(t *testing.T) {
	s := NewStatusBar(120)
	s.TransferPct = 0
	s.TransferMsg = "Downloading passwd"
	s.TransferRight = "15.2 KB"
	s.TransferUpload = false
	s.TransferAnimating = true

	got := ansi.Strip(s.View())
	if !strings.Contains(got, "Downloading passwd") {
		t.Fatalf("expected download label, got %q", got)
	}
	if !strings.Contains(got, "15.2 KB") {
		t.Fatalf("expected bytes text to remain visible, got %q", got)
	}
	if !strings.Contains(got, "/") {
		t.Fatalf("expected animated bar characters, got %q", got)
	}
}

func TestStatusBarDownloadAnimationAdvances(t *testing.T) {
	s := NewStatusBar(120)
	s.TransferPct = 0
	s.TransferMsg = "Downloading passwd"
	s.TransferRight = "15.2 KB"
	s.TransferUpload = false
	s.TransferAnimating = true
	s.TransferAnimPhase = 0

	first := ansi.Strip(s.View())
	s.StepTransferAnimation()
	second := ansi.Strip(s.View())

	if first == second {
		t.Fatalf("expected animation to change between frames, got %q", first)
	}
	if strings.Count(first, "/") < 2 || strings.Count(second, "/") < 2 {
		t.Fatalf("expected marquee-style band, got %q and %q", first, second)
	}
}

func TestTransferProgressDownloadEnablesAnimation(t *testing.T) {
	a := New(nil, "127.0.0.1:4444")
	a.statusBar.Width = 120

	model, cmd := a.Update(transferProgressMsg{Filename: "passwd", Pct: 0, Right: "15.2 KB", Upload: false})
	app := model.(App)

	if !app.statusBar.TransferAnimating {
		t.Fatal("expected download animation to be enabled")
	}
	if app.statusBar.TransferRight != "15.2 KB" {
		t.Fatalf("expected bytes text to stay in status bar, got %q", app.statusBar.TransferRight)
	}
	if cmd == nil {
		t.Fatal("expected download progress to schedule an animation tick")
	}
	if msg := cmd(); msg != nil {
		if _, ok := msg.(tea.Msg); !ok {
			t.Fatalf("expected tea msg from animation tick command, got %T", msg)
		}
	}
}

func TestTransferProgressDownloadDoesNotStartASecondAnimationLoop(t *testing.T) {
	a := New(nil, "127.0.0.1:4444")
	a.statusBar.Width = 120

	model, cmd := a.Update(transferProgressMsg{Filename: "passwd", Pct: 0, Right: "15.2 KB", Upload: false})
	if cmd == nil {
		t.Fatal("expected first download progress to start animation")
	}

	app := model.(App)
	_, cmd = app.Update(transferProgressMsg{Filename: "passwd", Pct: 0, Right: "16.4 KB", Upload: false})
	if cmd != nil {
		t.Fatal("expected repeated download progress to reuse the existing animation loop")
	}
}
