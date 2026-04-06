package internal

import "testing"

func TestShouldAttemptPTYUpgradeSkipsSSHFlavor(t *testing.T) {
	if shouldAttemptPTYUpgrade("linux", "ssh") {
		t.Fatal("expected ssh flavor to skip PTY upgrade")
	}
}

func TestShouldAttemptPTYUpgradeAllowsLinuxNonSSH(t *testing.T) {
	if !shouldAttemptPTYUpgrade("linux", "unknown") {
		t.Fatal("expected linux non-ssh session to attempt PTY upgrade")
	}
}

func TestIsSttyEchoSuppressesPTYProbeOutput(t *testing.T) {
	if !isSttyEcho([]byte("stty -echo; echo PTY_TEST_OK; stty echo\n")) {
		t.Fatal("expected PTY probe output to be treated as suppressible noise")
	}
}

func TestIsSttyEchoSuppressesStandalonePTYProbeMarker(t *testing.T) {
	if !isSttyEcho([]byte("PTY_TEST_OK\n")) {
		t.Fatal("expected PTY probe marker to be treated as suppressible noise")
	}
}
