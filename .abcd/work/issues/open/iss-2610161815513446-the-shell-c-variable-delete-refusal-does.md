---
schema_version: 1
id: "iss-2610161815513446"
slug: "the-shell-c-variable-delete-refusal-does"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "review of fix/checkable-shell-forms (abcd-05, Opus and Sonnet pair, 2026-10-16)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Read an eval operand inside a -c script with the same reading the -c string gets (payload.go, the ref.shellC branch), so the refusal and its rewrite reach it."
---

The shell-c-variable-delete refusal does not read an eval string inside a bash -c or sh -c script as a script of its own, so a delete whose target a variable decides, written through eval inside the -c string, passes the guard and the host still stops to ask a person before it runs.
