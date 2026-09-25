---
schema_version: 1
id: "iss-336"
slug: "checkgatelockstep-returns-clean-when-its-runbook-or-workflow"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/core/lint/lint.go"
resolution: "LoadConfig refuses an enabled gate_lockstep with a blank runbook or workflow, and an enabled surface_coverage with a blank registry."
impact: internal
resolved_by:
  commit: "7a848694"
---

checkGateLockstep returns clean when its runbook or workflow config path is blank while enabled:true, before the rule's own fail-closed guards — a defence-in-depth asymmetry against the rule's stated posture; functionally equivalent to enabled:false (no external arming path), so recorded not fixed

## Evidence

- `internal/core/lint/lint.go:1121-1124` -- checkGateLockstep returns nil,nil on a blank runbook or workflow before the rule's own fail-closed guards (`:1127`, `:1135-1136`, `:1164-1166`).

## Refuter verdict -- CONFIRMED, severity NITPICK (recorded, not fixed)

Reproduced empirically (blank workflow + enabled:true -> exit 0). But gate_lockstep has no external arming path (unlike receipt_gate, whose blank guards are load-bearing because its Commit is CI-supplied); the only actor who can blank a path is the one who can write enabled:false one field away, same file, same exit 0. Defence-in-depth consistency gap, not a bypass. Same family as #360/#361, both accepted low. Recording is the outcome this round; a durable fix is load-time severity validation plus an ArmGateLockstep, larger than an autonomous round should take on unprompted.

## Grounds

- pursued: such a config fails to load naming the blank key while a disabled rule with blank paths loads; an enabled rule with a blank input exiting 0 would show it wrong
