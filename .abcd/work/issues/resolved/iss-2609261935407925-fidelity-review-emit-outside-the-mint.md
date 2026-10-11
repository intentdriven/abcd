---
schema_version: 1
id: "iss-2609261935407925"
slug: "fidelity-review-emit-outside-the-mint"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25: fix2-drainA1 sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
resolution: "The fidelity-review emit now reads the shipped intent, judges its marker and parks the OWED stub in one hold of the intent store lock, so a disposition or verdict landing first is kept."
impact: fix
resolved_by:
  commit: "a27d06e08"
---

The fidelity-review emit is not under the intent mint lock: emitAuditWith (internal/core/intent/audit.go), reached from the spec-close ship move, the bundle close, abcd intent audit and the audit drain, reads a shipped intent, parks the OWED stub and writes the record back as a read-modify-write outside withIntentMintLock. A condition disposition or verdict ingest landing on the same record between the emit's read and its write is erased, and both verbs exit 0.

## Grounds

- pursued: a condition disposition landed in the window before a re-emit survives the parked stub; a disposition erased by the emit's write would show it wrong
