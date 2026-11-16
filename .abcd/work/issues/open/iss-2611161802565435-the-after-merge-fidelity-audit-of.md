---
schema_version: 1
id: "iss-2611161802565435"
slug: "the-after-merge-fidelity-audit-of"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "abcd intent audit ingest, the after-merge fidelity audit of itd-2610301346085246 (receipt rcp-12272bfa9de1)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2610301346085246-an-agent-working-in-an-abcd-managed.md"
remedy: "fix, then re-run the audit: make ac-1 NOT_MET of itd-2610301346085246 hold in the delivered code, then re-emit the request with `abcd intent audit itd-2610301346085246` and ingest the re-run's verdict; a verdict with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record"
related_intents: [itd-2610301346085246]
refines: [iss-2610241431221889, iss-2610241551259104, iss-2610241613191748]
---

The after-merge fidelity audit of itd-2610301346085246 judged a criterion unmet, so its check is still owed: ac-1 NOT_MET

The audit of itd-2610301346085246 (receipt rcp-12272bfa9de1) judged ac-1 NOT_MET. The intent stays in shipped/ and its changelog entry stands; its Audit Notes carry an audit-owed flag naming these criteria until a re-run of the audit passes (ruling DQ1c, 2026-09-30).

Remedy: fix, then re-run the audit. `abcd intent audit itd-2610301346085246` rewrites the review request for the re-run, and `abcd intent audit ingest --verdict-json <path>` ingests its verdict: one with no criterion NOT_MET or INCONCLUSIVE clears the flag and resolves this record.
