# Crush Study Notes - UI/UX Reference for Flame TUI

## Color Palette (Charmtone-based)

Crush uses `github.com/charmbracelet/x/exp/charmtone` for all colors.

### Primary Colors
- **Primary**: `charmtone.Charple` - Purple/magenta (highlights, accents)
- **Secondary**: `charmtone.Dolly` - Light secondary (decorative elements)
- **Tertiary**: `charmtone.Bok` - Green accent

### Text Hierarchy (IMPORTANT)
- **Base** (`charmtone.Ash`) - Primary content text
- **Muted** (`charmtone.Squid`) - Help text, secondary labels, file paths, placeholders
- **HalfMuted** (`charmtone.Smoke`) - Middle ground
- **Subtle** (`charmtone.Oyster`) - Hint text, decorative, less important

### Backgrounds
- **BgBase**: `charmtone.Pepper` - Main dark background
- **BgBaseLighter**: `charmtone.BBQ` - Slightly lighter for panels
- **BgSubtle**: `charmtone.Charcoal` - Code/content areas
- **BgOverlay**: `charmtone.Iron` - Semi-transparent overlay

### Status
- **Error**: `charmtone.Sriracha` (red)
- **Warning**: `charmtone.Zest` (yellow)
- **Info**: `charmtone.Malibu` (light blue)

## ASCII Art & Hatching

### Diagonal Character
- Uses `╱` (Unicode forward slash, NOT `/`)
- Styled in primary color (Charple/purple)
- Fills empty space between logo and details

### Logo
- "CRUSH" rendered with block characters (`▄`, `█`, `▀`)
- Horizontal gradient from TitleColorA to TitleColorB
- Left field: 6 diagonal chars
- Right field: Dynamic width, decreases top-to-bottom

### Small/Compact Logo
```
[Charm™ in secondary] [Crush in gradient] [╱╱╱ fill to width]
```

## Section Separators
- Character: `─` (horizontal line)
- Format: `Title ─────────────────────`
- Title in `fgSubtle`, line in `charmtone.Charcoal`

## Layout Architecture

### Framework: Ultraviolet + Lipgloss
- `uv.Screen` - canvas interface
- `uv.Rectangle` - positioning
- `layout.SplitHorizontal()` / `layout.SplitVertical()` - space division

### Responsive Breakpoints
- **Compact mode**: width < 120 OR height < 30
- **Wide mode**: sidebar (30 cols) on right

### Wide Mode
```
┌─────────────────────────────┬──────────────┐
│ Main Chat Area              │ Sidebar 30col│
│                             │  Logo        │
│                             │  Session     │
│                             │  Files       │
│                             │  LSP/MCP     │
├─────────────────────────────┤              │
│ Textarea (dynamic 3-15 ln)  │              │
├─────────────────────────────┴──────────────┤
│ Status/Help bar                            │
└────────────────────────────────────────────┘
```

### Compact Mode
```
┌────────────────────────────────────────────┐
│ Compact Header (1 line: logo + info)       │
├────────────────────────────────────────────┤
│ Main Chat Area (full width)                │
├────────────────────────────────────────────┤
│ Textarea (dynamic height)                  │
├────────────────────────────────────────────┤
│ Status/Help bar                            │
└────────────────────────────────────────────┘
```

Header content moves INTO sidebar when wide, stays as top bar when compact.

## Input Bar

### Prompt Function (Dynamic per line)
- Line 0, focused: `"  > "` (green `>`)
- Line 0, unfocused: `"::: "` (grip handles)
- Continuation, focused: styled prompt
- Continuation, unfocused: `":::"` (grip handles)

### Textarea Config
- `DynamicHeight = true` (grows 3-15 lines)
- `CharLimit = -1` (unlimited)
- `ShowLineNumbers = false`
- Placeholder rotates: "Ready!", "Thinking...", "Working!", "Brrrrr..."

### Textarea Styles
- Focused text: base color
- Blurred text: muted color
- Placeholder: subtle color (very dim)
- Prompt focused: tertiary color (green)
- Prompt blurred: muted color
- Cursor: secondary color, block, blink

## Status Bar / Help

### Help Styling
- **Key**: Muted gray (e.g., "ctrl+c")
- **Description**: Subtle gray (e.g., "quit")
- **Separator**: Border color (dot `•`)
- Format: `key desc • key desc • key desc`

### Status Messages
- Success: Green indicator "OKAY!" + green-dark background
- Error: Red indicator "ERROR" + red-dark background
- Warning: Yellow "WARNING"
- Info: Green "OKAY!"
- Update: Green "HEY!"

## Pubsub Broker

### Generic Broker[T]
```go
type Broker[T any] struct {
    subs map[chan Event[T]]struct{}
    mu   sync.RWMutex
}
```
- Buffer: 64 events per subscriber
- Non-blocking publish (drop on full)
- Context-aware auto-cleanup
- Thread-safe with RWMutex

## Key Symbols/Icons
- `✓` Success, `×` Error, `●` Pending
- `◇` Model, `→` Arrow, `◉`/`○` Radio
- `┃` Scrollbar thumb, `│` Scrollbar track
- `▌` Thick border (focused), `│` Thin border (blurred)

## Text Selection / Copy (CRITICAL for CTF tool)

### How Crush Enables Text Copying

Crush supports TWO copy methods:

#### 1. Mouse Drag Selection (like a normal text editor)
- **Mouse mode**: `tea.MouseModeCellMotion` set dynamically in View()
- Chat component tracks mouse state: `mouseDownItem`, `mouseDownX/Y`, `mouseDragX/Y`
- **Single click + drag**: Selects text range across cells
- **Double-click**: Selects word at position (`selectWord()`)
- **Triple-click**: Selects entire line (`selectLine()`)
- On mouse-up, if `HasHighlight()` is true, sends `copyChatHighlightMsg{}`
- Uses Ultraviolet's `ScreenBuffer` for cell-by-cell highlighting with `uv.AttrReverse`
- `HighlightContent()` extracts text from selected cells WITHOUT ANSI codes
- Mouse event filter throttles to 15ms intervals to prevent trackpad spam

#### 2. Keyboard Copy (focus chat → navigate → c/y)
- `Tab` toggles between `uiFocusEditor` and `uiFocusMain`
- In `uiFocusMain`: arrows navigate messages, `c`/`y` copies entire message
- Each message type (user, assistant, tool) has `HandleKeyEvent()` with copy logic
- Help bar updates to show `c/y copy` when chat is focused

#### Clipboard Implementation (dual method)
```go
// Uses BOTH for maximum compatibility:
tea.SetClipboard(text)           // OSC 52 (works over SSH!)
clipboard.WriteAll(text)         // Native system clipboard (go-nativeclipboard)
```

### Focus States
- `uiFocusEditor` - Input textarea active
- `uiFocusMain` - Chat viewport focused (enables message navigation + copy)
- `Tab` toggles between them
- Help bar updates to show available commands per focus state

### Program Initialization
```go
// Crush:
program := tea.NewProgram(model,
    tea.WithEnvironment(env),
    tea.WithContext(cmd.Context()),
    tea.WithFilter(ui.MouseEventFilter),  // 15ms throttle
)
// NOTE: AltScreen and MouseMode set dynamically in View()!
// v.AltScreen = true
// v.MouseMode = tea.MouseModeCellMotion

// Flame currently:
p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())
// Problem: mouse capture prevents native terminal text selection
```

### Key Insight for Flame

Our `WithMouseCellMotion()` captures ALL mouse events — terminal can't do native selection.
Crush solves this by implementing its own selection engine with UV ScreenBuffer.

**Our approach: Simplified selection engine (no UV needed)**

Crush has a complex item-based selection (multiple chat messages, each with borders).
Flame is simpler: ONE scrollable output pane. We can implement selection by:
1. Track mouse down/drag/up positions relative to viewport content
2. Map viewport coordinates to content lines (accounting for scroll offset)
3. During rendering, apply ANSI reverse video to selected range
4. On mouse up, extract plain text from selection, copy via OSC 52 + native clipboard
5. Mouse scroll continues to work for viewport scrolling

No UV dependency needed — just line/column math + ANSI manipulation.

### Crush Uses Bubble Tea v2

- Crush: `charm.land/bubbletea/v2 v2.0.2` — View() returns `tea.View` struct
- Flame: `github.com/charmbracelet/bubbletea v1.3.10` — View() returns string
- v2 has separate mouse events: MouseClickMsg, MouseMotionMsg, MouseReleaseMsg, MouseWheelMsg
- v1 has single MouseMsg with Type field
- v2 allows dynamic mouse mode per frame via `v.MouseMode`
- **Decision: Upgrade to v2 is recommended but can be done incrementally**

### Crush's Mouse Selection Architecture (Reference)

5-layer system:
1. **Filter** — 15ms throttle on motion/wheel events
2. **Mouse Tracking** — Chat tracks mouseDown/Drag state, multi-click (word/line select)
3. **Coordinate Translation** — Viewport coords → item index + local coords
4. **Highlight Rendering** — UV ScreenBuffer + AttrReverse on cells
5. **Text Extraction** — Walk highlighted cells, build plain text string

For Flame's simpler case (single output pane), layers 2-5 collapse into:
- Track start/end positions
- Map to content lines via scroll offset
- Apply reverse video during View() render
- Extract text between positions on mouse up

## Architecture Patterns for Flame

### What to Adopt
1. **Text hierarchy** (base/muted/subtle) for visual depth
2. **ASCII art header** with diagonal fill
3. **Header/sidebar integration** (responsive)
4. **Section separators** with `─` lines
5. **Dynamic textarea** that grows with input
6. **Focus-based help hints** that change per context
7. **Grip handles** `:::` for unfocused input
8. **Text selection support** - CRITICAL for CTF tool
9. **Pubsub broker** for async events

### What to Adapt for Flame
- Use terminal ANSI colors (0-15) instead of charmtone (Flame runs on any terminal)
- Magenta/cyan theme instead of purple/green
- Nerd Font symbols instead of Unicode-only
- Session list in sidebar instead of file/LSP/MCP info
- Shell output viewport instead of chat messages
