package tui

import (
	"strings"
	"testing"
)

func TestExtractTrailingShellPrompt(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "powershell prompt", in: "PS C:\\Users\\svc> ", want: "PS C:\\Users\\svc>"},
		{name: "cmd prompt", in: "C:\\Users\\svc> ", want: "C:\\Users\\svc>"},
		{name: "bash prompt", in: "svc@host:~$ ", want: "svc@host:~$"},
		{name: "no prompt", in: "whoami\nsvc\\user\n", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTrailingShellPrompt(tt.in)
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestFormatLocalShellEcho(t *testing.T) {
	got := formatLocalShellEcho("PS C:\\Users\\svc>", "whoami")
	want := "PS C:\\Users\\svc> whoami\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestApplyLocalShellEchoCompletesTrailingPrompt(t *testing.T) {
	content := "PS C:\\Users\\svc>"
	got := applyLocalShellEcho(content, "PS C:\\Users\\svc>", "whoami")
	want := "PS C:\\Users\\svc> whoami\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestApplyLocalShellEchoCompletesTrailingPromptWithTrailingSpace(t *testing.T) {
	content := "PS C:\\Users\\svc> "
	got := applyLocalShellEcho(content, "PS C:\\Users\\svc>", "whoami")
	want := "PS C:\\Users\\svc> whoami\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestApplyLocalShellEchoAppendsWhenNoTrailingPrompt(t *testing.T) {
	content := "output line\n"
	got := applyLocalShellEcho(content, "PS C:\\Users\\svc>", "whoami")
	want := "output line\nPS C:\\Users\\svc> whoami\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestExpandModuleSourceArg(t *testing.T) {
	args := []string{"~/Lab/tmp/test.ps1", "foo"}
	got := expandModuleSourceArg("ps1", args)
	if got[0] == args[0] || !strings.Contains(got[0], "/Lab/tmp/test.ps1") {
		t.Fatalf("expected first arg expanded, got %v", got)
	}
	if got[1] != "foo" {
		t.Fatalf("expected non-source args unchanged, got %v", got)
	}
}

func TestShouldLocallyEchoShellCommand(t *testing.T) {
	if !shouldLocallyEchoShellCommand("windows", "unknown") {
		t.Fatal("expected windows shell to use local echo")
	}
	if shouldLocallyEchoShellCommand("linux", "unknown") {
		t.Fatal("expected linux PTY shell to avoid local echo")
	}
	if shouldLocallyEchoShellCommand("macos", "unknown") {
		t.Fatal("expected macos PTY shell to avoid local echo")
	}
	if shouldLocallyEchoShellCommand("windows", "csharp") {
		t.Fatal("expected csharp windows shell to avoid local echo")
	}
}

func TestShouldDetachForBangUse(t *testing.T) {
	if !shouldDetachForBangUse(ContextShell) {
		t.Fatal("expected shell context to detach before bang use")
	}
	if shouldDetachForBangUse(ContextMenu) {
		t.Fatal("expected menu context to avoid detach before bang use")
	}
}
