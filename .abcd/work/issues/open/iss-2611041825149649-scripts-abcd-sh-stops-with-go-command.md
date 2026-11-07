---
schema_version: 1
id: "iss-2611041825149649"
slug: "scripts-abcd-sh-stops-with-go-command"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "planning itd-2611041815429986"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/abcd.sh"
remedy: "when go cannot be run, walk PATH for an entry named go that is a dead link or one this account cannot follow, and name it on stderr, because stdout carries --json; then refuse as today. Compose no path and run no copy found this way, which keeps to the rule of adr-2611041815445203 that a person says where a tool lives. Grounds: POSIX reports a command it cannot reach by exit 126 or 127 with no reason beyond the shell's own wording, and bash and zsh word this case differently, as seen here"
---

scripts/abcd.sh stopped with 'go: command not found' and 'cmd/abcd did not build' on 2026-11-04, on one machine, because the `go` link in that account's ~/.local/bin pointed into a home folder the account could not enter. Nothing had removed Go. bash reported the command as not found and zsh as permission denied, so the message pointed nowhere near the link, and tracing it took a person's session. The link has since been repointed on that machine, which fixes that account and not the message: every abcd invocation in a source checkout goes through this script, and pinned-toolchain.sh and make inherit the same failure.
