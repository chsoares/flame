package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	internal "github.com/chsoares/flame/internal"
)

func TestMenuPromptUsesFlameBrand(t *testing.T) {
	prompt := menuPrompt(0)
	if !strings.Contains(prompt, "flame") {
		t.Fatalf("expected flame in prompt, got %q", prompt)
	}
	if strings.Contains(prompt, "gummy") {
		t.Fatalf("did not expect legacy gummy prompt, got %q", prompt)
	}
}

func TestRenderBannerCompactUsesFlameBrand(t *testing.T) {
	banner := renderBannerCompact(40)
	if !strings.Contains(banner, "flame") {
		t.Fatalf("expected flame in compact banner, got %q", banner)
	}
	if strings.Contains(banner, "gummy") {
		t.Fatalf("did not expect legacy gummy brand in compact banner, got %q", banner)
	}
}

func TestRenderLogoUsesFlameAscii(t *testing.T) {
	logo := renderLogo()
	if len(logo) != 3 {
		t.Fatalf("expected 3 logo lines, got %d", len(logo))
	}
	if !strings.Contains(logo[0], "▀▀▀▀▀") {
		t.Fatalf("expected flame ascii in first line, got %q", logo[0])
	}
}

func TestRenderBannerFullPlacesVersionOnShellHandlerLine(t *testing.T) {
	raw := (App{width: 80}).padViewLines(renderBannerFull(80))
	if !strings.Contains(raw, styleMuted.Render(" shell handler")) {
		t.Fatalf("expected shell handler line to be gray, got %q", raw)
	}
	if !strings.Contains(raw, styleMuted.Render(internal.Version)) {
		t.Fatalf("expected version to be gray, got %q", raw)
	}
	view := ansi.Strip(raw)
	lines := strings.Split(view, "\n")
	line := strings.TrimRight(lines[2], " ")
	if got, want := lipgloss.Width(line), lipgloss.Width(renderLogo()[0]); got != want {
		t.Fatalf("expected shell handler line width %d, got %d (%q)", want, got, line)
	}
	if !strings.Contains(line, "shell handler") {
		t.Fatalf("expected shell handler line, got %q", line)
	}
	if !strings.HasSuffix(line, internal.Version) {
		t.Fatalf("expected version %q at the end of the shell handler line, got %q", internal.Version, line)
	}
	if strings.Contains(line, "╱") {
		t.Fatalf("did not expect hatching on the full banner shell handler line, got %q", line)
	}
}

func TestRenderBannerSplashPlacesVersionOnShellHandlerLine(t *testing.T) {
	raw := (App{width: 80}).padViewLines(renderBannerSplash(80))
	if !strings.Contains(raw, styleMuted.Render(" shell handler")) {
		t.Fatalf("expected shell handler line to be gray, got %q", raw)
	}
	if !strings.Contains(raw, styleMuted.Render(internal.Version)) {
		t.Fatalf("expected version to be gray, got %q", raw)
	}
	view := ansi.Strip(raw)
	lines := strings.Split(view, "\n")
	line := strings.TrimSpace(lines[0])
	if !strings.Contains(line, "shell handler") {
		t.Fatalf("expected shell handler line, got %q", line)
	}
	idx := strings.Index(line, internal.Version)
	if idx == -1 {
		t.Fatalf("expected version %q in line, got %q", internal.Version, line)
	}
	after := line[idx+len(internal.Version):]
	if !strings.HasPrefix(after, " ╱") {
		t.Fatalf("expected a space after version before right hatching, got %q", line)
	}
}
