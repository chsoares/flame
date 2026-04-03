package tui

// Rect represents a rectangular area on screen.
type Rect struct {
	X, Y int // Top-left corner
	W, H int // Width and height
}

// LayoutMode represents the responsive layout tier.
type LayoutMode int

const (
	LayoutCompact LayoutMode = iota // No sidebar, 1-line header
	LayoutMedium                    // Sidebar with compact banner (1-line logo)
	LayoutFull                      // Sidebar with full ASCII art banner
)

// Layout holds the calculated positions for all TUI components.
type Layout struct {
	Header    Rect
	Output    Rect
	Sidebar   Rect
	Input     Rect
	StatusBar Rect
	Mode      LayoutMode
}

const (
	headerHeight    = 1
	statusBarHeight = 1
	inputHeight     = 1
	sidebarWidth    = 32
	minMainWidth    = 40

	// Breakpoints
	compactBreakpoint = 70                       // Below this: no sidebar
	fullBreakpoint    = 15                        // Sidebar height needed for full banner
	fullBannerLines   = 7                         // 2 top hatch + text + 3 logo + 1 bottom hatch
)

// GenerateLayout calculates component positions for the given terminal size.
func GenerateLayout(width, height int, forceCompact ...bool) Layout {
	compact := width < (minMainWidth + sidebarWidth + 3)
	if len(forceCompact) > 0 && forceCompact[0] {
		compact = true
	}

	if compact {
		// Compact: header + output + blank + input + blank + statusbar
		outputH := height - headerHeight - 1 - inputHeight - 1 - statusBarHeight
		if outputH < 1 {
			outputH = 1
		}
		return Layout{
			Mode: LayoutCompact,
			Header: Rect{
				X: 0, Y: 0,
				W: width, H: headerHeight,
			},
			Output: Rect{
				X: 0, Y: headerHeight,
				W: width, H: outputH,
			},
			Input: Rect{
				X: 0, Y: headerHeight + outputH,
				W: width, H: inputHeight,
			},
			StatusBar: Rect{
				X: 0, Y: height - statusBarHeight,
				W: width, H: statusBarHeight,
			},
		}
	}

	// Wide: no header, sidebar on right with branding
	sidebarGap := 3 // empty space between main and sidebar
	sbW := sidebarWidth
	mainW := width - sbW - sidebarGap

	// Reserve lines: 1 blank above input, 1 input, 1 blank below input, 1 status
	outputH := height - 1 - inputHeight - 1 - statusBarHeight
	if outputH < 1 {
		outputH = 1
	}

	// Determine if we have enough space for full banner
	sidebarH := outputH + inputHeight
	mode := LayoutMedium
	if sidebarH >= fullBannerLines+fullBreakpoint {
		mode = LayoutFull
	}

	return Layout{
		Mode: mode,
		// No header in wide mode (W=0)
		Header: Rect{},
		Output: Rect{
			X: 0, Y: 0,
			W: mainW, H: outputH,
		},
		Input: Rect{
			X: 0, Y: outputH,
			W: mainW, H: inputHeight,
		},
		Sidebar: Rect{
			X: mainW + sidebarGap, Y: 0,
			W: sbW, H: sidebarH,
		},
		StatusBar: Rect{
			X: 0, Y: height - statusBarHeight,
			W: width, H: statusBarHeight,
		},
	}
}

// IsCompact returns true if the layout has no sidebar.
func (l Layout) IsCompact() bool {
	return l.Mode == LayoutCompact
}
