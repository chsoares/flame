package internal

import "testing"

func TestNewTransfererUsesSafeModeForWorkerSessions(t *testing.T) {
	progressCalled := false
	doneCalled := false
	s := &SessionInfo{
		ID:               "worker-1",
		Platform:         "linux",
		isWorker:         true,
		transferProgress: func(string) { progressCalled = true },
		transferDone:     func() { doneCalled = true },
	}
	tfer := s.newTransferer()

	if !tfer.ptyUpgraded {
		t.Fatal("expected worker transferer to use safe chunk mode")
	}
	if tfer.progressFn == nil {
		t.Fatal("expected worker transferer to expose progress callback")
	}
	if tfer.doneFn == nil {
		t.Fatal("expected worker transferer to expose done callback")
	}

	tfer.progressFn("test")
	tfer.doneFn()
	if !progressCalled {
		t.Fatal("expected worker progress callback to be used")
	}
	if !doneCalled {
		t.Fatal("expected worker done callback to be used")
	}
}
