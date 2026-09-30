---
schema_version: 1
id: "iss-2608261635558358"
slug: "audit-verdicts-propagate-to-spec-resolved-issues"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "abcd-ux-phase-session"
found_at: "intent audit / capture resolve seam"
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed A): Should a failed audit criterion reopen, or only annotate, the resolved issues that rode on it?"
remedy: "Waits on the M13 three-edges-of-resolution interview (reopen or annotate): if annotate, have intent audit ingest, on a NOT_MET or MET_WITH_CONCERNS criterion, name every resolved issue whose resolved_by points into the audited spec and file one new issue per rider citing the old disposition, per recurrence-is-signal; if reopen, move those issues back to open/ with a dated reopening section instead; either proven by an ingest test on a fixture spec carrying one riding issue."
---

Audit verdicts do not propagate to the resolved issues that rode the audited spec. An issue fixed inside a spec resolves at the fixing change (Resolves trailer, watched-fail test, RS001-RS003 provenance), which is correct and deliberate — but when the intent's audit later returns NOT_MET or MET_WITH_CONCERNS on a criterion, nothing walks back from the verdict to the resolved issues whose resolved_by points into that spec: they stay resolved while the audit says the delivery fell short. The design-consistent remedy is audit back-propagation, not a pending-verification state (a candidate queue is the forgotten post-merge step the close-with-the-act rule exists to avoid): the auditor's verdict names the resolved issues riding the failed criterion and files the recurrence treatment for each — a new issue citing the old disposition, per recurrence-is-signal — so a hollow fix is surfaced by the audit rather than waiting for the defect to bite. Kin: iss-2608241612007530 (a trailerless fix is invisible to the gate) and iss-2608250844259345 (resolution frontmatter unchecked against the body) — the three together are the closure loop's three unwatched edges.

## Remedy grounds (2026-09-29)

Why: the verdict is the only moment the tool holds both the failed criterion and the spec, so the walk-back belongs to ingest rather than to a later sweep. Rejected: a pending-verification state, which the record itself turns down as the forgotten post-merge step the close-with-the-act rule exists to avoid.
