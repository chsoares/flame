package internal

import (
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
	cmd := buildWindowsPowerShellB64Command("gummy_ps_var", []string{"foo", "bar baz"})
	checks := []string{
		"FromBase64String($gummy_ps_var)",
		"[scriptblock]::Create($decoded)",
		"& $sb 'foo' 'bar baz'",
		"Remove-Variable -Name gummy_ps_var",
	}
	for _, check := range checks {
		if !strings.Contains(cmd, check) {
			t.Fatalf("expected command to contain %q, got: %s", check, cmd)
		}
	}
}
