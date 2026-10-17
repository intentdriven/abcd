---
schema_version: 1
id: "iss-2610161815541336"
slug: "a-variable-inside-an-unquoted"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "review of fix/checkable-shell-forms (abcd-05, Opus and Sonnet pair, 2026-10-16)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Count parameter expansions in an unquoted here-document body as the script's variables when the body feeds a deleting command."
---

A variable inside an unquoted here-document within a bash -c or sh -c script is not counted when the shell-c-variable-delete refusal decides whether a variable chooses what the script deletes.
