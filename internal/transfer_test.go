package internal

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDisplayTransferNameStripsTmpPrefix(t *testing.T) {
	got := displayTransferName("/tmp/binbag/tmp_payload.exe")
	if got != "payload.exe" {
		t.Fatalf("expected payload.exe, got %q", got)
	}
}

func TestDisplayTransferNameKeepsNormalName(t *testing.T) {
	got := displayTransferName("/tmp/binbag/seatbelt.exe")
	if got != "seatbelt.exe" {
		t.Fatalf("expected seatbelt.exe, got %q", got)
	}
}

func TestEffectiveUploadRemotePathUsesDisplayNameWhenEmpty(t *testing.T) {
	got := effectiveUploadRemotePath("/tmp/binbag/tmp_shell.exe", "")
	if got != "shell.exe" {
		t.Fatalf("expected shell.exe, got %q", got)
	}
}

func TestUploadToBashVariableUsesSmallChunksInSafeMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script.sh")
	data := make([]byte, 2048)
	for i := range data {
		data[i] = 'a'
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write temp script: %v", err)
	}

	conn := &recordingConn{}
	tfer := &Transferer{conn: conn, platform: "linux", ptyUpgraded: true}
	if err := tfer.UploadToBashVariable(context.Background(), path, "_flame_script_test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	chunkWrites := 0
	for _, write := range conn.writes {
		if strings.Contains(write, "_flame_script_test+=") {
			chunkWrites++
		}
	}
	if chunkWrites < 2 {
		t.Fatalf("expected multiple safe-mode chunk writes, got %d writes: %#v", chunkWrites, conn.writes)
	}
}

func TestUploadToBashVariableReportsProgressViaCallback(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script.sh")
	data := make([]byte, 2048)
	for i := range data {
		data[i] = 'a'
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write temp script: %v", err)
	}

	conn := &recordingConn{}
	var updates []string
	tfer := &Transferer{
		conn:        conn,
		platform:    "linux",
		ptyUpgraded: true,
		progressFn:  func(text string) { updates = append(updates, text) },
	}
	if err := tfer.UploadToBashVariable(context.Background(), path, "_flame_script_test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(updates) == 0 {
		t.Fatal("expected progress callback updates for in-memory upload")
	}
	if !strings.Contains(updates[0], "Loading script.sh to memory") {
		t.Fatalf("expected upload progress text, got %#v", updates)
	}
}

func TestWriteWithContextCancelsOnRepeatedWriteTimeouts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := &timeoutWriteConn{}
	done := make(chan error, 1)
	go func() {
		done <- writeWithContext(ctx, conn, []byte("payload"))
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected context cancellation, got %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected writeWithContext to stop after cancellation")
	}
}

func TestUploadToBashVariableCallsDoneFnOnCancellation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script.sh")
	data := make([]byte, 2048)
	for i := range data {
		data[i] = 'a'
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write temp script: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	doneCalled := false
	tfer := &Transferer{
		conn:       &timeoutWriteConn{},
		platform:   "linux",
		progressFn: func(string) {},
		doneFn:     func() { doneCalled = true },
	}

	result := make(chan error, 1)
	go func() {
		result <- tfer.UploadToBashVariable(ctx, path, "_flame_script_test")
	}()

	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-result:
		if err == nil || !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected cancellation error, got %v", err)
		}
		if !doneCalled {
			t.Fatal("expected done callback to fire on cancellation")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected cancellation to finish upload")
	}
}

type recordingConn struct {
	writes []string
}

func (c *recordingConn) Read([]byte) (int, error) { return 0, readTimeoutErr{} }
func (c *recordingConn) Write(p []byte) (int, error) {
	c.writes = append(c.writes, string(p))
	return len(p), nil
}
func (c *recordingConn) Close() error                     { return nil }
func (c *recordingConn) LocalAddr() net.Addr              { return dummyAddr("local") }
func (c *recordingConn) RemoteAddr() net.Addr             { return dummyAddr("remote") }
func (c *recordingConn) SetDeadline(time.Time) error      { return nil }
func (c *recordingConn) SetReadDeadline(time.Time) error  { return nil }
func (c *recordingConn) SetWriteDeadline(time.Time) error { return nil }

type readTimeoutErr struct{}

func (readTimeoutErr) Error() string   { return "timeout" }
func (readTimeoutErr) Timeout() bool   { return true }
func (readTimeoutErr) Temporary() bool { return true }

type dummyAddr string

func (a dummyAddr) Network() string { return string(a) }
func (a dummyAddr) String() string  { return fmt.Sprintf("%s", string(a)) }

type timeoutWriteConn struct{}

func (c *timeoutWriteConn) Read([]byte) (int, error)         { return 0, readTimeoutErr{} }
func (c *timeoutWriteConn) Write([]byte) (int, error)        { return 0, readTimeoutErr{} }
func (c *timeoutWriteConn) Close() error                     { return nil }
func (c *timeoutWriteConn) LocalAddr() net.Addr              { return dummyAddr("local") }
func (c *timeoutWriteConn) RemoteAddr() net.Addr             { return dummyAddr("remote") }
func (c *timeoutWriteConn) SetDeadline(time.Time) error      { return nil }
func (c *timeoutWriteConn) SetReadDeadline(time.Time) error  { return nil }
func (c *timeoutWriteConn) SetWriteDeadline(time.Time) error { return nil }
