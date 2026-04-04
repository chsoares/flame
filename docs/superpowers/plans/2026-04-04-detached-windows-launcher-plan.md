# Detached Windows Launcher Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent Windows `spawn` and worker-session creation from hanging the originating shell by launching the PowerShell reverse shell in a detached process.

**Architecture:** Keep the existing reverse-shell script content, but add a detached Windows launcher path in the payload generator. Use that detached launcher specifically for Windows `spawn` and Windows worker creation, leaving Linux behavior unchanged. Validate with unit tests around payload selection and then real Windows retests.

**Tech Stack:** Go, existing payload generator in `internal/payloads.go`, session orchestration in `internal/session.go`, Go `_test.go` unit tests.

---

### Task 1: Add failing tests for detached launcher routing

**Files:**
- Modify: `internal/session_startmodule_test.go`
- Create/Modify: `internal/payloads_test.go`

- [ ] **Step 1: Write failing tests**
- [ ] **Step 2: Run `go test ./internal -run 'TestShouldUseWorkerForSpawn|Test.*Detached.*'` and verify they fail for missing functionality**
- [ ] **Step 3: Cover both payload generation and Windows routing decisions**

### Task 2: Implement detached Windows launcher

**Files:**
- Modify: `internal/payloads.go`
- Modify: `internal/session.go`

- [ ] **Step 1: Add a detached Windows payload generator variant**
- [ ] **Step 2: Keep the existing non-detached generator available for `rev` output and backward comparison if needed**
- [ ] **Step 3: Route Windows worker creation and Windows `spawn` to the detached variant**
- [ ] **Step 4: Keep Linux/macOS spawn behavior unchanged**

### Task 3: Verify and document

**Files:**
- Modify: `docs/testing/windows-baseline.md`
- Modify: `docs/current-status.md`

- [ ] **Step 1: Run targeted unit tests and `go build -o flame .`**
- [ ] **Step 2: Update handoff docs with the detached-launcher design and retest expectations**
- [ ] **Step 3: Leave the next manual retest steps explicit for Windows `run ps1` + `spawn`**
