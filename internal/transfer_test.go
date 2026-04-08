package internal

import "testing"

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
