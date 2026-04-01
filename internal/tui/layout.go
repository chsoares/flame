package tui

// Rect represents a rectangular area on screen.
type Rect struct {
	X, Y int // Top-left corner
	W, H int // Width and height
}

// Layout holds the calculated positions for all TUI components.
type Layout struct {
	Header    Rect
	Output    Rect
	Sidebar   Rect
	Input     Rect
	StatusBar Rect
}

const (
	headerHeight    = 1
	statusBarHeight = 1
	inputHeight     = 1
	sidebarWidth    = 28
	minMainWidth    = 40
)

// GenerateLayout calculates component positions for the given terminal size.
func GenerateLayout(width, height int) Layout {
	// Compact mode: no sidebar if terminal too narrow
	compact := width < (minMainWidth + sidebarWidth + 1)

	mainW := width
	sbW := 0
	if !compact {
		sbW = sidebarWidth
		mainW = width - sbW - 1 // 1 for separator
	}

	// Vertical split: header | output | input | statusbar
	outputH := height - headerHeight - inputHeight - statusBarHeight
	if outputH < 1 {
		outputH = 1
	}

	layout := Layout{
		Header: Rect{
			X: 0, Y: 0,
			W: width, H: headerHeight,
		},
		Output: Rect{
			X: 0, Y: headerHeight,
			W: mainW, H: outputH,
		},
		Input: Rect{
			X: 0, Y: headerHeight + outputH,
			W: mainW, H: inputHeight,
		},
		StatusBar: Rect{
			X: 0, Y: height - statusBarHeight,
			W: width, H: statusBarHeight,
		},
	}

	if !compact {
		layout.Sidebar = Rect{
			X: mainW + 1, Y: headerHeight,
			W: sbW, H: outputH + inputHeight,
		}
	}

	return layout
}

// IsCompact returns true if the layout has no sidebar.
func (l Layout) IsCompact() bool {
	return l.Sidebar.W == 0
}
