---
schema_version: 1
id: "iss-2610141706042163"
slug: "abcd-guard-check-help-mirrored-in-the"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "v0.14.1 release review, docs-currency-reviewer (Opus), content commit 74165cf97"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard.go"
remedy: "Qualify the clause as a parameter expansion of a variable the line does not itself assign, then run go generate ./internal/surface/cli so commands.md follows."
---

abcd guard check --help (mirrored in the generated docs/reference/cli/commands.md) says an allow still does not see a parameter expansion with no substitution ($VAR, ${VAR:-git}) wherever it stands, because the guard sees the variable, not its expansion; since the carried-value reading the guard does read a value the same line assigns and uses: P=pkill; $P make and X=rm-payload; sh -c "$X" both block. The help under-states the guard (it fails safe).
