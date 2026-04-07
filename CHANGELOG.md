# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project follows [Semantic Versioning](https://semver.org/).

## [0.9.0] - 2026-04-07

### Added

- First tagged public snapshot of Flame as a TUI-first reverse shell handler for CTF work.
- Multi-session listener and selection flow with interactive shell handoff.
- Built-in upload and download support with shared transfer plumbing.
- Built-in module registry for Linux, Windows, and generic runner modules.
- Payload generation for bash, PowerShell, C#, and PHP reverse shell flows.
- SSH-assisted session bootstrap with password and key-based modes.
- Persistent runtime configuration under `~/.flame/config.toml`.
- Bubble Tea TUI with output pane, help modal, clipboard support, status bar, and session-oriented interaction model.
- Session log persistence and app data management under `~/.flame/`.

### Changed

- Project direction is now explicitly TUI-first; legacy CLI-oriented behavior is no longer the architectural target.
- Release documentation now reflects the current command surface, module set, and configuration model.

### Fixed

- Release housekeeping for ignored artifacts, generated binaries, coverage output, and local environment files.

### Notes

- `0.9.0` is feature complete but not yet the final architectural baseline.
- The planned pre-`1.0` refactor and broader review pass live in `docs/superpowers/specs/` and `docs/superpowers/plans/`.
