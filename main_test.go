package main

import (
	"strings"
	"testing"

	"github.com/chsoares/flame/internal"
)

func TestTitleCaseFirst(t *testing.T) {
	if got := titleCaseFirst("interface 'enp34s' not found"); !strings.HasPrefix(got, "Interface") {
		t.Fatalf("expected title-cased first word, got %q", got)
	}
}

func TestRenderStartupSplashUsesSplashBanner(t *testing.T) {
	got := renderStartupSplash()
	if strings.Contains(got, "╭") || strings.Contains(got, "╰") {
		t.Fatalf("expected splash banner, got %q", got)
	}
}

func TestStartupOutputNoLeftPaddingForInterfaces(t *testing.T) {
	got := internal.FormatInterfaceList()
	if !strings.Contains(got, "  ") {
		t.Fatalf("expected interface list formatting to remain indented, got %q", got)
	}
}
