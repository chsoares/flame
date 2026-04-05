# Windows Baseline Validation

Status: in progress

## Goal

Validate the TUI against a real Windows target before refactoring Windows module execution.

## Priority

PowerShell first. Capture `cmd` behavior too if available.

## Local Test Artifacts

- PowerShell script: `/home/chsoares/Lab/tmp/flame_windows_test.ps1`
- Upload/download probe file: `/home/chsoares/Lab/tmp/flame_upload_test.txt`
- Optional `cmd` probe: `/home/chsoares/Lab/tmp/flame_cmd_probe.bat`

## First Pass Commands

Use these in order during the baseline run.

### 1. Shell detection and UX

Attach to the Windows session and run:

```powershell
whoami
hostname
pwd
$PSVersionTable.PSVersion
```

Then validate detach/attach and `Ctrl+C` with:

```powershell
ping 127.0.0.1 -n 6
```

### 2. Upload / download

From flame:

```text
upload /home/chsoares/Lab/tmp/flame_upload_test.txt C:\Windows\Temp\flame_upload_test.txt
download C:\Windows\Temp\flame_upload_test.txt /home/chsoares/Lab/tmp/flame_upload_test_returned.txt
```

### 3. PowerShell module-style execution baseline

```text
run ps1 /home/chsoares/Lab/tmp/flame_windows_test.ps1 foo bar
```

### 4. Spawn baseline

```text
spawn
```

### 5. Optional `cmd` observation

If you can land a `cmd` shell, run:

```bat
whoami
hostname
cd
type C:\Windows\Temp\flame_upload_test.txt
```

## Checklist

### Session detection

- session appears correctly in the TUI
- platform detection resolves to `windows`
- `whoami@hostname` detection is stable

### Shell UX

- attach with `shell` or `F12`
- detach with `F12`
- prompt behavior
- command relay
- history behavior
- `Ctrl+C`
- CRLF / formatting issues

### File operations

- `upload`
- `download`
- cancellation behavior

### Spawn and workers

- `spawn`
- module worker invisibility
- worker cleanup

### `cmd` follow-up

- does raw `cmd` work well enough in the TUI?
- should we add an explicit future `!psupgrade` command?

## Result

Not tested yet.

## Baseline Notes — 2026-04-04

Initial Windows PowerShell shell test:

- session detection worked
- platform detection resolved to `windows`
- `whoami@hostname` identification worked
- first attach showed a duplicated PowerShell prompt
- shell commands were sent successfully, but the submitted command was not echoed into the output pane before the response, which felt wrong compared to Linux shell behavior

Fixes applied after this baseline slice:

- removed the forced extra newline on shell attach, which was duplicating the prompt
- added prompt tracking in the TUI for remote shell sessions

Follow-up findings from the first fix attempt:

- using the remote PowerShell prompt as the prompt of the local input bar was the wrong model
- it caused the bottom input bar to look like a remote terminal line instead of a local editor widget
- it also contributed to prompt duplication when a command was echoed locally after a remote prompt was already visible

Second fix applied:

- the input bar now keeps a local shell prompt instead of reusing the remote PowerShell prompt
- local shell-command echo now completes the trailing remote prompt already present in the viewport instead of prefixing a second prompt
- Windows attach sends a newline only when the session viewport is empty, to fetch an initial prompt without constantly duplicating it

Pending retest:

- PowerShell attach should show a single prompt
- `whoami` and similar commands should appear in the viewport as `prompt + command`, with output on the next line

## PowerShell Shell UX Retest — 2026-04-04

Validated after the prompt/echo fixes:

- attach shows an initial PowerShell prompt in the viewport
- submitted commands reuse the prompt already visible at the end of the viewport
- command echo now looks shell-native (`PS ...> whoami`)
- output appears on following lines
- a fresh PowerShell prompt appears after command completion
- `whoami`, `hostname`, `pwd`, and `ls C:\` all rendered correctly in the TUI

Current Windows baseline status:

- shell detection: working
- platform detection: working
- `whoami@hostname` identification: working
- PowerShell shell UX: working for basic command flow
- remaining baseline work: `run ps1`, `spawn`, optional `cmd`

## Ctrl+C / Streaming Findings — 2026-04-04

Test used:

```powershell
ping 127.0.0.1 -n 10
```

Observed behavior:

- `Ctrl+C` did not interrupt the running command
- `ping` output was not reflected incrementally in the viewport
- the output arrived in one block only after command completion
- a duplicated prompt appeared after the failed interrupt attempt

Root-cause hypothesis:

- the current Windows reverse-shell payload in `internal/payloads.go` is command-response oriented, not a true interactive terminal stream
- it reads one full command from the socket, executes it via `iex ... | Out-String`, then sends the aggregated output back with a fresh prompt
- because the payload is busy executing the current command, incoming `\x03` is not processed as an interrupt during execution
- because the payload buffers through `Out-String`, long-running command output arrives as a block instead of streaming line by line

Implications:

- current PowerShell support is good enough for short command/response interaction
- current PowerShell support is not yet a true interactive shell for streaming commands or reliable `Ctrl+C`
- this is an architectural payload limitation, not just a TUI rendering bug

Follow-up ideas to investigate:

- design a better Windows payload focused on interactive behavior
- reintroduce and test `rev csharp` / generated `shell.exe` in the TUI branch
- check whether the C# payload provides better streaming and/or interrupt behavior than the current PowerShell payload

## `rev csharp` Direction — 2026-04-04

Progress:

- `rev` no longer accepts custom IP/port override args
- `rev` now uses the active listener/pivot IP and current listener port only
- `rev csharp` can print a C# reverse shell source payload
- `rev csharp shell.exe` can compile a local executable when `mcs` is available

Next Windows streaming question:

- validate whether the revived C# payload can become the better worker payload for long-running Windows modules

Remaining Windows baseline work after pause:

- `upload` / `download`
- `run ps1`
- `spawn`
- optional `cmd` baseline

## `run ps1` First Attempt — 2026-04-04

Observed behavior:

- `run ps1 ~/Lab/tmp/flame_windows_test.ps1 foo bar` failed because `run` was not expanding `~`
- using the absolute path launched the output terminal but produced no script output

Root cause found in code:

- `RunPowerShellInMemory()` was still using the old async internal goroutine model
- `StartModule()` therefore considered the module complete too early and could close the worker before real execution happened
- `run` command handling was not expanding the first source argument for custom modules (`ps1`, `sh`, `elf`, `dotnet`, `py`)

Fixes applied:

- `RunPowerShellInMemory()` now uses the blocking worker-session model with `ExecuteWithStreamingCtx`
- PowerShell script execution now uses explicit scriptblock execution helpers for HTTP and base64 paths
- `run` now expands `~` on the source argument for custom module commands
- module output filtering now strips pure trailing PowerShell prompt lines from the saved output when possible

Pending retest:

```text
run ps1 ~/Lab/tmp/flame_windows_test.ps1 foo bar
```

Open question still worth checking after retest:

- whether tab completion for `run ps1` is now perceived correctly once the source-path handling is aligned with `upload`

## `run ps1` Retest — 2026-04-04

Validated after the fixes:

- `run ps1 ~/Lab/tmp/flame_windows_test.ps1 foo bar` worked
- `~` expansion worked
- tab completion worked in practice during retest
- script output appeared in the spawned terminal window

Minor note:

- the output included PowerShell prompt lines around the script output; a low-risk filter was added to strip pure trailing prompt lines from streamed module output

## `spawn` Baseline — 2026-04-04

Validated:

- `spawn` produced a new visible Windows session successfully

Minor issue found and fixed:

- invisible worker sessions used by modules were still consuming the global numeric session counter internally
- result: visible sessions could jump from `1` to `3` after module activity
- fix: worker sessions remain invisible and no longer consume visible `NumID` values

## Detached Windows Launcher Change — 2026-04-04

Problem found during retest:

- Windows `spawn` could create the next shell, but the originating shell stayed effectively hung because the payload launcher remained in the foreground until the new reverse shell died
- this also cast doubt on Windows worker-session creation, because workers were being launched through the same blocking PowerShell payload path

Design change applied:

- added a detached Windows payload launcher variant in `internal/payloads.go`
- Windows `spawn` now uses the detached launcher
- Windows worker-session creation for modules now also uses the detached launcher
- Linux/macOS spawn behavior remains unchanged

Pending retest:

- run `run ps1 ~/Lab/tmp/flame_windows_test.ps1`
- then `spawn`
- confirm the original Windows shell stays responsive after the new shell connects

## Detached Windows Launcher Retest — 2026-04-04

Validated after the launcher change:

- `run ps1 ~/Lab/tmp/flame_windows_test.ps1` still works
- Windows worker creation still works
- `spawn` still creates the new shell successfully
- the original Windows shell remains responsive after `spawn`

Conclusion:

- the detached PowerShell launcher is now the correct path for Windows worker creation and Windows `spawn`
- this fixes the practical usability issue where the original shell became useless after spawning a second shell
- `spawn` also works from shell context through bang mode (`!spawn`)

## Bang Mode Session Switch Safety — 2026-04-04

Validated:

- `!use <id>` from shell context now detaches safely before switching sessions
- this prevents the TUI from ending up attached to one shell while the manager selects another

Behavior:

- in shell context, `!use <id>` now returns to menu context first, then switches the selected session
- this matches the existing defensive behavior already used for `!kill` on the active attached session

## `run dotnet` Preparation — 2026-04-04

Code changes applied before retest:

- `RunDotNetInMemory()` now uses the same blocking worker-session model as the fixed `run py` and `run ps1` paths
- both HTTP and base64 .NET execution paths now go through explicit command builders and `ExecuteWithStreamingCtx`

Local test assembly prepared:

- source: `/home/chsoares/Lab/tmp/FlameDotNetTest.cs`
- assembly: `/home/chsoares/Lab/tmp/FlameDotNetTest.exe`

Suggested retest command:

```text
run dotnet ~/Lab/tmp/FlameDotNetTest.exe foo bar
```

Success criteria:

- worker connects and stays invisible
- output appears in the spawned terminal window
- args arrive correctly in the assembly output
- worker cleans up after completion

## `run dotnet` Retest — 2026-04-04

Validated:

- `run dotnet ~/Lab/tmp/FlameDotNetTest.exe foo bar` worked
- output appeared in the spawned terminal window
- assembly arguments arrived correctly
- worker-session flow behaved correctly from the operator point of view

Memory model note:

- on the victim, the .NET assembly is executed in-memory via `DownloadData` or `FromBase64String` + `Reflection.Assembly.Load`
- that means the assembly itself is not written to disk on the target in the `run dotnet` path
- locally, flame still keeps a copy of the assembly source file and writes a local output log file for the operator

Conclusion:

- `run dotnet` is now validated in the TUI branch
- this unblocks the next Windows module checks that depend on the .NET runner (`winpeas`, `seatbelt`)

## `run winpeas` Validation — 2026-04-04

Validated with caveat:

- `run winpeas` works on the Windows target
- the worker/session model keeps the main shell responsive while WinPEAS runs

Caveat:

- WinPEAS output is buffered and appears in a large batch after the command finishes instead of streaming live
- this matches the current known limitation of the Windows PowerShell payload path

## `run seatbelt` Validation — 2026-04-04

Validated with caveat:

- `run seatbelt` works on the Windows target
- running `run seatbelt` with no explicit args uses the module default `-group=all`

Caveat:

- like `winpeas`, Seatbelt output is buffered instead of streaming live for long execution

## `run winpeas` Validation — 2026-04-04

Validated with caveat:

- `run winpeas` works on the Windows target
- the worker/session model keeps the main shell responsive while WinPEAS runs

Caveat:

- WinPEAS output is buffered and appears in a large batch after the command finishes instead of streaming live
- this matches the current known limitation of the Windows PowerShell payload path

## `run seatbelt` Validation — 2026-04-04

Validated with caveat:

- `run seatbelt` works on the Windows target
- running `run seatbelt` with no explicit args uses the module default `-group=all`

Caveat:

- like `winpeas`, Seatbelt output is buffered instead of streaming live for long execution

## `run bin` / `run elf` Outcome — 2026-04-04

Result after multiple Windows attempts:

- native Windows executable execution remained unreliable in this TUI branch
- several command-building and transfer fixes improved diagnostics, but the runner still did not produce trustworthy output on real Windows testing
- the project decision is to de-scope this path for now instead of carrying a flaky cross-platform abstraction

Local native Windows test binary prepared during investigation:

- source: `/home/chsoares/Lab/tmp/flame_run_bin_test.c`
- binary: `/home/chsoares/Lab/tmp/flame_run_bin_test.exe`

Decision captured:

- `run elf` is the Linux/native Unix binary runner name going forward
- `run elf` is Linux/native Unix only for now
- on Windows, the runner should fail early with a clear message telling the operator to use `run dotnet` or `run ps1`
- native Windows `.exe` execution remains a future dedicated project if needed

## Transfer Baseline — 2026-04-04

Validated on Windows:

- upload with binbag/HTTP: working
- download with binbag/HTTP: working
- upload without binbag (base64 fallback): working for small files
- download without binbag (base64 fallback): working for small files
- larger non-binbag transfers appear to work but are slow; no full end-to-end large-file wait was recorded yet

Bugs found and fixed during transfer baseline:

- HTTP upload progress was showing the internal `tmp_*` binbag filename instead of the user-facing filename
- HTTP transfer wait path did not respect cancellation context

After fixes:

- upload spinner now shows the correct user-facing filename
- `Ctrl+C` now cancels binbag/HTTP transfers from the TUI as expected

Current transfer status summary:

- Windows HTTP transfers: working
- Windows base64 fallback for small files: working
- transfer cancel via `Ctrl+C`: working
- large-file fallback performance: still worth characterizing later

## TUI Output Hygiene Note — 2026-04-04

Two maintainability/correctness concerns were identified during Windows module testing:

1. Old CLI-oriented download/progress paths can still leak raw spinner/stdout behavior into the TUI if they bypass TUI callbacks.
2. Some UI strings, symbols, and styles are still easier to change in one place than others because parts of the codebase bypass shared helpers.

Action taken:

- internal URL downloads used by runners/modules now use a quiet download path instead of the CLI spinner/stdout path

Still worth auditing later:

- legacy CLI-era input/output flows that may conflict with the TUI model
- hardcoded UI strings/colors/symbols that should use shared helpers for consistency

## TUI Output Hygiene Note — 2026-04-04

Two maintainability/correctness concerns were identified during Windows module testing:

1. Old CLI-oriented download/progress paths can still leak raw spinner/stdout behavior into the TUI if they bypass TUI callbacks.
2. Some UI strings, symbols, and styles are still easier to change in one place than others because parts of the codebase bypass shared helpers.

Action taken:

- internal URL downloads used by runners/modules now use a quiet download path instead of the CLI spinner/stdout path

Still worth auditing later:

- legacy CLI-era input/output flows that may conflict with the TUI model
- hardcoded UI strings/colors/symbols that should use shared helpers for consistency

## Current Hypotheses Before Testing

- PowerShell shell attach/detach is the highest-value baseline because Windows modules depend on it.
- The current Windows module runners still use the old async model and may reproduce the same class of bug that Linux `run py` had before refactor.
- Windows upload via base64 may fail or mis-handle absolute `remotePath` because the current decode path joins `Resolve-Path '.'` with the provided remote path. Validate with a full `C:\...` path first so we know for sure.
