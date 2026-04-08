package internal

import (
	"testing"
	"time"
)

func TestFindNewestWorkerSessionPrefersWorkerOverVisibleNumID(t *testing.T) {
	now := time.Now()
	sessions := map[string]*SessionInfo{
		"visible": {NumID: 3, CreatedAt: now.Add(2 * time.Second), isWorker: false},
		"worker":  {NumID: 0, CreatedAt: now.Add(1 * time.Second), isWorker: true},
	}

	got := findNewestWorkerSession(sessions)
	if got == nil {
		t.Fatal("expected worker session, got nil")
	}
	if !got.isWorker {
		t.Fatal("expected selected session to be worker")
	}
}

func TestShouldUseWorkerForSpawn(t *testing.T) {
	if shouldUseWorkerForSpawn("linux") {
		t.Fatal("expected linux spawn to avoid worker path")
	}
	if !shouldUseWorkerForSpawn("windows") {
		t.Fatal("expected windows spawn to use worker path")
	}
}
