package internal

import (
	"context"
	"testing"
	"time"
)

func TestWaitForTransferCtxCancels(t *testing.T) {
	fs := NewFileServer("/tmp", 8080)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := fs.WaitForTransferCtx(ctx, "file.txt", time.Second, nil)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
