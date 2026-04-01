# TUI Current Status — 2026-04-01

## What's Done (Phase 1 — Menu Mode + Visual Polish)

### Core TUI (5 earlier commits)
- Bubble Tea TUI replaces readline REPL
- All menu commands work via stdout capture (`ExecuteCommand`)
- Background notifications route to TUI (`notifyTUI` callback)
- Word-wrap, scroll, command history
- Pointer receivers fix (Builder copy panic)
- Deadlock fix (mutex before detectSessionInfo)

### Visual Redesign (this session)
- **styles.go** — Color palette: base(253)/muted(245)/subtle(240), magenta(5)/cyan(6) accents
- **logo.go** — ASCII art "GUMMY" in block chars (▄█▀), stretched U with smiley eyes (▀ ▀ in cyan)
- **3-tier responsive layout** (layout.go):
  - `LayoutFull` — wide+tall: sidebar with full ASCII banner (7 lines)
  - `LayoutMedium` — wide but short: sidebar with compact 1-line banner
  - `LayoutCompact` — narrow: no sidebar, 1-line compact header
- **Sidebar** (app.go renderSidebar):
  - Banner (adaptive full/compact)
  - Session name (date) + CWD
  - Listener with  icon (white)
  - Binbag status with 󰖟 icon (online=white+path+port, offline=muted)
  - Session count (singular/plural, 0=subtle)
  - Active section: platform icon + whoami + IP + platform
  - Sessions list
- **Header** (header.go) — compact: `gummy 󰗣 ╱╱╱╱ addr ╱╱ N sessions`
- **Status bar** (statusbar.go) — hotkey hints only, no mode badge
- **Input** (input.go) — `❯` arrow prompt, placeholder, magenta cursor
- **Splash screen** (app.go viewSplash + logo.go renderBannerSplash):
  - First screen before any Enter
  - Logo with hatching on sides (3 chars left, fills right)
  - "shell handler" label aligned with logo hatching
  - Info: listener address, help hint
  - Input bar active and functional during splash
  - Dismisses on first Enter (executes typed command if any)
- **No separator bar** between main and sidebar — 3-char space gap
- **Spacing** around input line (blank above + blank below)
- **session.go** — added `GetActiveSessionDisplay()` method

### Design Decisions
- Colors: terminal ANSI (0-15 + grays) for max compatibility, NOT charmtone
- Hatching `╱` in cyan (6), logo text in magenta (5), eyes in cyan (6)
- Sidebar banner: hatching top+bottom. Splash banner: hatching sides only
- Text hierarchy: base (bright) → muted (gray) → subtle (dark gray)
- Symbols from existing `ui/colors.go` where possible, Nerd Fonts required
- Binbag icon: `\U000f059f` (nf-md-web 󰖟)
- Listener icon: `\uf095` ()

## What's Next

### Immediate: Mouse Text Selection + Scroll (Phase 1 completion)
See `docs/mouse-selection-plan.md` for full details.

Summary:
1. Mouse scroll — wheel events → viewport scroll (already works via WithMouseCellMotion)
2. Mouse drag selection — track press/motion/release, map to content lines
3. Visual highlight — apply ANSI reverse video during View() render
4. Text extraction + clipboard — OSC 52 (works over SSH) + native clipboard
5. Double-click word selection, triple-click line selection
6. Mouse event throttling (15ms)

Key insight: We do NOT need Ultraviolet. Gummy has a single output pane (not a message list like Crush), so selection is simple line/column math + ANSI manipulation.

### Then: Phase 2 — Event Bridge + Sidebar
- Generic `Broker[T]` pub/sub
- Live session notifications via broker (not callback)
- Interactive sidebar navigation (j/k, Enter to select)

### Then: Phase 3 — Shell Context (p0wny-shell style)
- Line-buffered shell relay: type command → Enter → send to net.Conn
- Remote output streams into viewport
- Prompt tracking (PS1 detection)
- Context switching: `shell` enters, F12 exits

### Full plan: `TUI_PLAN.md`

## File Map

```
internal/tui/
├── app.go          # Root model, Update/View, splash, sidebar, executeInput
├── styles.go       # Color palette, text styles, hatching/sectionHeader helpers
├── logo.go         # ASCII art letters, renderBannerFull/Compact/Splash, renderLogoLine2
├── layout.go       # 3-tier layout (Full/Medium/Compact), Rect, GenerateLayout
├── header.go       # Compact header bar (narrow mode only)
├── statusbar.go    # Hotkey hints, transfer progress
├── input.go        # Text input with ❯ prompt, history, context switching
├── outputpane.go   # Scrollable viewport with word-wrap
├── focus.go        # FocusMode enum
└── messages.go     # All tea.Msg types (session, shell, transfer, module events)
```

## Branch: `tui` (based on `main`)
