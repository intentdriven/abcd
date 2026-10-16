---
schema_version: 1
id: "iss-2610161815523252"
slug: "a-find-command-whose-exec-runs-sh-c-or"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "review of fix/checkable-shell-forms (abcd-05, Opus and Sonnet pair, 2026-10-16)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Treat find -exec and -execdir as a launcher whose operand script is read like a -c string, so the refusal fires with its rewrite."
---

A find command whose -exec runs sh -c or bash -c with a delete a variable decides gets only the unrecognised-launcher warning, not the shell-c-variable-delete refusal, so the agent is not handed the rewrite and the host still stops to ask.
