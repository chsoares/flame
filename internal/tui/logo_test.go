package tui

import (
	"strings"
	"testing"
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
