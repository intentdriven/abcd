---
schema_version: 1
id: "iss-2610161815530723"
slug: "inside-a-bash-c-or-sh-c-script-find-with"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "review of fix/checkable-shell-forms (abcd-05, Opus and Sonnet pair, 2026-10-16)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Count find carrying -delete (and -exec rm) as a deleting command in the refusal's check, so its target is read the same way."
---

Inside a bash -c or sh -c script, find with -delete on a path a variable decides is not refused by shell-c-variable-delete, because find is not among the deleting commands the refusal reads.
