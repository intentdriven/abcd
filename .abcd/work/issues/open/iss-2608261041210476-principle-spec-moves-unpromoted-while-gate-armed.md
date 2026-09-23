---
schema_version: 1
id: "iss-2608261041210476"
slug: "principle-spec-moves-unpromoted-while-gate-armed"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "bughunt-a/round-7"
found_at: ".abcd/development/principles/spec-moves-with-the-surface.md"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (governance act owed: promote spec-moves-with-the-surface to a discipline intent or record why deferred). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

The principle spec-moves-with-the-surface names a promotion trigger that is already armed, yet sits unpromoted in the not-yet-enforced layer. The principle says a record-lint cross-check (every entry under commands/ and skills/ resolves to a brief surface row) WOULD promote it to a discipline; that is exactly surface_coverage Direction B, armed at blocker (lint.go, record-lint.json), documented live. principles/README.md makes promotion unconditional once any mechanical gate lands (the file is then retired to a discipline-kind intent, as itd-79 personas did), but no discipline intent exists for this principle and the file was never retired. The conditional 'would promote' predates the gate and was left stale — the inverse of the phantom-gate family (it understates its own enforcement). RECORDED NOT FIXED this round: the correct remedy is a governance act (mint a discipline-kind intent and retire the principle file, per the itd-79 precedent), which needs the maintainer rather than an autonomous edit. Proposed: promote per principles/README, or if promotion is deliberately deferred, record why on the principle.