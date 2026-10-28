---
schema_version: 1
id: "iss-2610252224285242"
slug: "intent-audit-judges-struck-acceptance"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "gap audit of the owed fidelity reviews, 2026-10-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/audit.go"
remedy: "Have the audit request mark a bullet that is struck through and labelled STRUCK as not judged, keep its slot so ac-N stays positional, and have the ingest and owedCriteria skip it; then re-emit itd-2610030810350727's audit, which clears iss-2610241604049754."
---

intent audit judges struck acceptance criteria. countAcceptanceCriteria (internal/core/intent/audit.go:2029) counts every top-level bullet of the Acceptance Criteria section, so a bullet struck by a later decision, such as itd-2610030810350727's A6 struck by decision 11, keeps its slot and must get a verdict. Nothing delivered can decide a struck outcome, so its INCONCLUSIVE files an owed check that no re-run can clear (iss-2610241604049754).
