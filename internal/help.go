package internal

import (
	"fmt"
	"sort"
	"strings"

	"github.com/chsoares/flame/internal/ui"
)

type HelpEntry struct {
	Topic    string
	Category string
	Aliases  []string
	Summary  string
	Usage    []string
	Details  []string
	Examples []string
}

var helpEntries = []HelpEntry{
	{Topic: "rev", Category: "connect", Aliases: []string{"rev bash", "rev sh"}, Summary: "Show reverse shell payloads.", Usage: []string{"rev", "rev bash", "rev sh", "rev ps1", "rev csharp [output.exe]", "rev php [output.php]"}},
	{Topic: "ssh", Category: "connect", Summary: "Connect over SSH and trigger a reverse shell.", Usage: []string{"ssh user@host (-p <password> | -i <key>) [--port <port>]"}},
	{Topic: "sessions", Category: "handler", Aliases: []string{"list", "ls"}, Summary: "List active sessions."},
	{Topic: "use", Category: "handler", Summary: "Select the active session by numeric ID.", Usage: []string{"use <id>"}},
	{Topic: "kill", Category: "handler", Summary: "Close a session by numeric ID.", Usage: []string{"kill <id>"}},
	{Topic: "shell", Category: "session", Summary: "Enter the interactive shell for the active session."},
	{Topic: "upload", Category: "session", Summary: "Upload a local file to the selected session.", Usage: []string{"upload <local_path> [remote_path]"}, Details: []string{"Uses binbag HTTP when available and falls back to chunked transfer otherwise."}},
	{Topic: "download", Category: "session", Summary: "Download a file from the selected session.", Usage: []string{"download <remote_path> [local_path]"}, Details: []string{"Uses the same shared transfer helpers as upload."}},
	{Topic: "spawn", Category: "session", Summary: "Spawn a new reverse shell from the active session."},
	{Topic: "modules", Category: "modules", Summary: "List available modules."},
	{Topic: "run", Category: "modules", Summary: "Run a module or custom runner.", Usage: []string{"run <module> [args...]"}, Details: []string{"Spawns a separate worker session so the main shell stays free.", "The command output opens in a separate terminal window while the original session remains usable.", "Use `help run <subtopic>` for runner-specific behavior."}},
	{Topic: "run peas", Category: "modules", Summary: "Run LinPEAS privilege escalation scanner in memory.", Usage: []string{"run peas [args]"}, Details: []string{"Good first pass when you want broad Linux privesc coverage without leaving disk artifacts."}},
	{Topic: "run lse", Category: "modules", Summary: "Run Linux Smart Enumeration in memory.", Usage: []string{"run lse [args]"}, Details: []string{"Handy when you want a second enumeration pass with slightly different checks than LinPEAS."}},
	{Topic: "run loot", Category: "modules", Summary: "Run the ezpz post-exploitation script in memory.", Usage: []string{"run loot [args]"}, Details: []string{"Useful after you have shell access and want quick credential and artifact triage."}},
	{Topic: "run pspy", Category: "modules", Summary: "Run the pspy process monitor.", Usage: []string{"run pspy [args]"}, Details: []string{"It is intentionally time-limited so cleanup can fire when the worker exits."}},
	{Topic: "run linexp", Category: "modules", Summary: "Run Linux Exploit Suggester in memory.", Usage: []string{"run linexp [args]"}, Details: []string{"Use it when you want a lightweight second opinion after a broader scan."}},
	{Topic: "run sh", Category: "modules", Aliases: []string{"run bash"}, Summary: "Run arbitrary bash script.", Usage: []string{"run sh <url|file> [args]"}, Details: []string{"Accepts small automation scripts or one-off payload glue without touching disk on the target."}},
	{Topic: "run ps1", Category: "modules", Summary: "Run a PowerShell script in memory.", Usage: []string{"run ps1 <url|file> [args]"}, Details: []string{"Windows output may arrive in blocks instead of a true stream, so a quiet window at the start is normal."}},
	{Topic: "run dotnet", Category: "modules", Summary: "Run a .NET assembly in memory.", Usage: []string{"run dotnet <url|file> [args]"}, Details: []string{"Windows output may arrive in blocks instead of a true stream, so a quiet window at the start is normal."}},
	{Topic: "run elf", Category: "modules", Summary: "Run a Linux or Unix binary.", Usage: []string{"run elf <url|file> [args]"}, Details: []string{"Use this for native binaries that need cleanup after execution."}},
	{Topic: "run py", Category: "modules", Summary: "Run a Python script.", Usage: []string{"run py <url|file> [args]"}, Details: []string{"Useful for quick cross-platform automation when a script URL is easier than a local file."}},
	{Topic: "run winpeas", Category: "modules", Summary: "Run WinPEAS privilege escalation scanner in memory.", Usage: []string{"run winpeas [args]"}, Details: []string{"Windows output may arrive in blocks instead of a true stream, so a quiet window at the start is normal."}},
	{Topic: "run seatbelt", Category: "modules", Summary: "Run Seatbelt system enumeration in memory.", Usage: []string{"run seatbelt [args]"}, Details: []string{"Defaults to -group=all when no arguments are provided.", "Windows output may arrive in blocks instead of a true stream, so a quiet window at the start is normal."}},
	{Topic: "binbag", Category: "network", Summary: "Manage the local HTTP file-serving directory.", Usage: []string{"binbag ls", "binbag on", "binbag off", "binbag path <dir>", "binbag port <N>"}, Details: []string{"HTTP is the fast path; use it when you care about speed or repeated transfers.", "Much faster than the base64 chunking fallback."}},
	{Topic: "pivot", Category: "network", Summary: "Rewrite URLs and payloads through a pivot IP.", Usage: []string{"pivot <ip>", "pivot off"}, Details: []string{"Use this when ligolo, chisel, or another forwarder is already carrying your traffic.", "It updates generated URLs and payloads globally so you do not have to hand-edit them."}},
	{Topic: "config", Category: "program", Summary: "Show the current configuration."},
	{Topic: "clear", Category: "program", Summary: "Clear the terminal screen."},
	{Topic: "exit", Category: "program", Aliases: []string{"quit", "q"}, Summary: "Exit Flame."},
}

func LookupHelpTopic(parts []string) (HelpEntry, bool) {
	query := strings.TrimSpace(strings.Join(parts, " "))
	if query == "" {
		return HelpEntry{}, false
	}
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

func HelpTopics() []string {
	topics := make([]string, 0, len(helpEntries))
	for _, entry := range helpEntries {
		topics = append(topics, entry.Topic)
	}
	return topics
}

func HelpTopicsForModal() []string {
	topics := make([]string, 0, len(helpEntries))
	for _, entry := range helpEntries {
		topics = append(topics, entry.Topic)
	}
	return topics
}

func HelpCategoriesForModal() []string {
	return []string{"connect", "handler", "session", "modules", "network", "program"}
}

func HelpTopicsForCompletion() []string {
	topics := make([]string, 0, len(helpEntries))
	seen := make(map[string]struct{}, len(helpEntries))
	for _, entry := range helpEntries {
		if _, ok := seen[entry.Topic]; !ok {
			topics = append(topics, entry.Topic)
			seen[entry.Topic] = struct{}{}
		}
		for _, alias := range entry.Aliases {
			if _, ok := seen[alias]; ok {
				continue
			}
			topics = append(topics, alias)
			seen[alias] = struct{}{}
		}
	}
	sort.Strings(topics)
	return topics
}

func RenderGeneralHelp() string {
	lines := []string{
		ui.CommandHelp("connect"),
		ui.Command("rev                          - Show reverse shell payloads"),
		ui.Command("ssh user@host                - Connect via SSH and execute revshell"),
		"",
		ui.CommandHelp("handler"),
		ui.Command("sessions, list               - List active sessions"),
		ui.Command("use <id>                     - Select session with given ID"),
		ui.Command("kill <id>                    - Kill session with given ID"),
		"",
		ui.CommandHelp("session"),
		ui.Command("shell                        - Enter interactive shell"),
		ui.Command("upload <local> [remote]      - Upload file to remote system"),
		ui.Command("download <remote> [local]    - Download file from remote system"),
		ui.Command("spawn                        - Spawn new shell from active session"),
		"",
		ui.CommandHelp("modules"),
		ui.Command("modules                      - List available modules"),
		ui.Command("run <module> [args]          - Run a module"),
		"",
		ui.CommandHelp("network"),
		ui.Command("binbag                      - Fast HTTP file serving"),
		ui.Command("pivot                       - Rewrite payload URLs via a forwarder"),
		"",
		ui.CommandHelp("program"),
		ui.Command("config                       - Show current configuration"),
		ui.Command("help                         - Show this help"),
		ui.Command("clear                        - Clear screen"),
		ui.Command("exit, quit                   - Exit Flame"),
		"",
		ui.HelpFooter("Type 'help <command>' for details"),
	}
	return ui.BoxWithTitle(fmt.Sprintf("%s Available Commands", ui.SymbolGem), lines)
}

func RenderHelpTopic(parts []string) (string, bool) {
	entry, ok := LookupHelpTopic(parts)
	if !ok {
		return "", false
	}

	lines := []string{}
	if entry.Summary != "" {
		lines = append(lines, entry.Summary)
	}
	if len(entry.Usage) > 0 {
		lines = append(lines, "", ui.CommandHelp("usage"))
		for _, usage := range entry.Usage {
			lines = append(lines, ui.Command(usage))
		}
	}
	if len(entry.Details) > 0 {
		lines = append(lines, "", ui.CommandHelp("details"))
		for _, detail := range entry.Details {
			lines = append(lines, detail)
		}
	}
	if len(entry.Examples) > 0 {
		lines = append(lines, "", ui.CommandHelp("examples"))
		for _, example := range entry.Examples {
			lines = append(lines, ui.Command(example))
		}
	}
	return ui.BoxWithTitle(entry.Topic, lines), true
}
