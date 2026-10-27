---
schema_version: 1
id: "iss-2610241613191748"
slug: "the-after-merge-fidelity-audit-of"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "abcd intent audit ingest, the after-merge fidelity audit of itd-2610111707085098 (receipt rcp-efcb50874c2d)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2610111707085098-the-status-line-should-show-which.md"
remedy: "fix, then re-run the audit: make ac-13 NOT_MET of itd-2610111707085098 hold in the delivered code, then re-emit the request with `abcd intent audit itd-2610111707085098` and ingest the re-run's verdict; a verdict with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record"
related_intents: [itd-2610111707085098]
refines: [iss-2610241427397332, iss-2610241431221889, iss-2610241551259104]
---

The after-merge fidelity audit of itd-2610111707085098 judged a criterion unmet, so its check is still owed: ac-13 NOT_MET

The audit of itd-2610111707085098 (receipt rcp-efcb50874c2d) judged ac-13 NOT_MET. The intent stays in shipped/ and its changelog entry stands; its Audit Notes carry an audit-owed flag naming these criteria until a re-run of the audit passes (ruling DQ1c, 2026-09-30).

Remedy: fix, then re-run the audit. `abcd intent audit itd-2610111707085098` rewrites the review request for the re-run, and `abcd intent audit ingest --verdict-json <path>` ingests its verdict: one with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record.
