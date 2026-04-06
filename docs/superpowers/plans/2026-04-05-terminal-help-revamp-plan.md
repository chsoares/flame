# Terminal Help Revamp Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Flame's one-screen help with a command-topic terminal help system that keeps `help` as the compact index, adds `help <topic>` detail pages, supports topic tab completion, and stops before any TUI help modal work begins.

**Architecture:** Extract help content into a focused helper in `internal/` so both the current menu command path and future TUI work can reuse one source of truth. Keep `internal/session.go` responsible for command dispatch and completion wiring, but move help-topic data and rendering into a dedicated helper with unit tests.

**Tech Stack:** Go, existing `internal/ui` helpers, current readline completer in `internal/session.go`, Go unit tests

---

## File Structure Map

- Create: `internal/help.go` - help topic registry, aliases, general-help rendering, detailed help rendering, topic list helpers
- Create: `internal/help_test.go` - tests for topic lookup, alias resolution, general help footer, and command-specific content rendering
- Modify: `internal/session.go` - route `help` and `help <topic>` through the help helper, extend tab completion with help topics, keep unknown-topic behavior strict
- Modify: `docs/current-status.md` - record that terminal help is the next UX phase, blocked only by the pending `rev`/`ssh` review, and that TUI help stays a later separate phase

### Task 1: Add a Dedicated Help Registry and Renderers

**Files:**
- Create: `internal/help.go`
- Create: `internal/help_test.go`

- [ ] **Step 1: Write failing tests for help topic lookup and alias resolution**

Create `internal/help_test.go` with:

```go
package internal

import "testing"

func TestLookupHelpTopicFindsPrimaryCommand(t *testing.T) {
	entry, ok := LookupHelpTopic([]string{"download"})
	if !ok {
		t.Fatal("expected download help topic")
	}
	if entry.Topic != "download" {
		t.Fatalf("expected topic download, got %q", entry.Topic)
	}
}

func TestLookupHelpTopicResolvesAlias(t *testing.T) {
	entry, ok := LookupHelpTopic([]string{"list"})
	if !ok {
		t.Fatal("expected list alias to resolve")
	}
	if entry.Topic != "sessions" {
		t.Fatalf("expected alias to resolve to sessions, got %q", entry.Topic)
	}
}

func TestLookupHelpTopicSupportsNestedRunTopic(t *testing.T) {
	entry, ok := LookupHelpTopic([]string{"run", "ps1"})
	if !ok {
		t.Fatal("expected nested run topic")
	}
	if entry.Topic != "run ps1" {
		t.Fatalf("expected topic run ps1, got %q", entry.Topic)
	}
}
```

- [ ] **Step 2: Run the focused tests and verify they fail**

Run: `go test ./internal -run 'TestLookupHelpTopic(FindsPrimaryCommand|ResolvesAlias|SupportsNestedRunTopic)' -v`

Expected: FAIL with `undefined: LookupHelpTopic`.

- [ ] **Step 3: Implement the help topic registry and lookup helper**

Create `internal/help.go` with:

```go
package internal

import "strings"

type HelpEntry struct {
	Topic    string
	Aliases  []string
	Summary  string
	Usage    []string
	Details  []string
	Examples []string
	Notes    []string
}

var helpEntries = []HelpEntry{
	{Topic: "sessions", Aliases: []string{"list"}, Summary: "List active sessions."},
	{Topic: "download", Summary: "Download a file from the remote target."},
	{Topic: "run", Summary: "Run a module or custom runner."},
	{Topic: "run ps1", Summary: "Run a PowerShell script through the module runner."},
}

func LookupHelpTopic(parts []string) (HelpEntry, bool) {
	query := strings.TrimSpace(strings.Join(parts, " "))
	for _, entry := range helpEntries {
		if entry.Topic == query {
			return entry, true
		}
		for _, alias := range entry.Aliases {
			if alias == query {
				return entry, true
			}
		}
	}
	return HelpEntry{}, false
}
```

- [ ] **Step 4: Re-run the lookup tests**

Run: `go test ./internal -run 'TestLookupHelpTopic(FindsPrimaryCommand|ResolvesAlias|SupportsNestedRunTopic)' -v`

Expected: PASS.

- [ ] **Step 5: Commit the help registry foundation**

```bash
git add internal/help.go internal/help_test.go
git commit -m "feat: add terminal help topic registry"
```

### Task 2: Render the Existing General Help From Shared Help Data

**Files:**
- Modify: `internal/help.go`
- Modify: `internal/help_test.go`
- Modify: `internal/session.go`

- [ ] **Step 1: Write failing tests for the general help screen**

Extend `internal/help_test.go` with:

```go
import "strings"

func TestRenderGeneralHelpIncludesDetailHint(t *testing.T) {
	rendered := RenderGeneralHelp()
	if !strings.Contains(rendered, "help <command>") {
		t.Fatalf("expected detail hint in general help, got %q", rendered)
	}
}

func TestRenderGeneralHelpListsRunCommand(t *testing.T) {
	rendered := RenderGeneralHelp()
	if !strings.Contains(rendered, "run <module> [args]") {
		t.Fatalf("expected run command in general help, got %q", rendered)
	}
}
```

- [ ] **Step 2: Run the new tests and verify they fail**

Run: `go test ./internal -run 'TestRenderGeneralHelp(IncludesDetailHint|ListsRunCommand)' -v`

Expected: FAIL with `undefined: RenderGeneralHelp`.

- [ ] **Step 3: Implement a shared general-help renderer and route `showHelp()` through it**

Update `internal/help.go` with a general-help renderer:

```go
func RenderGeneralHelp() string {
	var lines []string
	lines = append(lines, ui.CommandHelp("connect"))
	lines = append(lines, ui.Command("rev [csharp [file.exe]]       - Generate reverse shell payloads"))
	lines = append(lines, ui.Command("ssh user@host                - Connect via SSH and execute revshell"))
	lines = append(lines, "")
	lines = append(lines, ui.CommandHelp("program"))
	lines = append(lines, ui.Command("help                         - Show command help"))
	lines = append(lines, ui.HelpInfo("Type 'help <command>' for details"))
	return ui.BoxWithTitle(fmt.Sprintf("%s Available Commands", ui.SymbolGem), lines)
}
```

Then simplify `showHelp()` in `internal/session.go` to:

```go
func (m *Manager) showHelp() {
	fmt.Println(RenderGeneralHelp())
}
```

Also add the required imports in `internal/help.go`:

```go
import (
	"fmt"
	"strings"

	"github.com/chsoares/flame/internal/ui"
)
```

- [ ] **Step 4: Re-run the general-help tests**

Run: `go test ./internal -run 'TestRenderGeneralHelp(IncludesDetailHint|ListsRunCommand)' -v`

Expected: PASS.

- [ ] **Step 5: Commit the shared general-help rendering**

```bash
git add internal/help.go internal/help_test.go internal/session.go
git commit -m "refactor: render menu help from shared helper"
```

### Task 3: Add Detailed `help <topic>` Rendering

**Files:**
- Modify: `internal/help.go`
- Modify: `internal/help_test.go`
- Modify: `internal/session.go`

- [ ] **Step 1: Write failing tests for detailed help rendering**

Extend `internal/help_test.go` with:

```go
func TestRenderHelpTopicKeepsSimpleCommandsCompact(t *testing.T) {
	rendered, ok := RenderHelpTopic([]string{"use"})
	if !ok {
		t.Fatal("expected use help topic")
	}
	if !strings.Contains(rendered, "use <id>") {
		t.Fatalf("expected use syntax, got %q", rendered)
	}
	if strings.Contains(rendered, "Examples") {
		t.Fatalf("did not expect forced examples section for compact help, got %q", rendered)
	}
}

func TestRenderHelpTopicShowsDetailedSectionsWhenNeeded(t *testing.T) {
	rendered, ok := RenderHelpTopic([]string{"binbag"})
	if !ok {
		t.Fatal("expected binbag help topic")
	}
	checks := []string{"binbag on", "binbag path <dir>", "HTTP server", "upload"}
	for _, check := range checks {
		if !strings.Contains(rendered, check) {
			t.Fatalf("expected %q in binbag help, got %q", check, rendered)
		}
	}
}

func TestRenderHelpTopicSupportsRunSubtopics(t *testing.T) {
	rendered, ok := RenderHelpTopic([]string{"run", "ps1"})
	if !ok {
		t.Fatal("expected run ps1 help topic")
	}
	if !strings.Contains(rendered, "run ps1 <url|file> [args]") {
		t.Fatalf("expected run ps1 usage, got %q", rendered)
	}
}
```

- [ ] **Step 2: Run the detailed-help tests and verify they fail**

Run: `go test ./internal -run 'TestRenderHelpTopic(KeepsSimpleCommandsCompact|ShowsDetailedSectionsWhenNeeded|SupportsRunSubtopics)' -v`

Expected: FAIL with `undefined: RenderHelpTopic`.

- [ ] **Step 3: Implement detailed help rendering with selective sections**

Update `internal/help.go` with:

```go
func RenderHelpTopic(parts []string) (string, bool) {
	entry, ok := LookupHelpTopic(parts)
	if !ok {
		return "", false
	}

	var lines []string
	lines = append(lines, ui.Command(entry.Topic))
	if entry.Summary != "" {
		lines = append(lines, ui.Info(entry.Summary))
	}
	if len(entry.Usage) > 0 {
		lines = append(lines, "")
		lines = append(lines, ui.CommandHelp("usage"))
		for _, usage := range entry.Usage {
			lines = append(lines, ui.Command(usage))
		}
	}
	if len(entry.Details) > 0 {
		lines = append(lines, "")
		lines = append(lines, ui.CommandHelp("details"))
		for _, detail := range entry.Details {
			lines = append(lines, detail)
		}
	}
	if len(entry.Examples) > 0 {
		lines = append(lines, "")
		lines = append(lines, ui.CommandHelp("examples"))
		for _, example := range entry.Examples {
			lines = append(lines, ui.Command(example))
		}
	}
	if len(entry.Notes) > 0 {
		lines = append(lines, "")
		lines = append(lines, ui.CommandHelp("notes"))
		for _, note := range entry.Notes {
			lines = append(lines, note)
		}
	}

	return ui.BoxWithTitle(fmt.Sprintf("%s Help", ui.SymbolGem), lines), true
}
```

Also flesh out `helpEntries` with real content for at least these topics in the same task:

```go
{Topic: "use", Summary: "Select the active session by visible numeric ID.", Usage: []string{"use <id>"}},
{Topic: "binbag", Summary: "Manage the local HTTP file-serving directory used by uploads and module runners.", Usage: []string{"binbag ls", "binbag on", "binbag off", "binbag path <dir>", "binbag port <N>"}, Details: []string{"Starts or stops the binbag HTTP server.", "A configured binbag can speed up upload and runner paths that support HTTP delivery."}, Notes: []string{"Run commands that depend on file serving only after confirming the binbag path is correct."}},
{Topic: "run ps1", Summary: "Run a PowerShell script through the Windows in-memory runner path.", Usage: []string{"run ps1 <url|file> [args]"}, Details: []string{"Expands local paths before execution.", "Uses the worker-session flow on Windows."}},
```

- [ ] **Step 4: Wire `help <topic>` into command dispatch**

Update the `help` branch in `internal/session.go` to:

```go
case "help", "h":
	if len(parts) == 1 {
		m.showHelp()
		return
	}
	rendered, ok := RenderHelpTopic(parts[1:])
	if !ok {
		fmt.Println(ui.Warning(fmt.Sprintf("Unknown help topic: %s", strings.Join(parts[1:], " "))))
		return
	}
	fmt.Println(rendered)
```

- [ ] **Step 5: Re-run the detailed-help tests**

Run: `go test ./internal -run 'TestRenderHelpTopic(KeepsSimpleCommandsCompact|ShowsDetailedSectionsWhenNeeded|SupportsRunSubtopics)' -v`

Expected: PASS.

- [ ] **Step 6: Commit the detailed help topic support**

```bash
git add internal/help.go internal/help_test.go internal/session.go
git commit -m "feat: add per-command terminal help"
```

### Task 4: Add Help Topic Completion

**Files:**
- Modify: `internal/help.go`
- Modify: `internal/help_test.go`
- Modify: `internal/session.go`

- [ ] **Step 1: Write failing tests for help-topic completion data**

Extend `internal/help_test.go` with:

```go
func TestHelpTopicsForCompletionIncludesNestedRunTopics(t *testing.T) {
	topics := HelpTopicsForCompletion()
	checks := []string{"download", "binbag", "run", "run ps1", "run dotnet"}
	for _, check := range checks {
		found := false
		for _, topic := range topics {
			if topic == check {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected topic %q in completion list: %#v", check, topics)
		}
	}
}
```

- [ ] **Step 2: Run the new completion-data test and verify it fails**

Run: `go test ./internal -run TestHelpTopicsForCompletionIncludesNestedRunTopics -v`

Expected: FAIL with `undefined: HelpTopicsForCompletion`.

- [ ] **Step 3: Implement help-topic listing and hook it into the completer**

Add to `internal/help.go`:

```go
func HelpTopicsForCompletion() []string {
	topics := make([]string, 0, len(helpEntries))
	for _, entry := range helpEntries {
		topics = append(topics, entry.Topic)
	}
	sort.Strings(topics)
	return topics
}
```

Add the import required by that helper:

```go
import "sort"
```

Then update the `help` branch in `FlameCompleter.Do()` inside `internal/session.go`:

```go
case "help":
	if argCount >= 1 {
		prefix := strings.TrimSpace(strings.TrimPrefix(trimmed, "help"))
		prefix = strings.TrimLeft(prefix, " ")
		return c.completeFromList(prefix, HelpTopicsForCompletion())
	}
```

- [ ] **Step 4: Add and run a focused manager completion test**

Create `internal/session_help_test.go` with:

```go
package internal

import "testing"

func TestCompleteInputCompletesHelpTopic(t *testing.T) {
	m := &Manager{}
	got := m.CompleteInput("help dow")
	if got != "help download" {
		t.Fatalf("expected help topic completion, got %q", got)
	}
}
```

Run: `go test ./internal -run 'Test(HelpTopicsForCompletionIncludesNestedRunTopics|CompleteInputCompletesHelpTopic)' -v`

Expected: PASS.

- [ ] **Step 5: Commit the help-topic completion work**

```bash
git add internal/help.go internal/help_test.go internal/session.go internal/session_help_test.go
git commit -m "feat: add help topic completion"
```

### Task 5: Finalize Command Content and Update Handoff Docs

**Files:**
- Modify: `internal/help.go`
- Modify: `docs/current-status.md`

- [ ] **Step 1: Expand help content for the full terminal command surface**

Update `helpEntries` in `internal/help.go` so it covers at least these topics before validation:

```go
"rev",
"ssh",
"sessions",
"use",
"kill",
"shell",
"upload",
"download",
"spawn",
"modules",
"run",
"run ps1",
"run dotnet",
"run elf",
"run py",
"binbag",
"pivot",
"config",
"clear",
"exit",
```

Populate each topic with only the fields it needs. Keep `use`, `kill`, `clear`, and `exit` compact. Give `binbag`, `pivot`, `download`, `upload`, `run`, `rev`, and `ssh` the extra explanation they need.

- [ ] **Step 2: Run the package test suite and build**

Run: `go test ./...`

Expected: PASS.

Run: `go build -o flame .`

Expected: build succeeds.

- [ ] **Step 3: Perform the terminal help validation pass and stop there**

Run these manual checks in the built binary:

```text
help
help spawn
help download
help binbag
help pivot
help run
help run ps1
help rev
help ssh
```

Then validate completion and strict error behavior:

```text
help d<Tab>
help run p<Tab>
help does-not-exist
```

Expected:

- `help` stays compact and accurate
- detailed topics scale by command complexity
- `help run` explains the overview while subtopics carry runner detail
- tab completion resolves valid topics
- unknown topics fail plainly with no suggestion UX
- no TUI modal work starts yet

- [ ] **Step 4: Update the handoff docs with the saved objective and sequencing**

Append this exact section to `docs/current-status.md`:

```md
### Help UX roadmap
- Terminal help revamp is the next UI/UX phase once `rev` and `ssh` review work is done
- Phase 1 keeps `help` as a compact index and adds `help <command>` detail pages with tab completion
- `run` gets overview help plus specific subtopics such as `help run ps1`
- Phase 2 is a separate future TUI help-modal project and must not start until terminal help is implemented and validated
```

- [ ] **Step 5: Commit the finalized content and docs note**

```bash
git add internal/help.go docs/current-status.md
git commit -m "docs: capture terminal help roadmap"
```

## Self-Review Checklist

- Spec coverage: the plan covers shared help data, `help` index reuse, per-command detail pages, nested `run` topics, tab completion, strict unknown-topic behavior, and the explicit stop before TUI modal work.
- Placeholder scan: no task contains `TODO`, `TBD`, or vague “implement later” language.
- Type consistency: the plan uses one naming set throughout - `HelpEntry`, `LookupHelpTopic`, `RenderGeneralHelp`, `RenderHelpTopic`, and `HelpTopicsForCompletion`.
