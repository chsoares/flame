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

func TestRunScriptInMemoryUsesCancelableContextForUpload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sh")
	data := make([]byte, 512*1024)
	for i := range data {
		data[i] = 'a'
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write temp script: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := &SessionInfo{
		Conn:     &slowWriteTimeoutConn{},
		Platform: "linux",
		Handler:  &Handler{},
	}

	done := make(chan error, 1)
	go func() { done <- s.RunScriptInMemory(ctx, path, nil) }()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "context canceled") {
			t.Fatalf("expected cancellation error, got %v", err)
		}
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("expected RunScriptInMemory to stop after cancellation")
	}
}

type slowWriteTimeoutConn struct{}

func (c *slowWriteTimeoutConn) Read([]byte) (int, error)         { return 0, readTimeoutErr{} }
func (c *slowWriteTimeoutConn) Write([]byte) (int, error)        { return 0, readTimeoutErr{} }
func (c *slowWriteTimeoutConn) Close() error                     { return nil }
func (c *slowWriteTimeoutConn) LocalAddr() net.Addr              { return dummyTestAddr("local") }
func (c *slowWriteTimeoutConn) RemoteAddr() net.Addr             { return dummyTestAddr("remote") }
func (c *slowWriteTimeoutConn) SetDeadline(time.Time) error      { return nil }
func (c *slowWriteTimeoutConn) SetReadDeadline(time.Time) error  { return nil }
func (c *slowWriteTimeoutConn) SetWriteDeadline(time.Time) error { return nil }

type dummyTestAddr string

func (a dummyTestAddr) Network() string { return string(a) }
func (a dummyTestAddr) String() string  { return fmt.Sprintf("%s", string(a)) }
