---
schema_version: 1
id: "iss-2609261604485703"
slug: "the-tool-installer-s-15-minute-bound-is"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd63"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/install.go"
resolution: "runArgv sets cmd.WaitDelay, so the wait for output ends 10s after the step exits or is killed even when a re-grouped descendant holds the pipe; the re-group limit is named in the code, the ahoy brief chapter and the command page."
impact: fix
resolved_by:
  commit: "705b8216"
---

The tool installer's 15-minute bound is not a bound: runArgv in internal/core/tools/install.go kills the step's own process group on timeout and then waits for cmd.Wait, which waits for the stdout/stderr copier; a descendant that re-groups (setsid/setpgid) survives the group kill, and if it holds the pipe the wait never returns. The comment claims the kill reaches everything the step started, and nothing in the code, the ahoy brief chapter or the command page names the re-group limit.

## Grounds

- pursued: the install and verify calls return within their bound plus the grace whatever a descendant does with the pipe; a runArgv call that blocks past the bound with a Setpgid holder on the pipe would show it wrong
