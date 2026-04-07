package internal

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// PTYUpgrader gerencia upgrade de shells raw para PTY
type PTYUpgrader struct {
	conn         net.Conn
	sessionID    string
	stopResize   chan struct{} // Signal to stop resize handler goroutine
	overrideCols int           // If > 0, use this width instead of auto-detecting
	overrideRows int           // If > 0, use this height instead of auto-detecting
}

// NewPTYUpgrader cria um novo upgrader de PTY
func NewPTYUpgrader(conn net.Conn, sessionID string) *PTYUpgrader {
	return &PTYUpgrader{
		conn:      conn,
		sessionID: sessionID,
	}
}

// TryUpgrade tenta fazer upgrade da shell para PTY
func (p *PTYUpgrader) TryUpgrade() error {
	// Detecta shell type primeiro (silencioso)
	shellType, err := p.detectShell()
	if err != nil {
		return fmt.Errorf("failed to detect shell: %w", err)
	}

	// Tenta upgrade baseado no shell type
	switch shellType {
	case "bash", "sh", "dash":
		return p.upgradeBashShell()
	case "python", "python3":
		return p.upgradePythonShell()
	default:
		return p.upgradeGenericShell()
	}
}

// detectShell tenta detectar o tipo de shell
func (p *PTYUpgrader) detectShell() (string, error) {
	// Envia comando para detectar shell
	commands := []string{
		"echo $0",       // Detecta shell atual
		"which python3", // Verifica python3
		"which python",  // Verifica python
		"ps -p $$",      // Mostra processo atual
	}

	for _, cmd := range commands {
		p.conn.Write([]byte(cmd + "\n"))
		time.Sleep(100 * time.Millisecond)
	}

	// Lê resposta
	buffer := make([]byte, 4096)
	p.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := p.conn.Read(buffer)
	if err != nil {
		return "unknown", err
	}
	p.conn.SetReadDeadline(time.Time{})

	response := strings.ToLower(string(buffer[:n]))

	// Analisa resposta para detectar shell
	if strings.Contains(response, "python3") {
		return "python3", nil
	}
	if strings.Contains(response, "python") {
		return "python", nil
	}
	if strings.Contains(response, "bash") {
		return "bash", nil
	}
	if strings.Contains(response, "/sh") {
		return "sh", nil
	}

	return "bash", nil // Default fallback
}

// upgradePythonShell faz upgrade usando Python PTY
func (p *PTYUpgrader) upgradePythonShell() error {
	// Comandos PTY upgrade com Python (silenciosos)
	pythonCommands := []string{
		// Primeiro, tenta python3
		"python3 -c \"import pty; pty.spawn('/bin/bash')\"",
		// Fallback para python2
		"python -c \"import pty; pty.spawn('/bin/bash')\"",
	}

	for _, cmd := range pythonCommands {
		p.conn.Write([]byte(cmd + "\r\n"))
		time.Sleep(200 * time.Millisecond)

		// Testa se PTY está funcionando
		if p.testPTY() {
			return p.completePTYSetup()
		}
	}

	return fmt.Errorf("python PTY upgrade failed")
}

// upgradeBashShell faz upgrade usando recursos nativos do bash
func (p *PTYUpgrader) upgradeBashShell() error {
	// Comandos para criar script PTY upgrade (silenciosos)
	scriptCommands := []string{
		// Cria script temporário
		"echo '#!/bin/bash' > /tmp/pty_upgrade.sh",
		"echo 'script -qc /bin/bash /dev/null' >> /tmp/pty_upgrade.sh",
		"chmod +x /tmp/pty_upgrade.sh",
		"/tmp/pty_upgrade.sh",
	}

	for _, cmd := range scriptCommands {
		p.conn.Write([]byte(cmd + "\r\n"))
		time.Sleep(150 * time.Millisecond)
	}

	// Testa se funcionou
	if p.testPTY() {
		return p.completePTYSetup()
	}

	return fmt.Errorf("bash script PTY upgrade failed")
}

// upgradeGenericShell tenta métodos genéricos de upgrade
func (p *PTYUpgrader) upgradeGenericShell() error {
	// Lista de comandos PTY upgrade alternativos (silenciosos)
	genericCommands := []string{
		// Script command (disponível na maioria dos sistemas)
		"script -qc /bin/bash /dev/null",
		// Socat (se disponível)
		"socat - EXEC:'/bin/bash',pty,stderr,setsid,sigint,sane",
		// Busybox (em sistemas embarcados)
		"busybox script -qc /bin/bash /dev/null",
	}

	for _, cmd := range genericCommands {
		p.conn.Write([]byte(cmd + "\r\n"))
		time.Sleep(200 * time.Millisecond)

		if p.testPTY() {
			return p.completePTYSetup()
		}
	}

	return fmt.Errorf("generic PTY upgrade failed - raw shell will be used")
}

// testPTY testa se PTY upgrade foi bem-sucedido
func (p *PTYUpgrader) testPTY() bool {
	// Envia comando de teste silencioso.
	// On a real PTY, stty succeeds with no output; on a raw shell,
	// it typically prints a tty-related error we can detect.
	testCmd := "stty -echo; stty echo\n"
	p.conn.Write([]byte(testCmd))

	// Aguarda resposta
	buffer := make([]byte, 1024)
	p.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err := p.conn.Read(buffer)
	p.conn.SetReadDeadline(time.Time{})

	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return true
		}
		return false
	}

	response := string(buffer[:n])
	return !strings.Contains(strings.ToLower(response), "inappropriate ioctl") &&
		!strings.Contains(strings.ToLower(response), "not a tty") &&
		!strings.Contains(strings.ToLower(response), "bad file descriptor")
}

// completePTYSetup completa configuração PTY
func (p *PTYUpgrader) completePTYSetup() error {
	// Obter dimensões do terminal local
	width, height := p.getTerminalSize()

	// Limpar output anterior enviando vários enters
	p.conn.Write([]byte("\n\n\n"))
	time.Sleep(100 * time.Millisecond)

	// Configurar PTY na shell remota (silenciosamente)
	setupCommands := []string{
		fmt.Sprintf("stty rows %d cols %d", height, width),
		"export TERM=xterm-256color",
		"export SHELL=/bin/bash",
		"stty echo", // IMPORTANTE: Manter echo habilitado para shells interativas
		"clear",     // Limpar tela
	}

	for _, cmd := range setupCommands {
		p.conn.Write([]byte(cmd + "\r\n"))
		time.Sleep(30 * time.Millisecond)
	}

	// Aguarda comandos terminarem
	time.Sleep(200 * time.Millisecond)

	return nil
}

// SetSize overrides the terminal size used for PTY setup (for TUI mode).
func (p *PTYUpgrader) SetSize(cols, rows int) {
	p.overrideCols = cols
	p.overrideRows = rows
}

// getTerminalSize obtém dimensões do terminal local
func (p *PTYUpgrader) getTerminalSize() (int, int) {
	// Use override if set (TUI mode)
	if p.overrideCols > 0 && p.overrideRows > 0 {
		return p.overrideCols, p.overrideRows
	}

	// Usa stty para obter dimensões do terminal local
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	output, err := cmd.Output()
	if err != nil {
		return 80, 24 // Default fallback
	}

	var height, width int
	fmt.Sscanf(string(output), "%d %d", &height, &width)

	if width == 0 || height == 0 {
		return 80, 24 // Default fallback
	}

	return width, height
}

// SetupResizeHandler listens for SIGWINCH and sends stty resize commands to remote shell
func (p *PTYUpgrader) SetupResizeHandler() {
	p.stopResize = make(chan struct{})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)

	go func() {
		defer signal.Stop(sigChan)

		for {
			select {
			case <-p.stopResize:
				return
			case <-sigChan:
				width, height := p.getTerminalSize()
				cmd := fmt.Sprintf("stty rows %d cols %d\n", height, width)
				p.conn.Write([]byte(cmd))
			}
		}
	}()
}

// StopResizeHandler stops the SIGWINCH handler goroutine
func (p *PTYUpgrader) StopResizeHandler() {
	if p.stopResize != nil {
		close(p.stopResize)
	}
}
