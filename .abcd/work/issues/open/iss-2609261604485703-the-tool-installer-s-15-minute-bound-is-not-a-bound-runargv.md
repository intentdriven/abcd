---
schema_version: 1
id: "iss-2609261604485703"
slug: "the-tool-installer-s-15-minute-bound-is-not-a-bound-runargv"
severity: "major"
category: "bug"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review-itd63"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/tools/install.go"
---

The tool installer's 15-minute bound is not a bound: runArgv in internal/core/tools/install.go kills the step's own process group on timeout and then waits for cmd.Wait, which waits for the stdout/stderr copier; a descendant that re-groups (setsid/setpgid) survives the group kill, and if it holds the pipe the wait never returns. The comment claims the kill reaches everything the step started, and nothing in the code, the ahoy brief chapter or the command page names the re-group limit.
