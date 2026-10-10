---
schema_version: 1
id: "iss-2610102021189565"
slug: "the-shell-guard-s-hook-refused-an-abcd-capture-command-as"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "abcd-60 drain run 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Treat an operand as a script only in a position that executes it (bash f, source f, ./f, an interpreter's file argument, a startup-file variable), never an option value or a positional argument to a non-shell program; add a regression test with abcd capture --found-at naming a .json path after cd ... &&."
---

The shell guard's hook refused an abcd capture command as script-unread because one of its option values, the --found-at value, named a JSON file under internal/core/rules/defaults. The guard treated that file as a script the command runs and reported that it 'runs .abcd/docs-lint.json inside a chain of scripts deeper than the guard reads'. Nothing was executed: the path was a plain string argument to abcd capture, preceded by a cd and joined with &&. The same capture went through with the path reworded as prose (2026-10-10, abcd-60 drain run).
