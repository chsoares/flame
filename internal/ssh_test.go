package internal

import (
	"strings"
	"testing"
	"time"

	"github.com/chsoares/flame/internal/ui"
)

func TestBuildSSHCommandPasswordMode(t *testing.T) {
	connector := NewSSHConnector("10.10.14.2", 4444)
	connector.lookPath = func(file string) (string, error) {
		if file == "sshpass" {
			return "/usr/bin/sshpass", nil
		}
		return "", nil
	}

	cmd, err := connector.buildCommand(SSHArgs{Target: "user@host", Password: "secret", Port: 22})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(cmd.Args, " ")
	checks := []string{"sshpass", "-e", "ssh", "-T", "-p 22", "-o StrictHostKeyChecking=accept-new", "user@host"}
	for _, check := range checks {
		if !strings.Contains(joined, check) {
			t.Fatalf("expected %q in %q", check, joined)
		}
	}
	if got := cmd.Env[len(cmd.Env)-1]; got != "SSHPASS=secret" {
		t.Fatalf("expected SSHPASS env, got %q", got)
	}
}

func TestBuildSSHCommandKeyModeWithCustomPort(t *testing.T) {
	connector := NewSSHConnector("10.10.14.2", 4444)
	cmd, err := connector.buildCommand(SSHArgs{Target: "user@host", KeyPath: "id_rsa", Port: 2222})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(cmd.Args, " ")
	checks := []string{"ssh", "-T", "-i id_rsa", "-p 2222", "-o StrictHostKeyChecking=accept-new", "user@host"}
	for _, check := range checks {
		if !strings.Contains(joined, check) {
			t.Fatalf("expected %q in %q", check, joined)
		}
	}
}

func TestWaitForNewSessionTimesOutWhenCountDoesNotChange(t *testing.T) {
	err := waitForSessionCount(func() int { return 1 }, 1, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestSSHErrorMessageDoesNotAddTrailingNewline(t *testing.T) {
	errMsg := ui.Error("ssh timed out waiting for reverse shell")
	if strings.HasSuffix(errMsg, "\n") {
		t.Fatalf("expected no trailing newline, got %q", errMsg)
	}
}
