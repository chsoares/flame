package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var flameLogo = []string{
	"▀▀▀▀▀ ▀▀     ▀▀▀▀  ▀▀   ▀▀ ▀▀▀▀▀ ",
	"██▀▀  ██    ██▀▀██ ██▀▄▀██ ██▀▀  ",
	"▀▀    ▀▀▀▀▀ ▀▀  ▀▀ ▀▀   ▀▀ ▀▀▀▀▀ ",
}

// renderLogo returns the 3 lines of the FLAME ASCII art (plain, no color).
func renderLogo() []string {
	lines := append([]string(nil), flameLogo...)
	// Pad all lines to same width
	maxW := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > maxW {
			maxW = w
		}
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w < maxW {
			lines[i] = l + strings.Repeat(" ", maxW-w)
		}
	}
	return lines
}

// renderBannerFull renders the full banner inspired by Crush's layout:
//
//	╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
//	╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
//	 shell handler 
//	 ▀▀▀▀▀ ▀▀     ▀▀▀▀  ▀▀   ▀▀ ▀▀▀▀▀
//	 ██▀▀  ██    ██▀▀██ ██▀▄▀██ ██▀▀
//	 ▀▀    ▀▀▀▀▀ ▀▀  ▀▀ ▀▀   ▀▀ ▀▀▀▀▀
//	╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
func renderBannerFull(width int) string {
	logo := renderLogo()
	hatchRow := hatching(width)

	var lines []string

	// Top hatching (2 rows — more weight on top like Crush)
	lines = append(lines, hatchRow)
	lines = append(lines, hatchRow)

	// Text line
	label := " " + styleMuted.Render("shell handler")
	lines = append(lines, label)

	for _, line := range logo {
		lines = append(lines, " "+styleMagentaBold.Render(line))
	}

	// Bottom hatching (1 row)
	lines = append(lines, hatchRow)

	return strings.Join(lines, "\n")
}

// renderBannerCompact renders the compact 1-line banner for smaller sidebar.
// Format: flame  ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
func renderBannerCompact(width int) string {
	fire := ""
	logo := styleMagentaBold.Render("flame " + fire)
	logoW := lipgloss.Width(logo)
	hatchW := width - logoW - 1
	if hatchW < 0 {
		hatchW = 0
	}
	return logo + " " + hatching(hatchW)
}

// bannerFullHeight returns how many lines the full banner takes.
func bannerFullHeight() int {
	return 7 // 2 top hatch + 1 text + 3 logo + 1 bottom hatch
}

// renderBannerSplash renders the splash screen banner — hatching only on the sides of the logo.
// Like Crush's landing: logo left, hatching extends right. No top/bottom hatch rows.
//
//	shell handler
//	▄▀▀▀ █     █ █▄ ▄█ █▄ ▄█ █   █ ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
//	█ ▀█ █ ▀ ▀ █ █ ▀ █ █ ▀ █  ▀█▀  ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
//	 ▀▀▀  ▀▀▀▀▀  ▀   ▀ ▀   ▀   ▀   ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
func renderBannerSplash(width int) string {
	logo := renderLogo()

	leftHatch := 3
	var lines []string

	// Text line: left hatching + label + space pad to align with logo end + right hatching
	logoWidth := 0
	for _, line := range logo {
		if w := lipgloss.Width(line); w > logoWidth {
			logoWidth = w
		}
	}
	left := hatching(leftHatch)
	label := left + " " + styleMuted.Render("shell handler")
	labelW := lipgloss.Width(label)
	// Pad so right hatching starts at same position as logo lines
	logoEnd := leftHatch + 1 + logoWidth // left pad + logo
	padW := logoEnd - labelW
	if padW < 1 {
		padW = 1
	}
	rightFill := width - logoEnd
	if rightFill < 0 {
		rightFill = 0
	}
	lines = append(lines, label+strings.Repeat(" ", padW)+hatching(rightFill))

	// Logo lines with hatching on both sides
	for _, logoLine := range logo {
		rendered := styleMagentaBold.Render(logoLine)
		renderedW := lipgloss.Width(rendered)
		fillW := width - leftHatch - 1 - renderedW
		if fillW < 0 {
			fillW = 0
		}
		lines = append(lines, left+" "+rendered+hatching(fillW))
	}

	return strings.Join(lines, "\n")
}

// RenderExitBanner generates a banner for display after the TUI exits.
func RenderExitBanner(width int) string {
	return renderBannerSplash(width)
}
