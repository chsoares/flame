# Visual Redesign Plan — Crush-Inspired UI

## Goals
Transform Gummy's TUI from functional-but-plain to polished and beautiful, taking visual inspiration from Crush while keeping Gummy's identity (magenta/cyan terminal colors, Nerd Font symbols).

## Design Elements from Crush

### 1. ASCII Art Header with Hatching
Crush uses `╱` (diagonal slash) to fill empty space, creating a textured look:
```
Charm™ Crush ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
```

For Gummy:
```
gummy 󰗣 ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
```
Or with ASCII art logo:
```
 ╱╱ gummy 󰗣 ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
```

### 2. Header/Sidebar Integration
- **Wide mode (≥100 cols):** Header info appears at TOP of sidebar. No separate header bar.
- **Compact mode (<100 cols):** Header info appears as 1-line bar at top.
- This saves vertical space in wide mode.

### 3. Text Hierarchy (3 levels)
Using terminal ANSI colors:
- **Base** — Default terminal foreground (white/light) — primary content
- **Muted** — Color 245 (gray) — secondary labels, help text, separators
- **Subtle** — Color 240 (dark gray) — hints, placeholders, decorative

### 4. Section Separators
```
Sessions ───────────────────────
```
Title in muted color, line in subtle/dark gray.

### 5. Status Bar with Styled Hotkeys
```
MENU  Tab sidebar • Ctrl+D quit • PgUp/PgDn scroll
```
- Mode badge: colored background (magenta for MENU, cyan for SHELL)
- Key names: muted color
- Descriptions: subtle color
- Separator: `•` in subtle color

### 6. Input Prompt Styling
- Focused: `  > ` with colored `>` (magenta)
- Unfocused: `:::` grip handles in muted gray
- Placeholder: subtle gray ("Ready..." / "Type a command...")

## Color Palette (Terminal ANSI)

Using terminal colors for maximum compatibility:

```go
// Primary accent
Magenta = Color("5")     // Main brand color
Cyan    = Color("6")     // Secondary accent, info

// Text hierarchy
Base    = Color("15")    // Bright white — primary text (or just default fg)
Muted   = Color("245")   // Gray — secondary text, labels
Subtle  = Color("240")   // Dark gray — hints, decorative
Dim     = Color("238")   // Very dark — separators, borders

// Status
Green   = Color("2")     // Success
Red     = Color("1")     // Error
Yellow  = Color("3")     // Warning

// Backgrounds
BgDark  = Color("235")   // Header/status background
BgPanel = Color("236")   // Sidebar background (slightly lighter)
```

## Layout Design

### Wide Mode (≥100 cols)
```
┌─────────────────────────────────┬──────────────────────────┐
│                                 │ gummy 󰗣 ╱╱╱╱╱╱╱╱╱╱╱╱╱╱ │
│                                 │ 10.10.14.5:4444          │
│ OUTPUT PANE                     │                          │
│ (scrollable, selectable)        │ Sessions ─────────────── │
│                                 │  ▶ 1  root@10.10.11.5   │
│ gummy ❯ help                    │    2  www@10.10.11.23    │
│ Available commands:             │                          │
│   list     - List sessions      │ Info ────────────────── │
│   use <id> - Select session     │  Binbag: ~/Lab/binbag    │
│   ...                           │  Mode: stealth           │
│                                 │                          │
├─────────────────────────────────┤                          │
│  > _                            │                          │
├─────────────────────────────────┴──────────────────────────┤
│ MENU  Tab sidebar • Ctrl+D quit                            │
└────────────────────────────────────────────────────────────┘
```

### Compact Mode (<100 cols)
```
┌────────────────────────────────────────────┐
│ gummy 󰗣 ╱╱╱╱ 10.10.14.5:4444 │ 2 sessions│
├────────────────────────────────────────────┤
│ OUTPUT PANE (full width)                   │
│                                            │
│ gummy ❯ help                               │
│ Available commands:                        │
│   ...                                      │
├────────────────────────────────────────────┤
│  > _                                       │
├────────────────────────────────────────────┤
│ MENU  Tab sidebar • Ctrl+D quit            │
└────────────────────────────────────────────┘
```

## Sidebar Design

```
gummy 󰗣 ╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱╱
10.10.14.5:4444
2 sessions

Sessions ──────────────────────
 ▶ 1  root@target
    2  www@victim

Info ──────────────────────────
 Binbag  ~/Lab/binbag
 Mode    stealth
 HTTP    :8080
```

- Logo + hatching at top (magenta)
- Listener info below (muted gray)
- Section separators with `─` lines
- Active session with `▶` indicator (cyan/yellow)
- Platform icons:  Linux,  Windows
- Info section with config summary

## Components to Modify

### header.go → Compact header only
- 1-line compact header for narrow terminals
- Contains: logo + hatching + listener + session count

### sidebar.go (NEW or extracted from app.go)
- Full sidebar component with:
  - Logo + hatching at top
  - Listener info
  - Sessions section
  - Info section
- Dynamic height allocation for sections

### outputpane.go
- No visual changes needed (content area stays clean)
- Mouse selection rendering will be added separately

### input.go
- Styled prompt: `  > ` with magenta `>`
- Placeholder in subtle gray
- Consider grip handles `:::` when unfocused (future)

### statusbar.go
- Mode badge with colored background
- Hotkey hints: bold key + subtle description + `•` separator

### layout.go
- Update breakpoint (maybe 100 instead of 69)
- In wide mode: no header, sidebar contains branding
- In compact mode: 1-line header, no sidebar

## Implementation Status: DONE

All items implemented:
1. Color constants and text styles (`styles.go`)
2. ASCII art logo with hatching (`logo.go`) — stretched U with smiley eyes
3. 3-tier responsive layout: Full (big sidebar + ASCII art), Medium (compact sidebar), Compact (header only)
4. Sidebar: banner → session info → cwd → listener → binbag status → session count → Active section → Sessions list
5. Status bar: hotkey hints only (no mode badge)
6. Input prompt: `❯` arrow, placeholder text
7. No separator bar between main/sidebar — space gap like Crush
8. Spacing around input line

## Next Steps
- ~~Splash screen (like Crush's onboarding screen)~~ DONE
- Mouse text selection + scroll in viewport (see mouse-selection-plan.md)
