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
---

intent audit --owed parks an OWED stub with no request when the request write fails: emitAuditWith writes the intent file before the audit request, and NextOwedAudit moves on after the error, so an environment-shaped failure (the reviews directory unwritable, or a plain file) parks a stub on every markerless entry within the cap while each row still reads 'no receipt (one is minted on re-emit)'. The request should be written first (the updated content is already in memory), then the intent file, so a failed emit leaves the intent untouched and the row truthful.
