---
schema_version: 1
id: "iss-2610142235000811"
slug: "the-package-runners-shell-string-flags"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security review of feat/guard-port-blocks for v0.15.0"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/match.go"
remedy: "Read the -c and --call value of npx, npm exec and pnpm dlx as a shell string, as the value of sh -c is, so a blocker inside it is judged; prove it with kill-port and git push --force inside each spelling at block, and drop the limit from commands/guard.md."
---

The package runners' shell-string flags are not read: npx -c '<cmd>', npm exec --call '<cmd>' and pnpm dlx -c '<cmd>' run their value as a shell command, and the guard passes it unread, so npx -c 'kill-port 3000' is allowed while npx kill-port 3000 blocks. commands/guard.md and the runner change name this as a limit.
