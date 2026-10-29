---
schema_version: 1
id: "iss-2610241551259104"
slug: "the-after-merge-fidelity-audit-of"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "abcd intent audit ingest, the after-merge fidelity audit of itd-2610030720038073 (receipt rcp-d5bf5bbbdcef)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2610030720038073-abcd-keeps-its-home-folder-out-of.md"
remedy: "fix, then re-run the audit: make ac-6 NOT_MET of itd-2610030720038073 hold in the delivered code, then re-emit the request with `abcd intent audit itd-2610030720038073` and ingest the re-run's verdict; a verdict with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record"
related_intents: [itd-2610030720038073]
refines: [iss-2610241427397332, iss-2610241431221889, iss-2610241426406882]
resolution: "The measured receipt for ac-6 is taken in the real renamed store: 0.56% mean combined indexer CPU against 3.28% for the old shape (research note 2026-10-28-home-rename-receipt); the run-log half has no run log to read and is stated as such."
impact: internal
resolved_by:
  intent: "itd-2610030720038073"
---

The after-merge fidelity audit of itd-2610030720038073 judged a criterion unmet, so its check is still owed: ac-6 NOT_MET

The audit of itd-2610030720038073 (receipt rcp-d5bf5bbbdcef) judged ac-6 NOT_MET. The intent stays in shipped/ and its changelog entry stands; its Audit Notes carry an audit-owed flag naming these criteria until a re-run of the audit passes (ruling DQ1c, 2026-09-30).

Remedy: fix, then re-run the audit. `abcd intent audit itd-2610030720038073` rewrites the review request for the re-run, and `abcd intent audit ingest --verdict-json <path>` ingests its verdict: one with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record.
