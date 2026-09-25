---
schema_version: 1
id: "iss-2609251640462464"
slug: "a-blocked-command-piped-as-text-into-a-bare-shell-at-the-top"
severity: "major"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/payload.go"
---

A blocked command piped as text into a bare shell at the top level is allowed: echo or printf of the command string piped into sh, or into bash -s, runs it, but pipesIntoInterpreter is consulted only inside execute-a-string payloads, and the top-level segments carry no record of which operator joined them, so the guard cannot tell a shell reading the pipe from one running a script file without tracking pipes. Found by review-guard finding 6 (pre-existing).
