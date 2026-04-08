package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chsoares/flame/internal/ui"
)

func TestLookupHelpTopicFindsPrimaryCommand(t *testing.T) {
	entry, ok := LookupHelpTopic([]string{"download"})
	if !ok {
		t.Fatal("expected download help topic")
	}
	if entry.Topic != "download" {
		t.Fatalf("expected topic download, got %q", entry.Topic)
	}
}

func TestHelpTopicsForModalIncludesAllHelpEntries(t *testing.T) {
	topics := HelpTopicsForModal()
	for _, want := range []string{"rev", "ssh", "sessions", "use", "kill", "shell", "upload", "download", "spawn", "modules", "run", "run peas", "run lse", "run loot", "run pspy", "run linexp", "run sh", "run ps1", "run dotnet", "run elf", "run py", "run winpeas", "run seatbelt", "binbag", "pivot", "config", "clear", "exit"} {
		found := false
		for _, topic := range topics {
			if topic == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected %q in modal topics, got %v", want, topics)
		}
	}
	for _, bad := range []string{"list", "ls", "quit", "q"} {
		for _, topic := range topics {
			if topic == bad {
				t.Fatalf("did not expect alias or nested topic %q in modal topics, got %v", bad, topics)
			}
		}
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

func TestLookupHelpTopicResolvesRunAliases(t *testing.T) {
	checks := map[string]string{
		"run sh":   "run sh",
		"run bash": "run sh",
		"rev sh":   "rev",
		"rev bash": "rev",
	}
	for input, want := range checks {
		entry, ok := LookupHelpTopic(strings.Fields(input))
		if !ok {
			t.Fatalf("expected help topic for %q", input)
		}
		if entry.Topic != want {
			t.Fatalf("expected %q to resolve to %q, got %q", input, want, entry.Topic)
		}
	}
}

func TestRenderGeneralHelpIncludesDetailHint(t *testing.T) {
	rendered := RenderGeneralHelp()
	if !strings.Contains(rendered, "Type 'help <command>' for details") {
		t.Fatalf("expected detail hint in general help, got %q", rendered)
	}
	firstLine := strings.Split(rendered, "\n")[0]
	if !strings.Contains(firstLine, "Type 'help <command>' for details") {
		t.Fatalf("expected help footer on first line, got %q", firstLine)
	}
	if strings.Contains(rendered, "┌") || strings.Contains(rendered, "┐") || strings.Contains(rendered, "└") || strings.Contains(rendered, "┘") {
		t.Fatalf("expected general help without box borders, got %q", rendered)
	}
}

func TestHelpFooterIsSymbolFree(t *testing.T) {
	rendered := ui.HelpFooter("Type 'help <command>' for details")
	if strings.Contains(rendered, ui.SymbolCommand) {
		t.Fatalf("expected footer without command symbol, got %q", rendered)
	}
	if !strings.Contains(rendered, "Type 'help <command>' for details") {
		t.Fatalf("expected footer text to remain visible, got %q", rendered)
	}
}

func TestRenderGeneralHelpListsRunCommand(t *testing.T) {
	rendered := RenderGeneralHelp()
	if !strings.Contains(rendered, "run <module> [args]") {
		t.Fatalf("expected run command in general help, got %q", rendered)
	}
}

func TestRenderGeneralHelpGroupsNetworkCommands(t *testing.T) {
	rendered := RenderGeneralHelp()
	checks := []string{"network", "binbag", "pivot", "Type 'help <command>' for details"}
	for _, check := range checks {
		if !strings.Contains(rendered, check) {
			t.Fatalf("expected %q in general help, got %q", check, rendered)
		}
	}
	if strings.Contains(rendered, "binbag ls") || strings.Contains(rendered, "pivot <ip>") {
		t.Fatalf("expected general help to stay high level, got %q", rendered)
	}
}

func TestRenderHelpTopicKeepsSimpleCommandsCompact(t *testing.T) {
	rendered, ok := RenderHelpTopic([]string{"use"})
	if !ok {
		t.Fatal("expected use help topic")
	}
	if strings.Contains(rendered, ui.SymbolInfo) {
		t.Fatalf("expected plain summary without info symbol, got %q", rendered)
	}
	if !strings.Contains(rendered, "use <id>") {
		t.Fatalf("expected use syntax, got %q", rendered)
	}
	if strings.Contains(rendered, "examples") {
		t.Fatalf("did not expect forced examples section for compact help, got %q", rendered)
	}
	if strings.Contains(rendered, "┌") || strings.Contains(rendered, "┐") || strings.Contains(rendered, "└") || strings.Contains(rendered, "┘") {
		t.Fatalf("expected compact help without box borders, got %q", rendered)
	}
}

func TestExecuteCommandReportsBoxFreeHelpOutput(t *testing.T) {
	m := NewManager()
	got := m.ExecuteCommand("help use")
	if strings.Contains(got, "┌") || strings.Contains(got, "┐") || strings.Contains(got, "└") || strings.Contains(got, "┘") {
		t.Fatalf("expected help command output without box borders, got %q", got)
	}
}

func TestExecuteCommandReportsBoxFreeListOutput(t *testing.T) {
	m := NewManager()
	got := m.ExecuteCommand("sessions")
	if strings.Contains(got, "┌") || strings.Contains(got, "┐") || strings.Contains(got, "└") || strings.Contains(got, "┘") {
		t.Fatalf("expected sessions output without box borders, got %q", got)
	}
}

func TestExecuteCommandReportsBoxFreeModulesAndConfigOutput(t *testing.T) {
	m := NewManager()
	prev := GlobalRuntimeConfig
	defer func() { GlobalRuntimeConfig = prev }()
	GlobalRuntimeConfig = &RuntimeConfig{}
	for _, cmd := range []string{"modules", "config"} {
		got := m.ExecuteCommand(cmd)
		if strings.Contains(got, "┌") || strings.Contains(got, "┐") || strings.Contains(got, "└") || strings.Contains(got, "┘") {
			t.Fatalf("expected %s output without box borders, got %q", cmd, got)
		}
	}
}

func TestModulesOutputPutsLegendFirst(t *testing.T) {
	m := NewManager()
	got := m.ExecuteCommand("modules")
	firstLine := strings.Split(got, "\n")[0]
	if strings.Contains(firstLine, "linux") || strings.Contains(firstLine, "windows") || strings.Contains(firstLine, "misc") || strings.Contains(firstLine, "custom") {
		t.Fatalf("expected modules output to start with the legend line, got %q", firstLine)
	}
}

func TestRenderHelpTopicShowsDetailedSectionsWhenNeeded(t *testing.T) {
	rendered, ok := RenderHelpTopic([]string{"binbag"})
	if !ok {
		t.Fatal("expected binbag help topic")
	}
	checks := []string{"HTTP is the fast path", "Much faster than the base64 chunking fallback"}
	for _, check := range checks {
		if !strings.Contains(rendered, check) {
			t.Fatalf("expected %q in binbag help, got %q", check, rendered)
		}
	}
}

func TestRenderHelpTopicExplainsPivotUseCase(t *testing.T) {
	rendered, ok := RenderHelpTopic([]string{"pivot"})
	if !ok {
		t.Fatal("expected pivot help topic")
	}
	checks := []string{"ligolo", "chisel", "updates generated URLs"}
	for _, check := range checks {
		if !strings.Contains(rendered, check) {
			t.Fatalf("expected %q in pivot help, got %q", check, rendered)
		}
	}
}

func TestRenderHelpTopicExplainsRunWorkerFlow(t *testing.T) {
	rendered, ok := RenderHelpTopic([]string{"run"})
	if !ok {
		t.Fatal("expected run help topic")
	}
	checks := []string{"separate worker session", "separate terminal window", "main shell stays free"}
	for _, check := range checks {
		if !strings.Contains(rendered, check) {
			t.Fatalf("expected %q in run help, got %q", check, rendered)
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

func TestRenderHelpTopicSupportsRunShAndAliases(t *testing.T) {
	rendered, ok := RenderHelpTopic([]string{"run", "sh"})
	if !ok {
		t.Fatal("expected run sh help topic")
	}
	if !strings.Contains(rendered, "run sh <url|file> [args]") {
		t.Fatalf("expected run sh usage, got %q", rendered)
	}
	rendered, ok = RenderHelpTopic([]string{"run", "bash"})
	if !ok {
		t.Fatal("expected run bash help topic")
	}
	if !strings.Contains(rendered, "run sh <url|file> [args]") {
		t.Fatalf("expected run bash to reuse run sh content, got %q", rendered)
	}
}

func TestRenderHelpTopicExplainsWindowsBuffering(t *testing.T) {
	for _, topic := range []string{"run ps1", "run dotnet", "run winpeas", "run seatbelt"} {
		rendered, ok := RenderHelpTopic(strings.Fields(topic))
		if !ok {
			t.Fatalf("expected %s help topic", topic)
		}
		if !strings.Contains(rendered, "output may arrive in blocks") {
			t.Fatalf("expected buffering note in %q help, got %q", topic, rendered)
		}
	}
}

func TestHelpTopicsForCompletionIncludesNestedRunTopics(t *testing.T) {
	topics := HelpTopicsForCompletion()
	checks := []string{"download", "binbag", "run", "run sh", "run ps1", "run dotnet", "run elf", "run py"}
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

func TestLookupHelpTopicCoversRunModuleTopics(t *testing.T) {
	for _, topic := range []string{"run peas", "run winpeas", "run seatbelt"} {
		if _, ok := LookupHelpTopic(strings.Fields(topic)); !ok {
			t.Fatalf("expected help topic for %q", topic)
		}
	}
}

func TestExecuteCommandReportsUnknownHelpTopic(t *testing.T) {
	m := NewManager()
	got := m.ExecuteCommand("help run foo")
	if !strings.Contains(got, "Unknown help topic: run foo") {
		t.Fatalf("expected unknown topic error, got %q", got)
	}
}

func TestCompleteInputCompletesMultiWordHelpTopic(t *testing.T) {
	m := NewManager()
	got := m.CompleteInput("help run do")
	if got != "help run dotnet" {
		t.Fatalf("expected multi-word help topic completion, got %q", got)
	}
}

func TestCompleteInputIsCaseInsensitiveForCommands(t *testing.T) {
	m := NewManager()
	got := m.CompleteInput("HE")
	if got != "help" {
		t.Fatalf("expected case-insensitive command completion, got %q", got)
	}
}

func TestCompleteInputIsCaseInsensitiveForLocalPaths(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	if err := os.WriteFile(filepath.Join(dir, "Lab"), []byte(""), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	m := NewManager()
	got := m.CompleteInput("upload l")
	if got != "upload Lab" {
		t.Fatalf("expected case-insensitive local path completion, got %q", got)
	}
}
