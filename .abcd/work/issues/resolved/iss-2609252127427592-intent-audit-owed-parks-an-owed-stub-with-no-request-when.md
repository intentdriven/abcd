---
schema_version: 1
id: "iss-2609252127427592"
slug: "intent-audit-owed-parks-an-owed-stub-with-no-request-when"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
resolution: "The request is written before the intent file, so a failed request write parks no OWED stub; the drain row keeps the reader's receipt state (an already-parked receipt stays OWED)."
impact: fix
resolved_by:
  commit: "add09fd4"
---

intent audit --owed parks an OWED stub with no request when the request write fails: emitAuditWith writes the intent file before the audit request, and NextOwedAudit moves on after the error, so an environment-shaped failure (the reviews directory unwritable, or a plain file) parks a stub on every markerless entry within the cap while each row still reads 'no receipt (one is minted on re-emit)'. The request should be written first (the updated content is already in memory), then the intent file, so a failed emit leaves the intent untouched and the row truthful.

## Grounds

- pursued: with the reviews directory a plain file, intent audit --owed --max 3 leaves every shipped intent byte-identical and names the error on each row (TestNextOwedAuditFailedRequestWriteLeavesEveryIntentUntouched, TestIntentAuditOwedFailedRequestWriteParksNoStub); a modified intent file after such a run would show it wrong.
