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

func TestUploadToBashVariableUsesPTYFriendlyMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script.sh")
	data := make([]byte, 2048)
	for i := range data {
		data[i] = 'a'
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write temp script: %v", err)
	}

	conn := &recordingConn{}
	tfer := &Transferer{conn: conn, ptyUpgraded: true}
	if err := tfer.UploadToBashVariable(context.Background(), path, "_flame_script_test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(conn.writes) == 0 || conn.writes[0] != "stty -echo\n" {
		t.Fatalf("expected PTY echo suppression first, got %#v", conn.writes)
	}
	if !strings.Contains(strings.Join(conn.writes, ""), "stty echo\n") {
		t.Fatalf("expected PTY echo restoration, got %#v", conn.writes)
	}

	chunkWrites := 0
	for _, write := range conn.writes {
		if strings.Contains(write, "_flame_script_test+=") {
			chunkWrites++
		}
	}
	if chunkWrites < 2 {
		t.Fatalf("expected PTY mode to use smaller chunks, got %d chunk writes: %#v", chunkWrites, conn.writes)
	}
}

type recordingConn struct {
	writes []string
	reads  int
}

func (c *recordingConn) Read([]byte) (int, error) {
	c.reads++
	return 0, readTimeoutErr{}
}

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
