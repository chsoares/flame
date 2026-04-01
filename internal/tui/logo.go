package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ASCII art "GUMMY" in 3-line block characters.
// The U is stretched with "eyes" (▀ ▀) on line 2 — the wide ▀▀▀▀▀ bottom
// forms a smile, making the U look like a happy face :)
var logoLetters = [5][3]string{
	// G
	{" ▄▀▀▀", " █ ▀█", "  ▀▀▀"},
	// U (stretched, with eyes on line 2)
	{" █     █", " █ ▀ ▀ █", "  ▀▀▀▀▀ "},
	// M
	{" █▄ ▄█", " █ ▀ █", " ▀   ▀"},
	// M
	{" █▄ ▄█", " █ ▀ █", " ▀   ▀"},
	// Y
	{" █   █", "  ▀█▀ ", "   ▀  "},
}

// renderLogo returns the 3 lines of the GUMMY ASCII art (plain, no color).
func renderLogo() []string {
	lines := make([]string, 3)
	for line := 0; line < 3; line++ {
		for _, letter := range logoLetters {
			lines[line] += letter[line]
		}
	}
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

// renderLogoLine2 builds the second line of the logo with cyan eyes in the U.
func renderLogoLine2() string {
	eyeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	// G line 2 + U line 2 (with eyes) + M + M + Y line 2
	return styleMagentaBold.Render(" █ ▀█") +
		styleMagentaBold.Render(" █ ") + eyeStyle.Render("▀") + styleMagentaBold.Render(" ") + eyeStyle.Render("▀") + styleMagentaBold.Render(" █") +
		styleMagentaBold.Render(" █ ▀ █") +
		styleMagentaBold.Render(" █ ▀ █") +
		styleMagentaBold.Render("  ▀█▀ ")
}

// renderBannerFull renders the full banner inspired by Crush's layout:
//
//	╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
//	╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
//	 shell handler 󰗣
//	 ▄▀▀▀ █     █ █▄ ▄█ █▄ ▄█ █   █
//	 █ ▀█ █ ▀ ▀ █ █ ▀ █ █ ▀ █  ▀█▀
//	  ▀▀▀  ▀▀▀▀▀  ▀   ▀ ▀   ▀   ▀
//	╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
func renderBannerFull(width int) string {
	logo := renderLogo()
	hatchRow := hatching(width)

	var lines []string

	// Top hatching (2 rows — more weight on top like Crush)
	lines = append(lines, hatchRow)
	lines = append(lines, hatchRow)

	// Text line
	label := styleMuted.Render(" shell handler")
	lines = append(lines, label)

	// Logo lines
	// Line 0 and 2: pure magenta bold
	// Line 1: has pre-colored cyan eyes, render each letter part separately
	lines = append(lines, styleMagentaBold.Render(logo[0]))
	lines = append(lines, renderLogoLine2())
	lines = append(lines, styleMagentaBold.Render(logo[2]))

	// Bottom hatching (1 row)
	lines = append(lines, hatchRow)

	return strings.Join(lines, "\n")
}

// renderBannerCompact renders the compact 1-line banner for smaller sidebar.
// Format: gummy 󰗣 ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
func renderBannerCompact(width int) string {
	droplet := "󰗣"
	logo := styleMagentaBold.Render("gummy " + droplet)
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
