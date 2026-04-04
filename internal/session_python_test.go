package internal

import (
	"strings"
	"testing"
)

func TestBuildUnixPythonHTTPCommand(t *testing.T) {
	cmd := buildUnixPythonHTTPCommand("https://example.test/payload.py", []string{"foo", "bar baz"})

	checks := []string{
		"curl -fsSL 'https://example.test/payload.py'",
		"python3 -c",
		"sys.argv = ['script', 'foo', 'bar baz']",
		"exec(sys.stdin.read())",
	}

	for _, check := range checks {
		if !strings.Contains(cmd, check) {
			t.Fatalf("expected command to contain %q, got: %s", check, cmd)
		}
	}
}

func TestBuildUnixPythonB64Command(t *testing.T) {
	cmd := buildUnixPythonB64Command("gummy_py_var", []string{"foo", "bar baz"})

	checks := []string{
		"echo \"$gummy_py_var\"",
		"base64 -d",
		"python3 -c",
		"sys.argv = ['script', 'foo', 'bar baz']",
		"unset gummy_py_var",
	}

	for _, check := range checks {
		if !strings.Contains(cmd, check) {
			t.Fatalf("expected command to contain %q, got: %s", check, cmd)
		}
	}
}

func TestPythonArgvLiteralEscapesQuotes(t *testing.T) {
	got := pythonArgvLiteral([]string{"simple", "has space", "O'Brien"})
	expected := "['script', 'simple', 'has space', 'O\\'Brien']"
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}
