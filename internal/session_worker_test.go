package internal

import "testing"

func TestVisibleSessionIDAssignmentSkipsWorkers(t *testing.T) {
	m := NewManager()

	firstVisible := m.nextVisibleSessionID(false)
	workerVisibleID := m.nextVisibleSessionID(true)
	secondVisible := m.nextVisibleSessionID(false)

	if firstVisible != 1 {
		t.Fatalf("expected first visible id 1, got %d", firstVisible)
	}
	if workerVisibleID != 0 {
		t.Fatalf("expected worker visible id 0, got %d", workerVisibleID)
	}
	if secondVisible != 2 {
		t.Fatalf("expected second visible id 2, got %d", secondVisible)
	}
}
