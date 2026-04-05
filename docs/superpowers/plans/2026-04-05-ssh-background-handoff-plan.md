# SSH Background Handoff Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Flame's `ssh` command run the remote handoff in the background with password or key auth, then wait for the reverse shell session with spawn-style timeout/error behavior.

**Architecture:** Extend the existing CLI/TUI command path in `internal/session.go` with an explicit `ssh` argument parser and async wait loop. Keep SSH process construction in `internal/ssh.go`, where password mode uses `sshpass` and key mode uses native `ssh -i`, so session handling can stay centralized in the manager.

**Tech Stack:** Go, existing `internal/session.go` command dispatch/completion, `os/exec`, existing session-count timeout logic, Go unit tests

---

## File Structure Map

- Modify: `internal/session.go` - parse `ssh user@host -p pass| -i key [--port N]`, start async handoff, wait for session or timeout
- Modify: `internal/ssh.go` - add parsed options, build background SSH commands for password/key auth, validate `sshpass` presence when needed
- Modify: `internal/payloads.go` - expose the bash payload used by SSH through the existing generator only if needed by tests/helpers
- Modify: `internal/payloads_test.go` - cover the new SSH arg parser behavior in a focused way if parser stays in `session.go`
- Create: `internal/ssh_test.go` - cover option parsing/building and dependency validation without real SSH execution

### Task 1: Parse the new `ssh` command shape

**Files:**
- Modify: `internal/session.go`
- Modify: `internal/payloads_test.go`

- [ ] **Step 1: Write a failing parser test for password auth**

Add to `internal/payloads_test.go`:

```go
func TestParseSSHArgsPasswordMode(t *testing.T) {
	args, err := parseSSHArgs([]string{"user@10.10.10.10", "-p", "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.Target != "user@10.10.10.10" {
		t.Fatalf("expected target, got %q", args.Target)
	}
	if args.Password != "secret" {
		t.Fatalf("expected password mode, got %+v", args)
	}
	if args.Port != 22 {
		t.Fatalf("expected default port 22, got %d", args.Port)
	}
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./internal -run TestParseSSHArgsPasswordMode -v`

Expected: FAIL with `undefined: parseSSHArgs`.

- [ ] **Step 3: Add a failing parser test for key mode with custom port**

Add to `internal/payloads_test.go`:

```go
func TestParseSSHArgsKeyModeWithPort(t *testing.T) {
	args, err := parseSSHArgs([]string{"user@10.10.10.10", "-i", "id_rsa", "--port", "2222"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.KeyPath != "id_rsa" {
		t.Fatalf("expected key path, got %+v", args)
	}
	if args.Port != 2222 {
		t.Fatalf("expected custom port, got %d", args.Port)
	}
}
```

- [ ] **Step 4: Run the focused tests and verify they fail for the missing parser**

Run: `go test ./internal -run 'TestParseSSHArgs(PasswordMode|KeyModeWithPort)' -v`

Expected: FAIL with `undefined: parseSSHArgs`.

- [ ] **Step 5: Implement the minimal parser**

Add near `parseRevArgs` in `internal/session.go`:

```go
type SSHArgs struct {
	Target   string
	Password string
	KeyPath  string
	Port     int
}

func parseSSHArgs(args []string) (SSHArgs, error) {
	if len(args) < 3 {
		return SSHArgs{}, fmt.Errorf("usage: ssh user@host (-p <password> | -i <key>) [--port <port>]")
	}

	parsed := SSHArgs{Target: args[0], Port: 22}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-p":
			i++
			if i >= len(args) {
				return SSHArgs{}, fmt.Errorf("missing value for -p")
			}
			parsed.Password = args[i]
		case "-i":
			i++
			if i >= len(args) {
				return SSHArgs{}, fmt.Errorf("missing value for -i")
			}
			parsed.KeyPath = args[i]
		case "--port":
			i++
			if i >= len(args) {
				return SSHArgs{}, fmt.Errorf("missing value for --port")
			}
			port, err := strconv.Atoi(args[i])
			if err != nil || port <= 0 {
				return SSHArgs{}, fmt.Errorf("invalid SSH port: %s", args[i])
			}
			parsed.Port = port
		default:
			return SSHArgs{}, fmt.Errorf("unknown ssh flag: %s", args[i])
		}
	}

	if (parsed.Password == "" && parsed.KeyPath == "") || (parsed.Password != "" && parsed.KeyPath != "") {
		return SSHArgs{}, fmt.Errorf("use exactly one of -p <password> or -i <key>")
	}

	return parsed, nil
}
```

- [ ] **Step 6: Re-run the parser tests**

Run: `go test ./internal -run 'TestParseSSHArgs(PasswordMode|KeyModeWithPort)' -v`

Expected: PASS.

### Task 2: Build SSH commands for password and key auth

**Files:**
- Modify: `internal/ssh.go`
- Create: `internal/ssh_test.go`

- [ ] **Step 1: Write a failing test for password mode command construction**

Create `internal/ssh_test.go` with:

```go
package internal

import (
	"strings"
	"testing"
)

func TestBuildSSHCommandPasswordMode(t *testing.T) {
	connector := NewSSHConnector("10.10.14.2", 4444)
	cmd, err := connector.buildCommand(SSHArgs{Target: "user@host", Password: "secret", Port: 22})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(cmd.Args, " ")
	checks := []string{"sshpass", "-e", "ssh", "-T", "user@host"}
	for _, check := range checks {
		if !strings.Contains(joined, check) {
			t.Fatalf("expected %q in %q", check, joined)
		}
	}
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./internal -run TestBuildSSHCommandPasswordMode -v`

Expected: FAIL with `connector.buildCommand undefined`.

- [ ] **Step 3: Add a failing test for key mode command construction**

Extend `internal/ssh_test.go` with:

```go
func TestBuildSSHCommandKeyModeWithCustomPort(t *testing.T) {
	connector := NewSSHConnector("10.10.14.2", 4444)
	cmd, err := connector.buildCommand(SSHArgs{Target: "user@host", KeyPath: "id_rsa", Port: 2222})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	joined := strings.Join(cmd.Args, " ")
	checks := []string{"ssh", "-T", "-i", "id_rsa", "-p", "2222", "user@host"}
	for _, check := range checks {
		if !strings.Contains(joined, check) {
			t.Fatalf("expected %q in %q", check, joined)
		}
	}
}
```

- [ ] **Step 4: Run the focused tests and verify they fail**

Run: `go test ./internal -run 'TestBuildSSHCommand(PasswordMode|KeyModeWithCustomPort)' -v`

Expected: FAIL with `connector.buildCommand undefined`.

- [ ] **Step 5: Implement the minimal command builder**

Add to `internal/ssh.go`:

```go
func (s *SSHConnector) buildCommand(args SSHArgs) (*exec.Cmd, error) {
	payload := s.generatePayload()
	sshArgs := []string{"-T", "-p", strconv.Itoa(args.Port), args.Target, payload}

	if args.KeyPath != "" {
		sshArgs = append([]string{"-T", "-i", args.KeyPath, "-p", strconv.Itoa(args.Port), args.Target, payload}[:0], "-T", "-i", args.KeyPath, "-p", strconv.Itoa(args.Port), args.Target, payload)
		cmd := exec.Command("ssh", sshArgs...)
		return cmd, nil
	}

	if _, err := exec.LookPath("sshpass"); err != nil {
		return nil, fmt.Errorf("sshpass not found in PATH")
	}
	cmd := exec.Command("sshpass", append([]string{"-e", "ssh"}, sshArgs...)...)
	cmd.Env = append(os.Environ(), "SSHPASS="+args.Password)
	return cmd, nil
}
```

- [ ] **Step 6: Re-run the SSH command tests**

Run: `go test ./internal -run 'TestBuildSSHCommand(PasswordMode|KeyModeWithCustomPort)' -v`

Expected: PASS on systems with `sshpass`, or FAIL with the explicit missing-binary error in password mode until the test is adjusted to stub lookup. If it fails for that reason, refactor the connector with a lookPath function field and re-run until both tests pass deterministically.

### Task 3: Run SSH handoff asynchronously with timeout behavior

**Files:**
- Modify: `internal/session.go`
- Modify: `internal/ssh.go`
- Test: `internal/ssh_test.go`

- [ ] **Step 1: Write a failing test for async timeout behavior helper**

Add to `internal/ssh_test.go`:

```go
func TestWaitForNewSessionTimesOutWhenCountDoesNotChange(t *testing.T) {
	err := waitForSessionCount(func() int { return 1 }, 1, 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
```

- [ ] **Step 2: Run the focused timeout test and verify it fails**

Run: `go test ./internal -run TestWaitForNewSessionTimesOutWhenCountDoesNotChange -v`

Expected: FAIL with `undefined: waitForSessionCount`.

- [ ] **Step 3: Implement the minimal wait helper**

Add to `internal/session.go`:

```go
func waitForSessionCount(countFn func() int, initial int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if countFn() > initial {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("ssh timed out waiting for reverse shell")
}
```

- [ ] **Step 4: Re-run the timeout test**

Run: `go test ./internal -run TestWaitForNewSessionTimesOutWhenCountDoesNotChange -v`

Expected: PASS.

- [ ] **Step 5: Wire `handleSSH` to use parser + async command + timeout**

Update the `ssh` case in `internal/session.go` to pass all args after the command, then implement:

```go
func (m *Manager) handleSSH(args []string) {
	parsed, err := parseSSHArgs(args)
	if err != nil {
		fmt.Println(ui.Error(err.Error()))
		return
	}

	sshIP := GlobalRuntimeConfig.GetPivotIP()
	if sshIP == "" || m.listenerPort == 0 {
		fmt.Println(ui.Error("No listener IP/port available"))
		return
	}

	connector := NewSSHConnector(sshIP, m.listenerPort)
	spinID := m.startSpinner(fmt.Sprintf("SSH handoff via %s...", parsed.Target))
	initialCount := m.GetSessionCount()

	go func() {
		defer m.stopSpinner(spinID)
		if err := connector.ConnectBackground(parsed); err != nil {
			m.notify(ui.Error(err.Error()) + "\n")
			m.notifyOverlay(err.Error(), 2)
			return
		}
		if err := waitForSessionCount(m.GetSessionCount, initialCount, 5*time.Second); err != nil {
			m.notify(ui.Error(err.Error()) + "\n")
			m.notifyOverlay(err.Error(), 2)
			return
		}
	}()
}
```

And add to `internal/ssh.go`:

```go
func (s *SSHConnector) ConnectBackground(args SSHArgs) error {
	cmd, err := s.buildCommand(args)
	if err != nil {
		return err
	}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	cmd.Stdin = nil
	return cmd.Start()
}
```

- [ ] **Step 6: Run the SSH-focused tests**

Run: `go test ./internal -run 'Test(ParseSSHArgs|BuildSSHCommand|WaitForNewSession)' -v`

Expected: PASS.

- [ ] **Step 7: Run the full internal package tests**

Run: `go test ./internal/...`

Expected: PASS.
