package tui

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// copyToClipboard writes text to the system clipboard using multiple strategies:
// 1. OSC 52 escape sequence (works over SSH, in tmux, most modern terminals)
// 2. Native clipboard tool (wl-copy for Wayland, xclip/xsel for X11)
func copyToClipboard(text string) {
	// Strategy 1: OSC 52 — write directly to terminal
	// This works in kitty, alacritty, foot, tmux (with set-clipboard on), etc.
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	fmt.Fprintf(os.Stderr, "\033]52;c;%s\033\\", encoded)

	// Strategy 2: Native clipboard tool as fallback
	// Try in order: wl-copy (Wayland), xclip (X11), xsel (X11)
	if path, err := exec.LookPath("wl-copy"); err == nil {
		cmd := exec.Command(path)
		cmd.Stdin = strings.NewReader(text)
		_ = cmd.Run()
		return
	}
	if path, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command(path, "-selection", "clipboard")
		cmd.Stdin = strings.NewReader(text)
		_ = cmd.Run()
		return
	}
	if path, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command(path, "--clipboard", "--input")
		cmd.Stdin = strings.NewReader(text)
		_ = cmd.Run()
		return
	}
}
