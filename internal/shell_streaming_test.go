package internal

import "testing"

func TestFilterStreamingChunkRemovesTrailingPowerShellPrompt(t *testing.T) {
	chunk := "line1\nPS C:\\Users\\svc> "
	got := filterStreamingChunk(chunk, "__MARK__")
	want := "line1\n"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFilterStreamingChunkKeepsPowerShellPromptWithCommand(t *testing.T) {
	chunk := "PS C:\\Users\\svc> whoami\nsvc\\user\n"
	got := filterStreamingChunk(chunk, "__MARK__")
	if got != chunk {
		t.Fatalf("expected chunk unchanged, got %q", got)
	}
}
