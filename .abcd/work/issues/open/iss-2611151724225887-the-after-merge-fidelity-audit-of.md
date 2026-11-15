---
schema_version: 1
id: "iss-2611151724225887"
slug: "the-after-merge-fidelity-audit-of"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "abcd intent audit ingest, the after-merge fidelity audit of itd-2610230756259215 (receipt rcp-a2d9216f1974)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2610230756259215-questions-still-break-abcd-s-asking.md"
remedy: "fix, then re-run the audit: make ac-3 INCONCLUSIVE, ac-6 INCONCLUSIVE, ac-7 INCONCLUSIVE, ac-39 NOT_MET, ac-48 NOT_MET of itd-2610230756259215 hold in the delivered code, then re-emit the request with `abcd intent audit itd-2610230756259215` and ingest the re-run's verdict; a verdict with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record"
related_intents: [itd-2610230756259215]
deferred_after: "v0.16.2"
deferral_reason: "Deferred one cycle (dcn-2611151724464060): ac-3, ac-39 and ac-48 need a host session whose own hook runs the lab binary, queued for the person (dcn-2611151605470998); ac-6 and ac-7 need a lab material holding several decisions or too many answers, run with those halves."
refines: [iss-2610241431221889, iss-2610241551259104, iss-2610241613191748]
---

The after-merge fidelity audit of itd-2610230756259215 judged a criterion unmet, so its check is still owed: ac-3 INCONCLUSIVE, ac-6 INCONCLUSIVE, ac-7 INCONCLUSIVE, ac-39 NOT_MET, ac-48 NOT_MET

The audit of itd-2610230756259215 (receipt rcp-a2d9216f1974) judged ac-3 INCONCLUSIVE, ac-6 INCONCLUSIVE, ac-7 INCONCLUSIVE, ac-39 NOT_MET, ac-48 NOT_MET. The intent stays in shipped/ and its changelog entry stands; its Audit Notes carry an audit-owed flag naming these criteria until a re-run of the audit passes (ruling DQ1c, 2026-09-30).

Remedy: fix, then re-run the audit. `abcd intent audit itd-2610230756259215` rewrites the review request for the re-run, and `abcd intent audit ingest --verdict-json <path>` ingests its verdict: one with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record.
