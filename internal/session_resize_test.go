package internal

import "testing"

func TestShouldSendPTYResizeRequiresPTYUpgrade(t *testing.T) {
	if shouldSendPTYResize(nil) {
		t.Fatal("expected nil session to avoid PTY resize")
	}
	s := &SessionInfo{relayActive: true}
	if shouldSendPTYResize(s) {
		t.Fatal("expected session without handler/PTy to avoid PTY resize")
	}
}

func TestShouldSendRelayCleanupCtrlCRequiresPTYUpgrade(t *testing.T) {
	if shouldSendRelayCleanupCtrlC(nil) {
		t.Fatal("expected nil session to avoid relay cleanup ctrl-c")
	}
	s := &SessionInfo{relayActive: false}
	if shouldSendRelayCleanupCtrlC(s) {
		t.Fatal("expected session without PTY to avoid relay cleanup ctrl-c")
	}
}
