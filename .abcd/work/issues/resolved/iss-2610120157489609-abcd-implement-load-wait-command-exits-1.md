---
schema_version: 1
id: "iss-2610120157489609"
slug: "abcd-implement-load-wait-command-exits-1"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "review of the load waiter fix round"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/implement_load.go"
remedy: "exit 128+N, as a shell does, when the gated command is killed by signal N"
resolution: "runGated reads the command's wait status and exits 128 plus the signal number when a signal killed it (143 for SIGTERM), keeping 1 for any other non-positive code; the help, commands/implement.md and the CLI reference say so, and a unix test proves a SIGTERM-killed child makes the waiter exit 143"
impact: fix
resolved_by:
  commit: "10a0ae650"
---

abcd implement load --wait -- <command> exits 1 when the gated command is killed by a signal: runGated in internal/surface/cli/implement_load.go maps any non-positive exit code to 1. The help and commands/implement.md promise the waiter exits with the command's code, so a SIGTERM to the waiter leaves make preflight dead by SIGTERM and the waiter exits 1, which reads exactly like a preflight that failed a gate.
