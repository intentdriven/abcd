---
schema_version: 1
id: "iss-2610170717501963"
slug: "preflight-runs-on-macos-only-so"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "abcd-a2 overnight run, three Linux-only CI failures"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gittest"
remedy: "Make tests assert abcd's own words rather than git's stderr (gittest.ShimGitNamingNoGitfileTarget now rewrites one platform wording), give every fixture commit an explicit identity, and add a preflight step that runs the unit tests under a Linux container where one is available, saying so plainly when it is not."
---

Preflight runs on macOS only, so behaviour that differs on Linux reaches CI first: on 2026-10-16 and 17 three such failures cost a merge-queue round each: a test expecting git's wording for a broken .git file (Linux git prints '(null)'), a test argument over Linux's 128 KiB limit, and test fixtures committing in a bare repository with no git identity (macOS falls back to user@host, Linux refuses).

## Checked at the default branch 2026-10-28

Half the remedy is in: internal/gittest/shim.go gives the git shim a mode that names no gitfile target. Still open: no preflight step runs the tests under Linux, so a Linux-only failure still first shows in CI.
