# Linux `run py` Validation

Status: validated on Linux

## Goal

Validate `run py` against the current TUI worker-session model before changing Windows module runners.

## Checklist

- receive/select a Linux session in the TUI
- run `run py <url|binbag-file>`
- confirm worker session stays invisible
- confirm output opens in a new terminal window
- confirm module output reaches the local output file
- confirm worker cleanup happens after completion
- record whether `RunPythonInMemory` needs the same blocking refactor as Windows runners

## Baseline Attempt — 2026-04-04

Command used:

```text
run py /home/chsoares/Lab/tmp/flame_run_py_test.py foo bar
```

Observed behavior:

- spinner ran
- worker notification appeared in the viewport
- output terminal opened but stayed empty
- an error was later printed directly under the TUI, outside the viewport, breaking the layout

Observed error:

```text
Execution error: failed to send command: write tcp ... use of closed network connection
```

Root cause found in code:

1. `RunPythonInMemory()` returned immediately after spawning an internal goroutine, so `StartModule()` closed the worker connection too early.
2. `RunPythonInMemory()` used `UploadToPythonVariable()`, which sends raw Python assignment syntax to the remote shell. That is invalid for a normal Linux shell.
3. The delayed goroutine printed errors with `fmt.Println(...)`, which bypassed the TUI viewport and corrupted the layout.

Code changes made after the failed baseline:

- Linux `run py` now uses the blocking worker-session pattern with `ExecuteWithStreamingCtx`.
- Linux HTTP mode now pipes `curl` output directly into `python3`.
- Linux fallback now uses a bash variable plus `base64 -d | python3`, not the broken Python-variable uploader.
- The delayed goroutine path was removed for Linux `run py`, so this execution path no longer prints late errors directly to stdout.

## Next Retest

Re-run the same command after rebuilding:

```text
run py /home/chsoares/Lab/tmp/flame_run_py_test.py foo bar
```

Success criteria:

- output appears in the spawned terminal window
- worker stays invisible and cleans up after completion
- no error is printed outside the TUI viewport

## Retest Result — 2026-04-04

Retest outcome:

- output appeared correctly in the spawned terminal window
- arguments arrived correctly (`foo`, `bar`)
- worker session stayed invisible to the normal TUI flow
- TUI remained usable after execution
- the previous late stdout error did not reappear

Validated output sample:

```text
=== FLAME RUN PY TEST START ===
python=/usr/bin/python3
hostname=archlinux
cwd=/home/chsoares/Lab/tmp
args=["foo", "bar"]
tick=0
tick=1
tick=2
=== FLAME RUN PY TEST END ===
```

Conclusion:

- Linux `run py` now works in the worker-session model.
- The remaining `RunPythonInMemory` warning in the code/docs now applies to Windows refactor work, not the Linux path.
