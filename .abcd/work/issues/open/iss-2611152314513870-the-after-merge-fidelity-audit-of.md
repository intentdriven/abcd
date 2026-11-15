---
schema_version: 1
id: "iss-2611152314513870"
slug: "the-after-merge-fidelity-audit-of"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "abcd intent audit ingest, the after-merge fidelity audit of itd-2610301914350541 (receipt rcp-1c71e6f7520c)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2610301914350541-a-repository-sets-the-layout-limits-of.md"
remedy: "fix, then re-run the audit: make ac-1 INCONCLUSIVE, ac-5 NOT_MET, ac-6 INCONCLUSIVE, ac-15 NOT_MET, ac-21 INCONCLUSIVE, ac-23 NOT_MET of itd-2610301914350541 hold in the delivered code, then re-emit the request with `abcd intent audit itd-2610301914350541` and ingest the re-run's verdict; a verdict with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record"
related_intents: [itd-2610301914350541]
deferred_after: "v0.16.2"
deferral_reason: "Three of the six criteria (ac-1, ac-6, ac-21) need a person reading questions in a live session, which the unattended run cannot supply; ac-5 and ac-15 wait for the owner to rule on the spec's planned key sentence (dcn-2611152327210749); ac-23 is held by a test since 46bb49c47, so the re-run audit can clear it."
refines: [iss-2610241431221889, iss-2610241551259104, iss-2610241613191748]
---

The after-merge fidelity audit of itd-2610301914350541 judged a criterion unmet, so its check is still owed: ac-1 INCONCLUSIVE, ac-5 NOT_MET, ac-6 INCONCLUSIVE, ac-15 NOT_MET, ac-21 INCONCLUSIVE, ac-23 NOT_MET

The audit of itd-2610301914350541 (receipt rcp-1c71e6f7520c) judged ac-1 INCONCLUSIVE, ac-5 NOT_MET, ac-6 INCONCLUSIVE, ac-15 NOT_MET, ac-21 INCONCLUSIVE, ac-23 NOT_MET. The intent stays in shipped/ and its changelog entry stands; its Audit Notes carry an audit-owed flag naming these criteria until a re-run of the audit passes (ruling DQ1c, 2026-09-30).

Remedy: fix, then re-run the audit. `abcd intent audit itd-2610301914350541` rewrites the review request for the re-run, and `abcd intent audit ingest --verdict-json <path>` ingests its verdict: one with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record.
