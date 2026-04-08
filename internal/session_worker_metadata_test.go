package internal

import "testing"

func TestConsumePendingWorkerMetadataFallsBackSafely(t *testing.T) {
	m := NewManager()
	platform, whoami := m.consumePendingWorkerMetadata()
	if platform != "unknown" {
		t.Fatalf("expected unknown platform fallback, got %q", platform)
	}
	if whoami != "unknown" {
		t.Fatalf("expected unknown whoami fallback, got %q", whoami)
	}
}

func TestConsumePendingWorkerMetadataUsesParentSessionMetadata(t *testing.T) {
	m := NewManager()
	m.setPendingWorkerMetadata(&SessionInfo{Platform: "linux", Whoami: "chsoares@archlinux", ShellFlavor: "csharp"})

	platform, whoami := m.consumePendingWorkerMetadata()
	if platform != "linux" {
		t.Fatalf("expected inherited platform, got %q", platform)
	}
	if whoami != "chsoares@archlinux" {
		t.Fatalf("expected inherited whoami, got %q", whoami)
	}
}
